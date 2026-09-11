#!/usr/bin/env bash
# Smoke test: verify DRA allocation works before deploying KFP.
# Creates a temporary namespace, deploys a ResourceClaimTemplate + Pod,
# waits for the pod to reach Ready, and verifies resourceClaimStatuses.
# Env: NAMESPACE (driver namespace, for diagnostics)

set -euo pipefail

SMOKE_NS="dra-smoke-test"

cleanup() {
  echo "Cleaning up smoke test namespace..."
  kubectl delete namespace "${SMOKE_NS}" --ignore-not-found --wait=false
}
trap cleanup EXIT

# Stage 1: Verify driver pods are running
echo "=== Stage 1: Verifying DRA driver pods ==="
if ! kubectl get pods -n "${NAMESPACE}" -o wide; then
  echo "ERROR: Cannot list pods in driver namespace ${NAMESPACE}" >&2
  exit 1
fi

running=$(kubectl get pods -n "${NAMESPACE}" --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
if (( running == 0 )); then
  echo "ERROR: No running pods in driver namespace ${NAMESPACE}" >&2
  kubectl describe pods -n "${NAMESPACE}" >&2
  exit 1
fi
echo "Driver has ${running} running pod(s)."

# Stage 2: Create ResourceClaimTemplate + Pod and wait for Ready
echo "=== Stage 2: Testing DRA pod scheduling ==="
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
kubectl create namespace "${SMOKE_NS}"
kubectl apply -n "${SMOKE_NS}" -f "${SCRIPT_DIR}/smoke-test-resources.yaml"

echo "Waiting for smoke test pod to become Ready..."
kubectl wait --for=condition=Ready pod/dra-smoke-pod \
  -n "${SMOKE_NS}" --timeout=120s

# Stage 3: Verify resourceClaimStatuses shows the expected claim name
echo "=== Stage 3: Verifying DRA allocation ==="
claim_name=$(kubectl get pod/dra-smoke-pod -n "${SMOKE_NS}" \
  -o jsonpath='{.status.resourceClaimStatuses[0].name}')

if [[ -z "${claim_name}" || "${claim_name}" == "null" ]]; then
  echo "ERROR: Pod has no resourceClaimStatuses — DRA allocation did not happen" >&2
  kubectl get pod/dra-smoke-pod -n "${SMOKE_NS}" -o yaml >&2
  exit 1
fi

if [[ "${claim_name}" != "gpu" ]]; then
  echo "ERROR: Expected claim name 'gpu', got '${claim_name}'" >&2
  kubectl get pod/dra-smoke-pod -n "${SMOKE_NS}" -o jsonpath='{.status.resourceClaimStatuses}' >&2
  exit 1
fi

resource_claim_name=$(kubectl get pod/dra-smoke-pod -n "${SMOKE_NS}" \
  -o jsonpath='{.status.resourceClaimStatuses[0].resourceClaimName}')

if [[ -z "${resource_claim_name}" || "${resource_claim_name}" == "null" ]]; then
  echo "ERROR: resourceClaimName not set — claim was not bound" >&2
  kubectl get pod/dra-smoke-pod -n "${SMOKE_NS}" -o yaml >&2
  exit 1
fi

echo "Claim '${claim_name}' bound to ResourceClaim '${resource_claim_name}'."
echo "DRA smoke test passed: pod scheduled and resource claim allocated."
