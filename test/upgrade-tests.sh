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
    [--workflow_file        the file name of the argo workflow to run]
    [--test_result_bucket   the gcs bucket that argo workflow store the result to. Default is ml-pipeline-test
    [--test_result_folder   the gcs folder that argo workflow store the result to. Always a relative directory to gs://<gs_bucket>/[PULL_SHA]]
    [--timeout              timeout of the tests in seconds. Default is 1800 seconds. ]
    [-h help]"
}

PLATFORM=gcp
PROJECT=ml-pipeline-test
TEST_RESULT_BUCKET=ml-pipeline-test
GCR_IMAGE_BASE_DIR=gcr.io/ml-pipeline-test/${PULL_PULL_SHA}
TIMEOUT_SECONDS=1800
NAMESPACE=kubeflow

while [ "$1" != "" ]; do
    case $1 in
             --platform )             shift
                                      PLATFORM=$1
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

# Variables
TEST_RESULTS_GCS_DIR=gs://${TEST_RESULT_BUCKET}/${PULL_PULL_SHA}/${TEST_RESULT_FOLDER}
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"

echo "presubmit test starts"
source "${DIR}/test-prep.sh"
source "${DIR}/deploy-kubeflow.sh"
source "${DIR}/install-argo.sh"
source "${DIR}/build-system-images.sh"

# Run pre upgrade test before cluster is upgraded, to hydrate the cluster with some user data
echo "submitting argo workflow to run tests for commit ${PULL_PULL_SHA}..."
ARGO_WORKFLOW=`argo submit ${DIR}/upgrade_test.yaml \
-p image-build-context-gcs-uri="$remote_code_archive_uri" \
-p target-image-prefix="${GCR_IMAGE_BASE_DIR}/" \
-p test-results-gcs-dir="${TEST_RESULTS_GCS_DIR}" \
-p test-results-file-name="junit_UpgradeTestOutputBeforeUpgrade.xml" \
-p cleanup="false" \
-n ${NAMESPACE} \
--serviceaccount test-runner \
-o name
`
echo "pre upgrade test workflow submitted successfully"
source "${DIR}/check-argo-status.sh"
echo "test workflow completed"

# Upgrade the pipeline with latest built image
source ${DIR}/deploy-pipeline.sh --gcr_image_base_dir ${GCR_IMAGE_BASE_DIR}

# Run upgrade test after cluster is upgraded. Verify user dat can still be retrieved
echo "submitting argo workflow to run tests for commit ${PULL_PULL_SHA}..."
ARGO_WORKFLOW=`argo submit ${DIR}/upgrade_test.yaml \
-p image-build-context-gcs-uri="$remote_code_archive_uri" \
-p target-image-prefix="${GCR_IMAGE_BASE_DIR}/" \
-p test-results-gcs-dir="${TEST_RESULTS_GCS_DIR}" \
-p test-results-file-name="junit_UpgradeTestOutputAfterUpgrade.xml" \
-p cleanup="true" \
-n ${NAMESPACE} \
--serviceaccount test-runner \
-o name
`

echo "post upgrade test workflow submitted successfully"
source "${DIR}/check-argo-status.sh"
echo "test workflow completed"
