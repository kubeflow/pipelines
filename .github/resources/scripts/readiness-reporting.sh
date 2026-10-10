#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.

# Fault injection in the owned disposable Kind fixture only.
set -euo pipefail
reporting_phase=${1:?Specify source, target, or cleanup}
[[ "$reporting_phase" == source || "$reporting_phase" == target || "$reporting_phase" == cleanup ]]
source .github/resources/scripts/readiness-schedules.sh library
reporting_state=$state/reporting
mkdir -p "$reporting_state"
check() {
  python3 "$helpers/live_reporting_recovery.py" "$@" \
    --fixture-state "$fixture_dir/state.json" --state-dir "$reporting_state"
}
restore_faults() {
  local restore_result=0
  if [[ -f "$fixture_dir/state.json" ]]; then check restore || restore_result=1; fi
  if [[ -f "$reporting_state/agent-restore.json" ]]; then
    kube -n kubeflow patch deployment/ml-pipeline-persistenceagent --type=json \
      --patch-file "$reporting_state/agent-restore.json" >/dev/null || restore_result=1
    kube -n kubeflow rollout status deployment/ml-pipeline-persistenceagent --timeout=180s || restore_result=1
  fi
  return "$restore_result"
}
reporting_cleanup() {
  local result=$?
  restore_faults || result=1
  if [[ -n "${proxy_forward_pid:-}" ]]; then
    kill "$proxy_forward_pid" 2>/dev/null || true
    wait "$proxy_forward_pid" 2>/dev/null || true
  fi
  cleanup
  exit "$result"
}
trap reporting_cleanup EXIT
if [[ "$reporting_phase" == cleanup ]]; then exit 0; fi

if [[ "$reporting_phase" == source ]]; then
  # Source preparation restores controller namespaces after its own drain.
  configure_controllers
  start_forward
  mint_token
  check prepare --endpoint "$endpoint" --token-file "$state/token"
  restore_controller_namespaces
  exit 0
fi

configure_api enforce
configure_controllers
start_forward
mint_token
check recover --endpoint "$endpoint" --token-file "$state/token"

