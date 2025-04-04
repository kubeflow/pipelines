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

# This script builds KFP backend images and saves them to a tar archive.
# Usage: ./build-images.sh <output-tar-path>

# Remove the x if you need no print out of each command
set -e

if [ -z "$1" ]; then
  echo "Error: Output tar path argument is required."
  exit 1
fi
ARCHIVE_PATH="$1"

REGISTRY="${REGISTRY:-kind-registry:5000}"
echo "REGISTRY=$REGISTRY"
TAG="${TAG:-latest}"
EXIT_CODE=0

# Define image names
APISERVER_IMG="${REGISTRY}/apiserver:${TAG}"
PERSISTENCEAGENT_IMG="${REGISTRY}/persistenceagent:${TAG}"
SCHEDULEDWORKFLOW_IMG="${REGISTRY}/scheduledworkflow:${TAG}"
DRIVER_IMG="${REGISTRY}/driver:${TAG}"
LAUNCHER_IMG="${REGISTRY}/launcher:${TAG}"

# Array to hold all image names
IMAGES_TO_SAVE=()

docker system prune -a -f

echo "Building apiserver image: ${APISERVER_IMG}"
docker build --progress=plain -t "${APISERVER_IMG}" -f backend/Dockerfile . || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]; then
  echo "Failed to build apiserver image."
  exit $EXIT_CODE
fi
IMAGES_TO_SAVE+=("${APISERVER_IMG}")

echo "Building persistenceagent image: ${PERSISTENCEAGENT_IMG}"
docker build --progress=plain -t "${PERSISTENCEAGENT_IMG}" -f backend/Dockerfile.persistenceagent . || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]; then
  echo "Failed to build persistenceagent image."
  exit $EXIT_CODE
fi
IMAGES_TO_SAVE+=("${PERSISTENCEAGENT_IMG}")

echo "Building scheduledworkflow image: ${SCHEDULEDWORKFLOW_IMG}"
docker build --progress=plain -t "${SCHEDULEDWORKFLOW_IMG}" -f backend/Dockerfile.scheduledworkflow . || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]; then
  echo "Failed to build scheduledworkflow image."
  exit $EXIT_CODE
fi
IMAGES_TO_SAVE+=("${SCHEDULEDWORKFLOW_IMG}")

echo "Building driver image: ${DRIVER_IMG}"
docker build --progress=plain -t "${DRIVER_IMG}" -f backend/Dockerfile.driver . || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]; then
  echo "Failed to build driver image."
  exit $EXIT_CODE
fi
IMAGES_TO_SAVE+=("${DRIVER_IMG}")

echo "Building launcher image: ${LAUNCHER_IMG}"
docker build --progress=plain -t "${LAUNCHER_IMG}" -f backend/Dockerfile.launcher . || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]; then
  echo "Failed to build launcher image."
  exit $EXIT_CODE
fi
IMAGES_TO_SAVE+=("${LAUNCHER_IMG}")

echo "Saving images to ${ARCHIVE_PATH}:"
echo "${IMAGES_TO_SAVE[@]}"
docker save -o "${ARCHIVE_PATH}" "${IMAGES_TO_SAVE[@]}" || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]; then
  echo "Failed to save images to ${ARCHIVE_PATH}."
  exit $EXIT_CODE
fi

echo "Successfully built and saved images to ${ARCHIVE_PATH}"

KIND_REGISTRY="kind-registry:5000"
KIND_CLUSTER_NAME="kind"

echo "Pushing third-party images to local registry..."
docker pull google/cloud-sdk:279.0.0
docker tag google/cloud-sdk:279.0.0 "${KIND_REGISTRY}/google/cloud-sdk:279.0.0"
docker push "${KIND_REGISTRY}/google/cloud-sdk:279.0.0"

docker pull library/bash:4.4.23
docker tag library/bash:4.4.23 "${KIND_REGISTRY}/library/bash:4.4.23"
docker push "${KIND_REGISTRY}/library/bash:4.4.23"

echo "Loading images from archive ${ARCHIVE_PATH} into kind cluster ${KIND_CLUSTER_NAME}..."
kind load image-archive "${ARCHIVE_PATH}" --name "${KIND_CLUSTER_NAME}"

# clean up intermittent build caches to free up disk space
docker system prune -a -f

exit 0
