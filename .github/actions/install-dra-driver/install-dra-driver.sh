#!/usr/bin/env bash
# Install DRA Example Driver via Helm from a pinned commit.
# Env: DRIVER_COMMIT, NAMESPACE

set -euo pipefail

driver_dir="/tmp/dra-example-driver"
echo "Fetching dra-example-driver at ${DRIVER_COMMIT}..."
git init --quiet "${driver_dir}"
git -C "${driver_dir}" remote add origin \
  https://github.com/kubernetes-sigs/dra-example-driver.git
git -C "${driver_dir}" fetch --quiet --depth 1 origin "${DRIVER_COMMIT}"
git -C "${driver_dir}" checkout --quiet --detach FETCH_HEAD

actual_commit=$(git -C "${driver_dir}" rev-parse HEAD)
if [[ "${actual_commit}" != "${DRIVER_COMMIT}" ]]; then
  echo "ERROR: Expected DRA driver commit ${DRIVER_COMMIT}, got ${actual_commit}" >&2
  exit 1
fi

echo "Installing DRA Example Driver via Helm..."
helm upgrade -i --create-namespace \
  --namespace "${NAMESPACE}" \
  dra-example-driver \
  "${driver_dir}/deployments/helm/dra-example-driver" \
  --wait \
  --timeout 2m

echo "Waiting for DRA driver DaemonSet rollout..."
kubectl rollout status daemonset/dra-example-driver-kubeletplugin \
  -n "${NAMESPACE}" --timeout=120s

echo "DRA Example Driver installed successfully."
kubectl get pods -n "${NAMESPACE}" -o wide
