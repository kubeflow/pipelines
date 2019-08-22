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
  echo "$BUILT_IMAGES" | grep persistenceagent;
then
  echo "docker images for api-server, frontend, scheduledworkflow and \
    persistenceagent are already built in ${GCR_IMAGE_BASE_DIR}."
else
  echo "submitting cloud build to build docker images for commit ${PULL_PULL_SHA}..."
  IMAGES_BUILDING=true
  CLOUD_BUILD_COMMON_ARGS=(. --async --format='value(id)' --substitutions=_GCR_BASE=${GCR_IMAGE_BASE_DIR})
  # Use faster machine because this is CPU intensive
  BUILD_ID_API_SERVER=$(gcloud builds submit ${CLOUD_BUILD_COMMON_ARGS[@]} \
    --config ${DIR}/cloudbuild/api_server.yaml)
  BUILD_ID_FRONTEND=$(gcloud builds submit ${CLOUD_BUILD_COMMON_ARGS[@]} \
    --config ${DIR}/cloudbuild/frontend.yaml)
  BUILD_ID_SCHEDULED_WORKFLOW=$(gcloud builds submit ${CLOUD_BUILD_COMMON_ARGS[@]} \
    --config ${DIR}/cloudbuild/scheduled_workflow.yaml)
  BUILD_ID_PERSISTENCE_AGENT=$(gcloud builds submit ${CLOUD_BUILD_COMMON_ARGS[@]} \
    --config ${DIR}/cloudbuild/persistence_agent.yaml)

  BUILD_IDS=(
    "${BUILD_ID_API_SERVER}"
    "${BUILD_ID_FRONTEND}"
    "${BUILD_ID_SCHEDULED_WORKFLOW}"
    "${BUILD_ID_PERSISTENCE_AGENT}"
  )
  echo "Submitted the following cloud build jobs: ${BUILD_IDS[@]}"
fi
