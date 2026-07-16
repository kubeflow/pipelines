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

probe_iam_connection() {
  local timeout_milliseconds="$1"
  kubectl -n "$NAMESPACE" exec "deploy/$PROFILE_CONTROLLER" \
    -c profile-controller -- python -c \
    'import socket, sys
connection = socket.create_connection(
    (sys.argv[1], int(sys.argv[2])), timeout=int(sys.argv[3]) / 1000)
connection.close()' \
    "$SEAWEEDFS_HOST" "$SEAWEEDFS_IAM_PORT" "$timeout_milliseconds" \
    >/dev/null 2>&1
}

monotonic_milliseconds() {
  python3 -c 'import time; print(time.monotonic_ns() // 1_000_000)'
}

collect_failure_diagnostics() {
  echo "----- SeaweedFS Service -----"
  kubectl -n "$NAMESPACE" get service seaweedfs -o wide 2>&1 || true
  echo "----- SeaweedFS EndpointSlices -----"
  kubectl -n "$NAMESPACE" get endpointslice \
    -l kubernetes.io/service-name=seaweedfs -o wide 2>&1 || true
  echo "----- SeaweedFS pod state -----"
  kubectl -n "$NAMESPACE" get pods -l app=seaweedfs -o wide 2>&1 || true
  kubectl -n "$NAMESPACE" describe pods -l app=seaweedfs 2>&1 || true
  echo "----- profile controller state -----"
  kubectl -n "$NAMESPACE" get pods -l app=kubeflow-pipelines-profile-controller \
    -o wide 2>&1 || true
  kubectl -n "$NAMESPACE" logs "deploy/$PROFILE_CONTROLLER" \
    -c profile-controller --tail=100 2>&1 || true
  echo "----- metacontroller logs -----"
  kubectl -n "$NAMESPACE" logs statefulset/metacontroller \
    --all-containers=true --tail=100 2>&1 || true
  echo "----- recent kubeflow events -----"
  kubectl -n "$NAMESPACE" get events --sort-by=.metadata.creationTimestamp \
    2>&1 | tail -50 || true
}

if [[ "${1:-}" == "--diagnostics-only" ]]; then
  collect_failure_diagnostics
  exit 0
fi

echo "Waiting for SeaweedFS IAM at ${SEAWEEDFS_HOST}:${SEAWEEDFS_IAM_PORT} from the profile controller..."
start_milliseconds=$(monotonic_milliseconds)
deadline_milliseconds=$((start_milliseconds + TIMEOUT_SECONDS * 1000))
now_milliseconds=$start_milliseconds
while (( now_milliseconds < deadline_milliseconds )); do
  remaining_milliseconds=$((deadline_milliseconds - now_milliseconds))
  probe_timeout_milliseconds=$remaining_milliseconds
  if (( probe_timeout_milliseconds > 5000 )); then
    probe_timeout_milliseconds=5000
  fi

  if probe_iam_connection "$probe_timeout_milliseconds"; then
    now_milliseconds=$(monotonic_milliseconds)
    elapsed=$(((now_milliseconds - start_milliseconds) / 1000))
    echo "SeaweedFS IAM is reachable after ${elapsed}s."
    exit 0
  fi

  now_milliseconds=$(monotonic_milliseconds)
  elapsed=$(((now_milliseconds - start_milliseconds) / 1000))
  if (( now_milliseconds >= deadline_milliseconds )); then
    break
  fi
  echo "Waiting for SeaweedFS IAM... (${elapsed}s/${TIMEOUT_SECONDS}s)"
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
