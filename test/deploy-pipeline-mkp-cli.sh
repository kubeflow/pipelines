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

echo "==================================================START DEPLOY CLUSTER================================================="
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
  NODE_POOL_CONFIG_ARG="--num-nodes=2 --machine-type=n1-standard-8 \
    --enable-autoscaling --max-nodes=8 --min-nodes=2"
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
  gcloud beta container clusters create ${TEST_CLUSTER} --zone "us-east4-c" --machine-type "n1-standard-2"
fi

gcloud container clusters get-credentials ${TEST_CLUSTER} --zone "us-east4-c"



echo "==================================================START MKP DEPLOY======================================================"

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

kubectl create namespace $NAMESPACE

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

mpdev

KFP_MANIFEST_DIR=${DIR}/../manifests/gcp_marketplace
pushd ${KFP_MANIFEST_DIR}
# Update the tag value on schema.yaml
sed -ri 's/publishedVersion:.*/publishedVersion: '"$GCR_IMAGE_TAG"'/' schema.yaml

# Build new deployer
export REGISTRY=gcr.io/$(gcloud config get-value project | tr ':' '/')
export APP_NAME=$COMMIT_SHA
export DEPLOYER_NAME=$REGISTRY/$APP_NAME/deployer

docker build --tag $DEPLOYER_NAME -f deployer/Dockerfile .

docker push $DEPLOYER_NAME:$GCR_IMAGE_TAG

# Run install script
mpdev scripts/install  --deployer=$DEPLOYER_NAME:$GCR_IMAGE_TAG   --parameters='{"name": "'$APP_INSTANCE_NAME'", "namespace": "'$NAMESPACE'"}'
#mpdev scripts/install  --deployer=gcr.io/ml-pipeline/google/pipelines/deployer:0.1   --parameters='{"name": "'$APP_INSTANCE_NAME'", "namespace": "'$NAMESPACE'"}'

echo "Status of pods after mpdev install"
kubectl get pods -n ${NAMESPACE}

if [ "$ENABLE_WORKLOAD_IDENTITY" = true ]; then
  # Use static GSAs for testing, so we don't need to GC them.
  export SYSTEM_GSA="test-kfp-system"
  export USER_GSA="test-kfp-user"

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