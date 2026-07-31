#!/usr/bin/env bash
# Install Fake GPU Operator (fake backend) via Helm.
# Env: CHART_VERSION, NAMESPACE, NODE_POOL, GPU_COUNT, HELM_TIMEOUT

set -euo pipefail

helm repo add fake-gpu-operator \
  https://runai.jfrog.io/artifactory/api/helm/fake-gpu-operator-charts-prod \
  --force-update
helm repo update fake-gpu-operator

helm upgrade -i fake-gpu-operator fake-gpu-operator/fake-gpu-operator \
  --version "${CHART_VERSION}" \
  --namespace "${NAMESPACE}" \
  --create-namespace \
  --set "topology.nodePools.${NODE_POOL}.gpuCount=${GPU_COUNT}" \
  --wait \
  --timeout "${HELM_TIMEOUT}"
