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

# Env inputs:
# * $GCR_IMAGE_BASE_DIR
# * $GCR_IMAGE_TAG
# * $KFP_DEPLOY_RELEASE
GCR_IMAGE_TAG=${GCR_IMAGE_TAG:-latest}

KFP_MANIFEST_DIR="${DIR}/manifests"
pushd ${KFP_MANIFEST_DIR}

if ! which kustomize; then
  # Download kustomize cli tool
  TOOL_DIR=${DIR}/bin
  mkdir -p ${TOOL_DIR}
  wget --no-verbose https://github.com/kubernetes-sigs/kustomize/releases/download/v3.1.0/kustomize_3.1.0_linux_amd64 \
    -O ${TOOL_DIR}/kustomize --no-verbose
  chmod +x ${TOOL_DIR}/kustomize
  PATH=${PATH}:${TOOL_DIR}
fi

if [ -z "$KFP_DEPLOY_RELEASE" ]; then
  echo "Deploying KFP in working directory..."

  # This is the recommended approach to do this.
  # reference: https://github.com/kubernetes-sigs/kustomize/blob/master/docs/eschewedFeatures.md#build-time-side-effects-from-cli-args-or-env-variables
  kustomize edit set image gcr.io/ml-pipeline/api-server=${GCR_IMAGE_BASE_DIR}/api-server:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/persistenceagent=${GCR_IMAGE_BASE_DIR}/persistenceagent:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/scheduledworkflow=${GCR_IMAGE_BASE_DIR}/scheduledworkflow:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/frontend=${GCR_IMAGE_BASE_DIR}/frontend:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/viewer-crd-controller=${GCR_IMAGE_BASE_DIR}/viewer-crd-controller:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/visualization-server=${GCR_IMAGE_BASE_DIR}/visualization-server:${GCR_IMAGE_TAG}
  kustomize edit set image gcr.io/ml-pipeline/inverse-proxy-agent=${GCR_IMAGE_BASE_DIR}/inverse-proxy-agent:${GCR_IMAGE_TAG}
  cat kustomization.yaml

  kustomize build . | kubectl apply -f -
else
  KFP_LATEST_RELEASE=$(git tag --sort=v:refname | tail -1)
  echo "Deploying KFP release $KFP_LATEST_RELEASE"

  # temporarily checkout last release tag
  git checkout $KFP_LATEST_RELEASE

  kustomize build . | kubectl apply -f -

  # go back to previous commit
  git checkout -
fi

# show current info
echo "Status of pods after kubectl apply"
kubectl get pods -n ${NAMESPACE}

# wait for all deployments to be successful
# note, after we introduce statefulsets or daemonsets, we need to wait their rollout status here too
for deployment in $(kubectl get deployments -n ${NAMESPACE} -o name)
do
  kubectl rollout status $deployment -n ${NAMESPACE}
done

echo "Status of pods after rollouts are successful"
kubectl get pods -n ${NAMESPACE}

popd
