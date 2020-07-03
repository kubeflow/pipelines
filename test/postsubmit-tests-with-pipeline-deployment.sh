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
    [--kfp_deployment       the deployment method of kfp. Valid values are: [standalone, mkp]. Default is standalone.]
    [--workflow_file        the file name of the argo workflow to run]
    [--test_result_bucket   the gcs bucket that argo workflow store the result to. Default is ml-pipeline-test
    [--test_result_folder   the gcs folder that argo workflow store the result to. Always a relative directory to gs://<gs_bucket>/[PULL_SHA]]
    [--timeout              timeout of the tests in seconds. Default is 1800 seconds. ]
    [-h help]"
}

PLATFORM=gcp
PROJECT=ml-pipeline-test
KFP_DEPLOYMENT=standalone
TEST_RESULT_BUCKET=ml-pipeline-test
CLOUDBUILD_PROJECT=ml-pipeline-test
GCR_IMAGE_BASE_DIR=gcr.io/ml-pipeline-test
TARGET_IMAGE_BASE_DIR=gcr.io/ml-pipeline-test/${PULL_BASE_SHA}
TIMEOUT_SECONDS=1800
NAMESPACE=kubeflow
ENABLE_WORKLOAD_IDENTITY=true
COMMIT_SHA="$PULL_BASE_SHA"

while [ "$1" != "" ]; do
    case $1 in
             --platform )             shift
                                      PLATFORM=$1
                                      ;;
             --kfp_deployment )       shift
                                      KFP_DEPLOYMENT=$1
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
# Refer to https://github.com/kubernetes/test-infra/blob/e357ffaaeceafe737bd6ab89d2feff132d92ea50/prow/jobs.md for the Prow job environment variables
TEST_RESULTS_GCS_DIR=gs://${TEST_RESULT_BUCKET}/${PULL_BASE_SHA}/${TEST_RESULT_FOLDER}
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"

if [ ${KFP_DEPLOYMENT} != standalone ]; then
  ENABLE_WORKLOAD_IDENTITY=false
fi

echo "postsubmit test starts"

source "${DIR}/test-prep.sh"

## Wait for the cloudbuild job to be started
CLOUDBUILD_TIMEOUT_SECONDS=3600
PULL_CLOUDBUILD_STATUS_MAX_ATTEMPT=$(expr ${CLOUDBUILD_TIMEOUT_SECONDS} / 20 )
CLOUDBUILD_STARTED=TIMEOUT
CLOUDBUILD_FILTER="substitutions.COMMIT_SHA:${PULL_BASE_SHA} AND tags:build-each-commit"

for i in $(seq 1 ${PULL_CLOUDBUILD_STATUS_MAX_ATTEMPT})
do
  output=`gcloud builds list --project="$CLOUDBUILD_PROJECT" --filter="$CLOUDBUILD_FILTER"`
  if [[ ${output} != "" ]]; then
    CLOUDBUILD_STARTED=True
    break
  fi
  sleep 20
done

if [[ ${CLOUDBUILD_STARTED} == TIMEOUT ]];then
  echo "Wait for cloudbuild job to start, timeout exiting..."
  exit 1
fi

## Wait for the cloudbuild job to complete
CLOUDBUILD_FINISHED=TIMEOUT
for i in $(seq 1 ${PULL_CLOUDBUILD_STATUS_MAX_ATTEMPT})
do
  output=`gcloud builds list --project="$CLOUDBUILD_PROJECT" --filter="$CLOUDBUILD_FILTER"`
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
  echo "Cloud build failure, postsubmit tests cannot proceed. exiting..."
  exit 1
elif [[ ${CLOUDBUILD_FINISHED} == TIMEOUT ]];then
  echo "Wait for cloudbuild job to finish, timeout exiting..."
  exit 1
fi

source "${DIR}/deploy-cluster.sh"
echo "cluster deployed"

# Install Argo
source "${DIR}/install-argo.sh"
echo "argo installed"

# Deploy the pipeline
GCR_IMAGE_TAG=${PULL_BASE_SHA}
if [ ${KFP_DEPLOYMENT} == standalone ]; then
  time source "${DIR}/deploy-pipeline-lite.sh"
  echo "KFP standalone deployed"
  # Submit the argo job and check the results
  echo "submitting argo workflow for commit ${PULL_BASE_SHA}..."
  ARGO_WORKFLOW=`argo submit ${DIR}/${WORKFLOW_FILE} \
  -p image-build-context-gcs-uri="$remote_code_archive_uri" \
  -p commit-sha="${PULL_BASE_SHA}" \
  -p component-image-prefix="${GCR_IMAGE_BASE_DIR}/" \
  -p target-image-prefix="${TARGET_IMAGE_BASE_DIR}/" \
  -p test-results-gcs-dir="${TEST_RESULTS_GCS_DIR}" \
  -n ${NAMESPACE} \
  --serviceaccount test-runner \
  -o name
  `
  echo "argo workflow submitted successfully"
  source "${DIR}/check-argo-status.sh"
  echo "test workflow completed"
else
  SEM_VERSION="$(cat ${DIR}/../VERSION)"
  source "${DIR}/deploy-pipeline-mkp-cli.sh" $SEM_VERSION $COMMIT_SHA ${DIR}
  exit $?
fi
