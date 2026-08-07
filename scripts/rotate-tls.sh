#!/usr/bin/env bash
# Usage: ./scripts/rotate-tls.sh <namespace> <secret-name> <cert-file> <key-file>
# Defaults: namespace=kubeflow, secret=kfp-pod-tls, cert=server.crt, key=server.key

NS="${1:-kubeflow}"
SECRET="${2:-kfp-api-tls-cert}"
CERT="${3:-server.crt}"
KEY="${4:-server.key}"

set -euo pipefail

if [[ ! -f "${CERT}" || ! -f "${KEY}" ]]; then
  echo "Cert or key file missing. Provide paths to cert and key."
  echo "Usage: $0 <namespace> <secret-name> <cert-file> <key-file>"
  exit 1
fi

echo "Applying TLS secret ${SECRET} in namespace ${NS} (dry-run -> apply)..."
kubectl create secret tls "${SECRET}" --cert="${CERT}" --key="${KEY}" --dry-run=client -o yaml | kubectl apply -f -

echo "Secret applied. Now finding deployments that reference ${SECRET} ..."
readarray -t DEPS < <(kubectl get deploy -n "${NS}" -o json | jq -r --arg SECRET "${SECRET}" '
  .items[] |
  select(
    (.spec.template.spec.volumes[]? | select(.secret and .secret.secretName == $SECRET)) or
    (.spec.template.spec.containers[]?.env[]? | select(.valueFrom and .valueFrom.secretKeyRef and .valueFrom.secretKeyRef.name == $SECRET))
  ) | .metadata.name
')

if [[ "${#DEPS[@]}" -eq 0 ]]; then
  echo "No deployments found referencing secret ${SECRET} in namespace ${NS}."
  echo "You may need to restart specific deployments manually."
  exit 0
fi

echo "Found deployments:"
for d in "${DEPS[@]}"; do
  echo "  - ${d}"
done

echo
for d in "${DEPS[@]}"; do
  echo "Rolling restart: ${d}"
  kubectl rollout restart "deploy/${d}" -n "${NS}"
  echo "Waiting for rollout of ${d}..."
  kubectl rollout status "deploy/${d}" -n "${NS}"
done

echo "All rollouts triggered. Verify with: kubectl get pods -n ${NS}"
