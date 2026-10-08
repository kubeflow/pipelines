#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.

# Disposable Kind fixture only; the original rejection/recreation lane is separate.
set -euo pipefail
adoption_phase=${1:?Specify source or target}
[[ "$adoption_phase" == source || "$adoption_phase" == target ]]
source .github/resources/scripts/readiness-schedules.sh library
check() { python3 "$helpers/live_adoption_check.py" "$@" --state "$state"; }
stop_deployment() {
  local name=$1 selector
  selector=$(kube -n kubeflow get "deployment/$name" -o json |
    jq -r '.spec.selector.matchLabels | to_entries | map(.key + "=" + .value) | join(",")')
  [[ -n "$selector" ]]
  kube -n kubeflow scale "deployment/$name" --replicas=0
  kube -n kubeflow wait --for=delete pod -l "$selector" --timeout=120s
}

if [[ "$adoption_phase" == source ]]; then
  # The preceding source step has populated and drained real 2.17.2 history.
  configure_controllers
  start_forward
  mint_token
  # Hold an actual API-persisted source submission, not a manufactured SQL row.
  # Argo is stopped until suspend is set, avoiding a race with trivial workloads.
  stop_deployment workflow-controller
  check active
  stop_deployment ml-pipeline-scheduledworkflow
  stop_deployment ml-pipeline-persistenceagent
  stop_forward
  stop_deployment ml-pipeline
  check snapshot
  # Avoid value/valueFrom conflicts when the candidate canonical manifest applies.
  for controller in ml-pipeline-scheduledworkflow ml-pipeline-persistenceagent; do
    kube -n kubeflow patch "deployment/$controller" --type=strategic -p \
      "{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"$controller\",\"env\":[{\"name\":\"NAMESPACE\",\"value\":null,\"valueFrom\":{\"fieldRef\":{\"fieldPath\":\"metadata.namespace\"}}}]}]}}}}"
  done
  # The normal image loader/deployer is reused, but it must never start writers
  # before adoption. This edits only the disposable CI checkout's overlay.
  python3 - <<'PY'
from pathlib import Path
import yaml
path = Path('.github/resources/manifests/standalone/default/kustomization.yaml')
manifest = yaml.safe_load(path.read_text())
manifest['replicas'] = [dict(name=name, count=0) for name in (
    'ml-pipeline', 'ml-pipeline-scheduledworkflow',
    'ml-pipeline-persistenceagent', 'workflow-controller')]
path.write_text(yaml.safe_dump(manifest, sort_keys=False))
PY
else
  # Configuration may change only while all writer replicas stay at zero.
  check stopped
  set_api_env enforce
  configure_controllers
  for attempt in first repeat; do
    check job --job-name "readiness-adopt-$attempt"
    check wait-job --job-name "readiness-adopt-$attempt"
    if [[ "$attempt" == first ]]; then check adopted; else check idempotent; fi
  done
  kube -n kubeflow scale deployment/ml-pipeline --replicas=1
  # Full policy cutover requires live Pods, so verify only after API resumes,
  # while all submitting controllers remain stopped.
  configure_api enforce
  start_forward
  mint_token
  for controller in ml-pipeline-persistenceagent workflow-controller ml-pipeline-scheduledworkflow; do
    kube -n kubeflow scale "deployment/$controller" --replicas=1
    kube -n kubeflow rollout status "deployment/$controller" --timeout=300s
  done
  # More than two 30-second intervals: the adopted active run must occupy the
  # stored max_concurrency=1 slot while two disabled schedules stay disabled.
  for observation in {1..15}; do
    check held
    sleep 5
  done
  active_name=$(jq -r .name "$state/active.json")
  kube -n "$namespace" patch "workflow/$active_name" --type=merge -p '{"spec":{"suspend":false}}'
  complete=false
  for observation in {1..90}; do
    if check completed; then complete=true; break; fi
    sleep 5
  done
  [[ "$complete" == true ]]
  # Stop new submissions, drain all created runs, then verify final inventory.
  stop_deployment ml-pipeline-scheduledworkflow
  fixture --phase disable
  drained=false
  for observation in {1..60}; do
    if check drained; then drained=true; break; fi
    sleep 5
  done
  [[ "$drained" == true ]]
  python3 - "$state" <<'PY'
from pathlib import Path
import sys
from provision_live_schedules import read_object, write_object
state = Path(sys.argv[1])
fixture = read_object(state / 'fixture/state.json')
# Completion validation above records preserved enablement before cleanup.
write_object(state / 'reports/adoption-complete.json', dict(
    scope='populated_legacy_adoption', outcome='passed',
    active_case='persisted_suspended_workflow', source_version='2.17.2',
    schedule_uids=[case['schedule_uid'] for case in fixture['schedules']]))
PY
fi
