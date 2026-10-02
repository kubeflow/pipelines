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
[[ "$phase" == source || "$phase" == target ]]
[[ $(kubectl config current-context) == "$context" ]]
preflight
kube() { kubectl --context "$context" --request-timeout=30s "$@"; }
fixture() {
  python3 "$helpers/provision_live_schedules.py" --context "$context" \
    --allow-test-cluster-mutations --state-dir "$state/fixture" \
    --endpoint "$endpoint" --token-file "$state/token" "$@"
}
cleanup() {
  # Disable only schedules owned by this fixture. Never print token or raw API/log data.
  if [[ -f "$state/token" && -f "$state/fixture/state.json" ]]; then
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
  kube -n kubeflow set env deployment/ml-pipeline \
    MULTIUSER=true TOKEN_REVIEW_AUDIENCE=pipelines.kubeflow.org \
    KUBEFLOW_USERID_HEADER=kubeflow-userid KUBEFLOW_USERID_PREFIX= \
    DEFAULTPIPELINERUNNERSERVICEACCOUNT=pipeline-runner \
    ALLOWEDSERVICEACCOUNTS=readiness-granted,readiness-denied \
    COMPILED_PIPELINE_SPEC_PATCH='{}' \
    KFP_SECURITY_SERVICE_ACCOUNT_MODE="$mode" KFP_SECURITY_WORKFLOW_IDENTITY_MODE=enforce
  kube -n kubeflow rollout status deployment/ml-pipeline --timeout=300s
}
configure_controllers() {
  kube -n kubeflow set env deployment/ml-pipeline-scheduledworkflow MULTIUSER=true NAMESPACE="$namespace"
  kube -n kubeflow set env deployment/ml-pipeline-persistenceagent NAMESPACE="$namespace"
  if ! kube -n kubeflow get deployment/workflow-controller -o json |
      jq -e --arg flag "--managed-namespace=$namespace" '.spec.template.spec.containers[0].args | index($flag)' >/dev/null; then
    kube -n kubeflow patch deployment/workflow-controller --type=json \
      -p "[{\"op\":\"add\",\"path\":\"/spec/template/spec/containers/0/args/-\",\"value\":\"--managed-namespace=$namespace\"}]"
  fi
  for controller in ml-pipeline-scheduledworkflow ml-pipeline-persistenceagent workflow-controller; do
    kube -n kubeflow rollout status "deployment/$controller" --timeout=300s
  done
}
start_forward() {
  kube -n kubeflow port-forward service/ml-pipeline 8888:8888 >"$state/port-forward.log" 2>&1 &
  forward_pid=$!
  for attempt in {1..30}; do
    if curl --noproxy '*' --silent --fail --max-time 2 "$endpoint/apis/v2beta1/healthz" >/dev/null; then
      return
    fi
    kill -0 "$forward_pid"
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
  local require_success=$1
  python3 - "$state" "$require_success" <<'PYDRAIN'
import json
from pathlib import Path
import sys
import time
from kfp_http import Client
from live_schedule_check import list_runs
state = Path(sys.argv[1])
fixture = json.loads((state / 'fixture/state.json').read_text())
terminal = {'SUCCEEDED', 'FAILED', 'CANCELED', 'SKIPPED'}
try:
    deadline = time.monotonic() + 300
    while time.monotonic() < deadline:
        client = Client('http://127.0.0.1:8888', state / 'token')
        complete = True
        for case in fixture['schedules']:
            records = list_runs(client, fixture['namespace'], case['schedule_uid'])
            if any(run.get('state') not in terminal for run in records):
                complete = False
            if sys.argv[2] == 'source' and not any(run.get('state') == 'SUCCEEDED' for run in records):
                complete = False
        if complete:
            if sys.argv[2] == 'source':
                (state / 'reports/source-completion.json').write_text(json.dumps({
                    'source_version': '2.17.2', 'scope': 'fixture_run_completion',
                    'outcome': 'passed', 'all_scenarios_succeeded': True}))
            break
        time.sleep(5)
    else:
        raise ValueError('fixture_runs_not_drained')
except Exception:
    sys.exit('Fixture drain failed: disable schedules and establish terminal runs before continuing.')
PYDRAIN
}
observe() {
  local mode=$1
  fixture --phase enable
  python3 "$helpers/live_schedule_check.py" --context "$context" --namespace "$namespace" \
    --kfp-endpoint "$endpoint" --kfp-token-file "$state/token" \
    --expectations "$reports/$mode-baseline.json" --prediction-report "$reports/$mode-prediction.json" \
    --not-before "$(cat "$state/fixture/activation-start.txt")" --timeout-seconds 180 \
    >"$reports/$mode-observed.json"
  fixture --phase disable
  drain target
}

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
  # Establish source-version run creation for every fixture before predicting an upgrade.
  python3 - "$state" <<'PY'
import json
from pathlib import Path
import sys
import time
from kfp_http import Client
from live_schedule_check import field, list_runs, timestamp
state = Path(sys.argv[1])
fixture = json.loads((state / 'fixture/state.json').read_text())
start = timestamp(fixture['activation_start'])
seen = set()
try:
    deadline = time.monotonic() + 180
    while time.monotonic() < deadline:
        client = Client('http://127.0.0.1:8888', state / 'token')
        for case in fixture['schedules']:
            records = list_runs(client, fixture['namespace'], case['schedule_uid'])
            if any(timestamp(field(r, 'created_at', 'createdAt')) >= start and
                   field(r, 'service_account', 'serviceAccount') == case['service_account']
                   for r in records):
                seen.add(case['scenario'])
        if len(seen) == 3:
            break
        time.sleep(5)
    if len(seen) != 3:
        raise ValueError('source_schedule_firing_not_established')
except Exception:
    sys.exit('Source schedule-firing preparation failed; no upgrade will be attempted.')
(state / 'reports/source-observed.json').write_text(json.dumps({
    'source_version': '2.17.2', 'scope': 'run_creation_only',
    'scenarios': sorted(seen), 'outcome': 'passed'}))
PY
  fixture --phase disable
  drain source
  # Full cluster snapshot is intentional for this controlled RBAC-only fixture.
  kube get roles,rolebindings,clusterroles,clusterrolebindings --all-namespaces -o json >"$state/source-rbac.json"
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
    capture "$mode"
    cp "$reports/$mode-baseline.json" "$reports/source-$mode-baseline.json"
  done
  printf '{"source_version":"2.17.2","target_revision":"%s"}\n' "$revision" >"$reports/revisions.json"
else
  configure_api enforce
  configure_controllers
  kube -n kubeflow rollout status deployment/ml-pipeline-persistenceagent --timeout=300s
  start_forward
  mint_token
  observe enforce
  stop_forward
  configure_api audit
  start_forward
  # Use the source-generated audit prediction, with a fresh baseline after enforce.
  capture audit
  observe audit
fi
