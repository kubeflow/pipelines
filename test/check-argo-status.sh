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


#Wait for the workflow to appear
while [ "$(kubectl get workflow/$ARGO_WORKFLOW --namespace "$NAMESPACE" -o=jsonpath='{.status.phase}')" == "" ]; do
  echo "Workflow has not started yet"
  sleep 10s
done

#Wait for the workflow to start Running (to stop Pending)
#TODO: Remove after https://github.com/argoproj/argo/issues/989 is fixed.
while [ "$(kubectl get workflow/$ARGO_WORKFLOW --namespace "$NAMESPACE" -o=jsonpath='{.status.phase}')" == "Pending" ]; do
  echo "Workflow is still Pending"
  sleep 10s
done
sleep 2m #Preventing errors: "warning msg="container 'main' of pod 'build-images-xxxx-yyyyyyy' has not started within expected timeout". See https://github.com/argoproj/argo/issues/1104

#Stream workflow logs
#kubectl logs --follow --pod-running-timeout 30m workflow/$ARGO_WORKFLOW --namespace "$NAMESPACE" #Currently we cannot use kubectl for Workflow. See https://github.com/argoproj/argo/issues/1108
argo logs --follow --workflow $ARGO_WORKFLOW --namespace "$NAMESPACE" --timestamps

WORKFLOW_STATUS=$(kubectl get workflow/$ARGO_WORKFLOW --namespace "$NAMESPACE" -o=jsonpath='{.status.phase}')
echo Workflow finished with status: $WORKFLOW_STATUS.

#Print workflow object summary
argo get $ARGO_WORKFLOW

if [ "$WORKFLOW_STATUS" != "Succeeded" ]; then
  exit 1
fi

if [[ ! -z "$TEST_RESULT_FOLDER" ]]
then
  echo "Copy test result"
  mkdir -p $ARTIFACT_DIR
  gsutil cp -r "${TEST_RESULTS_GCS_DIR}"/* "${ARTIFACT_DIR}" || true
fi
