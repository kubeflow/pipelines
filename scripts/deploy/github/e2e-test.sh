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

# Remove the x if you need no print out of each command
set -e

# Need the following env
# - KUBEFLOW_NS:                            kubeflow namespace

KUBEFLOW_NS="${KUBEFLOW_NS:-kubeflow}"
TEST_SCRIPT="${TEST_SCRIPT:="test-flip-coin.sh"}"

C_DIR="${BASH_SOURCE%/*}"
if [[ ! -d "$C_DIR" ]]; then C_DIR="$PWD"; fi
source "${C_DIR}/helper-functions.sh"

POD_NAME=$(kubectl get pod -n kubeflow -l app=ml-pipeline -o json | jq -r '.items[] | .metadata.name ')
kubectl port-forward -n "$KUBEFLOW_NS" "$POD_NAME" 8888:8888 2>&1 > /dev/null &
# wait for the port-forward
sleep 5

if [ -n "$TEST_SCRIPT" ]; then
  source "${C_DIR}/${TEST_SCRIPT}"
fi

kill %1

if [[ "$RESULT" -ne 0 ]]; then
  echo "e2e test ${STATUS_MSG}"
  exit 1
fi

echo "e2e test ${STATUS_MSG}"