# Capture terminal reports through the real persistence worker, then delete the
# exact Workflow UID before forwarding to the real API. No synthetic reports.
proxy_build=$reporting_state/proxy-build
mkdir -p "$proxy_build"
go test ./tools/upgrade-readiness/reporting-proxy
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$proxy_build/reporting-proxy" ./tools/upgrade-readiness/reporting-proxy
printf 'FROM scratch\nCOPY reporting-proxy /reporting-proxy\nUSER 65532:65532\nENTRYPOINT ["/reporting-proxy"]\n' >"$proxy_build/Dockerfile"
docker build -t kfp-reporting-proxy:fixture "$proxy_build"
kind load docker-image --name kfp-readiness kfp-reporting-proxy:fixture
python3 - "$reporting_state" <<'PY'
import json
from pathlib import Path
import sys
state = Path(sys.argv[1])
records = json.loads((state / 'reporting-source.json').read_text())['deletion_runs']
namespace, name = 'kfp-readiness-test', 'kfp-reporting-proxy'
labels = dict(app=name)
targets = [{k: r[k] for k in ('run_id', 'workflow_name', 'workflow_uid')} for r in records]
objects = [
    dict(apiVersion='v1', kind='ServiceAccount', metadata=dict(name=name, namespace='kubeflow')),
    dict(apiVersion='rbac.authorization.k8s.io/v1', kind='Role', metadata=dict(name=name, namespace=namespace),
         rules=[dict(apiGroups=['argoproj.io'], resources=['workflows'],
                     resourceNames=[r['workflow_name'] for r in records], verbs=['get', 'delete'])]),
    dict(apiVersion='rbac.authorization.k8s.io/v1', kind='RoleBinding', metadata=dict(name=name, namespace=namespace),
         subjects=[dict(kind='ServiceAccount', name=name, namespace='kubeflow')],
         roleRef=dict(apiGroup='rbac.authorization.k8s.io', kind='Role', name=name)),
    dict(apiVersion='apps/v1', kind='Deployment', metadata=dict(name=name, namespace='kubeflow'),
         spec=dict(replicas=1, selector=dict(matchLabels=labels), template=dict(metadata=dict(labels=labels),
         spec=dict(serviceAccountName=name, securityContext=dict(runAsNonRoot=True, seccompProfile=dict(type='RuntimeDefault')),
         containers=[dict(name=name, image='kfp-reporting-proxy:fixture', imagePullPolicy='Never',
              env=[dict(name='FIXTURE_NAMESPACE', value=namespace), dict(name='FIXTURE_TARGETS_JSON', value=json.dumps(targets))],
              securityContext=dict(allowPrivilegeEscalation=False, readOnlyRootFilesystem=True, capabilities=dict(drop=['ALL'])),
              resources=dict(requests=dict(cpu='25m', memory='32Mi'), limits=dict(cpu='500m', memory='128Mi')),
              readinessProbe=dict(httpGet=dict(path='/healthz', port=8888), initialDelaySeconds=2, periodSeconds=2))])))),
    dict(apiVersion='v1', kind='Service', metadata=dict(name=name, namespace='kubeflow'),
         spec=dict(selector=labels, ports=[dict(name='grpc', port=8887), dict(name='http', port=8888)])),
    dict(apiVersion='networking.k8s.io/v1', kind='NetworkPolicy', metadata=dict(name=name, namespace='kubeflow'),
         spec=dict(podSelector=dict(matchLabels=labels), policyTypes=['Ingress'],
                   ingress=[{'from': [dict(podSelector=dict(matchLabels=dict(app='ml-pipeline-persistenceagent')))],
                             'ports': [dict(protocol='TCP', port=8887), dict(protocol='TCP', port=8888)]}]))
]
(state / 'proxy-resources.json').write_text(json.dumps(dict(apiVersion='v1', kind='List', items=objects)))
PY
kube apply -f "$reporting_state/proxy-resources.json" >/dev/null
kube -n kubeflow rollout status deployment/kfp-reporting-proxy --timeout=180s
kube -n kubeflow get deployment/ml-pipeline-persistenceagent -o json >"$reporting_state/agent-before.json"
python3 - "$reporting_state" <<'PY'
import json
from pathlib import Path
import sys
state = Path(sys.argv[1])
d = json.loads((state / 'agent-before.json').read_text())
containers = d['spec']['template']['spec']['containers']
indices = [i for i, c in enumerate(containers) if c['name'] == 'ml-pipeline-persistenceagent']
assert len(indices) == 1
index = indices[0]
c = containers[index]
path = '/spec/template/spec/containers/' + str(index) + '/args'
args = c.get('args', [])
assert not any('mlPipelineAPIServerName' in a for a in args)
(state / 'agent-restore.json').write_text(json.dumps([dict(op='add', path=path, value=args)]))
(state / 'agent-proxy.json').write_text(json.dumps([dict(op='add', path=path, value=args + ['-mlPipelineAPIServerName=kfp-reporting-proxy'])]))
PY
kube -n kubeflow patch deployment/ml-pipeline-persistenceagent --type=json --patch-file "$reporting_state/agent-proxy.json" >/dev/null
kube -n kubeflow rollout status deployment/ml-pipeline-persistenceagent --timeout=180s
kubectl --context "$context" -n kubeflow port-forward service/kfp-reporting-proxy 18888:8888 --address 127.0.0.1 >"$reporting_state/proxy-forward.log" 2>&1 &
proxy_forward_pid=$!
for attempt in {1..30}; do
  if curl -fsS --max-time 2 http://127.0.0.1:18888/healthz >/dev/null; then break; fi
  sleep 1
done
curl -fsS --max-time 2 http://127.0.0.1:18888/healthz >/dev/null
mint_token
check delete --endpoint "$endpoint" --token-file "$state/token" --proxy-endpoint http://127.0.0.1:18888
