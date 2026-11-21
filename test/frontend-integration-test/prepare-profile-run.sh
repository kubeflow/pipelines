#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0

# Seed the disposable multi-user CI cluster through the same UI proxy the browser uses.
set -euo pipefail

api="${PROFILE_UI_URL:-http://127.0.0.1:3000}/apis/v2beta1"
spec=$(
  yq -o=json '.' test_data/sdk_compiled_pipelines/valid/critical/two_step_pipeline_containerized.yaml |
    jq --rawfile producer test/frontend-integration-test/profile-upgrade-output.py '
      .root.inputDefinitions.parameters.score = {parameterType: "NUMBER_INTEGER"}
      | .root.dag.tasks.component1.inputs.parameters.score = {componentInputParameter: "score"}
      | .components["comp-component1"].inputDefinitions.parameters.score = {parameterType: "NUMBER_INTEGER"}
      | .components["comp-component1"].outputDefinitions.artifacts.metrics = {
          artifactType: {schemaTitle: "system.Metrics", schemaVersion: "0.0.1"}
        }
      | .deploymentSpec.executors["exec-component1"].container |= (
          .image = "python:3.11"
          | .command = ["python3", "-c", $producer]
          | .args = [.args[0], "{{$.inputs.parameters[\u0027score\u0027]}}", .args[1], "{{$}}"]
        )
    '
)

prepare_run() {
  local namespace=$1 owner=$2 score=$3 run state attempt
  # A unique input forces a fresh producer execution even on cache-enabled clusters.
  local name="profile-run-details-$(date +%s)-$RANDOM"
  prepared_run_name=$name
  prepared_run_id=$(
    jq --arg namespace "$namespace" --arg name "$name" --argjson score "$score" \
        '{namespace: $namespace, display_name: $name, pipeline_spec: .,
          service_account: "default-editor", runtime_config: {parameters: {text: $name, score: $score}}}' <<< "$spec" |
      curl --fail-with-body -sS -H "kubeflow-userid: $owner" -H 'Content-Type: application/json' \
        --data-binary @- "$api/runs" |
      jq -er '.run_id | select(type == "string" and length > 0)'
  )

  for attempt in {1..120}; do
    run=$(curl --fail-with-body -sS -H "kubeflow-userid: $owner" "$api/runs/$prepared_run_id")
    state=$(jq -er '.state // "RUNTIME_STATE_UNSPECIFIED"' <<<"$run")
    case "$state" in
      SUCCEEDED) break ;;
      FAILED|CANCELED|CANCELING|SKIPPED|PAUSED)
        echo "Fixture run $prepared_run_id in $namespace reached $state" >&2
        kubectl -n "$namespace" get workflows -l "pipeline/runid=$prepared_run_id" -o json |
          jq '[.items[].status.nodes[]? | select(.phase == "Failed" or .phase == "Error") |
            {displayName, phase, message}]' >&2
        exit 1 ;;
    esac
    if [[ $attempt == 120 ]]; then
      echo "Fixture run $prepared_run_id timed out in $state" >&2
      exit 1
    fi
    sleep 5
  done
}

allow_profile_workloads() {
  local profile_namespace=$1
  local selector='.spec.ingress[0].from[0].namespaceSelector.matchExpressions[0].values'
  if ! kubectl -n kubeflow get networkpolicy allow-user-namespace-access -o json |
    jq -e --arg namespace "$profile_namespace" "$selector | index(\$namespace) != null" >/dev/null; then
    kubectl -n kubeflow patch networkpolicy allow-user-namespace-access --type=json \
      -p "$(jq -cn --arg ns "$profile_namespace" '[{op:"add",
        path:"/spec/ingress/0/from/0/namespaceSelector/matchExpressions/0/values/-", value:$ns}]')"
  fi
}

if [[ ${1:-} == --verify-writes ]]; then
  : "${PROFILE_OWNER_NAMESPACE:?set PROFILE_OWNER_NAMESPACE}"
  : "${PROFILE_FOREIGN_NAMESPACE:?set PROFILE_FOREIGN_NAMESPACE}"
  : "${PROFILE_OWNER:?set PROFILE_OWNER}"
  : "${PROFILE_FOREIGN:?set PROFILE_FOREIGN}"
  allow_profile_workloads "$PROFILE_FOREIGN_NAMESPACE"
  prepare_run "$PROFILE_OWNER_NAMESPACE" "$PROFILE_OWNER" 3131
  echo "Owner write succeeded on the candidate: $prepared_run_id"
  prepare_run "$PROFILE_FOREIGN_NAMESPACE" "$PROFILE_FOREIGN" 3232
  echo "Second profile write succeeded on the candidate: $prepared_run_id"
  exit
