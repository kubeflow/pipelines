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

REGISTRY1="${REGISTRY1:-docker.io/aipipeline}"
REGISTRY2="${REGISTRY2:-gcr.io/ml-pipeline}"
TAG1="${TAG1:-latest}"
TAG2="${TAG2:-latest}"

docker system prune -a -f

declare -a IMAGES=(apiserver persistenceagent scheduledworkflow tekton-driver)

for IMAGE in "${IMAGES[@]}"; do
    docker pull "${REGISTRY1}/${IMAGE}:${TAG1}"
    docker tag "${REGISTRY1}/${IMAGE}:${TAG1}" "${REGISTRY2}/${IMAGE}:${TAG2}"
    docker push "${REGISTRY2}/${IMAGE}:${TAG2}"
done

# clean up intermittent build caches to free up disk space
docker system prune -a -f
