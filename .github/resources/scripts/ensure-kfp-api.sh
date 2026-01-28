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

dump_ml_pipeline_logs() {
  if ! command -v kubectl >/dev/null 2>&1; then
    log "kubectl not available; skipping ml-pipeline log collection."
    return
  fi

  log "ml-pipeline deployment logs (last 200 lines):"
  kubectl -n "${NAMESPACE}" logs deploy/"${SERVICE_NAME}" --tail=200 || true

  if [[ -d /tmp ]]; then
    mkdir -p /tmp/kfp-debug
    kubectl -n "${NAMESPACE}" logs deploy/"${SERVICE_NAME}" > /tmp/kfp-debug/ml-pipeline.log || true
  fi

  log "Current pods in namespace ${NAMESPACE}:"
  kubectl -n "${NAMESPACE}" get pods -o wide || true

  log "Listing TLS cert directory contents inside ${SERVICE_NAME} deployment:"
  kubectl -n "${NAMESPACE}" exec deploy/"${SERVICE_NAME}" -- ls -lah /etc/pki/tls/certs || true
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

PORT_FORWARD_LOG="/tmp/${SERVICE_NAME}-port-forward.log"

log "Starting port-forward for ${SERVICE_NAME} service on localhost:${API_PORT}..."
pkill -f "kubectl port-forward.*${SERVICE_NAME}" || true
kubectl port-forward -n "${NAMESPACE}" svc/"${SERVICE_NAME}" "${API_PORT}:${API_PORT}" >"${PORT_FORWARD_LOG}" 2>&1 &
KFP_PORT_FORWARD_PID=$!

if [[ -n "${GITHUB_ENV:-}" ]]; then
  echo "KFP_PORT_FORWARD_PID=${KFP_PORT_FORWARD_PID}" >> "${GITHUB_ENV}"
else
  export KFP_PORT_FORWARD_PID
fi

SCHEME="http"
TARGET_HOST="localhost"
CURL_TIMEOUT_SECONDS=5
declare -a CURL_OPTS=(-sS -f --max-time "${CURL_TIMEOUT_SECONDS}")

if [[ "${TLS_ENABLED}" == "true" ]]; then
  SCHEME="https"
  TARGET_HOST="${SERVICE_NAME}.${NAMESPACE}"
  CURL_OPTS+=(--resolve "${TARGET_HOST}:${API_PORT}:127.0.0.1")
  if [[ -n "${CA_CERT_PATH}" ]]; then
    CURL_OPTS+=(--cacert "${CA_CERT_PATH}")
  else
    CURL_OPTS+=(-k)
  fi
fi

CURL_URL="${SCHEME}://${TARGET_HOST}:${API_PORT}/apis/v2beta1/healthz"

log "Probing ${CURL_URL}"

for attempt in {1..30}; do
  if OUTPUT=$(curl "${CURL_OPTS[@]}" -o /dev/null "${CURL_URL}" 2>&1); then
    log "KFP API server is ready."
    exit 0
  fi

  curl_status=$?

  if ! kill -0 "${KFP_PORT_FORWARD_PID}" 2>/dev/null; then
    log "Port-forward process ${KFP_PORT_FORWARD_PID} exited unexpectedly (curl exit ${curl_status})."
    if [[ -f "${PORT_FORWARD_LOG}" ]]; then
      log "Recent port-forward logs:"
      tail -n 20 "${PORT_FORWARD_LOG}"
    else
      log "Port-forward log ${PORT_FORWARD_LOG} not found."
    fi
    if [[ -n "${OUTPUT}" ]]; then
      log "Last curl output: ${OUTPUT}"
    fi
    dump_ml_pipeline_logs
    exit 1
  fi

  if [[ "${attempt}" -eq 30 ]]; then
    log "KFP API server did not become reachable within expected time (curl exit ${curl_status})."
    if [[ -n "${OUTPUT}" ]]; then
      log "Last curl output: ${OUTPUT}"
    fi
    if [[ -f "${PORT_FORWARD_LOG}" ]]; then
      log "Recent port-forward logs:"
      tail -n 20 "${PORT_FORWARD_LOG}"
    else
      log "Port-forward log ${PORT_FORWARD_LOG} not found."
    fi
    dump_ml_pipeline_logs
    exit 1
  fi

  if [[ -n "${OUTPUT}" ]]; then
    log "Curl attempt ${attempt} failed (exit ${curl_status}): ${OUTPUT}"
  else
    log "Curl attempt ${attempt} failed (exit ${curl_status})."
  fi
  log "Waiting for KFP API server... (attempt ${attempt}/30)"
  sleep 5
done

exit 1
