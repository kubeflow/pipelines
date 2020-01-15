#!/bin/bash
#
# Copyright 2018 Google LLC
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

echo "===========START DEPLOY CLUSTER==============="
TEST_CLUSTER_PREFIX=${WORKFLOW_FILE%.*}
TEST_CLUSTER_DEFAULT=$(echo $TEST_CLUSTER_PREFIX | cut -d _ -f 1)-${COMMIT_SHA:0:7}-${RANDOM}
TEST_CLUSTER=${TEST_CLUSTER:-${TEST_CLUSTER_DEFAULT}}
ENABLE_WORKLOAD_IDENTITY=${ENABLE_WORKLOAD_IDENTITY:-false}

cd ${DIR}
# test if ${TEST_CLUSTER} exists or not
if gcloud container clusters describe ${TEST_CLUSTER} &>/dev/null; then
  echo "Use existing test cluster: ${TEST_CLUSTER}"
else
  echo "Creating a new test cluster: ${TEST_CLUSTER}"
  SHOULD_CLEANUP_CLUSTER=true
  # Machine type and cluster size is the same as kubeflow deployment to
  # easily compare performance. We can reduce usage later.
  NODE_POOL_CONFIG_ARG="--num-nodes=3 --machine-type=n1-standard-8 \
    --enable-autoscaling --max-nodes=8 --min-nodes=3"
  # Use new kubernetes master to improve workload identity stability.
  KUBERNETES_VERSION_ARG="--cluster-version=1.14.8-gke.17"
  if [ "$ENABLE_WORKLOAD_IDENTITY" = true ]; then
    WI_ARG="--identity-namespace=$PROJECT.svc.id.goog"
    SCOPE_ARG=
  else
    WI_ARG=
    # "storage-rw" is needed to allow VMs to push to gcr.io when using default GCE service account.
    # reference: https://cloud.google.com/compute/docs/access/service-accounts#accesscopesiam
    SCOPE_ARG="--scopes=storage-rw,cloud-platform"
  fi
  #  gcloud beta container clusters create ${TEST_CLUSTER} ${SCOPE_ARG} --zone "us-east4-c" --machine-type "n1-standard-2"
  gcloud beta container clusters create ${TEST_CLUSTER} ${SCOPE_ARG} ${NODE_POOL_CONFIG_ARG} ${WI_ARG} ${KUBERNETES_VERSION_ARG}

fi

gcloud container clusters get-credentials ${TEST_CLUSTER} --zone "us-east1-b"

# when we reuse a cluster when debugging, clean up its kfp installation first
# this does nothing with a new cluster
kubectl delete namespace ${NAMESPACE} --wait || echo "No need to delete ${NAMESPACE} namespace. It doesn't exist."
kubectl create namespace ${NAMESPACE} --dry-run -o yaml | kubectl apply -f -

echo "=================Install Argo===================="
# Tests work without these lines. TODO: Verify and remove these lines
kubectl config set-context $(kubectl config current-context) --namespace=default
echo "Add necessary cluster role bindings"
ACCOUNT=$(gcloud info --format='value(config.account)')
kubectl create clusterrolebinding PROW_BINDING --clusterrole=cluster-admin --user=$ACCOUNT --dry-run -o yaml | kubectl apply -f -
kubectl create clusterrolebinding DEFAULT_BINDING --clusterrole=cluster-admin --serviceaccount=default:default --dry-run -o yaml | kubectl apply -f -

ARGO_VERSION=v2.3.0

# if argo is not installed
if ! which argo; then
  echo "install argo"
  mkdir -p ~/bin/
  export PATH=~/bin/:$PATH
  curl -sSL -o ~/bin/argo https://github.com/argoproj/argo/releases/download/$ARGO_VERSION/argo-linux-amd64
  chmod +x ~/bin/argo
fi

ARGO_KSA="test-runner"

# Some workflows are deployed to the non-default namespace where the GCP credential secret is stored
# In this case, the default service account in that namespace doesn't have enough permission
echo "add service account for running the test workflow"
kubectl create serviceaccount ${ARGO_KSA} -n ${NAMESPACE} --dry-run -o yaml | kubectl apply -f -
kubectl create clusterrolebinding test-admin-binding --clusterrole=cluster-admin --serviceaccount=${NAMESPACE}:${ARGO_KSA} --dry-run -o yaml | kubectl apply -f -

if [ "$ENABLE_WORKLOAD_IDENTITY" = true ]; then
  ARGO_GSA="test-argo"
  # Util library including create_gsa_if_not_present and bind_gsa_and_ksa functions.
  source "$DIR/../manifests/kustomize/wi-utils.sh"
  create_gsa_if_not_present $ARGO_GSA

  gcloud projects add-iam-policy-binding $PROJECT \
    --member="serviceAccount:$ARGO_GSA@$PROJECT.iam.gserviceaccount.com" \
    --role="roles/editor" \
    > /dev/null # hide verbose output
  bind_gsa_and_ksa $ARGO_GSA $ARGO_KSA $PROJECT $NAMESPACE

  verify_workload_identity_binding $ARGO_KSA $NAMESPACE
fi

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

if [ "$ENABLE_WORKLOAD_IDENTITY" = true ]; then
  # Use static GSAs for testing, so we don't need to GC them.
  export SYSTEM_GSA="test-kfp-system"
  export USER_GSA="test-kfp-user"

  kubectl create serviceaccount --namespace $NAMESPACE ml-pipeline-ui
  kubectl create serviceaccount --namespace $NAMESPACE pipeline-runner

  yes | PROJECT_ID=$PROJECT CLUSTER_NAME=$TEST_CLUSTER NAMESPACE=$NAMESPACE \
    ${DIR}/../manifests/kustomize/gcp-workload-identity-setup.sh

  gcloud projects add-iam-policy-binding $PROJECT \
    --member="serviceAccount:$SYSTEM_GSA@$PROJECT.iam.gserviceaccount.com" \
    --role="roles/editor"
  gcloud projects add-iam-policy-binding $PROJECT \
    --member="serviceAccount:$USER_GSA@$PROJECT.iam.gserviceaccount.com" \
    --role="roles/editor"

  source "$DIR/../manifests/kustomize/wi-utils.sh"
  verify_workload_identity_binding "pipeline-runner" $NAMESPACE
fi

popd

# Waiting for the KFP resources are ready. TODO: verification of KFP resources
sleep 60