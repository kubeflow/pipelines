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

function has_images_been_built {
  local images=$(gcloud container images list --repository=${GCR_IMAGE_BASE_DIR})
  local result=1
  echo "$images" | grep api-server && \
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

# Image caching can be turned off by setting $DISABLE_IMAGE_CACHING env flag.
# Note that GCR_IMAGE_BASE_DIR contains commit hash, so whenever there's a code
# change, we won't use caches for sure.
if
  test -z "$DISABLE_IMAGE_CACHING" && has_images_been_built
then
  echo "docker images for api-server, frontend, scheduledworkflow, \
    persistenceagent, viewer-crd-controller, inverse-proxy-agent, metadata-writer, cache-server, \
    cache-deployer and visualization-server are already built in ${GCR_IMAGE_BASE_DIR}."
else
  # sleep randomly to reduce chance of submitting two cloudbuild jobs at the same time
  sleep $((RANDOM%30))

  # The object name is inited as ${TEST_RESULTS_GCS_DIR} below, so it always has ${COMMIT_SHA} in it.
  ongoing_build_ids=($(gcloud builds list --filter='source.storageSource.object~'${COMMIT_SHA}' AND (status=QUEUED OR status=WORKING)' --format='value(id)'))
  if [ "${#ongoing_build_ids[@]}" -gt "0" ]; then
    echo "There is an existing cloud build job, wait for it: id=${ongoing_build_ids[0]}"
    IMAGES_BUILDING=true
    BUILD_IDS=(${ongoing_build_ids[0]})
  else
    echo "submitting cloud build to build docker images for commit ${COMMIT_SHA}..."
    IMAGES_BUILDING=true
    CLOUD_BUILD_COMMON_ARGS=( \
      $remote_code_archive_uri \
      --async \
      --format='value(id)' \
      --substitutions=_GCR_BASE=${GCR_IMAGE_BASE_DIR} \
    )
    BUILD_ID_BATCH=$(gcloud builds submit ${CLOUD_BUILD_COMMON_ARGS[@]} \
      --config ${DIR}/cloudbuild/batch_build.yaml \
      --gcs-source-staging-dir ${TEST_RESULTS_GCS_DIR}/cloudbuild)

    BUILD_IDS=("${BUILD_ID_BATCH}")
    echo "Submitted the following cloud build jobs: ${BUILD_IDS[@]}"
  fi
fi
