#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

SQUID_CONTAINER="${SQUID_CONTAINER:-kfp-local-squid}"
SQUID_PORT="${SQUID_PORT:-}"
API_SERVER_HTTP_PORT="${API_SERVER_HTTP_PORT:-}"
API_SERVER_RPC_PORT="${API_SERVER_RPC_PORT:-}"
TEST_LABEL="${TEST_LABEL:-FullRegression}"
NUM_NODES="${NUM_NODES:-6}"
RUN_PROXY_TESTS="${RUN_PROXY_TESTS:-false}"
DB_BACKEND="${DB_BACKEND:-sqlite}"
OBJECT_STORE_BACKEND="${OBJECT_STORE_BACKEND:-file}"

TMP_DIR="$(mktemp -d)"
SQLITE_DB_PATH="${SQLITE_DB_PATH:-${TMP_DIR}/metadata.sqlite}"
ARTIFACT_BUCKET_ROOT="${ARTIFACT_BUCKET_ROOT:-${TMP_DIR}/artifacts}"
SDK_WHEEL_DIR="${TMP_DIR}/sdk-wheel"
SDK_BUILD_ROOT="${TMP_DIR}/sdk-build"
API_SERVER_LOG="${TMP_DIR}/local-apiserver.log"
API_SERVER_PID=""
KUBECONFIG_PATH="${TMP_DIR}/kubeconfig"
SCRIPT_FAILED=false