fi

: "${GITHUB_OUTPUT:?set GITHUB_OUTPUT to the fixture output file}"
namespace="${USER_NAMESPACE:-kubeflow-user-example-com}"
owner=$(kubectl get profile "$namespace" -o jsonpath='{.spec.owner.name}')
foreign='profile-run-reader@example.com'
foreign_namespace=$(kubectl create -f - -o jsonpath='{.metadata.name}' <<YAML
apiVersion: kubeflow.org/v1beta1
kind: Profile
metadata:
  generateName: profile-run-reader-
spec:
  owner:
    kind: User
    name: $foreign
YAML
)

# This CI overlay installs KFP's aggregate roles independently of the dashboard.
# Give each profile user only the permissions needed in their own namespace.
kubectl wait --for=create "namespace/$foreign_namespace" --timeout=120s
kubectl create rolebinding "profile-run-owner-${foreign_namespace#profile-run-reader-}" \
  -n "$namespace" --clusterrole=kubeflow-pipelines-edit --user="$owner"
kubectl create rolebinding profile-run-reader \
  -n "$foreign_namespace" --clusterrole=kubeflow-pipelines-view --user="$foreign"

# Establish a working second profile before checking access to the owner's run.
foreign_runs="$api/runs?namespace=$foreign_namespace&page_size=1"
for attempt in {1..60}; do
  status=$(curl -sS -o /dev/null -w '%{http_code}' -H "kubeflow-userid: $foreign" "$foreign_runs")
  case "$status" in
    200) break ;;
    403|404) ;;
    *) echo "Listing the second profile's runs returned HTTP $status" >&2; exit 1 ;;
  esac
  if [[ $attempt == 60 ]]; then
    echo "Timed out waiting for run-list access in $foreign_namespace (HTTP $status)" >&2
    exit 1
  fi
  sleep 3
done


prepare_run "$namespace" "$owner" 1717
run_id=$prepared_run_id
run_name=$prepared_run_name
prepare_run "$namespace" "$owner" 1212
comparison_run_id=$prepared_run_id
comparison_run_name=$prepared_run_name

status=$(curl -sS -o /dev/null -w '%{http_code}' -H "kubeflow-userid: $foreign" "$api/runs/$run_id")
if [[ $status != 403 && $status != 404 ]]; then
  echo "Expected the second profile to be denied run $run_id, got HTTP $status" >&2
  exit 1
fi

kubectl create rolebinding profile-catalog-producer \
  -n "$foreign_namespace" --clusterrole=kubeflow-pipelines-edit --user="$foreign"
kubectl -n "$foreign_namespace" wait --for=create serviceaccount/default-editor --timeout=120s
kubectl -n "$foreign_namespace" wait --for=create secret/mlpipeline-minio-artifact --timeout=300s
# The CI deployment's workload ingress policy lists only its original test profile.
# Give the second fixture's workload the same network access inside this disposable cluster.
allow_profile_workloads "$foreign_namespace"
prepare_run "$foreign_namespace" "$foreign" 2727
foreign_run_id=$prepared_run_id
foreign_run_name=$prepared_run_name
prepare_run "$foreign_namespace" "$foreign" 2323
foreign_comparison_run_id=$prepared_run_id
foreign_comparison_run_name=$prepared_run_name

{
  echo "run_id=$run_id"
  echo "run_name=$run_name"
  echo "comparison_run_id=$comparison_run_id"
  echo "comparison_run_name=$comparison_run_name"
  echo "foreign_run_id=$foreign_run_id"
  echo "foreign_run_name=$foreign_run_name"
  echo "foreign_comparison_run_id=$foreign_comparison_run_id"
  echo "foreign_comparison_run_name=$foreign_comparison_run_name"
  echo "owner_namespace=$namespace"
  echo "foreign_namespace=$foreign_namespace"
  echo "owner=$owner"
  echo "foreign=$foreign"
} >> "$GITHUB_OUTPUT"
