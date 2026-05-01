#!/bin/bash
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Purpose:
# This script configures KFP to use an already-deployed MLflow instance for
# MLflow E2E tests.
#
# CI helper: patch the KFP API server with plugins.mlflow, roll it out, and
# port-forward the API server and MLflow so E2E tests can reach both.
# It also exports workspace/auth variables used by MLflow test helpers.
#
# Usage: configure-mlflow.sh <KFP_NAMESPACE> <MLFLOW_NAMESPACE> <CONFIG_JSON_PATH>

set -e

KFP_NAMESPACE="${1:?KFP namespace required}"
MLFLOW_NAMESPACE="${2:?MLflow namespace required}"
CONFIG_JSON_PATH="${3:?Path to source config.json required}"

echo "Services in ${MLFLOW_NAMESPACE} namespace:"
kubectl get svc -n "$MLFLOW_NAMESPACE" --no-headers
MLFLOW_SVC=$(kubectl get svc -n "$MLFLOW_NAMESPACE" --no-headers -o custom-columns=":metadata.name" | grep -i mlflow | head -1)
if [ -z "$MLFLOW_SVC" ]; then
  echo "ERROR: No service matching 'mlflow' found in namespace $MLFLOW_NAMESPACE"
  exit 1
fi
MLFLOW_PORT=$(kubectl get svc -n "$MLFLOW_NAMESPACE" "$MLFLOW_SVC" -o jsonpath='{.spec.ports[0].port}')
MLFLOW_HOST="${MLFLOW_SVC}.${MLFLOW_NAMESPACE}.svc.cluster.local"
MLFLOW_ENDPOINT="https://${MLFLOW_HOST}:${MLFLOW_PORT}"
echo "MLflow service: $MLFLOW_SVC port=$MLFLOW_PORT endpoint=$MLFLOW_ENDPOINT"

MLFLOW_PATCH=$(jq -n --arg endpoint "$MLFLOW_ENDPOINT" '{
  endpoint: $endpoint,
  tls: { insecureSkipVerify: true },
  settings: { workspacesEnabled: true }
}')

jq --argjson mlflow "$MLFLOW_PATCH" '. + { plugins: { mlflow: $mlflow } }' \
  "$CONFIG_JSON_PATH" > /tmp/kfp-config.json

echo "Patched config.json plugins.mlflow:"
jq '.plugins.mlflow' /tmp/kfp-config.json

kubectl create configmap kfp-mlflow-config -n "$KFP_NAMESPACE" \
  --from-file=config.json=/tmp/kfp-config.json --dry-run=client -o yaml | kubectl apply -f -
kubectl patch deployment ml-pipeline -n "$KFP_NAMESPACE" --type=strategic -p \
  '{"spec":{"template":{"spec":{"volumes":[{"name":"mlflow-cfg","configMap":{"name":"kfp-mlflow-config"}}],"containers":[{"name":"ml-pipeline-api-server","volumeMounts":[{"name":"mlflow-cfg","mountPath":"/config/config.json","subPath":"config.json"}]}]}}}}'
kubectl rollout status deployment/ml-pipeline -n "$KFP_NAMESPACE" --timeout=180s

pkill -f "kubectl port-forward.*ml-pipeline.*8888" || true
sleep 2

C_DIR="${BASH_SOURCE%/*}"
"${C_DIR}/forward-port.sh" "$KFP_NAMESPACE" ml-pipeline 8888 8888

for i in $(seq 1 12); do
  if curl -sf http://localhost:8888/apis/v1beta1/healthz > /dev/null 2>&1; then
    echo "API server is healthy on localhost:8888"
    break
  fi
  echo "Waiting for API server to become healthy... ($i/12)"
  sleep 5
done
curl -sf http://localhost:8888/apis/v1beta1/healthz > /dev/null 2>&1 || {
  echo "ERROR: API server not reachable at localhost:8888"
  exit 1
}

SA_TOKEN=$(kubectl create token ml-pipeline -n "$KFP_NAMESPACE" --duration=1h 2>/dev/null || true)
if [ -n "${GITHUB_ENV:-}" ]; then
  echo "MLFLOW_WORKSPACE=$KFP_NAMESPACE" >> "$GITHUB_ENV"
  # Later workflow steps need these to re-establish port-forward: background jobs from this step
  # are terminated when the step exits, so test-and-report starts kubectl port-forward again.
  echo "MLFLOW_PORT_FORWARD_NS=$MLFLOW_NAMESPACE" >> "$GITHUB_ENV"
  echo "MLFLOW_PORT_FORWARD_SVC=$MLFLOW_SVC" >> "$GITHUB_ENV"
  echo "MLFLOW_PORT_FORWARD_REMOTE_PORT=$MLFLOW_PORT" >> "$GITHUB_ENV"
  if [ -n "$SA_TOKEN" ]; then
    echo "MLFLOW_BEARER_TOKEN=$SA_TOKEN" >> "$GITHUB_ENV"
    echo "Exported MLFLOW_BEARER_TOKEN and MLFLOW_WORKSPACE for test helpers"
  else
    echo "WARNING: Could not create SA token; MLflow requests may be unauthenticated"
    echo "Exported MLFLOW_WORKSPACE only"
  fi
fi

kubectl port-forward -n "$MLFLOW_NAMESPACE" "svc/$MLFLOW_SVC" "8080:$MLFLOW_PORT" &
sleep 3

HEALTH_URL="https://localhost:8080/api/2.0/mlflow/experiments/search"
CURL_HEADERS=(-H "X-MLflow-Workspace: $KFP_NAMESPACE")
[ -n "$SA_TOKEN" ] && CURL_HEADERS+=(-H "Authorization: Bearer $SA_TOKEN")

STATUS=000
for i in $(seq 1 30); do
  STATUS=$(curl -sk -o /dev/null -w '%{http_code}' --connect-timeout 5 --max-time 10 \
    "${CURL_HEADERS[@]}" "$HEALTH_URL" 2>/dev/null || echo "000")
  if [ "$STATUS" != "000" ] && [ "$STATUS" -lt 500 ] 2>/dev/null; then
    echo "MLflow backend is healthy on localhost:8080 (HTTPS, status=$STATUS)"
    break
  fi
  echo "Waiting for MLflow backend... ($i/30, status=$STATUS)"
  sleep 5
done
if [ "$STATUS" = "000" ] || { [ "$STATUS" -ge 500 ] 2>/dev/null; }; then
  echo "ERROR: MLflow backend not healthy after 30 attempts (last status=$STATUS)"
  exit 1
fi
