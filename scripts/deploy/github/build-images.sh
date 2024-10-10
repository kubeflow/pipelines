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
# source: https://raw.githubusercontent.com/open-toolchain/commons/master/scripts/check_registry.sh

# Remove the x if you need no print out of each command
set -e

REGISTRY="${REGISTRY:-kind-registry:5000}"
echo "REGISTRY=$REGISTRY"
TAG="${TAG:-latest}"
EXIT_CODE=0

docker system prune -a -f

docker build -q -t "${REGISTRY}/apiserver:${TAG}" -f backend/Dockerfile . && docker push "${REGISTRY}/apiserver:${TAG}" || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Failed to build apiserver image."
  exit $EXIT_CODE
fi

docker build -q -t "${REGISTRY}/persistenceagent:${TAG}" -f backend/Dockerfile.persistenceagent . && docker push "${REGISTRY}/persistenceagent:${TAG}" || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Failed to build persistenceagent image."
  exit $EXIT_CODE
fi

docker build -q -t "${REGISTRY}/scheduledworkflow:${TAG}" -f backend/Dockerfile.scheduledworkflow . && docker push "${REGISTRY}/scheduledworkflow:${TAG}" || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Failed to build scheduledworkflow image."
  exit $EXIT_CODE
fi

docker build -q -t "${REGISTRY}/driver:${TAG}" -f backend/Dockerfile.driver . && docker push "${REGISTRY}/driver:${TAG}" || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Failed to build driver image."
  exit $EXIT_CODE
fi

docker build -q -t "${REGISTRY}/launcher:${TAG}" -f backend/Dockerfile.launcher . && docker push "${REGISTRY}/launcher:${TAG}" || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Failed to build launcher image."
  exit $EXIT_CODE
fi

# clean up intermittent build caches to free up disk space
docker system prune -a -f
