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
set -o pipefail

usage()
{
    echo "usage: multiuser-tests.sh
    [--project              the gcp project. Default is ml-pipeline-test. Only used when platform is gcp.]
    [--workflow_file        the file name of the argo workflow to run]
    [--test_result_bucket   the gcs bucket that argo workflow store the result to. Default is ml-pipeline-test]
    [--test_result_folder   the gcs folder that argo workflow store the result to. Always a relative directory to gs://<gs_bucket>/[COMMIT_SHA]]
    [--timeout              timeout of the tests in seconds. Default is 2700 seconds. ]
    [-h help]"
}

COMMIT_SHA=${PULL_BASE_SHA:-$(git rev-parse HEAD)}

KFP_DEPLOYMENT=kubeflow
PROJECT=ml-pipeline-test
TEST_RESULT_BUCKET=ml-pipeline-test
TEST_RESULT_FOLDER=test-result
TIMEOUT_SECONDS=1800
NAMESPACE=kubeflow
ENABLE_WORKLOAD_IDENTITY=true

while [ "$1" != "" ]; do
    case $1 in
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

CLOUDBUILD_PROJECT=${PROJECT}
GCR_IMAGE_BASE_DIR=gcr.io/${PROJECT}/${COMMIT_SHA}
GCR_IMAGE_TAG=${GCR_IMAGE_TAG:-latest}
TEST_RESULTS_GCS_DIR=gs://${TEST_RESULT_BUCKET}/${COMMIT_SHA}/${TEST_RESULT_FOLDER}
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"

# Configure `time` command output format.
TIMEFORMAT="[test-timing] It took %lR."

time source "${DIR}/test-prep.sh"

TEST_CLUSTER_PREFIX=${WORKFLOW_FILE%.*}
TEST_CLUSTER_DEFAULT=$(echo $TEST_CLUSTER_PREFIX | cut -d _ -f 1)-${COMMIT_SHA:0:7}-${RANDOM}
TEST_CLUSTER=${TEST_CLUSTER:-${TEST_CLUSTER_DEFAULT}}

# We don't wait for image building here, because cluster can be deployed in
# parallel so that we save a few minutes of test time.
time source "${DIR}/build-images.sh"
echo "KFP images cloudbuild jobs submitted"

time source "${DIR}/check-build-image-status.sh"
echo "KFP images built"

time source "${DIR}/deploy-multiuser-kfp.sh"
echo "multi user kfp deployed"

time source "${DIR}/install-argo.sh"
echo "argo installed"

kubectl create ns argo --dry-run -o yaml | kubectl apply -f -
kubectl apply -n argo -f https://raw.githubusercontent.com/argoproj/argo/$ARGO_VERSION/manifests/install.yaml

if [ "$ENABLE_WORKLOAD_IDENTITY" = true ]; then
  # Use static GSAs for testing, so we don't need to GC them.
  export SYSTEM_GSA="test-kfp-system"
  export USER_GSA="test-kfp-user"
  source "${DIR}/scripts/retry.sh"

  retry gcloud projects add-iam-policy-binding $PROJECT \
    --member="serviceAccount:$SYSTEM_GSA@$PROJECT.iam.gserviceaccount.com" \
    --role="roles/editor"
  retry gcloud projects add-iam-policy-binding $PROJECT \
    --member="serviceAccount:$USER_GSA@$PROJECT.iam.gserviceaccount.com" \
    --role="roles/editor"
fi

# Create test profiles in prepare for test scenarios
kubectl create -f ${DIR}/multi-user-test/profile.yaml

set +x
KFP_HOST="https://${TEST_CLUSTER}.endpoints.${PROJECT}.cloud.goog/pipeline"
CLIENT_ID=${CLIENT_ID:-$(gsutil cat gs://ml-pipeline-test-keys/oauth_client_id)}
OTHER_CLIENT_ID=${OTHER_CLIENT_ID:-$(gsutil cat gs://ml-pipeline-test-keys/other_client_id)}
OTHER_CLIENT_SECRET=${OTHER_CLIENT_SECRET:-$(gsutil cat gs://ml-pipeline-test-keys/other_client_secret)}
REFRESH_TOKEN_A=${REFRESH_TOKEN_A:-$(gsutil cat gs://ml-pipeline-test-keys/cerseistark_token)}
USER_NAMESPACE_A=${USER_NAMESPACE_A:-"kubeflow-cerseistark"}
REFRESH_TOKEN_B=${REFRESH_TOKEN_B:-$(gsutil cat gs://ml-pipeline-test-keys/briennebolton_token)}
USER_NAMESPACE_B=${USER_NAMESPACE_B:-"kubeflow-briennebolton"}

# Submit the argo job and check the results
echo "submitting argo workflow for commit ${COMMIT_SHA}..."
ARGO_WORKFLOW=`argo submit ${DIR}/${WORKFLOW_FILE} \
-p image-build-context-gcs-uri="$remote_code_archive_uri" \
-p commit-sha="${COMMIT_SHA}" \
-p component-image-prefix="${GCR_IMAGE_BASE_DIR}/" \
-p target-image-prefix="${GCR_IMAGE_BASE_DIR}/" \
-p test-results-gcs-dir="${TEST_RESULTS_GCS_DIR}" \
-p host="${KFP_HOST}" \
-p client-id="${CLIENT_ID}" \
-p other-client-id="${OTHER_CLIENT_ID}" \
-p other-client-secret="${OTHER_CLIENT_SECRET}" \
-p refresh-token-a="${REFRESH_TOKEN_A}" \
-p user-namespace-a="${USER_NAMESPACE_A}" \
-p refresh-token-b="${REFRESH_TOKEN_B}" \
-p user-namespace-b="${USER_NAMESPACE_B}" \
-n ${NAMESPACE} \
--serviceaccount test-runner \
-o name
`
set -x

echo "argo workflow submitted successfully"
source "${DIR}/check-argo-status.sh"
echo "test workflow completed"
