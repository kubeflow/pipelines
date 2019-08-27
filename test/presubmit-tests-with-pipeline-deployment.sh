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

set -ex

usage()
{
    echo "usage: deploy.sh
    [--platform             the deployment platform. Valid values are: [gcp, minikube]. Default is gcp.]
    [--project              the gcp project. Default is ml-pipeline-test. Only used when platform is gcp.]
    [--workflow_file        the file name of the argo workflow to run]
    [--test_result_bucket   the gcs bucket that argo workflow store the result to. Default is ml-pipeline-test
    [--test_result_folder   the gcs folder that argo workflow store the result to. Always a relative directory to gs://<gs_bucket>/[PULL_SHA]]
    [--timeout              timeout of the tests in seconds. Default is 1800 seconds. ]
    [-h help]"
}

PLATFORM=gcp
PROJECT=ml-pipeline-test
TEST_RESULT_BUCKET=ml-pipeline-test
TIMEOUT_SECONDS=1800
NAMESPACE=kubeflow

while [ "$1" != "" ]; do
    case $1 in
             --platform )             shift
                                      PLATFORM=$1
                                      ;;
             --project )              shift
                                      PROJECT=$1
                                      ;;
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

# PULL_PULL_SHA is empty whne Pros/Tide tests the batches.
# PULL_BASE_SHA cannot be used here as it still points to master tip in that case.
if [ -z "${PULL_PULL_SHA:-''}" ]; then
    PULL_PULL_SHA=$(git rev-parse HEAD)
fi

# Variables
GCR_IMAGE_BASE_DIR=gcr.io/${PROJECT}/${PULL_PULL_SHA}
TEST_RESULTS_GCS_DIR=gs://${TEST_RESULT_BUCKET}/${PULL_PULL_SHA}/${TEST_RESULT_FOLDER}
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"

# Configure `time` command output format.
TIMEFORMAT="[test-timing] It took %lR."

echo "presubmit test starts"
time source "${DIR}/test-prep.sh"
echo "test env prepared"

time source "${DIR}/build-images.sh"
echo "KFP images cloudbuild jobs submitted"

time COMMIT_SHA=$PULL_PULL_SHA source "${DIR}/deploy-cluster.sh"
echo "cluster deployed"

time source "${DIR}/check-build-image-status.sh"
echo "KFP images built"

# Install Argo CLI and test-runner service account
time source "${DIR}/install-argo.sh"
echo "argo installed"

time source "${DIR}/deploy-pipeline-lite.sh"
echo "KFP lite deployed"

echo "submitting argo workflow to run tests for commit ${PULL_PULL_SHA}..."
ARGO_WORKFLOW=`argo submit ${DIR}/${WORKFLOW_FILE} \
-p image-build-context-gcs-uri="$remote_code_archive_uri" \
${IMAGE_BUILDER_ARG} \
-p target-image-prefix="${GCR_IMAGE_BASE_DIR}/" \
-p test-results-gcs-dir="${TEST_RESULTS_GCS_DIR}" \
-p cluster-type="${CLUSTER_TYPE}" \
-n ${NAMESPACE} \
--serviceaccount test-runner \
-o name
`
echo "test workflow submitted successfully"
time source "${DIR}/check-argo-status.sh"
echo "test workflow completed"
