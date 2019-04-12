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

ARTIFACT_DIR=$WORKSPACE/_artifacts
WORKFLOW_COMPLETE_KEYWORD="completed=true"
WORKFLOW_FAILED_KEYWORD="phase=Failed"
PULL_ARGO_WORKFLOW_STATUS_MAX_ATTEMPT=$(expr $TIMEOUT_SECONDS / 20 )

echo "Waiting for Argo workflow $ARGO_WORKFLOW to complete (time-out after $(expr $TIMEOUT_SECONDS / 60 ) minutes)..."
(set +x
for i in $(seq 1 ${PULL_ARGO_WORKFLOW_STATUS_MAX_ATTEMPT})
do
  WORKFLOW_STATUS=`kubectl get workflow $ARGO_WORKFLOW -n ${NAMESPACE} --show-labels --no-headers=true`
  echo $WORKFLOW_STATUS
  if echo $WORKFLOW_STATUS | grep ${WORKFLOW_COMPLETE_KEYWORD}; then
    s=0
    break
  else
    s=1
    sleep 20
  fi
done

# Check whether the argo workflow finished or not and exit if not.
if [[ $s != 0 ]]; then
 echo "Prow job Failed: Argo workflow timeout.."
 argo logs -w ${ARGO_WORKFLOW} -n ${NAMESPACE}
 exit $s
fi
)

echo "Argo workflow finished."

if [[ ! -z "$TEST_RESULT_FOLDER" ]]
then
  echo "Copy test result"
  mkdir -p $ARTIFACT_DIR
  gsutil cp -r "${TEST_RESULTS_GCS_DIR}"/* "${ARTIFACT_DIR}" -q || true
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