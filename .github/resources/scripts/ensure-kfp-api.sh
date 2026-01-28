#!/usr/bin/env bash

set -euo pipefail

NAMESPACE="${1:-kubeflow}"
SERVICE_NAME="${2:-ml-pipeline}"
API_PORT="${3:-8888}"
TLS_ENABLED="${4:-false}"
CA_CERT_PATH="${5:-}"

log() {
  echo "[ensure-kfp-api] $*"
}

if ! kubectl version --request-timeout=5s >/dev/null 2>&1; then
  log "Kubernetes API server is not reachable; skipping KFP API readiness check."
  exit 0
fi

if ! kubectl get deploy/"${SERVICE_NAME}" -n "${NAMESPACE}" >/dev/null 2>&1; then
  log "Deployment ${SERVICE_NAME} not found in namespace ${NAMESPACE}; skipping readiness check."
  exit 0
fi

log "Waiting for ${SERVICE_NAME} deployment to become ready in namespace ${NAMESPACE}..."
kubectl rollout status deploy/"${SERVICE_NAME}" -n "${NAMESPACE}" --timeout=5m

log "Starting port-forward for ${SERVICE_NAME} service on localhost:${API_PORT}..."
pkill -f "kubectl port-forward.*${SERVICE_NAME}" || true
kubectl port-forward -n "${NAMESPACE}" svc/"${SERVICE_NAME}" "${API_PORT}:${API_PORT}" >/tmp/"${SERVICE_NAME}"-port-forward.log 2>&1 &
KFP_PORT_FORWARD_PID=$!

if [[ -n "${GITHUB_ENV:-}" ]]; then
  echo "KFP_PORT_FORWARD_PID=${KFP_PORT_FORWARD_PID}" >> "${GITHUB_ENV}"
else
  export KFP_PORT_FORWARD_PID
fi

SCHEME="http"
TARGET_HOST="localhost"
declare -a CURL_OPTS=(-sf)

if [[ "${TLS_ENABLED}" == "true" ]]; then
  SCHEME="https"
  TARGET_HOST="${SERVICE_NAME}.${NAMESPACE}.svc.cluster.local"
  CURL_OPTS+=(--resolve "${TARGET_HOST}:${API_PORT}:127.0.0.1")
  if [[ -n "${CA_CERT_PATH}" ]]; then
    CURL_OPTS+=(--cacert "${CA_CERT_PATH}")
  else
    CURL_OPTS+=(-k)
  fi
fi

for attempt in {1..30}; do
  if curl "${CURL_OPTS[@]}" "${SCHEME}://${TARGET_HOST}:${API_PORT}/apis/v2beta1/healthz" >/dev/null; then
    log "KFP API server is ready."
    exit 0
  fi

  if [[ "${attempt}" -eq 30 ]]; then
    log "KFP API server did not become reachable within expected time."
    exit 1
  fi

  log "Waiting for KFP API server... (attempt ${attempt}/30)"
  sleep 5
done

exit 1
