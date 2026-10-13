#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.

# Mutating CI fixture only. Never point this script at an operator installation.
set -euo pipefail
umask 077
phase=${1:?Specify preflight, source, or target}
context=kind-kfp-readiness
namespace=kfp-readiness-test
state=${RUNNER_TEMP:?RUNNER_TEMP is required}/readiness-schedules
reports=$state/reports
fixture_dir=$state/fixture
fixture_helper=provision_live_schedules.py
helpers=tools/upgrade-readiness
endpoint=http://127.0.0.1:8888
export PYTHONPATH="$PWD/$helpers${PYTHONPATH:+:$PYTHONPATH}"
mkdir -p "$reports"

preflight() {
  # The lane must fail, not silently exercise the older scheduling path.
  grep -q 'KFP_SECURITY_SERVICE_ACCOUNT_MODE' backend/src/apiserver/common/config.go &&
  grep -q 'KFP_SECURITY_WORKFLOW_IDENTITY_MODE' backend/src/apiserver/common/config.go &&
  grep -q 'authorizeServiceAccountWithPolicy' backend/src/apiserver/resource/resource_manager.go &&
  grep -q 'run.ServiceAccount = job.ServiceAccount' backend/src/apiserver/resource/recurring_run.go &&
  grep -q 'authorizeStoredRunServiceAccount' backend/src/apiserver/resource/recurring_run.go &&
  [[ $(git rev-parse HEAD) =~ ^[0-9a-f]{40}$ ]]
}

if [[ "$phase" == preflight ]]; then
  if ! preflight; then
    echo '::error::Integrate service-account audit and authorized scheduling policy prerequisites before enabling readiness schedule CI.'
    exit 1
  fi
  exit 0
fi
[[ "$phase" == source || "$phase" == target || "$phase" == library ]]
[[ $(kubectl config current-context) == "$context" ]]
preflight
kube() { kubectl --context "$context" --request-timeout=30s "$@"; }
fixture() {
  python3 "$helpers/$fixture_helper" --context "$context" \
    --allow-test-cluster-mutations --state-dir "$fixture_dir" \
    --endpoint "$endpoint" --token-file "$state/token" "$@"
}
cleanup() {
  # Disable only schedules owned by this fixture. Never print token or raw API/log data.
  if [[ -f "$state/token" && -f "$fixture_dir/state.json" ]]; then
    fixture --phase disable >/dev/null 2>&1 || true
  fi
  if [[ -n "${forward_pid:-}" ]]; then
    kill "$forward_pid" 2>/dev/null || true
    wait "$forward_pid" 2>/dev/null || true
  fi
  rm -f "$state/token"
}
trap cleanup EXIT

