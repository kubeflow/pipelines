#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Cursor terminals can prepend a bundled Node.js helper ahead of /usr/local/bin.
# Prefer the globally installed Node.js toolchain for this script.
if [[ -x "/usr/local/bin/node" || -x "/usr/local/sbin/node" ]]; then
  PATH="/usr/local/sbin:/usr/local/bin:${PATH}"
fi

UI_PORT="${UI_PORT:-3000}"
API_SERVER_HTTP_PORT="${API_SERVER_HTTP_PORT:-3002}"
API_SERVER_RPC_PORT="${API_SERVER_RPC_PORT:-3003}"

TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/kfp-local-ui.XXXXXX")"
SQLITE_DB_PATH="${SQLITE_DB_PATH:-${TMP_ROOT}/metadata.sqlite}"
ARTIFACT_BUCKET_ROOT="${ARTIFACT_BUCKET_ROOT:-${TMP_ROOT}/artifacts}"
SDK_WHEEL_DIR="${TMP_ROOT}/sdk-wheel"
SDK_BUILD_ROOT="${TMP_ROOT}/sdk-build"
LOG_DIR="${TMP_ROOT}/logs"
KUBECONFIG_PATH="${TMP_ROOT}/kubeconfig"
GO_CACHE_DIR="${TMP_ROOT}/go-cache"
GO_TMP_DIR="${TMP_ROOT}/go-tmp"
API_SERVER_LOG="${LOG_DIR}/apiserver.log"
UI_SERVER_LOG="${LOG_DIR}/ui-server.log"

SCRIPT_FAILED=false
declare -a PROCESS_GROUPS=()

usage() {
  cat <<EOF
Usage: $(basename "$0")

Starts the built UI and a local API server from the current checkout.
By default it runs:

  UI:     http://127.0.0.1:${UI_PORT}
  API:    http://127.0.0.1:${API_SERVER_HTTP_PORT}

Environment overrides:
  UI_PORT
  API_SERVER_HTTP_PORT
  API_SERVER_RPC_PORT
  SQLITE_DB_PATH
  ARTIFACT_BUCKET_ROOT
  KEEP_LOCAL_POC_UI_ARTIFACTS=true  # preserve logs on exit
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -gt 0 ]]; then
  echo "This script now runs from the current checkout only and does not accept a commit argument." >&2
  usage >&2
  exit 1
fi

mkdir -p "${LOG_DIR}"
mkdir -p "${GO_CACHE_DIR}" "${GO_TMP_DIR}"
mkdir -p "$(dirname "${SQLITE_DB_PATH}")" "${ARTIFACT_BUCKET_ROOT}" "${SDK_WHEEL_DIR}" "${SDK_BUILD_ROOT}"

require_command() {
  local command_name="$1"
  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "Missing required command: ${command_name}" >&2
    exit 1
  fi
}

listener_pids_for_port() {
  local port="$1"
  ss -ltnp "sport = :${port}" 2>/dev/null | python3 -c '
import re
import sys

output = sys.stdin.read()
for pid in sorted(set(re.findall(r"pid=(\\d+)", output))):
    print(pid)
'
}

is_managed_listener_command() {
  local port_name="$1"
  local command_line="$2"

  case "${port_name}" in
    "UI")
      [[ "${command_line}" == *"server/dist/server.js"* ]]
      ;;
    "API HTTP" | "API RPC")
      [[ "${command_line}" == *"apiserver"* ]]
      ;;
    *)
      return 1
      ;;
  esac
}

stop_managed_listener() {
  local port_name="$1"
  local port="$2"
  local signal="${3:-TERM}"

  while read -r pid; do
    [[ -z "${pid}" ]] && continue

    local command_line
    command_line="$(ps -p "${pid}" -o args= 2>/dev/null || true)"
    if ! is_managed_listener_command "${port_name}" "${command_line}"; then
      continue
    fi

    echo "Stopping stale ${port_name} listener on port ${port} (pid ${pid})"
    kill -"${signal}" "${pid}" >/dev/null 2>&1 || true
  done < <(listener_pids_for_port "${port}")
}

cleanup_previous_run_artifacts() {
  stop_managed_listener "UI" "${UI_PORT}" TERM
  stop_managed_listener "API HTTP" "${API_SERVER_HTTP_PORT}" TERM
  stop_managed_listener "API RPC" "${API_SERVER_RPC_PORT}" TERM

  sleep 2

  stop_managed_listener "UI" "${UI_PORT}" KILL
  stop_managed_listener "API HTTP" "${API_SERVER_HTTP_PORT}" KILL
  stop_managed_listener "API RPC" "${API_SERVER_RPC_PORT}" KILL
}

