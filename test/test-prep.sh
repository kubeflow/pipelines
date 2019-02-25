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

# activating the service account
gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"
gcloud config set compute/zone us-east1-b
gcloud config set core/project ${PROJECT}

#Uploading the source code to GCS:
local_code_archive_file=$(mktemp)
date_string=$(TZ=PST8PDT date +%Y-%m-%d_%H-%M-%S_%Z)
code_archive_prefix="gs://${TEST_RESULT_BUCKET}/${PULL_BASE_SHA}/source_code"
remote_code_archive_uri="${code_archive_prefix}_${PULL_BASE_SHA}_${date_string}.tar.gz"

tar -czf "$local_code_archive_file" .
gsutil cp "$local_code_archive_file" "$remote_code_archive_uri"

TEST_CLUSTER_PREFIX=${WORKFLOW_FILE%.*}
TEST_CLUSTER=$(echo $TEST_CLUSTER_PREFIX | cut -d _ -f 1)-${PULL_BASE_SHA:0:7}-${RANDOM}
