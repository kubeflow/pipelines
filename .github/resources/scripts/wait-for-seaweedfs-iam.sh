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
  kubectl -n "$NAMESPACE" exec "deploy/$PROFILE_CONTROLLER" \
    -c profile-controller -- python -c \
    'import socket; connection = socket.create_connection(("seaweedfs.kubeflow", 8111), timeout=5); connection.close()' \
    >/dev/null 2>&1
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
elapsed=0
while (( elapsed < TIMEOUT_SECONDS )); do
  if probe_iam_connection; then
    echo "SeaweedFS IAM is reachable after ${elapsed}s."
    exit 0
  fi

  echo "Waiting for SeaweedFS IAM... (${elapsed}s/${TIMEOUT_SECONDS}s)"
  sleep "$INTERVAL_SECONDS"
  elapsed=$((elapsed + INTERVAL_SECONDS))
done

echo "ERROR: SeaweedFS IAM at ${SEAWEEDFS_HOST}:${SEAWEEDFS_IAM_PORT} was not reachable after ${TIMEOUT_SECONDS}s."
collect_failure_diagnostics
exit 1
