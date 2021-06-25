#!/bin/bash
#
# Copyright 2021 The Kubeflow Authors
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

# This script builds argo images and pushes them to KFP's gcr container registry.
# Usage: ./release.sh
# It can be run anywhere.

set -ex

# Get this bash script's dir.
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"

PROJECT=ml-pipeline
TAG_ARGO="$(cat ${DIR}/VERSION)"
TAG=${TAG_ARGO}-license-compliance
IMAGE_ARGOEXEC="gcr.io/${PROJECT}/argoexec:${TAG}"
IMAGE_WORKFLOW_CONTROLLER="gcr.io/${PROJECT}/workflow-controller:${TAG}"

# make sure we don't override existing remote images
docker pull "${IMAGE_ARGOEXEC}" && (echo "Error: ${IMAGE_ARGOEXEC} already exists remotely" && exit 1)
docker pull "${IMAGE_WORKFLOW_CONTROLLER}" && (echo "Error: ${IMAGE_WORKFLOW_CONTROLLER} already exists remotely" && exit 1)

# build images locally
docker build -t "${IMAGE_ARGOEXEC}" -f Dockerfile.argoexec "${DIR}" --build-arg TAG=${TAG_ARGO}
docker build -t "${IMAGE_WORKFLOW_CONTROLLER}" -f Dockerfile.workflow-controller "${DIR}" --build-arg TAG=${TAG_ARGO}

# push them to remote
docker push "${IMAGE_ARGOEXEC}"
docker push "${IMAGE_WORKFLOW_CONTROLLER}"
