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

set -e

MLFLOW_NAMESPACE="${1:?MLflow namespace required}"
MLFLOW_AUTH_TYPE="${2:?MLflow auth type required}"
ARTIFACT_STORAGE="${3:?MLflow artifact storage required}"
DEFAULT_MLFLOW_IMAGE="ghcr.io/mlflow/mlflow:v3.13.0-full"
MLFLOW_IMAGE="${4:-$DEFAULT_MLFLOW_IMAGE}"

AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-minio}"
AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-minio123}"
AWS_S3_BUCKET="${AWS_S3_BUCKET:-mlpipeline}"

case "$MLFLOW_AUTH_TYPE" in
  basic-auth|none)
    ;;
  *)
    echo "ERROR: deploy-direct-mlflow.sh only supports basic-auth and none, got $MLFLOW_AUTH_TYPE"
    exit 1
    ;;
esac

C_DIR="${BASH_SOURCE%/*}"
MANIFESTS_DIR="${C_DIR}/../mlflow-direct/manifests"

ARTIFACTS_DESTINATION="file:///mlflow/artifacts"
MLFLOW_S3_ENDPOINT_URL=""
if [ "$ARTIFACT_STORAGE" = "s3" ]; then
  ARTIFACTS_DESTINATION="s3://${AWS_S3_BUCKET}/artifacts"
  MLFLOW_S3_ENDPOINT_URL="http://seaweedfs.kubeflow.svc.cluster.local:9000"
fi

kubectl create namespace "$MLFLOW_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

CONFIGMAP_ARGS=(
  --from-literal=MLFLOW_AUTH_TYPE="$MLFLOW_AUTH_TYPE"
  --from-literal=ARTIFACTS_DESTINATION="$ARTIFACTS_DESTINATION"
  --from-literal=MLFLOW_S3_ENDPOINT_URL="$MLFLOW_S3_ENDPOINT_URL"
)
if [ "$MLFLOW_AUTH_TYPE" = "none" ]; then
  CONFIGMAP_ARGS+=(--from-literal=MLFLOW_SERVER_DISABLE_SECURITY_MIDDLEWARE=true)
fi

kubectl create configmap mlflow-direct-config -n "$MLFLOW_NAMESPACE" \
  "${CONFIGMAP_ARGS[@]}" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret generic mlflow-direct-auth -n "$MLFLOW_NAMESPACE" \
  --from-literal=flask-secret-key=mlflow-ci-secret-key \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret generic aws-credentials -n "$MLFLOW_NAMESPACE" \
  --from-literal=AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" \
  --from-literal=AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl kustomize "$MANIFESTS_DIR" \
  | sed "s#ghcr.io/mlflow/mlflow:v3.13.0-full#${MLFLOW_IMAGE}#g" \
  | kubectl apply -f -

kubectl -n "$MLFLOW_NAMESPACE" wait --for=condition=available deployment/postgres-deployment --timeout=180s
kubectl -n "$MLFLOW_NAMESPACE" wait --for=condition=available deployment/mlflow --timeout=300s
kubectl -n "$MLFLOW_NAMESPACE" get pods -l app=mlflow -o wide