require_free_port() {
  local port_name="$1"
  local port="$2"
  python3 - "${port_name}" "${port}" <<'PY'
import socket
import sys

port_name = sys.argv[1]
port = int(sys.argv[2])
sock = socket.socket()
try:
    sock.bind(("127.0.0.1", port))
except OSError as exc:
    print(f"{port_name} port {port} is not available: {exc}", file=sys.stderr)
    sys.exit(1)
finally:
    sock.close()
PY
}

wait_for_http() {
  local name="$1"
  local url="$2"
  local log_file="$3"
  local attempts="${4:-120}"

  for _ in $(seq 1 "${attempts}"); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  echo "${name} did not become ready at ${url}" >&2
  if [[ -f "${log_file}" ]]; then
    echo "--- ${name} log ---" >&2
    sed -n '1,200p' "${log_file}" >&2 || true
    echo "--- end ${name} log ---" >&2
  fi
  return 1
}

start_process_group() {
  local name="$1"
  local log_file="$2"
  local command="$3"

  setsid bash -lc "${command}" >"${log_file}" 2>&1 &
  local pid=$!
  PROCESS_GROUPS+=("${pid}")
  echo "Started ${name} (pid ${pid}), log: ${log_file}"
}

cleanup() {
  local exit_code=$?
  set +e

  if [[ ${exit_code} -ne 0 ]]; then
    SCRIPT_FAILED=true
  fi

  for pid in "${PROCESS_GROUPS[@]}"; do
    if kill -0 "${pid}" >/dev/null 2>&1; then
      kill -TERM -- "-${pid}" >/dev/null 2>&1 || kill -TERM "${pid}" >/dev/null 2>&1 || true
    fi
  done

  for pid in "${PROCESS_GROUPS[@]}"; do
    wait "${pid}" >/dev/null 2>&1 || true
  done

  if [[ "${KEEP_LOCAL_POC_UI_ARTIFACTS:-false}" == "true" || "${SCRIPT_FAILED}" == "true" ]]; then
    echo "Preserving local POC UI artifacts in ${TMP_ROOT}"
    exit "${exit_code}"
  fi

  rm -rf "${TMP_ROOT}"

  exit "${exit_code}"
}

trap cleanup EXIT SIGINT SIGTERM SIGHUP

for command_name in docker go npm node ps python3 curl setsid ss; do
  require_command "${command_name}"
done

cleanup_previous_run_artifacts

require_free_port "UI" "${UI_PORT}"
require_free_port "API HTTP" "${API_SERVER_HTTP_PORT}"
require_free_port "API RPC" "${API_SERVER_RPC_PORT}"

HOST_GATEWAY="$(docker network inspect bridge --format '{{(index .IPAM.Config 0).Gateway}}' 2>/dev/null || true)"
if [[ -z "${HOST_GATEWAY}" ]]; then
  HOST_GATEWAY="172.17.0.1"
fi

cat > "${KUBECONFIG_PATH}" <<EOF
apiVersion: v1
kind: Config
clusters:
- cluster:
    insecure-skip-tls-verify: true
    server: https://127.0.0.1:65535
  name: local
contexts:
- context:
    cluster: local
    namespace: kubeflow
    user: local
  name: local
current-context: local
users:
- name: local
  user:
    token: dummy
EOF

echo "Building local KFP SDK wheel"
(
  cd "${REPO_ROOT}"
  cp -a "${REPO_ROOT}/sdk/python/." "${SDK_BUILD_ROOT}/"
  python3 -m pip wheel --no-deps --wheel-dir "${SDK_WHEEL_DIR}" "${SDK_BUILD_ROOT}" >/dev/null
)

SDK_WHEEL_PATH="$(ls "${SDK_WHEEL_DIR}"/kfp-*.whl | sed -n '1p')"
if [[ -z "${SDK_WHEEL_PATH}" ]]; then
  echo "Failed to build local kfp wheel" >&2
  exit 1
fi

LOCAL_API_SERVER_SDK_PACKAGE_PATH="/kfp-local-sdk/$(basename "${SDK_WHEEL_PATH}")"

