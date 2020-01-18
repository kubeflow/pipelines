#!/bin/bash
#
# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -ex

echo "=================START MKP DEPLOY================"

GCR_IMAGE_TAG=${GCR_IMAGE_TAG:-latest}

# Configure gcloud as a Docker credential helper
gcloud auth configure-docker

# Delete argo first because KFP comes with argo too
kubectl delete namespace argo --wait || echo "No argo installed"

# Install the application resource definition
kubectl apply -f "https://raw.githubusercontent.com/GoogleCloudPlatform/marketplace-k8s-app-tools/master/crd/app-crd.yaml"

# Grant user ability for using Role-Based Access Control
kubectl create clusterrolebinding cluster-admin-binding \
  --clusterrole cluster-admin \
  --user $(gcloud config get-value account)

export APP_INSTANCE_NAME=kubeflow-pipelines-test

# Install mpdev
BIN_FILE="$HOME/bin/mpdev"
if ! which mpdev; then
  echo "Install mpdev"
  docker pull gcr.io/cloud-marketplace-tools/k8s/dev
  mkdir -p $HOME/bin/
  touch $BIN_FILE
  export PATH=$HOME/bin:$PATH
  docker run gcr.io/cloud-marketplace-staging/marketplace-k8s-app-tools/k8s/dev:remove-ui-ownerrefs cat /scripts/dev > "$BIN_FILE"
  chmod +x "$BIN_FILE"
fi

export MARKETPLACE_TOOLS_TAG=remove-ui-ownerrefs
export MARKETPLACE_TOOLS_IMAGE=gcr.io/cloud-marketplace-staging/marketplace-k8s-app-tools/k8s/dev

KFP_MANIFEST_DIR=${DIR}/../manifests/gcp_marketplace
pushd ${KFP_MANIFEST_DIR}
# Update the version value on schema.yaml and application.yaml
sed -ri 's/publishedVersion:.*/publishedVersion: '"$GCR_IMAGE_TAG"'/' schema.yaml
sed -ri 's/version:.*/version: '"$GCR_IMAGE_TAG"'/' ./chart/kubeflow-pipelines/templates/application.yaml

# Build new deployer
export REGISTRY=gcr.io/$(gcloud config get-value project | tr ':' '/')
export APP_NAME=$COMMIT_SHA
export DEPLOYER_NAME=$REGISTRY/$APP_NAME/deployer

docker build --tag $DEPLOYER_NAME -f deployer/Dockerfile .

docker push $DEPLOYER_NAME:$GCR_IMAGE_TAG

# Copy rest images and rename current images
export MKP_VERSION=$GCR_IMAGE_TAG
export GCR_FOLDER=$REGISTRY/$APP_NAME

echo MKP_VERSION:$MKP_VERSION, GCR_FOLDER:$GCR_FOLDER

export ARGO_VERSION=v2.3.0-license-compliance
export CLOUDSQL_PROXY_VERSION=1.14
export MLMD_SERVER_VERSION=0.14.0
export MLMD_ENVOY_VERSION=initial
export MINIO_VERSION=RELEASE.2019-08-14T20-37-41Z-license-compliance
export MYSQL_VERSION=5.6

gcloud container images add-tag --quiet gcr.io/ml-pipeline/argoexec:$ARGO_VERSION $GCR_FOLDER/argoexecutor:$MKP_VERSION
gcloud container images add-tag --quiet gcr.io/ml-pipeline/workflow-controller:$ARGO_VERSION $GCR_FOLDER/argoworkflowcontroller:$MKP_VERSION
gcloud container images add-tag --quiet gcr.io/cloudsql-docker/gce-proxy:$CLOUDSQL_PROXY_VERSION $GCR_FOLDER/cloudsqlproxy:$MKP_VERSION
gcloud container images add-tag --quiet gcr.io/tfx-oss-public/ml_metadata_store_server:$MLMD_SERVER_VERSION $GCR_FOLDER/metadataserver:$MKP_VERSION
gcloud container images add-tag --quiet gcr.io/ml-pipeline/envoy:$MLMD_ENVOY_VERSION $GCR_FOLDER/metadataenvoy:$MKP_VERSION
gcloud container images add-tag --quiet gcr.io/ml-pipeline/minio:$MINIO_VERSION $GCR_FOLDER/minio:$MKP_VERSION
gcloud container images add-tag --quiet gcr.io/ml-pipeline/mysql:$MYSQL_VERSION $GCR_FOLDER/mysql:$MKP_VERSION

gcloud container images add-tag --quiet $GCR_FOLDER/api-server:$MKP_VERSION $GCR_FOLDER/apiserver:$MKP_VERSION
gcloud container images add-tag --quiet $GCR_FOLDER/inverse-proxy-agent:$MKP_VERSION $GCR_FOLDER/proxyagent:$MKP_VERSION
gcloud container images add-tag --quiet $GCR_FOLDER/viewer-crd-controller:$MKP_VERSION $GCR_FOLDER/viewercrd:$MKP_VERSION
gcloud container images add-tag --quiet $GCR_FOLDER/visualization-server:$MKP_VERSION $GCR_FOLDER/visualizationserver:$MKP_VERSION
gcloud container images add-tag --quiet $GCR_FOLDER/deployer:$GCR_IMAGE_TAG $GCR_FOLDER/pipelines-test/deployer:$GCR_IMAGE_TAG

# Run install script
mpdev scripts/install  --deployer=$DEPLOYER_NAME:$GCR_IMAGE_TAG   --parameters='{"name": "'$APP_INSTANCE_NAME'", "namespace": "'$NAMESPACE'"}'

echo "Status of pods after mpdev install"
kubectl get pods -n ${NAMESPACE}

popd

# Waiting for the KFP resources are ready. TODO: verification of KFP resources
sleep 60