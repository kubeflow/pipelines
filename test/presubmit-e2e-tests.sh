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

TEST_CLUSTER=spinner
TEST_WORKFLOW=integration_test_gke.yaml
PULL_ARGO_WORKFLOW_STATUS_MAX_ATTEMPT=60
ARTIFACT_DIR=$WORKSPACE/_artifacts
WORKFLOW_COMPLETE_KEYWORD="completed=true"
WORKFLOW_FAILED_KEYWORD="phase=Failed"

echo presubmit test starts...

gcloud container clusters get-credentials ${TEST_CLUSTER} --zone us-west1-a --project ml-pipeline-test
kubectl config set-context $(kubectl config current-context) --namespace=default

echo "submitting argo workflow for commit ${PULL_PULL_SHA}..."
ARGO_WORKFLOW=`argo submit $(dirname $0)/${TEST_WORKFLOW} -p commit-sha="${PULL_PULL_SHA}" | awk '/Name:/{print $NF}'`
echo argo workflow submitted successfully

echo check status of argo workflow $ARGO_WORKFLOW....

# probing the argo workflow status until it completed. Timeout after 20 minutes
for i in $(seq 1 ${PULL_ARGO_WORKFLOW_STATUS_MAX_ATTEMPT})
do
  WORKFLOW_STATUS=`kubectl get wf $ARGO_WORKFLOW --show-labels`
  echo $WORKFLOW_STATUS | grep ${WORKFLOW_COMPLETE_KEYWORD} && s=0 && break || s=$? && printf "Workflow ${ARGO_WORKFLOW} is not finished.\n${WORKFLOW_STATUS}\nSleep for 20 seconds...\n" && sleep 20
done

# Check whether the argo workflow finished or not and exit if not.
if [[ $s != 0 ]]
 then echo "Prow job Failed: Argo workflow timeout.." && exit $s
fi

echo Argo workflow finished....

echo Copy test result...
mkdir -p $WORKSPACE/_artifacts
gsutil mv -r gs://ml-pipeline-test/${PULL_PULL_SHA}/* ${ARTIFACT_DIR}

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