configure_api() {
  local mode=$1
  set_api_env "$mode"
  kube -n kubeflow rollout status deployment/ml-pipeline --timeout=300s
  # Rollout readiness excludes terminating replicas. Existing controller gRPC
  # connections can still reach an old audit process during its grace period.
  # Establish full policy cutover before enabling any fixture schedules.
  local deadline=$((SECONDS + 120)) pods
  while ((SECONDS < deadline)); do
    if pods=$(kube -n kubeflow get pods -l app=ml-pipeline -o json) &&
      jq -e --arg mode "$mode" '
        (.items | type == "array" and length > 0) and
        all(.items[];
          .kind == "Pod" and .metadata.namespace == "kubeflow" and
          (.metadata.uid | type == "string" and length > 0) and
          .metadata.deletionTimestamp == null and .status.phase == "Running" and
          ([.status.conditions[] | select(.type == "Ready")] |
            length == 1 and .[0].status == "True") and
          ([.spec.containers[] | select(.name == "ml-pipeline-api-server")] |
            length == 1 and (.[0].env |
              ([.[] | select(.name == "MULTIUSER")] |
                length == 1 and .[0].value == "true" and (.[0] | has("valueFrom") | not)) and
              ([.[] | select(.name == "KFP_SECURITY_SERVICE_ACCOUNT_MODE")] |
                length == 1 and .[0].value == $mode and (.[0] | has("valueFrom") | not)) and
              ([.[] | select(.name == "KFP_SECURITY_WORKFLOW_IDENTITY_MODE")] |
                length == 1 and .[0].value == "enforce" and (.[0] | has("valueFrom") | not)))))
      ' <<<"$pods" >/dev/null 2>&1; then
      api_policy_cutover_count=$(( ${api_policy_cutover_count:-0} + 1 ))
      jq --arg mode "$mode" '{scope: "isolated_api_policy_cutover", outcome: "passed",
        service_account_mode: $mode, workflow_identity_mode: "enforce",
        observed_at: (now | todateiso8601), pod_uids: ([.items[].metadata.uid] | sort)}' \
        <<<"$pods" >"$reports/api-policy-cutover-$phase-$api_policy_cutover_count-$mode.json"
      return
    fi
    sleep 2
  done
  echo '::error::API policy cutover could not exclude stale, terminating, or unready Pods.'
  return 1
}
set_api_env() {
  local mode=$1
  kube -n kubeflow set env deployment/ml-pipeline \
    MULTIUSER=true TOKEN_REVIEW_AUDIENCE=pipelines.kubeflow.org \
    KUBEFLOW_USERID_HEADER=kubeflow-userid KUBEFLOW_USERID_PREFIX= \
    DEFAULTPIPELINERUNNERSERVICEACCOUNT=pipeline-runner \
    ALLOWEDSERVICEACCOUNTS=readiness-granted,readiness-denied \
    COMPILED_PIPELINE_SPEC_PATCH='{}' \
    KFP_SECURITY_SERVICE_ACCOUNT_MODE="$mode" KFP_SECURITY_WORKFLOW_IDENTITY_MODE=enforce
}
configure_controllers() {
  kube -n kubeflow set env deployment/ml-pipeline-scheduledworkflow MULTIUSER=true NAMESPACE="$namespace"
  kube -n kubeflow set env deployment/ml-pipeline-persistenceagent NAMESPACE="$namespace"
  local patch
  patch=$(kube -n kubeflow get deployment/workflow-controller -o json |
    jq -c --arg namespace "$namespace" '
      .spec.template.spec.containers[0].args as $old |
      reduce (($old // [])[]) as $arg ({args: [], skip: false};
        if .skip then .skip = false
        elif $arg == "--managed-namespace" then .skip = true
        elif ($arg | startswith("--managed-namespace=")) then .
        else .args += [$arg] end) |
      .args += ["--managed-namespace=" + $namespace] |
      if .args == $old then [] else
        [{op: "add", path: "/spec/template/spec/containers/0/args", value: .args}]
      end')
  if [[ "$patch" != '[]' ]]; then
    kube -n kubeflow patch deployment/workflow-controller --type=json -p "$patch"
  fi
  for controller in ml-pipeline-scheduledworkflow ml-pipeline-persistenceagent workflow-controller; do
    kube -n kubeflow rollout status "deployment/$controller" --timeout=300s
  done
}
restore_controller_namespaces() {
  # Restore the canonical downward-API shape before kubectl apply. A literal
  # value added with set env otherwise survives alongside candidate valueFrom.
  local controller patch
  for controller in ml-pipeline-scheduledworkflow ml-pipeline-persistenceagent; do
    patch=$(jq -cn --arg name "$controller" '{spec:{template:{spec:{containers:[{name:$name,env:[{name:"NAMESPACE",value:null,valueFrom:{fieldRef:{fieldPath:"metadata.namespace"}}}]}]}}}}')
    kube -n kubeflow patch "deployment/$controller" --type=strategic -p "$patch"
    kube -n kubeflow rollout status "deployment/$controller" --timeout=300s
    kube -n kubeflow get "deployment/$controller" -o json |
      jq -e --arg name "$controller" '[.spec.template.spec.containers[] | select(.name == $name) | .env[] | select(.name == "NAMESPACE")] | length == 1 and (.[0] | (has("value") | not) and .valueFrom.fieldRef.fieldPath == "metadata.namespace")' >/dev/null
  done
}
start_forward() {
  # exec makes forward_pid the actual listener, so stop_forward cannot leave
  # an orphan behind when an API rollout replaces the selected Pod.
  (exec kubectl --context "$context" --request-timeout=30s -n kubeflow \
    port-forward --address=127.0.0.1 service/ml-pipeline 8888:8888) >"$state/port-forward.log" 2>&1 &
  forward_pid=$!
  for attempt in {1..30}; do
    kill -0 "$forward_pid" 2>/dev/null || return 1
    # A healthy unrelated/old listener is not evidence that this child bound.
    if grep -q '^Forwarding from 127\.0\.0\.1:8888 -> ' "$state/port-forward.log" &&
        curl --noproxy '*' --silent --fail --max-time 2 "$endpoint/apis/v2beta1/healthz" >/dev/null; then
      kill -0 "$forward_pid" 2>/dev/null || return 1
      return
    fi
    sleep 2
  done
  echo '::error::The isolated API port-forward did not become ready.'
  return 1
}
stop_forward() {
  kill "$forward_pid" 2>/dev/null || true
  wait "$forward_pid" 2>/dev/null || true
  forward_pid=
}
mint_token() {
  kube -n "$namespace" create token fixture-owner --audience=pipelines.kubeflow.org --duration=30m >"$state/token"
}
capture() {
  local mode=$1
  python3 "$helpers/capture_live_schedule_baseline.py" --context "$context" --namespace "$namespace" \
    --kfp-endpoint "$endpoint" --kfp-token-file "$state/token" \
    --cases "$state/$mode-cases.json" --prediction-report "$reports/$mode-prediction.json" \
    >"$reports/$mode-baseline.json"
}
drain() {
  local mode=$1
  python3 - "$state" "$mode" "$fixture_dir" <<'PYDRAIN'
import json
from pathlib import Path
import sys
import time
from kfp_http import Client
from live_schedule_check import FAILED_STATES, run_evidence, timestamp
from source_schedule_check import source_run_evidence, diagnostics
state = Path(sys.argv[1])
fixture_dir = Path(sys.argv[3]) if len(sys.argv) > 3 else state / 'fixture'
fixture = json.loads((fixture_dir / 'state.json').read_text())
mode = sys.argv[2]
start = timestamp((fixture_dir / 'activation-start.txt').read_text().strip())
if mode == 'source':
    cases = [dict(case, baseline_run_ids=[], expected_outcome='run_succeeded')
             for case in fixture['schedules']]
else:
    cases = json.loads((state / f'reports/{mode}-baseline.json').read_text())['cases']
observed = json.loads((state / f'reports/{mode}-observed.json').read_text())
known_ids = {case['schedule_uid']: {run['run_id'] for run in case['runs']}
             for case in observed['cases']}
last_evidence = {}
failure_reason = 'collection_failed'
try:
    # Persistence retries back off for up to 360 seconds. Allow that existing
    # delay plus execution/report grace; do not reset either controller.
    deadline = time.monotonic() + 600
    while time.monotonic() < deadline:
        client = Client('http://127.0.0.1:8888', state / 'token')
        complete = True
        evidence = []
        for case in cases:
            collect = source_run_evidence if mode == 'source' else run_evidence
            records = collect(client, fixture['namespace'], case, start)
            known_ids.setdefault(case['schedule_uid'], set()).update(
                run['run_id'] for run in records)
            record = {'scenario': case['scenario'], 'schedule_uid': case['schedule_uid'],
                      'service_account': case['service_account'], 'runs': records}
            evidence.append(record)
            last_evidence[case['scenario']] = record
            if case['expected_outcome'] == 'blocked':
                if records:
                    failure_reason = 'blocked_schedule_created_run'
                    raise ValueError(failure_reason)
            elif any(run['state'] in FAILED_STATES for run in records):
                failure_reason = 'fixture_run_did_not_succeed'
                raise ValueError(failure_reason)
            elif (not records or any(run['state'] != 'SUCCEEDED' for run in records) or
                  not known_ids.get(case['schedule_uid'], set()).issubset(
                      {run['run_id'] for run in records})):
                complete = False
        if complete:
            (state / f'reports/{mode}-completion.json').write_text(json.dumps({
                'scope': 'fixture_run_completion', 'mode': mode, 'outcome': 'passed',
                'namespace': fixture['namespace'], 'observation_start': start.isoformat(),
                'all_expected_runs_succeeded': True, 'cases': evidence}))
            break
        time.sleep(5)
    else:
        failure_reason = 'fixture_runs_not_drained'
        raise ValueError(failure_reason)
except Exception:
    (state / f'reports/{mode}-completion.json').write_text(json.dumps({
        'scope': 'fixture_run_completion', 'mode': mode, 'outcome': 'inconclusive',
        'reason': failure_reason, 'evidence_scope': 'last_successful_collection_per_case',
        'cases': list(last_evidence.values()),
        'known_run_ids': {uid: sorted(ids) for uid, ids in known_ids.items()},
        'diagnostics': diagnostics()}))
    sys.exit('Fixture completion failed: disable schedules and establish successful expected runs before continuing.')
PYDRAIN
}
observe() {
  local mode=$1 timeout=180
  # Denials leave up to 360 seconds of controller retry backoff. Observe the
  # live policy transition with that delay plus execution grace, without reset.
  [[ "$mode" == enforce || "$mode" == v1 ]] || timeout=600
  mint_token
  fixture --phase enable
  python3 "$helpers/schedule_diagnostics.py" --context "$context" --namespace "$namespace" \
    --cases "$reports/$mode-baseline.json" --wait-enabled >"$reports/$mode-activation-scheduler.json"
  if ! python3 "$helpers/live_schedule_check.py" --context "$context" --namespace "$namespace" \
    --kfp-endpoint "$endpoint" --kfp-token-file "$state/token" \
    --expectations "$reports/$mode-baseline.json" --prediction-report "$reports/$mode-prediction.json" \
    --not-before "$(cat "$fixture_dir/activation-start.txt")" --timeout-seconds "$timeout" --require-run-success \
    >"$reports/$mode-observed.json"; then
    python3 "$helpers/capture_run_diagnostics.py" --context "$context" --namespace "$namespace" \
      --baseline "$reports/$mode-baseline.json" --observed "$reports/$mode-observed.json" \
      --activation-start-file "$fixture_dir/activation-start.txt" --endpoint "$endpoint" \
      --token-file "$state/token" >"$reports/$mode-failed-runs.json" || true
    python3 "$helpers/schedule_diagnostics.py" --context "$context" --namespace "$namespace" \
      --cases "$reports/$mode-baseline.json" >"$reports/$mode-failed-scheduler.json" || true
    return 1
  fi
  fixture --phase disable
  if ! drain "$mode"; then
    python3 "$helpers/capture_run_diagnostics.py" --context "$context" --namespace "$namespace" \
      --baseline "$reports/$mode-baseline.json" --observed "$reports/$mode-observed.json" \
      --completion "$reports/$mode-completion.json" \
      --activation-start-file "$fixture_dir/activation-start.txt" --endpoint "$endpoint" \
      --token-file "$state/token" >"$reports/$mode-failed-runs.json" || true
    python3 "$helpers/schedule_diagnostics.py" --context "$context" --namespace "$namespace" \
      --cases "$reports/$mode-baseline.json" >"$reports/$mode-failed-scheduler.json" || true
    return 1
  fi
}

# Shared primitives for the separate adoption fixture; no source/target actions.
[[ "$phase" != library ]] || return 0

if [[ "$phase" == source ]]; then
  kube apply -k 'https://github.com/kubeflow/pipelines/manifests/kustomize/cluster-scoped-resources?ref=2.17.2'
  kube apply -k 'https://github.com/kubeflow/pipelines/manifests/kustomize/env/platform-agnostic?ref=2.17.2'
  kube -n kubeflow rollout status deployment/ml-pipeline --timeout=600s
  configure_api enforce
  kube -n kubeflow rollout status deployment/ml-pipeline-persistenceagent --timeout=300s
  start_forward
  fixture --phase rbac
  configure_controllers
  mint_token
  python3 - "$state/pipeline.json" <<'PY'
import json
import sys
import yaml
with open('test_data/sdk_compiled_pipelines/valid/hello_world.yaml') as stream:
    pipeline = yaml.safe_load(stream)
# The acceptance test observes schedule firing, not caching behavior.
for task in pipeline['root']['dag']['tasks'].values():
    task['cachingOptions'] = {'enableCache': False}
with open(sys.argv[1], 'w') as stream:
    json.dump(pipeline, stream)
PY
  fixture --phase prepare --pipeline-spec "$state/pipeline.json"
  fixture --phase enable
  # Source 2.17.2 persistence omits API service_account for embedded workflows.
  python3 "$helpers/source_schedule_check.py" --state-dir "$state/fixture" \
    --endpoint "$endpoint" --token-file "$state/token" \
    --output "$reports/source-observed.json"
  fixture --phase disable
  drain source
  # Full cluster snapshot is intentional for this controlled RBAC-only fixture.
  kube get roles,rolebindings,clusterroles,clusterrolebindings --all-namespaces -o json >"$state/source-rbac.json"
  # This isolated fixture enables multi-user mode on the standalone overlay.
  # Explicitly include its handoff grant; ordinary standalone installs omit it.
  python3 - <<'HANDOFF'
from pathlib import Path
import yaml
path = Path('.github/resources/manifests/standalone/default/kustomization.yaml')
manifest = yaml.safe_load(path.read_text())
resource = '../../../../../manifests/kustomize/base/installs/multi-user/managed-schedule-writers.yaml'
if resource not in manifest.setdefault('resources', []):
    manifest['resources'].append(resource)
manifest['namespace'] = 'kubeflow'
path.write_text(yaml.safe_dump(manifest, sort_keys=False))
HANDOFF
  kube kustomize --load-restrictor LoadRestrictionsNone .github/resources/manifests/standalone/default >"$state/candidate.yaml"
  kube kustomize manifests/kustomize/cluster-scoped-resources >"$state/candidate-cluster.yaml"
  python3 - "$state" <<'PY'
import json
from pathlib import Path
import sys
import yaml
state = Path(sys.argv[1])
items = []
for name in ('candidate.yaml', 'candidate-cluster.yaml'):
    with (state / name).open() as stream:
        items.extend(o for o in yaml.safe_load_all(stream) if isinstance(o, dict) and
                     o.get('kind') in ('Role', 'RoleBinding', 'ClusterRole', 'ClusterRoleBinding'))
(state / 'candidate-rbac.json').write_text(json.dumps({'items': items}))
cases = json.loads((state / 'fixture/cases.json').read_text())
(state / 'enforce-cases.json').write_text(json.dumps(cases))
for case in cases['cases']:
    case['expected_outcome'] = 'run_created'
    if case['scenario'] == 'denied':
        case['expected_prediction'] = 'operational_impact'
(state / 'audit-cases.json').write_text(json.dumps(cases))
PY
  revision=$(git rev-parse HEAD)
  for mode in enforce audit; do
    python3 "$helpers/build_live_policy.py" --source-rbac "$state/source-rbac.json" \
      --candidate-rbac "$state/candidate-rbac.json" --target-revision "$revision" --mode "$mode" \
      >"$state/$mode-policy.json"
    result=0
    python3 "$helpers/readiness.py" --context "$context" --system-namespace kubeflow \
      --namespace "$namespace" --source-version 2.17.2 --include-schedules \
      --schedule-policy "$state/$mode-policy.json" --kfp-endpoint "$endpoint" \
      --kfp-token-file "$state/token" --format json >"$reports/$mode-prediction.json" || result=$?
    [[ "$result" == 2 ]] # Reports remain explicitly incomplete, even in this fixture.
    cp "$reports/$mode-prediction.json" "$reports/source-$mode-prediction.json"
  done
  python3 "$helpers/capture_live_schedule_baseline.py" --context "$context" --namespace "$namespace" \
    --kfp-endpoint "$endpoint" --kfp-token-file "$state/token" \
    --cases "$state/fixture/cases.json" --legacy-migration >"$reports/source-legacy-baseline.json"
  printf '{"source_version":"2.17.2","target_revision":"%s"}\n' "$revision" >"$reports/revisions.json"
  # Fixtures are disabled and source runs drained before controllers leave the
  # fixture namespace. Target configuration restores this scope after upgrade.
  python3 "$helpers/live_adoption_check.py" source-disabled --state "$state"
  restore_controller_namespaces
else
  configure_api enforce
  configure_controllers
  kube -n kubeflow rollout status deployment/ml-pipeline-persistenceagent --timeout=300s
  start_forward
  mint_token
  # Automatic adoption must finish on the original disabled source IDs before
  # account-policy observation enables them. No deletion or recreation occurs.
  adopted=false
  for observation in {1..60}; do
    if python3 "$helpers/live_adoption_check.py" adopted-disabled --state "$state"; then
      adopted=true
      break
    fi
    sleep 5
  done
  [[ "$adopted" == true ]]
  for mode in enforce audit; do
    python3 "$helpers/check_fixture_policy.py" --context "$context" \
      --fixture-state "$state/fixture/state.json" --policy "$state/$mode-policy.json" \
      --endpoint "$endpoint" --token-file "$state/token" >"$reports/$mode-prediction.json"
  done
  capture enforce
  observe enforce
  # Retained V1 API and raw Workflow path: runtime contract, not scanner output.
  fixture_dir=$state/v1-fixture
  fixture_helper=provision_v1_schedules.py
  mint_token
  fixture --phase prepare --parent-state "$state/fixture/state.json"
  cp "$fixture_dir/cases.json" "$state/v1-cases.json"
  cp "$fixture_dir/expectations.json" "$reports/v1-prediction.json"
  capture v1
  observe v1
  fixture --phase verify
  cp "$fixture_dir/workflow-evidence.json" "$reports/v1-workflow-evidence.json"
  fixture_dir=$state/fixture
  fixture_helper=provision_live_schedules.py
  stop_forward
  configure_api audit
  start_forward
  # Use the adopted-fixture target policy check, with a fresh baseline after enforce.
  capture audit
  observe audit
  python3 "$helpers/verify_live_audit.py" --context "$context" \
    --not-before "$(cat "$state/fixture/activation-start.txt")" \
    --completion-report "$reports/audit-completion.json" >"$reports/audit-emission.json"
  # All schedules are disabled and every audit run is terminal before enforcing.
  stop_forward
  configure_api enforce
  start_forward
  for transition in audit-enforce revoked restored; do
    mint_token
    python3 "$helpers/prepare_schedule_transition.py" --context "$context" \
      --state-dir "$state" --phase "$transition"
    python3 "$helpers/check_fixture_policy.py" --context "$context" \
      --fixture-state "$state/fixture/state.json" --policy "$state/$transition-policy.json" \
      --endpoint "$endpoint" --token-file "$state/token" >"$reports/$transition-prediction.json"
    capture "$transition"
    observe "$transition"
  done

fi
