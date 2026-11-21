#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0

# Provision the disposable two-profile cluster and render the pipeline file
# that the browser uploads through the normal KFP Pipelines UI.
set -euo pipefail

api="${PROFILE_UI_URL:-http://127.0.0.1:3000}/apis/v2beta1"
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

if [[ ${1:-} == --restore-fixture-network ]]; then
  : "${PROFILE_FOREIGN_NAMESPACE:?set PROFILE_FOREIGN_NAMESPACE}"
  allow_profile_workloads "$PROFILE_FOREIGN_NAMESPACE"
  exit
fi

: "${PROFILE_PIPELINE_FILE:?set PROFILE_PIPELINE_FILE to the browser upload file}"
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


kubectl create rolebinding profile-catalog-producer \
  -n "$foreign_namespace" --clusterrole=kubeflow-pipelines-edit --user="$foreign"
kubectl -n "$foreign_namespace" wait --for=create serviceaccount/default-editor --timeout=120s
kubectl -n "$foreign_namespace" wait --for=create secret/mlpipeline-minio-artifact --timeout=300s
# The CI deployment's workload ingress policy lists only its original test profile.
# Give the second fixture's workload the same network access inside this disposable cluster.
allow_profile_workloads "$foreign_namespace"
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
printf '%s\n' "$spec" > "$PROFILE_PIPELINE_FILE"

{
  echo "owner_namespace=$namespace"
  echo "foreign_namespace=$foreign_namespace"
  echo "owner=$owner"
  echo "foreign=$foreign"
} >> "$GITHUB_OUTPUT"
