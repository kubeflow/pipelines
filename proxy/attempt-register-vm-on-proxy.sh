#!/bin/bash
#
# Copyright 2019 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -ex

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"

function run-proxy-agent {
  # Start the proxy process
  # https://github.com/google/inverting-proxy/blob/master/agent/Dockerfile
  # Connect proxy agent to Kubeflow Pipelines UI
  /opt/bin/proxy-forwarding-agent \
        --debug=${DEBUG} \
        --proxy=${PROXY_URL} \
        --proxy-timeout=${PROXY_TIMEOUT} \
        --backend=${BACKEND_ID} \
        --host=${ML_PIPELINE_UI_SERVICE_HOST}:${ML_PIPELINE_UI_SERVICE_PORT} \
        --shim-websockets=true \
        --shim-path=websocket-shim \
        --health-check-path=${HEALTH_CHECK_PATH} \
        --health-check-interval-seconds=${HEALTH_CHECK_INTERVAL_SECONDS} \
        --health-check-unhealthy-threshold=${HEALTH_CHECK_UNHEALTHY_THRESHOLD}
}

# Check if the cluster already have proxy agent installed by checking ConfigMap.
if kubectl get configmap inverse-proxy-config; then
  # If ConfigMap already exist, reuse the existing endpoint (a.k.a BACKEND_ID) and same ProxyUrl.
  PROXY_URL=$(kubectl get configmap inverse-proxy-config -o json | jq -r ".data.ProxyUrl")
  BACKEND_ID=$(kubectl get configmap inverse-proxy-config -o json | jq -r ".data.BackendId")
  run-proxy-agent
  exit 0
fi

# Activate service account for gcloud SDK first
if [[ ! -z "${GOOGLE_APPLICATION_CREDENTIALS}" ]]; then
  gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"
fi

INSTANCE_ZONE="/"$(curl http://metadata.google.internal/computeMetadata/v1/instance/zone -H "Metadata-Flavor: Google")
INSTANCE_ZONE="${INSTANCE_ZONE##/*/}"

# Get latest Proxy server URL
curl -O https://storage.googleapis.com/dl-platform-public-configs/proxy-agent-config.json
PROXY_URL=$(python ${DIR}/get_proxy_url.py --config-file-path "proxy-agent-config.json" --location "${INSTANCE_ZONE}" --version "latest")
if [[ -z "${PROXY_URL}" ]]; then
    echo "Proxy URL for the zone ${INSTANCE_ZONE} no found, exiting."
    exit 1
fi
echo "Proxy URL from the config: ${PROXY_URL}"

# Register the proxy agent
VM_ID=$(curl -H 'Metadata-Flavor: Google' "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?format=full&audience=${PROXY_URL}/request-service-account-endpoint"  2>/dev/null)
RESULT_JSON=$(curl -H "Authorization: Bearer $(gcloud auth print-access-token)" -H "X-Inverting-Proxy-VM-ID: ${VM_ID}" -d "" "${PROXY_URL}/request-service-account-endpoint" 2>/dev/null)
echo "Response from the registration server: ${RESULT_JSON}"

HOSTNAME=$(echo "${RESULT_JSON}" | jq -r ".hostname")
BACKEND_ID=$(echo "${RESULT_JSON}" | jq -r ".backendID")
echo "Hostname: ${HOSTNAME}"
echo "Backend id: ${BACKEND_ID}"

# Store the registration information in a ConfigMap
kubectl create configmap inverse-proxy-config \
        --from-literal=ProxyUrl=${PROXY_URL} \
        --from-literal=BackendId=${BACKEND_ID} \
        --from-literal=Hostname=${HOSTNAME}

run-proxy-agent