cleanup() {
  if [[ -n "${API_SERVER_PID}" ]] && kill -0 "${API_SERVER_PID}" >/dev/null 2>&1; then
    kill "${API_SERVER_PID}" || true
    wait "${API_SERVER_PID}" || true
  fi
  if [[ "${KEEP_LOCAL_DOCKER_E2E_ARTIFACTS:-false}" == "true" || "${SCRIPT_FAILED}" == "true" ]]; then
    echo "Preserving local Docker E2E artifacts in ${TMP_DIR}"
    return
  fi
  docker rm -f "${SQUID_CONTAINER}" >/dev/null 2>&1 || true
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

pick_port() {
  python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
}

if [[ -z "${SQUID_PORT}" ]]; then SQUID_PORT="$(pick_port)"; fi
if [[ -z "${API_SERVER_HTTP_PORT}" ]]; then API_SERVER_HTTP_PORT="$(pick_port)"; fi
if [[ -z "${API_SERVER_RPC_PORT}" ]]; then API_SERVER_RPC_PORT="$(pick_port)"; fi

if [[ "${DB_BACKEND}" != "sqlite" ]]; then
  echo "Unsupported DB_BACKEND=${DB_BACKEND}; only sqlite is currently supported." >&2
  exit 1
fi
if [[ "${OBJECT_STORE_BACKEND}" != "file" ]]; then
  echo "Unsupported OBJECT_STORE_BACKEND=${OBJECT_STORE_BACKEND}; only file is currently supported." >&2
  exit 1
fi

mkdir -p "$(dirname "${SQLITE_DB_PATH}")" "${ARTIFACT_BUCKET_ROOT}"
mkdir -p "${SDK_WHEEL_DIR}" "${SDK_BUILD_ROOT}"

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

HTTP_PROXY_VALUE=""
HTTPS_PROXY_VALUE=""
NO_PROXY_VALUE="127.0.0.1,localhost,${HOST_GATEWAY}"
if [[ "${RUN_PROXY_TESTS}" == "true" ]]; then
  docker rm -f "${SQUID_CONTAINER}" >/dev/null 2>&1 || true
  docker build -t kfp-local-squid -f "${REPO_ROOT}/.github/resources/squid/Containerfile" "${REPO_ROOT}/.github/resources/squid" >/dev/null
  docker run -d --name "${SQUID_CONTAINER}" -p "${SQUID_PORT}:3128" kfp-local-squid >/dev/null
  for _ in $(seq 1 30); do
    if python3 - "${HOST_GATEWAY}" "${SQUID_PORT}" <<'PY'
import socket
import sys

host = sys.argv[1]
port = int(sys.argv[2])
sock = socket.socket()
sock.settimeout(1)
try:
    sock.connect((host, port))
except OSError:
    sys.exit(1)
finally:
    sock.close()
PY
    then
      break
    fi
    sleep 1
  done
  HTTP_PROXY_VALUE="http://${HOST_GATEWAY}:${SQUID_PORT}"
  HTTPS_PROXY_VALUE="${HTTP_PROXY_VALUE}"
fi

pushd "${REPO_ROOT}" >/dev/null

cp -a "${REPO_ROOT}/sdk/python/." "${SDK_BUILD_ROOT}/"
python3 -m pip wheel --no-deps --wheel-dir "${SDK_WHEEL_DIR}" "${SDK_BUILD_ROOT}" >/dev/null
SDK_WHEEL_PATH="$(ls "${SDK_WHEEL_DIR}"/kfp-*.whl | sed -n '1p')"
if [[ -z "${SDK_WHEEL_PATH}" ]]; then
  echo "Failed to build local kfp wheel" >&2
  exit 1
fi

env \
  KUBECONFIG="${KUBECONFIG_PATH}" \
  POD_NAMESPACE=kubeflow \
  INITCONNECTIONTIMEOUT=60s \
  KFP_V2_LOCAL_SDK_ROOT="${SDK_WHEEL_DIR}" \
  DBDRIVERNAME=sqlite \
  DBCONFIG_SQLITECONFIG_DATASOURCENAME="file:${SQLITE_DB_PATH}?_busy_timeout=5000&_journal_mode=WAL" \
  OBJECTSTORECONFIG_BUCKETURL="file://${ARTIFACT_BUCKET_ROOT}" \
  KFP_DEFAULT_PIPELINE_ROOT="file://${ARTIFACT_BUCKET_ROOT}/v2/artifacts" \
  ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_HOST=127.0.0.1 \
  ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_PORT=8889 \
  KFP_V2_RUNTIME_MODE=coordinator-poc \
  KFP_V2_RUNTIME_EXECUTOR=docker \
  LOCAL_API_SERVER=true \
  KFP_V2_LOCAL_API_SERVER_ADDRESS="${HOST_GATEWAY}" \
  KFP_V2_LOCAL_API_SERVER_PORT="${API_SERVER_RPC_PORT}" \
  http_proxy="${HTTP_PROXY_VALUE}" \
  https_proxy="${HTTPS_PROXY_VALUE}" \
  no_proxy="${NO_PROXY_VALUE}" \
  HTTP_PROXY="${HTTP_PROXY_VALUE}" \
  HTTPS_PROXY="${HTTPS_PROXY_VALUE}" \
  NO_PROXY="${NO_PROXY_VALUE}" \
  go run ./backend/src/apiserver --config ./backend/src/apiserver/config -logtostderr=true --rpcPortFlag=":${API_SERVER_RPC_PORT}" --httpPortFlag=":${API_SERVER_HTTP_PORT}" > "${API_SERVER_LOG}" 2>&1 &
API_SERVER_PID=$!

for _ in $(seq 1 90); do
  if ! kill -0 "${API_SERVER_PID}" >/dev/null 2>&1; then
    echo "Local API server exited before becoming ready"
    SCRIPT_FAILED=true
    tail -n 200 "${API_SERVER_LOG}" || true
    exit 1
  fi
  if curl -fsS "http://127.0.0.1:${API_SERVER_HTTP_PORT}/apis/v2beta1/healthz" >/dev/null; then
    break
  fi
  sleep 2
done

if ! curl -fsS "http://127.0.0.1:${API_SERVER_HTTP_PORT}/apis/v2beta1/healthz" >/dev/null; then
  echo "Local API server did not become ready"
  SCRIPT_FAILED=true
  tail -n 200 "${API_SERVER_LOG}" || true
  exit 1
fi

export LOCAL_API_SERVER=true
export LOCAL_API_SERVER_SDK_PACKAGE_PATH="/kfp-local-sdk/$(basename "${SDK_WHEEL_PATH}")"
export KFP_LOCAL_PIPELINE_ROOT="file://${ARTIFACT_BUCKET_ROOT}/v2/artifacts"

if ! go run github.com/onsi/ginkgo/v2/ginkgo \
  -r -v --keep-going --github-output=true \
  --nodes="${NUM_NODES}" \
  --label-filter="${TEST_LABEL}" \
  --silence-skips=true \
  ./backend/test/local_docker_e2e \
  -- \
  -namespace=kubeflow \
  -useProxy="${RUN_PROXY_TESTS}" \
  -apiUrl="http://127.0.0.1:${API_SERVER_HTTP_PORT}" \
  -apiScheme=http \
  -disableTlsCheck=false \
  -pullNumber="" \
  -repoName="kubeflow/pipelines"; then
  SCRIPT_FAILED=true
  exit 1
fi

docker rm -f "${SQUID_CONTAINER}" >/dev/null 2>&1 || true

popd >/dev/null
