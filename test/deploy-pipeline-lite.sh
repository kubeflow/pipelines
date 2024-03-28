#!/bin/bash
#
# Copyright 2018 The Kubeflow Authors
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

# Env inputs:
# * $GCR_IMAGE_BASE_DIR
# * $GCR_IMAGE_TAG
# * $KFP_DEPLOY_RELEASE
# * $ENABLE_WORKLOAD_IDENTITY
GCR_IMAGE_TAG=${GCR_IMAGE_TAG:-latest}
ENABLE_WORKLOAD_IDENTITY=${ENABLE_WORKLOAD_IDENTITY:-false}

KFP_MANIFEST_DIR="${DIR}/manifests"
pushd ${KFP_MANIFEST_DIR}

if ! which kustomize; then
  # Download kustomize cli tool
  TOOL_DIR=${DIR}/bin
  mkdir -p ${TOOL_DIR}
  # Use 5.2.1 because we want it to be compatible with latest kustomize syntax changes
  # See discussions tracked in https://github.com/kubeflow/manifests/issues/2388 and https://github.com/kubeflow/manifests/pull/2653.
  wget --no-verbose https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv5.2.1/kustomize_v5.2.1_linux_amd64.tar.gz \
    -O kustomize_linux_amd64.tar.gz  
  tar -xzvf kustomize_linux_amd64.tar.gz kustomize  
  mv kustomize ${TOOL_DIR}/kustomize 
  chmod +x ${TOOL_DIR}/kustomize
  PATH=${PATH}:${TOOL_DIR}
fi

if [ -z "$KFP_DEPLOY_RELEASE" ]; then
  echo "Deploying KFP in working directory..."
  KFP_MANIFEST_DIR=${DIR}/manifests

  pushd ${KFP_MANIFEST_DIR}/cluster-scoped-resources
  kustomize build | kubectl apply -f -
  kubectl wait --for condition=established --timeout=60s crd/applications.app.k8s.io
  popd

  pushd ${KFP_MANIFEST_DIR}/dev
  # This is the recommended approach to do this.
  # reference: https://github.com/kubernetes-sigs/kustomize/blob/master/docs/eschewedFeatures.md#build-time-side-effects-from-cli-args-or-env-variables
  kustomize edit set image gcr.io/ml-pipeline/api-server=${GCR_IMAGE_BASE_DIR}/api-server:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/persistenceagent=${GCR_IMAGE_BASE_DIR}/persistenceagent:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/scheduledworkflow=${GCR_IMAGE_BASE_DIR}/scheduledworkflow:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/frontend=${GCR_IMAGE_BASE_DIR}/frontend:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/viewer-crd-controller=${GCR_IMAGE_BASE_DIR}/viewer-crd-controller:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/visualization-server=${GCR_IMAGE_BASE_DIR}/visualization-server:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/inverse-proxy-agent=${GCR_IMAGE_BASE_DIR}/inverse-proxy-agent:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/metadata-writer=${GCR_IMAGE_BASE_DIR}/metadata-writer:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/cache-server=${GCR_IMAGE_BASE_DIR}/cache-server:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/cache-deployer=${GCR_IMAGE_BASE_DIR}/cache-deployer:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/metadata-envoy=${GCR_IMAGE_BASE_DIR}/metadata-envoy:${GCR_IMAGE_TAG}
  cat kustomization.yaml

  kustomize build | kubectl apply -f -
  popd
else
  # exclude SDK release tags
  KFP_LATEST_RELEASE=$(git tag --sort=v:refname | grep -v "sdk-" | tail -1)
  echo "Deploying KFP release $KFP_LATEST_RELEASE"

  # temporarily checkout last release tag
  git checkout $KFP_LATEST_RELEASE

  pushd ${KFP_MANIFEST_DIR}/cluster-scoped-resources
  kustomize build | kubectl apply -f -

  kubectl wait --for condition=established --timeout=60s crd/applications.app.k8s.io
  popd

  pushd ${KFP_MANIFEST_DIR}/dev
  kustomize build | kubectl apply -f -
  popd

  # go back to previous commit
  git checkout -
fi

# show current info
echo "Status of pods after kubectl apply"
kubectl get pods -n ${NAMESPACE}

# wait for all deployments to be successful
# note, after we introduce statefulset and daemonsets, we need to wait their rollout status here too
for deployment in $(kubectl get deployments -n ${NAMESPACE} -o name)
do
  kubectl rollout status $deployment -n ${NAMESPACE}
done

echo "Status of pods after rollouts are successful"
kubectl get pods -n ${NAMESPACE}

if [ "$ENABLE_WORKLOAD_IDENTITY" = true ]; then
  # Use static GSAs for testing, so we don't need to GC them.
  export SYSTEM_GSA="test-kfp-system"
  export USER_GSA="test-kfp-user"
  source "${DIR}/scripts/retry.sh"

  function setup_workload_identity {
    # Workaround for flakiness from gcp-workload-identity-setup.sh:
    # When two tests add iam policy bindings at the same time, one will fail because
    # there could be two concurrent changes.
    # Wait here randomly to reduce chance both scripts are run at the same time
    # between tests. Unless for testing scenario like this, it won't
    # meet the concurrent change issue.
    sleep $((RANDOM%30))
    yes | PROJECT_ID=$PROJECT RESOURCE_PREFIX=$TEST_CLUSTER NAMESPACE=$NAMESPACE \
      ${DIR}/../manifests/kustomize/gcp-workload-identity-setup.sh
  }
  retry setup_workload_identity

  retry gcloud projects add-iam-policy-binding $PROJECT \
    --member="serviceAccount:$SYSTEM_GSA@$PROJECT.iam.gserviceaccount.com" \
    --role="roles/editor"
  retry gcloud projects add-iam-policy-binding $PROJECT \
    --member="serviceAccount:$USER_GSA@$PROJECT.iam.gserviceaccount.com" \
    --role="roles/editor"

  source "$DIR/../manifests/kustomize/wi-utils.sh"
  # TODO(Bobgy): re-enable this after temporary flakiness is resolved.
  # verify_workload_identity_binding "pipeline-runner" $NAMESPACE
  sleep 30
fi

popd
