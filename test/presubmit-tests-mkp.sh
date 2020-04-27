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

set -ex

echo "Presubmit test with Hosted/MKP. It's also runable before sending out PR"

if [[ ! -z "${GOOGLE_APPLICATION_CREDENTIALS}" ]]; then
  # activating the service account
  gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"
fi

# Build all required images used for Hosted (sync style)
echo "Building, it may take ~30 minutes"
echo "Images can be found in gcr.io/ml-pipeline-test/hosted/$(git rev-parse HEAD)/"
gcloud builds submit --config=.cloudbuild.yaml --substitutions=COMMIT_SHA="$(git rev-parse HEAD)" --project=ml-pipeline-test

# Install & uninstsall
echo "Verify hosted images, it may take ~10 minutes"
MM_VER=$(cat VERSION | sed -e "s#[^0-9]*\([0-9]*\)[.]\([0-9]*\)[.]\([0-9]*\)#\1.\2#")
gcloud builds submit --config=test/cloudbuild/mkp_verify.yaml --substitutions=COMMIT_SHA="$(git rev-parse HEAD)",_DEPLOYER_VERSION=$MM_VER --project=ml-pipeline-test

echo "Well done!"
