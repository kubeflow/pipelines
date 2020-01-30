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

function deploy_bucket() {
  # Helper function to deploy bucket with a unique name. The unique name is ${BASE_NAME} + random
  # unique string.
  #
  # Usage:
  # deploy_bucket BASE_NAME NUM_RETRIES
  BASE_NAME=$1
  NUM_RETRIES=$2
  for i in $(seq 1 ${NUM_RETRIES})
  do
    bucket_is_set=true
    bucket_name="${BASE_NAME}-$(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 10 | head -n 1)"
    gsutil mb "gs://${bucket_name}/" || bucket_is_set=false
    if [ ! "$bucket_is_set" = true ]; then
      continue
    fi
    # Populate configmap, with name gcs-config
    kubectl create configmap -n "${NAMESPACE}" "gcs-config" --from-literal bucket_name="${bucket_name}"
    break
  done
  if [ ! "$bucket_is_set" = true ]; then
    echo "Cannot successfully create bucket after ${NUM_RETRIES} attempts."
    return 1
  fi
}

# Helper script for auto-provision bucket in KFP MKP deployment.
NAME="$(/bin/print_config.py \
    --xtype NAME \
    --values_mode raw)"
NAMESPACE="$(/bin/print_config.py \
    --xtype NAMESPACE \
    --values_mode raw)"
export NAME
export NAMESPACE

deploy_bucket "${NAME}-default" 10

# Invoke normal deployer routine.
/bin/bash /bin/core_deploy.sh