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

IMAGE_BUILDER_ARG=""
if [ "$PROJECT" != "ml-pipeline-test" ]; then
  COPIED_IMAGE_BUILDER_IMAGE=${GCR_IMAGE_BASE_DIR}/image-builder
  echo "Copy image builder image to ${COPIED_IMAGE_BUILDER_IMAGE}"
  yes | gcloud container images add-tag \
    gcr.io/ml-pipeline-test/image-builder:v20181128-0.1.3-rc.1-109-ga5a14dc-e3b0c4 \
    ${COPIED_IMAGE_BUILDER_IMAGE}:latest
  IMAGE_BUILDER_ARG="-p image-builder-image=${COPIED_IMAGE_BUILDER_IMAGE}"
fi

# Image caching can be turned off by setting $DISABLE_IMAGE_CACHING env flag.
# Note that GCR_IMAGE_BASE_DIR contains commit hash, so whenever there's a code
# change, we won't use caches for sure.
BUILT_IMAGES=$(gcloud container images list --repository=${GCR_IMAGE_BASE_DIR})
if
  test -z "$DISABLE_IMAGE_CACHING" && \
  echo "$BUILT_IMAGES" | grep api-server && \
  echo "$BUILT_IMAGES" | grep frontend && \
  echo "$BUILT_IMAGES" | grep scheduledworkflow && \
  echo "$BUILT_IMAGES" | grep persistenceagent;
then
  echo "docker images for api-server, frontend, scheduledworkflow and \
    persistenceagent are already built in ${GCR_IMAGE_BASE_DIR}."
else
  echo "submitting argo workflow to build docker images for commit ${PULL_PULL_SHA}..."
  # Build Images
  ARGO_WORKFLOW=`argo submit ${DIR}/build_image.yaml \
  -p image-build-context-gcs-uri="$remote_code_archive_uri" \
  ${IMAGE_BUILDER_ARG} \
  -p api-image="${GCR_IMAGE_BASE_DIR}/api-server" \
  -p frontend-image="${GCR_IMAGE_BASE_DIR}/frontend" \
  -p scheduledworkflow-image="${GCR_IMAGE_BASE_DIR}/scheduledworkflow" \
  -p persistenceagent-image="${GCR_IMAGE_BASE_DIR}/persistenceagent" \
  -n ${NAMESPACE} \
  --serviceaccount test-runner \
  -o name
  `
  echo "build docker images workflow submitted successfully"
  source "${DIR}/check-argo-status.sh"
  echo "build docker images workflow completed"
fi
