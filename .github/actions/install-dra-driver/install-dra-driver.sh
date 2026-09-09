#!/usr/bin/env bash
# Install DRA Example Driver via Helm from a pinned commit.
# Env: NAMESPACE

set -euo pipefail

driver_commit="bb67e6bac6b045a80498017ad60a1aa6b23eb9ce" # v0.2.1; latest driver tag compatible with Kubernetes 1.34.
driver_dir="/tmp/dra-example-driver"
echo "Fetching dra-example-driver at ${driver_commit}..."
git init --quiet "${driver_dir}"
git -C "${driver_dir}" remote add origin \
  https://github.com/kubernetes-sigs/dra-example-driver.git
git -C "${driver_dir}" fetch --quiet --depth 1 origin "${driver_commit}"
git -C "${driver_dir}" checkout --quiet --detach FETCH_HEAD

actual_commit=$(git -C "${driver_dir}" rev-parse HEAD)
if [[ "${actual_commit}" != "${driver_commit}" ]]; then
  echo "ERROR: Expected DRA driver commit ${driver_commit}, got ${actual_commit}" >&2
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
