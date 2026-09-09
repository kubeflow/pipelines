#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -u

NAMESPACE="kubeflow"
PROFILE_CONTROLLER="kubeflow-pipelines-profile-controller"
SEAWEEDFS_HOST="seaweedfs.kubeflow"
SEAWEEDFS_IAM_PORT="8111"
TIMEOUT_SECONDS="${SEAWEEDFS_IAM_WAIT_TIMEOUT_SECONDS:-120}"
INTERVAL_SECONDS="${SEAWEEDFS_IAM_WAIT_INTERVAL_SECONDS:-5}"
REQUIRED_CONSECUTIVE_SUCCESSES="${SEAWEEDFS_IAM_REQUIRED_CONSECUTIVE_SUCCESSES:-3}"
KUBECTL_REQUEST_TIMEOUT="${SEAWEEDFS_IAM_KUBECTL_REQUEST_TIMEOUT:-10s}"
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SEAWEEDFS_SERVICE_IP="-"
SEAWEEDFS_ENDPOINT_HOST="-"
SEAWEEDFS_ENDPOINT_PORT="-"

kubectl_bounded() {
  kubectl --request-timeout="$KUBECTL_REQUEST_TIMEOUT" "$@"
}

resolve_iam_paths() {
  local target_directory
  target_directory=$(mktemp -d)
  if kubectl_bounded -n "$NAMESPACE" get service seaweedfs -o json \
    >"$target_directory/service.json" 2>/dev/null \
    && kubectl_bounded -n "$NAMESPACE" get endpointslice \
      -l kubernetes.io/service-name=seaweedfs -o json \
      >"$target_directory/endpointslices.json" 2>/dev/null; then
    python3 "$SCRIPT_DIR/service_path_probe.py" resolve \
      --pair seaweedfs-iam \
      --service "$target_directory/service.json" \
      --endpoint-slices "$target_directory/endpointslices.json" \
      --port-name http-iam >"$target_directory/targets.tsv" 2>/dev/null || true
    SEAWEEDFS_SERVICE_IP=$(
      awk -F '\t' '$2 == "vip" {print $3; exit}' "$target_directory/targets.tsv"
    )
    SEAWEEDFS_ENDPOINT_HOST=$(
      awk -F '\t' '$2 == "endpoint" {print $3; exit}' "$target_directory/targets.tsv"
    )
    SEAWEEDFS_ENDPOINT_PORT=$(
      awk -F '\t' '$2 == "endpoint" {print $4; exit}' "$target_directory/targets.tsv"
    )
  fi
  SEAWEEDFS_SERVICE_IP="${SEAWEEDFS_SERVICE_IP:--}"
  SEAWEEDFS_ENDPOINT_HOST="${SEAWEEDFS_ENDPOINT_HOST:--}"
  SEAWEEDFS_ENDPOINT_PORT="${SEAWEEDFS_ENDPOINT_PORT:--}"
  rm -rf "$target_directory"
}

