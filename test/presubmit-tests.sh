#!/bin/bash

# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://www.apache.org/licenses/LICENSE-2.0
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
    [--test_result_folder   the gcs folder that argo workflow store the result to. Always a relative directory to gs://ml-pipeline-test/[PULL_SHA]]
    [-h help]"
}

while [ "$1" != "" ]; do
    case $1 in
             --workflow_file )        shift
                                      WORKFLOW_FILE=$1
                                      ;;
             --test_result_folder )   shift
                                      TEST_RESULT_FOLDER=$1
                                      ;;
             -h | --help )            usage
                                      exit
                                      ;;
             * )                      usage
                                      exit 1
    esac
    shift
done

TEST_CLUSTER_PREFIX=${WORKFLOW_FILE%.*}
TEST_CLUSTER=${TEST_CLUSTER_PREFIX//_}-${PULL_PULL_SHA:0:10}
ZONE=us-west1-a
PULL_ARGO_WORKFLOW_STATUS_MAX_ATTEMPT=90
ARTIFACT_DIR=$WORKSPACE/_artifacts
WORKFLOW_COMPLETE_KEYWORD="completed=true"
WORKFLOW_FAILED_KEYWORD="phase=Failed"

echo "presubmit test starts"

echo "create test cluster"
gcloud config set project ml-pipeline-test
gcloud config set compute/zone us-west1-a
gcloud container clusters create ${TEST_CLUSTER} \
  --scopes cloud-platform \
  --enable-cloud-logging \
  --enable-cloud-monitoring \
  --machine-type n1-standard-2 \
  --num-nodes 3

function delete_cluster {
  echo "Delete cluster..."
  gcloud container clusters delete ${TEST_CLUSTER} --async
}
trap delete_cluster EXIT

gcloud container clusters get-credentials ${TEST_CLUSTER}
kubectl config set-context $(kubectl config current-context) --namespace=default

echo "Add necessary cluster role bindings"
ACCOUNT=$(gcloud info --format='value(config.account)')
kubectl create clusterrolebinding PROW_BINDING --clusterrole=cluster-admin --user=$ACCOUNT
kubectl create clusterrolebinding DEFAULT_BINDING --clusterrole=cluster-admin --serviceaccount=default:default

echo "Create k8s secret for github SSH credentials"
cp /etc/ssh-knative/ssh-knative ./id_rsa
kubectl create secret generic ssh-key-secret --from-file=id_rsa=./id_rsa

echo "install argo"
argo install

echo "submitting argo workflow for commit ${PULL_PULL_SHA}..."
ARGO_WORKFLOW=`argo submit $(dirname $0)/${WORKFLOW_FILE} -p commit-sha="${PULL_PULL_SHA}" | awk '/Name:/{print $NF}'`
echo argo workflow submitted successfully

echo "check status of argo workflow $ARGO_WORKFLOW...."
# probing the argo workflow status until it completed. Timeout after 20 minutes
for i in $(seq 1 ${PULL_ARGO_WORKFLOW_STATUS_MAX_ATTEMPT})
do
  WORKFLOW_STATUS=`kubectl get workflow $ARGO_WORKFLOW --show-labels`
  echo $WORKFLOW_STATUS | grep ${WORKFLOW_COMPLETE_KEYWORD} && s=0 && break || s=$? && printf "Workflow ${ARGO_WORKFLOW} is not finished.\n${WORKFLOW_STATUS}\nSleep for 20 seconds...\n" && sleep 20
done

# Check whether the argo workflow finished or not and exit if not.
if [[ $s != 0 ]]
 then echo "Prow job Failed: Argo workflow timeout.." && exit $s
fi

echo "Argo workflow finished. Copy test result"
mkdir -p $ARTIFACT_DIR
gsutil cp -r gs://ml-pipeline-test/${PULL_PULL_SHA}/${TEST_RESULT_FOLDER}/* ${ARTIFACT_DIR} || true

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
