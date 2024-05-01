#!/bin/bash
#
# Copyright 2023 kubeflow.org
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

retry() {
  local max=$1; shift
  local interval=$1; shift

  until "$@"; do
    echo "trying.."
    max=$((max-1))
    if [[ "$max" -eq 0 ]]; then
      return 1
    fi
    sleep "$interval"
  done
}

collect_pipeline_artifacts() {
  pipeline_uid=$1
  kubeflow_ns=kubeflow

  local log_dir=$(mktemp -d)

  #pods_kubeflow=$(kubectl get pods -n $kubeflow_ns --no-headers -o custom-columns=NAME:.metadata.name -l pipeline/runid=$pipeline_uid)
  pods_kubeflow=$(kubectl get pods -n $kubeflow_ns --no-headers -o custom-columns=NAME:.metadata.name)

  echo "Collecting pod names for run $pipeline_uid"
  echo $pods_kubeflow > $log_dir/pods.log

  echo "Collecting pod logs for run $pipeline_uid"
  for pod in $pods_kubeflow; do
      kubectl logs -n $kubeflow_ns $pod --all-containers=true > $log_dir/$pod.log
  done

  echo "Collecting events for run $pipeline_uid"
  kubectl get events -n $kubeflow_ns > $log_dir/events.log

  echo "Collection Tekton pipeline runs for run $pipeline_uid"
  kubectl get pipelineruns -n $kubeflow_ns -o yaml > $log_dir/pipelineruns.log

  echo "Collection Tekton task runs for run $pipeline_uid"
  kubectl get taskruns -n $kubeflow_ns -o yaml > $log_dir/taskruns.log

  echo "Collection Tekton kfptask runs for run $pipeline_uid"
  kubectl get kfptasks -n $kubeflow_ns -o yaml > $log_dir/kfptasks.log

  echo "Collection Tekton custom runs for run $pipeline_uid"
  kubectl get customruns -n $kubeflow_ns -o yaml > $log_dir/customruns.log
}

# compile the python to a pipeline yaml, upload the pipeline, create a run,
# and wait until the run finishes.
run_test_case() {
  if [[ $# -ne 4 ]]
  then
      echo "Usage: run_test_case test-case-name python-file condition-string wait-time"
      return 1
  fi
  local REV=1
  local TEST_CASE=$1
  shift
  local PY_FILE=$1
  shift
  local F_STATUS=$1
  shift
  local DURATION=$1
  shift
  local PIPELINE_ID
  local RUN_ID
  local KFP_COMMAND="kfp"
  local PIPELINE_NAME="${TEST_CASE}-$((RANDOM%10000+1))"
  local YAML_FILE=$(echo "${PY_FILE}" | sed "s/\.py$/\.yaml/")

  echo " =====   ${TEST_CASE}  ====="
  $KFP_COMMAND dsl compile --py "${PY_FILE}" --output "${YAML_FILE}"
  retry 3 3 $KFP_COMMAND --endpoint http://localhost:8888 pipeline create -p "$PIPELINE_NAME" "${YAML_FILE}" 2>&1 || :
  PIPELINE_ID=$($KFP_COMMAND --endpoint http://localhost:8888 pipeline list 2>&1| grep "$PIPELINE_NAME" | awk '{print $1}')
  if [[ -z "$PIPELINE_ID" ]]; then
    echo "Failed to upload pipeline"
    return "$REV"
  fi
  VERSION_ID=$($KFP_COMMAND --endpoint http://localhost:8888 pipeline list-versions "${PIPELINE_ID}" 2>&1| grep "$PIPELINE_NAME" | awk '{print $1}')

  local RUN_NAME="${PIPELINE_NAME}-run"
  retry 3 3 $KFP_COMMAND --endpoint http://localhost:8888 run create -e "exp-${TEST_CASE}" -r "$RUN_NAME" -p "$PIPELINE_ID" -v "$VERSION_ID" 2>&1 || :
  RUN_ID=$($KFP_COMMAND --endpoint http://localhost:8888  run list 2>&1| grep "$RUN_NAME" | awk '{print $1}')
  if [[ -z "$RUN_ID" ]]; then
    echo "Failed to submit a run for ${TEST_CASE} pipeline"
    return "$REV"
  fi

  local RUN_STATUS
  ENDTIME=$(date -ud "$DURATION minute" +%s)
  while [[ "$(date -u +%s)" -le "$ENDTIME" ]]; do
    RUN_STATUS=$($KFP_COMMAND --endpoint http://localhost:8888 run list 2>&1| grep "$RUN_NAME" | awk '{print $4}')
    if [[ "$RUN_STATUS" == "$F_STATUS" ]]; then
      REV=0
      break;
    fi
    echo "  Status of ${TEST_CASE} run: $RUN_STATUS"
    if [[ "$RUN_STATUS" == "FAILED" ]]; then
      REV=1
      break;
    fi
    sleep 10
  done

  if [[ "$REV" -eq 0 ]]; then
    echo " =====   ${TEST_CASE} PASSED ====="
  else
    echo " =====   ${TEST_CASE} FAILED ====="
  fi

  collect_pipeline_artifacts $RUN_ID

  echo 'y' | $KFP_COMMAND --endpoint http://localhost:8888 run delete "$RUN_ID" || :

  return "$REV"
}
