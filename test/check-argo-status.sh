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

set -e

WORKFLOW_COMPLETE_KEYWORD="completed=true"
WORKFLOW_FAILED_KEYWORD="phase=Failed"
PULL_ARGO_WORKFLOW_STATUS_MAX_ATTEMPT=$(expr $TIMEOUT_SECONDS / 20 )

workflow_completed=false
workflow_failed=false

echo "check status of argo workflow $ARGO_WORKFLOW...."
# probing the argo workflow status until it completed. Timeout after 30 minutes
for i in $(seq 1 ${PULL_ARGO_WORKFLOW_STATUS_MAX_ATTEMPT})
do
  WORKFLOW_STATUS=`kubectl get workflow $ARGO_WORKFLOW -n ${NAMESPACE} --show-labels 2>&1` \
    || echo kubectl get workflow failed with "$WORKFLOW_STATUS" # Tolerate temporary network failure during kubectl get workflow
  if echo $WORKFLOW_STATUS | grep "${WORKFLOW_COMPLETE_KEYWORD}" --quiet; then
    workflow_completed=true
    if echo $WORKFLOW_STATUS | grep "${WORKFLOW_FAILED_KEYWORD}" --quiet; then
      workflow_failed=true
    fi
    break
  else
    echo "Workflow ${ARGO_WORKFLOW} is not finished: ${WORKFLOW_STATUS} - Sleep for 20 seconds..."
    sleep 20
  fi
done

if [[ "$workflow_completed" == "false" ]] || [[ "$workflow_failed" != "false" ]]; then
  # Handling failed workflow
  if [[ "$workflow_completed" == "false" ]]; then
    echo "Argo workflow timed out."
  else
    echo "Argo workflow failed."
  fi

  echo "=========Argo Workflow Logs========="
  argo logs "${ARGO_WORKFLOW}" -n "${NAMESPACE}"

  echo "========All workflows============="

  argo --namespace "${NAMESPACE}" list --output=name |
    while read workflow_id; do
      echo "========${workflow_id}============="
      argo get "${workflow_id}" -n "${NAMESPACE}"
    done

  echo "=========Main workflow=============="
  argo get "${ARGO_WORKFLOW}" -n "${NAMESPACE}"

  exit 1
fi

echo "Argo workflow finished successfully."
if [[ -n "$TEST_RESULT_FOLDER" ]]; then
  echo "Copy test result"
  gsutil cp -r "${TEST_RESULTS_GCS_DIR}"/* "${ARTIFACTS}" || true
fi

echo "=========Main workflow=============="
argo get "${ARGO_WORKFLOW}" -n "${NAMESPACE}"
