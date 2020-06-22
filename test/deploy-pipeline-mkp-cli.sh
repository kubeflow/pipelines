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


# Test before submit:
# test/deploy-pipline-mkp-cli.sh 0.3 $(git rev-parse HEAD) $(pwd)/test

set -ex

VERSION=$1
COMMIT_SHA=$2
TEST_FOLDER=$3

# sync trigger to avoid wait
gcloud builds submit --config=$TEST_FOLDER/cloudbuild/mkp_verify.yaml --substitutions=_DEPLOYER_VERSION="$VERSION",COMMIT_SHA="$COMMIT_SHA" --project=ml-pipeline-test