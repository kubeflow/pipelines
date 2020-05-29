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
  # Helper function to deploy bucket with a unique name.
  # Also detect the current GCP project ID and populate the properties into a
  # config map.
  #
  # Usage:
  # set_bucket_and_configmap
  CONFIG_NAME="gcp-default-config"

  # Detect GCP project
  GCP_PROJECT_ID=$(curl -H "Metadata-Flavor: Google" -w '\n' "http://metadata.google.internal/computeMetadata/v1/project/project-id")

  # Check whether ConfigMap is already exist
  if kubectl get configmap ${CONFIG_NAME}; then
    echo "Already has a configmap map there"
    return 0
  fi

  bucket_name="${GCP_PROJECT_ID}-kubeflowpipelines-default"
  bucket_is_set=true
  gsutil ls gs://${bucket_name} || bucket_is_set=false
  if [ "$bucket_is_set" = false ]; then
    bucket_is_set=true
    gsutil mb -p ${GCP_PROJECT_ID} "gs://${bucket_name}/" || bucket_is_set=false
  fi

  # Populate configmap, with name gcp-default-config
  if [ "${bucket_is_set}" = true ]; then
    kubectl create configmap -n "${NAMESPACE}" "${CONFIG_NAME}" \
      --from-literal bucket_name="${bucket_name}" \
      --from-literal has_default_bucket="true" \
      --from-literal project_id="${GCP_PROJECT_ID}"
  else
    echo "Cannot successfully create bucket. Fall back to not specifying default bucket."
    kubectl create configmap -n "${NAMESPACE}" "${CONFIG_NAME}" \
      --from-literal bucket_name="<your-bucket>" \
      --from-literal has_default_bucket="false" \
      --from-literal project_id="${GCP_PROJECT_ID}"
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

set_bucket_and_configmap

echo "init_action done"
