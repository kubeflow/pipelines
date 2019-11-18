#!/bin/bash
#
# Copyright 2019 Google LLC
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

function create_gsa_if_not_present {
  local name=${1}
  local already_present=$(gcloud iam service-accounts list --filter='name:'$name'' --format='value(name)')
  if [ -n "$already_present" ]; then
    echo "Service account $name already exists"
  else
    gcloud iam service-accounts create $name
  fi
}

# Bind KSA to GSA through workload identity.
# Documentation: https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity
function bind_gsa_and_ksa {
  local gsa=${1}
  local ksa=${2}
  local project=${3:-$PROJECT_ID}
  local gsa_full="$gsa@$project.iam.gserviceaccount.com"
  local namespace=${4:-$NAMESPACE}

  gcloud iam service-accounts add-iam-policy-binding $gsa_full \
    --member="serviceAccount:$project.svc.id.goog[$namespace/$ksa]" \
    --role="roles/iam.workloadIdentityUser" \
    > /dev/null # hide verbose output
  kubectl annotate serviceaccount \
    --namespace $namespace \
    --overwrite \
    $ksa \
    iam.gke.io/gcp-service-account=$gsa_full
  echo "* Bound KSA $ksa in namespace $namespace to GSA $gsa_full"
}
