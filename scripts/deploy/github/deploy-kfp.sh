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

REGISTRY="${REGISTRY:-kind-registry:5000}"
EXIT_CODE=0

C_DIR="${BASH_SOURCE%/*}"
if [[ ! -d "$C_DIR" ]]; then C_DIR="$PWD"; fi
source "${C_DIR}/helper-functions.sh"

kubectl apply -k "manifests/kustomize/cluster-scoped-resources/"
kubectl wait crd/applications.app.k8s.io --for condition=established --timeout=60s || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Failed to deploy cluster-scoped resources."
  exit $EXIT_CODE
fi

PLATFORM_AGNOSTIC_MANIFESTS="manifests/kustomize/env/platform-agnostic"

kubectl apply -k "${PLATFORM_AGNOSTIC_MANIFESTS}" || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Deploy unsuccessful. Failure applying ${PLATFORM_AGNOSTIC_MANIFESTS}."
  exit 1
fi

# Check if all pods are running - (10 minutes)
wait_for_pods || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Deploy unsuccessful. Not all pods running."
  exit 1
fi

collect_artifacts kubeflow

echo "Finished KFP deployment."

