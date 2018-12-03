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

set -xe

usage()
{
    echo "usage: deploy.sh
    [--workflow_file        the file name of the argo workflow to run]
    [--test_result_bucket   the gcs bucket that argo workflow store the result to. Default is ml-pipeline-test
    [--test_result_folder   the gcs folder that argo workflow store the result to. Always a relative directory to gs://<gs_bucket>/[PULL_SHA]]
    [--cluster-type         the type of cluster to use for the tests. One of: create-gke,none. Default is create-gke ]
    [--timeout              timeout of the tests in seconds. Default is 1800 seconds. ]
    [-h help]"
}

PROJECT=ml-pipeline-test
TEST_RESULT_BUCKET=ml-pipeline-test
GCR_IMAGE_BASE_DIR=gcr.io/ml-pipeline-test/${PULL_PULL_SHA}
CLUSTER_TYPE=create-gke
TIMEOUT_SECONDS=1800
NAMESPACE=default

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
             --cluster-type )         shift
                                      CLUSTER_TYPE=$1
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

echo "presubmit test starts"

repo_root=$(dirname "$0")/..
cd "$repo_root"

# activating the service account
gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"
gcloud config set compute/zone us-central1-a

#Uploading the source code to GCS:
local_code_archive_file=$(mktemp)
date_string=$(TZ=PST8PDT date +%Y-%m-%d_%H-%M-%S_%Z)
code_archive_prefix="gs://${TEST_RESULT_BUCKET}/${PULL_PULL_SHA}/source_code"
remote_code_archive_uri="${code_archive_prefix}_${PULL_BASE_SHA}_${date_string}.tar.gz"

tar -czf "$local_code_archive_file" .
gsutil cp "$local_code_archive_file" "$remote_code_archive_uri"

#The file can be inaccessible right after upload.
#ServiceException: 401 Anonymous caller does not have storage.objects.get access to ml-pipeline-test
#See https://github.com/kubeflow/pipelines/issues/403
#Let's try to wait for it. This might not be enough to fix the issue though.
until gsutil ls "$remote_code_archive_uri"; do
  sleep 5s
done

#Creating a new GKE cluster if needed
if [ "$CLUSTER_TYPE" == "create-gke" ]; then
  echo "create test cluster"
  TEST_CLUSTER_PREFIX=${WORKFLOW_FILE%.*}
  TEST_CLUSTER=${TEST_CLUSTER_PREFIX//_}-${PULL_PULL_SHA:0:10}-${RANDOM}

  function delete_cluster {
    echo "Delete cluster..."
    gcloud container clusters delete ${TEST_CLUSTER} --async
  }
  trap delete_cluster EXIT

  gcloud config set project ml-pipeline-test
  gcloud config set compute/zone us-central1-a
  gcloud container clusters create ${TEST_CLUSTER} \
    --scopes cloud-platform \
    --enable-cloud-logging \
    --enable-cloud-monitoring \
    --machine-type n1-standard-2 \
    --num-nodes 3 \
    --network test \
    --subnetwork test-1

  gcloud container clusters get-credentials ${TEST_CLUSTER}
fi

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"
source "${DIR}/install-argo.sh"

echo "submitting argo workflow for commit ${PULL_PULL_SHA}..."
ARGO_WORKFLOW=`argo submit ${DIR}/${WORKFLOW_FILE} \
-p image-build-context-gcs-uri="$remote_code_archive_uri" \
-p target-image-prefix="${GCR_IMAGE_BASE_DIR}/" \
-p test-results-gcs-dir="${TEST_RESULTS_GCS_DIR}" \
-p cluster-type="${CLUSTER_TYPE}" \
-p bootstrapper-image="${GCR_IMAGE_BASE_DIR}/bootstrapper" \
-p api-image="${GCR_IMAGE_BASE_DIR}/api" \
-p frontend-image="${GCR_IMAGE_BASE_DIR}/frontend" \
-p scheduledworkflow-image="${GCR_IMAGE_BASE_DIR}/scheduledworkflow" \
-p persistenceagent-image="${GCR_IMAGE_BASE_DIR}/persistenceagent" \
-o name
`
echo argo workflow submitted successfully

source "${DIR}/check-argo-status.sh"
