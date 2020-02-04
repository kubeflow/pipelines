#!/bin/bash
#
# Copyright 2020 Google LLC
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

function set_bucket_and_configmap() {
  # Helper function to deploy bucket with a unique name. The unique name is ${BASE_NAME} + random
  # unique string. Also detect the current GCP project ID and populate the properties into a
  # config map.
  #
  # Usage:
  # set_bucket_and_configmap BASE_NAME NUM_RETRIES
  BASE_NAME=$1
  NUM_RETRIES=$2
  CONFIG_NAME="gcp-default-config"

  # Detect GCP project
  GCP_PROJECT_ID=$(curl -H "Metadata-Flavor: Google" -w '\n' "http://metadata.google.internal/computeMetadata/v1/project/project-id")

  for i in $(seq 1 ${NUM_RETRIES})
  do
    bucket_is_set=true
    bucket_name="${BASE_NAME}-$(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 10 | head -n 1)"
    gsutil mb -p ${GCP_PROJECT_ID} "gs://${bucket_name}/" || bucket_is_set=false
    if [ "$bucket_is_set" = true ]; then
      break
    fi
  done
  
  # Update value of configmap gcp-default-config
  PATCH_TEMP='{"data": {"bucket_name":"'${bucket_name}'","has_default_bucket":"'${bucket_is_set}'","project_id":"'${GCP_PROJECT_ID}'"}}'
  PATCH_JSON=$(printf "${PATCH_TEMP}" "${bucket_name}" "${bucket_is_set}" "${GCP_PROJECT_ID}")
  echo "PACTH_JSON: ${PATCH_JSON}"

  kubectl patch configmap/gcp-default-config \
    --type merge \
    --patch "${PATCH_JSON}"

  echo "Patched configmap/gcp-default-config"
}

# Helper script for auto-provision bucket in KFP MKP deployment.
set_bucket_and_configmap "${NAME}-default" 10