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

set -ex
set -o pipefail

usage()
{
    echo "usage: deploy.sh
    [--workflow_file        the file name of the argo workflow to run]
    [--test_result_bucket   the gcs bucket that argo workflow store the result to. Default is ml-pipeline-test
    [--test_result_folder   the gcs folder that argo workflow store the result to. Always a relative directory to gs://<gs_bucket>/[PULL_SHA]]
    [--timeout              timeout of the tests in seconds. Default is 1800 seconds. ]
    [-h help]"
}

PROJECT=ml-pipeline-test
TEST_RESULT_BUCKET=ml-pipeline-test
CLOUDBUILD_PROJECT=ml-pipeline-test
GCR_IMAGE_BASE_DIR=gcr.io/ml-pipeline-test
TARGET_IMAGE_BASE_DIR=gcr.io/ml-pipeline-test/${PULL_BASE_SHA}
TIMEOUT_SECONDS=1800
NAMESPACE=kubeflow
COMMIT_SHA="$PULL_BASE_SHA"

while [ "$1" != "" ]; do
    case $1 in
             --workflow_file )        shift
                                      WORKFLOW_FILE=$1
                                      ;;
             --test_result_bucket )   shift
                                      TEST_RESULT_BUCKET=$1
                                      ;;
             --test_result_folder )   shift
                                      TEST_RESULT_FOLDER=$1
                                      ;;
             --timeout )              shift
                                      TIMEOUT_SECONDS=$1
                                      ;;
             -h | --help )            usage
                                      exit
                                      ;;
             * )                      usage
                                      exit 1
    esac
    shift
done

# Variables
# Refer to https://github.com/kubernetes/test-infra/blob/e357ffaaeceafe737bd6ab89d2feff132d92ea50/prow/jobs.md for the Prow job environment variables
TEST_RESULTS_GCS_DIR=gs://${TEST_RESULT_BUCKET}/${PULL_BASE_SHA}/${TEST_RESULT_FOLDER}
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"

# Configure `time` command output format.
TIMEFORMAT="[test-timing] It took %lR."

time source "${DIR}/test-prep.sh"

TEST_CLUSTER_PREFIX=${WORKFLOW_FILE%.*}
TEST_CLUSTER_DEFAULT=$(echo $TEST_CLUSTER_PREFIX | cut -d _ -f 1)-${COMMIT_SHA:0:7}-${RANDOM}
TEST_CLUSTER=${TEST_CLUSTER:-${TEST_CLUSTER_DEFAULT}}

time source "${DIR}/subscripts/deploy-multi-user-kfp.sh"
echo "multi user kfp deployed"

time source "${DIR}/install-argo.sh"
echo "argo installed"

# Submit the argo job and check the results
echo "submitting argo workflow for commit ${PULL_BASE_SHA}..."
ARGO_WORKFLOW=`argo submit ${DIR}/${WORKFLOW_FILE} \
-p image-build-context-gcs-uri="$remote_code_archive_uri" \
-p commit-sha="${PULL_BASE_SHA}" \
-p component-image-prefix="${GCR_IMAGE_BASE_DIR}/" \
-p target-image-prefix="${TARGET_IMAGE_BASE_DIR}/" \
-p test-results-gcs-dir="${TEST_RESULTS_GCS_DIR}" \
-p cluster-type="${CLUSTER_TYPE}" \
-n ${NAMESPACE} \
--serviceaccount test-runner \
-o name
`
echo "argo workflow submitted successfully"
source "${DIR}/check-argo-status.sh"
echo "test workflow completed"
