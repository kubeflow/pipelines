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

if ! which kustomize; then
  # Download kustomize cli tool
  TOOL_DIR=${DIR}/bin
  mkdir -p ${TOOL_DIR}
  wget https://github.com/kubernetes-sigs/kustomize/releases/download/v3.1.0/kustomize_3.1.0_linux_amd64 -O ${TOOL_DIR}/kustomize
  chmod +x ${TOOL_DIR}/kustomize
  PATH=${PATH}:${TOOL_DIR}
fi

# delete argo first because KFP comes with argo too
kubectl delete namespace argo --wait || echo "No argo installed"

KFP_MANIFEST_DIR=${DIR}/manifests
pushd ${KFP_MANIFEST_DIR}

# This is the recommended approach to do this.
# reference: https://github.com/kubernetes-sigs/kustomize/blob/master/docs/eschewedFeatures.md#build-time-side-effects-from-cli-args-or-env-variables
kustomize edit set image gcr.io/ml-pipeline/api-server=${GCR_IMAGE_BASE_DIR}/api-server:latest
kustomize edit set image gcr.io/ml-pipeline/persistenceagent=${GCR_IMAGE_BASE_DIR}/persistenceagent:latest
kustomize edit set image gcr.io/ml-pipeline/scheduledworkflow=${GCR_IMAGE_BASE_DIR}/scheduledworkflow:latest
kustomize edit set image gcr.io/ml-pipeline/frontend=${GCR_IMAGE_BASE_DIR}/frontend:latest
cat kustomization.yaml

kustomize build . | kubectl apply -f -
# show current info
echo "Status of pods after kubectl apply"
kubectl get pods -n ${NAMESPACE}

popd
