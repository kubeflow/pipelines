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

function has_batch_images_been_built {
  local images=$(gcloud container images list --repository=${GCR_IMAGE_BASE_DIR})
  local result=1
  echo "$images" | grep frontend && \
  echo "$images" | grep scheduledworkflow && \
  echo "$images" | grep persistenceagent && \
  echo "$images" | grep viewer-crd-controller && \
  echo "$images" | grep inverse-proxy-agent && \
  echo "$images" | grep metadata-writer && \
  echo "$images" | grep cache-server && \
  echo "$images" | grep cache-deployer && \
  echo "$images" | grep visualization-server && result=0
  return $result
}

function has_api_server_image_been_built {
  local images=$(gcloud container images list --repository=${GCR_IMAGE_BASE_DIR})
  local result=1
  echo "$images" | grep api-server && result=0
  return $result
}

BUILD_IDS=()
function build_image {
  local build_target=$1
  # sleep randomly to reduce chance of submitting two cloudbuild jobs at the same time
  sleep $((RANDOM%30))

  # The object name is inited as ${TEST_RESULTS_GCS_DIR} below, so it always has ${COMMIT_SHA} in it.
  local ongoing_build_ids=($(gcloud builds list \
    --filter='source.storageSource.object~'${COMMIT_SHA}.*/${build_target}' AND (status=QUEUED OR status=WORKING)' \
    --format='value(id)'))
  if [ "${#ongoing_build_ids[@]}" -gt "0" ]; then
    echo "There is an existing cloud build job for ${build_target}, wait for it: id=${ongoing_build_ids[0]}"
    BUILD_IDS+=("${ongoing_build_ids[0]}")
  else
    echo "submitting cloud build to build docker images for commit ${COMMIT_SHA} and target ${build_target}..."
    local common_args=( \
      $remote_code_archive_uri \
      --async \
      --format='value(id)' \
      --substitutions=_GCR_BASE=${GCR_IMAGE_BASE_DIR} \
    )
    local build_id=$(gcloud builds submit ${common_args[@]} \
      --config ${DIR}/cloudbuild/${build_target}.yaml \
      --gcs-source-staging-dir "${TEST_RESULTS_GCS_DIR}/cloudbuild/${build_target}")

    BUILD_IDS+=("${build_id}")
    echo "Submitted ${build_target} cloud build job: ${build_id}"
  fi
}

# Image caching can be turned off by setting $DISABLE_IMAGE_CACHING env flag.
# Note that GCR_IMAGE_BASE_DIR contains commit hash, so whenever there's a code
# change, we won't use caches for sure.
#
# Split into two tasks because api_server builds slowly, use a separate task to speed up.
if
  test -z "$DISABLE_IMAGE_CACHING" && has_batch_images_been_built
then
  echo "docker images for frontend, scheduledworkflow, \
    persistenceagent, viewer-crd-controller, inverse-proxy-agent, metadata-writer, cache-server, \
    cache-deployer and visualization-server are already built in ${GCR_IMAGE_BASE_DIR}."
else
  IMAGES_BUILDING=true
  build_image "batch_build"
fi

if
  test -z "$DISABLE_IMAGE_CACHING" && has_api_server_image_been_built
then
  echo "docker image for api-server is already built in ${GCR_IMAGE_BASE_DIR}."
else
  IMAGES_BUILDING=true
  build_image "api_server"
fi
