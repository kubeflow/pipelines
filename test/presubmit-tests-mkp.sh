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

# This file is forked from presubmit-test-with-pipoeline-deployment.sh
# as the test can't be run inside Argo.

# This is merged commit's SHA.
COMMIT_SHA="$(git rev-parse HEAD)"
GCR_IMAGE_BASE_DIR=gcr.io/${PROJECT}/presubmit/${COMMIT_SHA}
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"

# Configure `time` command output format.
TIMEFORMAT="[test-timing] It took %lR."

echo "presubmit test starts"
if [ -n "$PULL_PULL_SHA" ]; then
    echo "PR commit is ${PULL_PULL_SHA}"
fi
time source "${DIR}/test-prep.sh"
echo "test env prepared"

# We don't wait for image building here, because cluster can be deployed in
# parallel so that we save a few minutes of test time.
time source "${DIR}/build-images.sh"
echo "KFP images cloudbuild jobs submitted"

time source "${DIR}/deploy-cluster.sh"
echo "cluster deployed"

time source "${DIR}/check-build-image-status.sh"
echo "KFP images built"

