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
# * ENABLE_WORKLOAD_IDENTITY
GCR_IMAGE_TAG=${GCR_IMAGE_TAG:-latest}
ENABLE_WORKLOAD_IDENTITY=${ENABLE_WORKLOAD_IDENTITY:-false}

if ! which kustomize; then
  # Download kustomize cli tool
  TOOL_DIR=${DIR}/bin
  mkdir -p ${TOOL_DIR}
  wget --no-verbose https://github.com/kubernetes-sigs/kustomize/releases/download/v3.1.0/kustomize_3.1.0_linux_amd64 \
    -O ${TOOL_DIR}/kustomize --no-verbose
  chmod +x ${TOOL_DIR}/kustomize
  PATH=${PATH}:${TOOL_DIR}
fi

# delete argo first because KFP comes with argo too
kubectl delete namespace argo --wait || echo "No argo installed"

KFP_MANIFEST_DIR=${DIR}/manifests

pushd ${KFP_MANIFEST_DIR}/crd
kustomize build . | kubectl apply -f -
popd

kubectl wait --for condition=established --timeout=60s crd/applications.app.k8s.io

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
kustomize edit set image gcr.io/ml-pipeline-test/cache-server=${GCR_IMAGE_BASE_DIR}/cache-server:${GCR_IMAGE_TAG}
kustomize edit set image gcr.io/ml-pipeline-test/cache-deployer=${GCR_IMAGE_BASE_DIR}/cache-deployer:${GCR_IMAGE_TAG}
cat kustomization.yaml

kustomize build . | kubectl apply -f -

# show current info
echo "Status of pods after kubectl apply"
kubectl get pods -n ${NAMESPACE}

if [ "$ENABLE_WORKLOAD_IDENTITY" = true ]; then
  # Use static GSAs for testing, so we don't need to GC them.
  export SYSTEM_GSA="test-kfp-system"
  export USER_GSA="test-kfp-user"

  # Workaround for flakiness from gcp-workload-identity-setup.sh:
  # When two tests add iam policy bindings at the same time, one will fail because
  # there could be two concurrent changes.
  # Wait here randomly to reduce chance both scripts are run at the same time
  # between tests. gcp-workload-identity-setup.sh is user facing, we'd better
  # not add retry there. Also unless for testing scenario like this, it won't
  # meet the concurrent change issue.
  sleep $((RANDOM%30))
  yes | PROJECT_ID=$PROJECT CLUSTER_NAME=$TEST_CLUSTER NAMESPACE=$NAMESPACE \
    ${DIR}/../manifests/kustomize/gcp-workload-identity-setup.sh

  source "${DIR}/scripts/retry.sh"
  retry gcloud projects add-iam-policy-binding $PROJECT \
    --member="serviceAccount:$SYSTEM_GSA@$PROJECT.iam.gserviceaccount.com" \
    --role="roles/editor"
  retry gcloud projects add-iam-policy-binding $PROJECT \
    --member="serviceAccount:$USER_GSA@$PROJECT.iam.gserviceaccount.com" \
    --role="roles/editor"

  source "$DIR/../manifests/kustomize/wi-utils.sh"
  verify_workload_identity_binding "pipeline-runner" $NAMESPACE
fi

popd
