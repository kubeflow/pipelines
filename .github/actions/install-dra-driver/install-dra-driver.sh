#!/usr/bin/env bash
# Install DRA Example Driver via Helm from a cloned repo.
# Env: DRIVER_VERSION, NAMESPACE

set -euo pipefail

echo "Cloning dra-example-driver at ${DRIVER_VERSION}..."
git clone --branch "${DRIVER_VERSION}" --depth 1 \
  https://github.com/kubernetes-sigs/dra-example-driver.git \
  /tmp/dra-example-driver

echo "Installing DRA Example Driver via Helm..."
helm upgrade -i --create-namespace \
  --namespace "${NAMESPACE}" \
  dra-example-driver \
  /tmp/dra-example-driver/deployments/helm/dra-example-driver \
  --wait \
  --timeout 2m

echo "Waiting for DRA driver DaemonSet rollout..."
kubectl rollout status daemonset/dra-example-driver-kubeletplugin \
  -n "${NAMESPACE}" --timeout=120s

echo "DRA Example Driver installed successfully."
kubectl get pods -n "${NAMESPACE}" -o wide
