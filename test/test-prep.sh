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

set -x

ZONE=${ZONE:-us-east1-b}
if [[ ! -z "${GOOGLE_APPLICATION_CREDENTIALS}" ]]; then
  # activating the service account
  gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"
fi
gcloud config set compute/zone ${ZONE}
gcloud config set core/project ${PROJECT}

#Uploading the source code to GCS:
local_code_archive_file=$(mktemp)
date_string=$(TZ=PST8PDT date +%Y-%m-%d_%H-%M-%S_%Z)
code_archive_prefix="${TEST_RESULTS_GCS_DIR}/source_code"
remote_code_archive_uri="${code_archive_prefix}_${PULL_BASE_SHA}_${date_string}.tar.gz"

tar -czf "$local_code_archive_file" .
gsutil cp "$local_code_archive_file" "$remote_code_archive_uri"
