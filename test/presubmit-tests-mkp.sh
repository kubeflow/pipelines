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

# TODO: extract this part to be a dedicated file for reusage.
# =========================================================================
## Wait for the cloudbuild job to be started (triggerd by PR commit)
CLOUDBUILD_TIMEOUT_SECONDS=3600
PULL_CLOUDBUILD_STATUS_MAX_ATTEMPT=$(expr ${CLOUDBUILD_TIMEOUT_SECONDS} / 20 )
CLOUDBUILD_STARTED=False

for i in $(seq 1 ${PULL_CLOUDBUILD_STATUS_MAX_ATTEMPT})
do
  output=`gcloud builds list --project="$CLOUDBUILD_PROJECT" --filter="sourceProvenance.resolvedRepoSource.commitSha:${COMMIT_SHA}"`
  if [[ ${output} != "" ]]; then
    CLOUDBUILD_STARTED=True
    break
  fi
  sleep 20
done

if [[ ${CLOUDBUILD_STARTED} == False ]];then
  echo "CloudBuild job for the commit not be triggered before timeout, exiting..."
  exit 1
fi

## Wait for the cloudbuild job to complete, if success, all images ready
CLOUDBUILD_FINISHED=TIMEOUT
for i in $(seq 1 ${PULL_CLOUDBUILD_STATUS_MAX_ATTEMPT})
do
  output=`gcloud builds list --project="$CLOUDBUILD_PROJECT" --filter="sourceProvenance.resolvedRepoSource.commitSha:${COMMIT_SHA}"`
  if [[ ${output} == *"SUCCESS"* ]]; then
    CLOUDBUILD_FINISHED=SUCCESS
    break
  elif [[ ${output} == *"FAILURE"* ]]; then
    CLOUDBUILD_FINISHED=FAILURE
    break
  fi
  sleep 20
done

if [[ ${CLOUDBUILD_FINISHED} == FAILURE ]];then
  echo "Cloud build failure, cannot proceed. exiting..."
  exit 1
elif [[ ${CLOUDBUILD_FINISHED} == TIMEOUT ]];then
  echo "Cloud build timeout, cannot proceed. exiting..."
  exit 1
fi

# =========================================================================
## Create Cluster (delete in clean up)

# =========================================================================
## All images ready, trigger the MKP verification. Here we can't use Argo and have to use CloudBuild again
## Run CB job in sync mode to easily get test result.

## TODO
## 
# gcloud builds submit --config=test/cloudbuild/mkp_verify.yaml --substitutions=_MM_VERSION="0.2" --project=ml-pipeline-test

## ??? how test cluster get deleted?