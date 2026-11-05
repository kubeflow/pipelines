#!/usr/bin/env bash
# Fence the released API writers before either MLMD migration entry point runs.
set -euo pipefail

: "${KFP_MLMD_MYSQL_DSN:?set the KFP MySQL DSN for the drain checks}"
: "${KFP_MLMD_MAINTENANCE_URL:?set the public maintenance URL returning HTTP 503}"

namespace="${KFP_MLMD_NAMESPACE:-kubeflow}"
api_deployment="${KFP_MLMD_API_DEPLOYMENT:-ml-pipeline}"
recurring_deployment="${KFP_MLMD_RECURRING_DEPLOYMENT:-ml-pipeline-scheduledworkflow}"

response_file="$(mktemp)"
trap 'rm -f "$response_file"' EXIT
status="$(curl --silent --show-error --output "$response_file" --write-out '%{http_code}' "${KFP_MLMD_MAINTENANCE_URL%/}/apis/v2/runs")"
if [[ "$status" != "503" ]] || ! grep -qi 'maintenance' "$response_file"; then
  echo 'Public API maintenance gate is not returning HTTP 503 with a maintenance message; leave old writers running.' >&2
  exit 1
fi

wait_for_pods() {
  local selector="$1" deadline=$((SECONDS + 1200))
  while [[ -n "$(kubectl -n "$namespace" get pods -l "$selector" -o name)" ]]; do
    if (( SECONDS >= deadline )); then
      echo "Timed out waiting for pods matching $selector to terminate." >&2
      exit 1
    fi
    sleep 5
  done
}

kubectl -n "$namespace" scale "deployment/$recurring_deployment" --replicas=0
wait_for_pods "app=$recurring_deployment"

# Keep old API reporting available until all runs and Workflows are terminal.
drain_deadline=$((SECONDS + 3600))
while ! drain_result="$(mlmd-migrate --drain-check --namespace="$namespace" --deployment="$api_deployment" --recurring-deployment="$recurring_deployment" 2>&1)"; do
  case "$drain_result" in
    *'lack a stored terminal report'*|*'has no stored terminal report'*|*'is not terminal'*) ;;
    *) echo "$drain_result" >&2; exit 1 ;;
  esac
  if (( SECONDS >= drain_deadline )); then
    echo 'Timed out waiting for terminal run and Workflow reports; postpone migration.' >&2
    exit 1
  fi
  sleep 10
done
echo "$drain_result"

kubectl -n "$namespace" scale "deployment/$api_deployment" --replicas=0
wait_for_pods "app=$api_deployment"
mlmd-migrate --gate-only --namespace="$namespace" --deployment="$api_deployment" --recurring-deployment="$recurring_deployment"

echo 'Old API writers are fenced. Run the CLI or deploy the new API image; keep the public maintenance gate and recurring controller paused until migration completes.'