probe_iam_connection() {
  local timeout_milliseconds="$1"
  local output status
  output=$(kubectl_bounded -n "$NAMESPACE" exec "deploy/$PROFILE_CONTROLLER" \
    -c profile-controller -- python -c \
    'from concurrent.futures import ThreadPoolExecutor
import json
import socket
import sys
import time

timeout = int(sys.argv[6]) / 1000
targets = (
    ("service-dns", sys.argv[1], int(sys.argv[2])),
    ("service-vip", sys.argv[3], int(sys.argv[2])),
    ("endpoint", sys.argv[4], int(sys.argv[5]) if sys.argv[5] != "-" else 0),
)

def probe(target):
    label, host, port = target
    if host == "-" or not port:
        return label, {"result": "unavailable"}
    started = time.monotonic_ns()
    try:
        connection = socket.create_connection((host, port), timeout=timeout)
        connection.close()
        result = "success"
        error = None
    except socket.timeout:
        result = "timeout"
        error = "TimeoutError"
    except OSError as exception:
        result = "error"
        error = type(exception).__name__
    return label, {
        "result": result,
        "error": error,
        "latency_ms": round((time.monotonic_ns() - started) / 1_000_000, 3),
    }

with ThreadPoolExecutor(max_workers=3) as executor:
    results = dict(executor.map(probe, targets))
print(json.dumps(results, sort_keys=True))
sys.exit(0 if results["service-dns"]["result"] == "success" else 1)' \
    "$SEAWEEDFS_HOST" "$SEAWEEDFS_IAM_PORT" \
    "$SEAWEEDFS_SERVICE_IP" "$SEAWEEDFS_ENDPOINT_HOST" \
    "$SEAWEEDFS_ENDPOINT_PORT" "$timeout_milliseconds" 2>/dev/null) && status=0 || status=$?
  echo "SeaweedFS IAM path probe: ${output:-unavailable}"
  return "$status"
}

monotonic_milliseconds() {
  python3 -c 'import time; print(time.monotonic_ns() // 1_000_000)'
}

collect_failure_diagnostics() {
  echo "----- SeaweedFS Service -----"
  kubectl_bounded -n "$NAMESPACE" get service seaweedfs -o wide 2>&1 || true
  echo "----- SeaweedFS EndpointSlices -----"
  kubectl_bounded -n "$NAMESPACE" get endpointslice \
    -l kubernetes.io/service-name=seaweedfs -o wide 2>&1 || true
  echo "----- SeaweedFS pod state -----"
  kubectl_bounded -n "$NAMESPACE" get pods -l app=seaweedfs -o wide 2>&1 || true
  kubectl_bounded -n "$NAMESPACE" describe pods -l app=seaweedfs 2>&1 || true
  echo "----- profile controller state -----"
  kubectl_bounded -n "$NAMESPACE" get pods -l app=kubeflow-pipelines-profile-controller \
    -o wide 2>&1 || true
  kubectl_bounded -n "$NAMESPACE" logs "deploy/$PROFILE_CONTROLLER" \
    -c profile-controller --tail=100 2>&1 || true
  echo "----- metacontroller logs -----"
  kubectl_bounded -n "$NAMESPACE" logs statefulset/metacontroller \
    --all-containers=true --tail=100 2>&1 || true
  echo "----- recent kubeflow events -----"
  kubectl_bounded -n "$NAMESPACE" get events --sort-by=.metadata.creationTimestamp \
    2>&1 | tail -50 || true
}

if [[ "${1:-}" == "--diagnostics-only" ]]; then
  collect_failure_diagnostics
  exit 0
fi

echo "Waiting for SeaweedFS IAM at ${SEAWEEDFS_HOST}:${SEAWEEDFS_IAM_PORT} from the profile controller..."
resolve_iam_paths
echo "SeaweedFS IAM direct targets: Service VIP ${SEAWEEDFS_SERVICE_IP}:${SEAWEEDFS_IAM_PORT}; Endpoint ${SEAWEEDFS_ENDPOINT_HOST}:${SEAWEEDFS_ENDPOINT_PORT}"
start_milliseconds=$(monotonic_milliseconds)
deadline_milliseconds=$((start_milliseconds + TIMEOUT_SECONDS * 1000))
now_milliseconds=$start_milliseconds
# A single successful handshake did not predict that the immediately following
# Profile reconciliation could reach the same Service. Require a short stable
# streak while retaining the existing wall-clock deadline.
consecutive_successes=0
while (( now_milliseconds < deadline_milliseconds )); do
  remaining_milliseconds=$((deadline_milliseconds - now_milliseconds))
  probe_timeout_milliseconds=$remaining_milliseconds
  if (( probe_timeout_milliseconds > 5000 )); then
    probe_timeout_milliseconds=5000
  fi

  if probe_iam_connection "$probe_timeout_milliseconds"; then
    consecutive_successes=$((consecutive_successes + 1))
    now_milliseconds=$(monotonic_milliseconds)
    elapsed=$(((now_milliseconds - start_milliseconds) / 1000))
    if (( consecutive_successes >= REQUIRED_CONSECUTIVE_SUCCESSES )); then
      echo "SeaweedFS IAM is stable after ${elapsed}s (${consecutive_successes} consecutive successful probes)."
      exit 0
    fi
    echo "SeaweedFS IAM probe succeeded (${consecutive_successes}/${REQUIRED_CONSECUTIVE_SUCCESSES}); confirming stability..."
  else
    consecutive_successes=0
  fi

  now_milliseconds=$(monotonic_milliseconds)
  elapsed=$(((now_milliseconds - start_milliseconds) / 1000))
  if (( now_milliseconds >= deadline_milliseconds )); then
    break
  fi
  if (( consecutive_successes == 0 )); then
    echo "Waiting for SeaweedFS IAM... (${elapsed}s/${TIMEOUT_SECONDS}s)"
  fi
  remaining_milliseconds=$((deadline_milliseconds - now_milliseconds))
  sleep_milliseconds=$((INTERVAL_SECONDS * 1000))
  if (( sleep_milliseconds > remaining_milliseconds )); then
    sleep_milliseconds=$remaining_milliseconds
  fi
  printf -v sleep_seconds '%d.%03d' \
    "$((sleep_milliseconds / 1000))" "$((sleep_milliseconds % 1000))"
  sleep "$sleep_seconds"
  now_milliseconds=$(monotonic_milliseconds)
done

now_milliseconds=$(monotonic_milliseconds)
elapsed=$(((now_milliseconds - start_milliseconds) / 1000))
echo "ERROR: SeaweedFS IAM at ${SEAWEEDFS_HOST}:${SEAWEEDFS_IAM_PORT} was not reachable after ${elapsed}s."
collect_failure_diagnostics
exit 1
