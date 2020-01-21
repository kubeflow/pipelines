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

IMAGES_BUILDING=false

# Image caching can be turned off by setting $DISABLE_IMAGE_CACHING env flag.
# Note that GCR_IMAGE_BASE_DIR contains commit hash, so whenever there's a code
# change, we won't use caches for sure.
BUILT_IMAGES=$(gcloud container images list --repository=${GCR_IMAGE_BASE_DIR})
if
  test -z "$DISABLE_IMAGE_CACHING" && \
  echo "$BUILT_IMAGES" | grep api-server && \
  echo "$BUILT_IMAGES" | grep frontend && \
  echo "$BUILT_IMAGES" | grep scheduledworkflow && \
  echo "$BUILT_IMAGES" | grep persistenceagent && \
  echo "$BUILT_IMAGES" | grep viewer-crd-controller && \
  echo "$BUILT_IMAGES" | grep inverse-proxy-agent && \
  echo "$BUILT_IMAGES" | grep metadata-writer && \
  echo "$BUILT_IMAGES" | grep visualization-server;
then
  echo "docker images for api-server, frontend, scheduledworkflow, \
    persistenceagent, viewer-crd-controller, inverse-proxy-agent, metadata-writer, and visualization-server \
    are already built in ${GCR_IMAGE_BASE_DIR}."
else
  echo "submitting cloud build to build docker images for commit ${COMMIT_SHA}..."
  IMAGES_BUILDING=true
  CLOUD_BUILD_COMMON_ARGS=(. --async --format='value(id)' --substitutions=_GCR_BASE=${GCR_IMAGE_BASE_DIR})
  # Split into two tasks because api_server builds slowly, use a separate task
  # to make it faster.
  BUILD_ID_API_SERVER=$(gcloud builds submit ${CLOUD_BUILD_COMMON_ARGS[@]} \
    --config ${DIR}/cloudbuild/api_server.yaml)
  BUILD_ID_BATCH=$(gcloud builds submit ${CLOUD_BUILD_COMMON_ARGS[@]} \
    --config ${DIR}/cloudbuild/batch_build.yaml)

  BUILD_IDS=(
    "${BUILD_ID_API_SERVER}"
    "${BUILD_ID_BATCH}"
  )
  echo "Submitted the following cloud build jobs: ${BUILD_IDS[@]}"
fi
