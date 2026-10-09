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
availability() { python3 "$helpers/live_adoption_availability.py" "$@" --state "$state"; }

if [[ "$adoption_phase" == source ]]; then
  # Keep all four controllers serving throughout the upgrade. Source cleanup
  # must not disable the enabled schedule when this preparation step exits.
  trap 'stop_forward; rm -f "$state/token"' EXIT
  configure_controllers
  start_forward
  mint_token
  check active
  check snapshot --online
  availability start
  # Preserve this isolated fixture's installation configuration on the ordinary
  # candidate apply. These patches change neither replicas nor rollout strategy.
  python3 - <<'PATCH'
from pathlib import Path
import yaml
path = Path('.github/resources/manifests/standalone/default/kustomization.yaml')
manifest = yaml.safe_load(path.read_text())
envs = {
    'MULTIUSER': 'true', 'TOKEN_REVIEW_AUDIENCE': 'pipelines.kubeflow.org',
    'KUBEFLOW_USERID_HEADER': 'kubeflow-userid', 'KUBEFLOW_USERID_PREFIX': '',
    'DEFAULTPIPELINERUNNERSERVICEACCOUNT': 'pipeline-runner',
    'ALLOWEDSERVICEACCOUNTS': 'readiness-granted,readiness-denied',
    'COMPILED_PIPELINE_SPEC_PATCH': '{}',
    'KFP_SECURITY_SERVICE_ACCOUNT_MODE': 'enforce',
    'KFP_SECURITY_WORKFLOW_IDENTITY_MODE': 'enforce',
}
for name, container, values in (
    ('ml-pipeline', 'ml-pipeline-api-server', envs),
    ('ml-pipeline-scheduledworkflow', 'ml-pipeline-scheduledworkflow',
     {'MULTIUSER': 'true', 'NAMESPACE': 'kfp-readiness-test'}),
    ('ml-pipeline-persistenceagent', 'ml-pipeline-persistenceagent',
     {'NAMESPACE': 'kfp-readiness-test'}),
):
    env = [dict(name=k, value=v, valueFrom=None) for k, v in values.items()]
    patch = dict(apiVersion='apps/v1', kind='Deployment', metadata=dict(name=name),
                 spec=dict(template=dict(spec=dict(containers=[dict(name=container, env=env)]))))
    manifest.setdefault('patches', []).append(dict(patch=yaml.safe_dump(patch)))
manifest['patches'].append(dict(
    target=dict(kind='Deployment', name='workflow-controller'),
    patch=yaml.safe_dump([dict(op='add', path='/spec/template/spec/containers/0/args/-',
                               value='--managed-namespace=kfp-readiness-test')])))
path.write_text(yaml.safe_dump(manifest, sort_keys=False))
PATCH
else
  start_forward
  mint_token
  adopted=false
  for observation in {1..60}; do
    if check adopted --online; then adopted=true; break; fi
    sleep 5
  done
  [[ "$adopted" == true ]]
  availability stop
  # More than two 30-second intervals: the adopted active run must occupy the
  # stored max_concurrency=1 slot while two disabled schedules stay disabled.
  for observation in {1..15}; do
    check held
    check idempotent --online
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
    upgrade_mode='ordinary_rolling_apply',
    schedule_uids=[case['schedule_uid'] for case in fixture['schedules']]))
PY
fi
