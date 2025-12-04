#!/usr/bin/env bash
# Usage: ./scripts/find-tls-refs.sh <namespace> <secret-name>
# Defaults: namespace=kubeflow, secret=kfp-pod-tls

NS="${1:-kubeflow}"
SECRET="${2:-kfp-pod-tls}"

set -euo pipefail

echo "Looking in cluster namespace: ${NS} for secret: ${SECRET}"
echo

echo "Pods that reference the secret via volumes or env:"
kubectl get pods -n "${NS}" -o json | jq -r --arg SECRET "${SECRET}" '
  .items[] |
  select(
    (.spec.volumes[]? | select(.secret and .secret.secretName == $SECRET)) or
    (.spec.containers[]?.env[]? | select(.valueFrom and .valueFrom.secretKeyRef and .valueFrom.secretKeyRef.name == $SECRET))
  ) | .metadata.name
' || true

echo
echo "Deployments referencing the secret:"
kubectl get deploy -n "${NS}" -o json | jq -r --arg SECRET "${SECRET}" '
  .items[] |
  select(
    (.spec.template.spec.volumes[]? | select(.secret and .secret.secretName == $SECRET)) or
    (.spec.template.spec.containers[]?.env[]? | select(.valueFrom and .valueFrom.secretKeyRef and .valueFrom.secretKeyRef.name == $SECRET))
  ) | .metadata.name
' || true
