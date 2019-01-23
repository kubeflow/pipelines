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

TEST_RESULTS_GCS_DIR=gs://${TEST_RESULT_BUCKET}/${PULL_PULL_SHA}/${TEST_RESULT_FOLDER}
ARTIFACT_DIR=$WORKSPACE/_artifacts
WORKFLOW_COMPLETE_KEYWORD="completed=true"
WORKFLOW_FAILED_KEYWORD="phase=Failed"
PULL_ARGO_WORKFLOW_STATUS_MAX_ATTEMPT=$(expr $TIMEOUT_SECONDS / 20 )
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"

echo "presubmit test starts"

# activating the service account
gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"
gcloud config set compute/zone us-central1-a
gcloud config set core/project ${PROJECT}

#Uploading the source code to GCS:
local_code_archive_file=$(mktemp)
date_string=$(TZ=PST8PDT date +%Y-%m-%d_%H-%M-%S_%Z)
code_archive_prefix="gs://${TEST_RESULT_BUCKET}/${PULL_PULL_SHA}/source_code"
remote_code_archive_uri="${code_archive_prefix}_${PULL_BASE_SHA}_${date_string}.tar.gz"

tar -czf "$local_code_archive_file" .
gsutil cp "$local_code_archive_file" "$remote_code_archive_uri"

# Install ksonnet
KS_VERSION="0.13.0"
curl -LO https://github.com/ksonnet/ksonnet/releases/download/v${KS_VERSION}/ks_${KS_VERSION}_linux_amd64.tar.gz
tar -xzf ks_${KS_VERSION}_linux_amd64.tar.gz
chmod +x ./ks_${KS_VERSION}_linux_amd64/ks
mv ./ks_${KS_VERSION}_linux_amd64/ks /usr/local/bin/

# Download kubeflow master
KUBEFLOW_MASTER=${DIR}/kubeflow_master
git clone https://github.com/kubeflow/kubeflow.git ${KUBEFLOW_MASTER}

## Download latest kubeflow release source code
KUBEFLOW_SRC=${DIR}/kubeflow_latest_release
mkdir ${KUBEFLOW_SRC}
cd ${KUBEFLOW_SRC}
export KUBEFLOW_TAG=v0.3.1
curl https://raw.githubusercontent.com/kubeflow/kubeflow/${KUBEFLOW_TAG}/scripts/download.sh | bash

## Override the pipeline config with code from master
cp -r ${KUBEFLOW_MASTER}/kubeflow/pipeline ${KUBEFLOW_SRC}/kubeflow/pipeline
cp -r ${KUBEFLOW_MASTER}/kubeflow/argo ${KUBEFLOW_SRC}/kubeflow/argo

# TODO temporarily set KUBEFLOW_SRC as KUBEFLOW_MASTER. This should be deleted when latest release have the pipeline entry
KUBEFLOW_SRC=${KUBEFLOW_MASTER}

TEST_CLUSTER_PREFIX=${WORKFLOW_FILE%.*}
TEST_CLUSTER=$(echo $TEST_CLUSTER_PREFIX | cut -d _ -f 1)-${PULL_PULL_SHA:0:7}-${RANDOM}

export CLIENT_ID=${RANDOM}
export CLIENT_SECRET=${RANDOM}
KFAPP=${TEST_CLUSTER}

function clean_up {
  echo "Clean up..."
  cd ${KFAPP}
  ${KUBEFLOW_SRC}/scripts/kfctl.sh delete all
}
trap clean_up EXIT

${KUBEFLOW_SRC}/scripts/kfctl.sh init ${KFAPP} --platform ${PLATFORM} --project ${PROJECT} --skipInitProject

cd ${KFAPP}
${KUBEFLOW_SRC}/scripts/kfctl.sh generate platform
${KUBEFLOW_SRC}/scripts/kfctl.sh apply platform
${KUBEFLOW_SRC}/scripts/kfctl.sh generate k8s

## Update pipeline component image
pushd ks_app
ks param set pipeline apiImage ${GCR_IMAGE_BASE_DIR}/api
ks param set pipeline persistenceAgentImage ${GCR_IMAGE_BASE_DIR}/persistenceagent
ks param set pipeline scheduledWorkflowImage ${GCR_IMAGE_BASE_DIR}/scheduledworkflow
ks param set pipeline uiImage ${GCR_IMAGE_BASE_DIR}/frontend
popd

${KUBEFLOW_SRC}/scripts/kfctl.sh apply k8s

gcloud container clusters get-credentials ${TEST_CLUSTER}

source "${DIR}/install-argo.sh"

echo "submitting argo workflow for commit ${PULL_PULL_SHA}..."
ARGO_WORKFLOW=`argo submit ${DIR}/${WORKFLOW_FILE} \
-p image-build-context-gcs-uri="$remote_code_archive_uri" \
-p target-image-prefix="${GCR_IMAGE_BASE_DIR}/" \
-p test-results-gcs-dir="${TEST_RESULTS_GCS_DIR}" \
-p cluster-type="${CLUSTER_TYPE}" \
-n ${NAMESPACE} \
--serviceaccount test-runner \
-o name
`

echo argo workflow submitted successfully

source "${DIR}/check-argo-status.sh"