echo "Installing frontend dependencies in ${REPO_ROOT}/frontend"
(
  cd "${REPO_ROOT}/frontend"
  NPM_CONFIG_INCLUDE=optional npm ci
)

echo "Building frontend bundle"
(
  cd "${REPO_ROOT}/frontend"
  npm run build:tailwind >/dev/null
  ./node_modules/.bin/vite build >/dev/null
)

echo "Building frontend node server"
(
  cd "${REPO_ROOT}/frontend/server"
  npm run build >/dev/null
)

start_process_group \
  "API server" \
  "${API_SERVER_LOG}" \
  "cd \"${REPO_ROOT}\" && exec env \
KUBECONFIG=\"${KUBECONFIG_PATH}\" \
GOCACHE=\"${GO_CACHE_DIR}\" \
GOTMPDIR=\"${GO_TMP_DIR}\" \
POD_NAMESPACE=kubeflow \
INITCONNECTIONTIMEOUT=60s \
KFP_V2_LOCAL_SDK_ROOT=\"${SDK_WHEEL_DIR}\" \
DBDRIVERNAME=sqlite \
DBCONFIG_SQLITECONFIG_DATASOURCENAME=\"file:${SQLITE_DB_PATH}?_busy_timeout=5000&_journal_mode=WAL\" \
OBJECTSTORECONFIG_BUCKETURL=\"file://${ARTIFACT_BUCKET_ROOT}\" \
KFP_DEFAULT_PIPELINE_ROOT=\"file://${ARTIFACT_BUCKET_ROOT}/v2/artifacts\" \
ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_HOST=127.0.0.1 \
ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_PORT=8889 \
KFP_V2_RUNTIME_MODE=coordinator-poc \
KFP_V2_RUNTIME_EXECUTOR=docker \
LOCAL_API_SERVER=true \
LOCAL_API_SERVER_SDK_PACKAGE_PATH=\"${LOCAL_API_SERVER_SDK_PACKAGE_PATH}\" \
KFP_V2_LOCAL_API_SERVER_ADDRESS=\"${HOST_GATEWAY}\" \
KFP_V2_LOCAL_API_SERVER_PORT=\"${API_SERVER_RPC_PORT}\" \
go run ./backend/src/apiserver --config ./backend/src/apiserver/config -logtostderr=true --rpcPortFlag=\":${API_SERVER_RPC_PORT}\" --httpPortFlag=\":${API_SERVER_HTTP_PORT}\""
wait_for_http "API server" "http://127.0.0.1:${API_SERVER_HTTP_PORT}/apis/v2beta1/healthz" "${API_SERVER_LOG}"

start_process_group \
  "UI server" \
  "${UI_SERVER_LOG}" \
  "cd \"${REPO_ROOT}/frontend\" && exec env \
ARTIFACTS_LOCAL_ROOT=\"${ARTIFACT_BUCKET_ROOT}\" \
ML_PIPELINE_SERVICE_HOST=127.0.0.1 \
ML_PIPELINE_SERVICE_PORT=\"${API_SERVER_HTTP_PORT}\" \
ML_PIPELINE_SERVICE_SCHEME=http \
KFP_V2_RUNTIME_MODE=coordinator-poc \
KFP_V2_RUNTIME_EXECUTOR=docker \
node server/dist/server.js build \"${UI_PORT}\""
wait_for_http "UI server" "http://127.0.0.1:${UI_PORT}/apis/v2beta1/healthz" "${UI_SERVER_LOG}"
wait_for_http "UI root" "http://127.0.0.1:${UI_PORT}/" "${UI_SERVER_LOG}"

cat <<EOF

UI is ready at:          http://127.0.0.1:${UI_PORT}
API server is ready at:  http://127.0.0.1:${API_SERVER_HTTP_PORT}
Repo checkout:           ${REPO_ROOT}
SQLite DB:               ${SQLITE_DB_PATH}
Artifact root:           ${ARTIFACT_BUCKET_ROOT}
Local SDK wheel:         ${SDK_WHEEL_PATH}
Logs:                    ${LOG_DIR}

Press Ctrl+C or close this terminal to stop the UI and API server.
EOF

while true; do
  for pid in "${PROCESS_GROUPS[@]}"; do
    if ! kill -0 "${pid}" >/dev/null 2>&1; then
      SCRIPT_FAILED=true
      child_exit_code=0
      wait "${pid}" || child_exit_code=$?
      exit "${child_exit_code}"
    fi
  done
  sleep 2
done
