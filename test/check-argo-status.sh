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


echo "check status of argo workflow $ARGO_WORKFLOW...."
# probing the argo workflow status until it completed. Timeout after 30 minutes
for i in $(seq 1 ${PULL_ARGO_WORKFLOW_STATUS_MAX_ATTEMPT})
do
  WORKFLOW_STATUS=`kubectl get workflow $ARGO_WORKFLOW -n ${NAMESPACE} --show-labels`
  echo $WORKFLOW_STATUS | grep ${WORKFLOW_COMPLETE_KEYWORD} && s=0 && break || s=$? && printf "Workflow ${ARGO_WORKFLOW} is not finished.\n${WORKFLOW_STATUS}\nSleep for 20 seconds...\n" && sleep 20
done

# Check whether the argo workflow finished or not and exit if not.
if [[ $s != 0 ]]; then
 echo "Prow job Failed: Argo workflow timeout.."
 argo logs -w ${ARGO_WORKFLOW} -n ${NAMESPACE}
 exit $s
fi

echo "Argo workflow finished."

if [[ ! -z "$TEST_RESULT_FOLDER" ]]
then
  echo "Copy test result"
  mkdir -p $ARTIFACT_DIR
  gsutil cp -r "${TEST_RESULTS_GCS_DIR}"/* "${ARTIFACT_DIR}" || true
fi

if [[ $WORKFLOW_STATUS = *"${WORKFLOW_FAILED_KEYWORD}"* ]]; then
  echo "Test workflow failed."
  echo "=========Argo Workflow Logs========="
  argo logs -w ${ARGO_WORKFLOW} -n ${NAMESPACE}
  echo "===================================="
  argo get ${ARGO_WORKFLOW} -n ${NAMESPACE}
  exit 1
else
  argo get ${ARGO_WORKFLOW} -n ${NAMESPACE}
fi