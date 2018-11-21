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

# activating the service account
gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"
gcloud config set compute/zone us-central1-a

if [ "$WORKFLOW_FILE" == "build_image.yaml" ]; then
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

  echo "install argo"
  ARGO_VERSION=v2.2.0
  mkdir -p ~/bin/
  export PATH=~/bin/:$PATH
  curl -sSL -o ~/bin/argo https://github.com/argoproj/argo/releases/download/$ARGO_VERSION/argo-linux-amd64
  chmod +x ~/bin/argo
  kubectl create ns argo
  kubectl apply -n argo -f https://raw.githubusercontent.com/argoproj/argo/$ARGO_VERSION/manifests/install.yaml
fi

if [ "$WORKFLOW_FILE" == "e2e_test_gke.yaml" ]; then
  # Install ksonnet
  KS_VERSION="0.11.0"
  curl -LO https://github.com/ksonnet/ksonnet/releases/download/v${KS_VERSION}/ks_${KS_VERSION}_linux_amd64.tar.gz
  tar -xzf ks_${KS_VERSION}_linux_amd64.tar.gz
  chmod +x ./ks_${KS_VERSION}_linux_amd64/ks
  mv ./ks_${KS_VERSION}_linux_amd64/ks /usr/local/bin/

  # Install kubeflow
  KUBEFLOW_MASTER=$(pwd)/kubeflow_master
  git clone https://github.com/kubeflow/kubeflow.git ${KUBEFLOW_MASTER}

  ## Download latest release source code
  KUBEFLOW_SRC=$(pwd)/kubeflow_latest_release
  mkdir ${KUBEFLOW_SRC}
  cd ${KUBEFLOW_SRC}
  export KUBEFLOW_TAG=v0.3.1
  curl https://raw.githubusercontent.com/kubeflow/kubeflow/${KUBEFLOW_TAG}/scripts/download.sh | bash

  ## Override the pipeline config with code from master
  cp -r ${KUBEFLOW_MASTER}/kubeflow/pipeline ${KUBEFLOW_SRC}/kubeflow/pipeline
  cp -r ${KUBEFLOW_MASTER}/kubeflow/argo ${KUBEFLOW_SRC}/kubeflow/argo

  TEST_CLUSTER_PREFIX=${WORKFLOW_FILE%.*}
  TEST_CLUSTER=$(echo $TEST_CLUSTER_PREFIX | cut -d _ -f 1)-${PULL_PULL_SHA:0:7}-${RANDOM}
  function delete_cluster {
    echo "Delete cluster..."
    gcloud container clusters delete ${TEST_CLUSTER} --async
  }
#  trap delete_cluster EXIT

  KFAPP=${TEST_CLUSTER}
  ${KUBEFLOW_SRC}/scripts/kfctl.sh init ${KFAPP} --platform gcp --project ${PROJECT}
  cd ${KFAPP}
  ${KUBEFLOW_SRC}/scripts/kfctl.sh generate platform
  ${KUBEFLOW_SRC}/scripts/kfctl.sh apply platform
  ${KUBEFLOW_SRC}/scripts/kfctl.sh generate k8s

  ## Update pipeline component image
  ks param set pipeline apiImage ${GCR_IMAGE_BASE_DIR}/api:${PULL_PULL_SHA}
  ks param set pipeline persistenceAgentImage ${GCR_IMAGE_BASE_DIR}/persistenceagent:${PULL_PULL_SHA}
  ks param set pipeline scheduledWorkflowImage ${GCR_IMAGE_BASE_DIR}/scheduledworkflow:${PULL_PULL_SHA}
  ks param set pipeline uiImage ${GCR_IMAGE_BASE_DIR}/frontend:${PULL_PULL_SHA}
  ${KUBEFLOW_SRC}/scripts/kfctl.sh apply k8s

  gcloud container clusters get-credentials ${TEST_CLUSTER}
fi

# TODO install argo if not install kubeflow

echo "submitting argo workflow for commit ${PULL_PULL_SHA}..."
ARGO_WORKFLOW=`argo submit $(dirname $0)/${WORKFLOW_FILE} \
-p commit-sha="${PULL_PULL_SHA}" \
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

echo "check status of argo workflow $ARGO_WORKFLOW...."
# probing the argo workflow status until it completed. Timeout after 30 minutes
for i in $(seq 1 ${PULL_ARGO_WORKFLOW_STATUS_MAX_ATTEMPT})
do
  WORKFLOW_STATUS=`kubectl get workflow $ARGO_WORKFLOW --show-labels`
  echo $WORKFLOW_STATUS | grep ${WORKFLOW_COMPLETE_KEYWORD} && s=0 && break || s=$? && printf "Workflow ${ARGO_WORKFLOW} is not finished.\n${WORKFLOW_STATUS}\nSleep for 20 seconds...\n" && sleep 20
done

# Check whether the argo workflow finished or not and exit if not.
if [[ $s != 0 ]]; then
 echo "Prow job Failed: Argo workflow timeout.."
 argo logs -w ${ARGO_WORKFLOW}
 exit $s
fi

echo "Argo workflow finished."

if [[ ! -z "$TEST_RESULT_FOLDER" ]]
then
  echo "Copy test result"
  mkdir -p $ARTIFACT_DIR
  gsutil cp -r "${TEST_RESULTS_GCS_DIR}"/* "${ARTIFACT_DIR}" || true
fi

ARGO_WORKFLOW_DETAILS=`argo get ${ARGO_WORKFLOW}`
ARGO_WORKFLOW_LOGS=`argo logs -w ${ARGO_WORKFLOW}`

if [[ $WORKFLOW_STATUS = *"${WORKFLOW_FAILED_KEYWORD}"* ]]; then
  printf "The argo workflow failed.\n =========Argo Workflow=========\n${ARGO_WORKFLOW_DETAILS}\n==================\n"
  printf "=========Argo Workflow Logs=========\n${ARGO_WORKFLOW_LOGS}\n==================\n"
  exit 1
else
  printf ${ARGO_WORKFLOW_DETAILS}
  exit 0
fi
