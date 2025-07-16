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

TEST_MANIFESTS=".github/resources/manifests/argo"
PIPELINES_STORE="database"
USE_PROXY=false
CACHE_DISABLED=false

# Loop over script arguments passed. This uses a single switch-case
# block with default value in case we want to make alternative deployments
# in the future.
while [ "$#" -gt 0 ]; do
  case "$1" in
    --deploy-k8s-native)
      PIPELINES_STORE="kubernetes"
      shift
      ;;
    --proxy)
      USE_PROXY=true
      shift
      ;;
    --cache-disabled)
      CACHE_DISABLED=true
      shift
      ;;
  esac
done

if [ "${USE_PROXY}" == "true" && "${PIPELINES_STORE}" == "kubernetes" ]; then
  echo "ERROR: Kubernetes Pipeline store cannot be deployed with proxy support."
  exit 1
fi

kubectl apply -k "manifests/kustomize/cluster-scoped-resources/"
kubectl wait crd/applications.app.k8s.io --for condition=established --timeout=60s || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Failed to deploy cluster-scoped resources."
  exit $EXIT_CODE
fi

# If pipelines store is set to 'kubernetes', cert-manager must be deployed
if [ "${PIPELINES_STORE}" == "kubernetes" ]; then
  #Install cert-manager
  make -C ./backend install-cert-manager || EXIT_CODE=$?
  if [[ $EXIT_CODE -ne 0 ]]
  then
    echo "Failed to deploy cert-manager."
    exit $EXIT_CODE
  fi
fi

# Manifests will be deployed according to the flag provided
if $CACHE_DISABLED; then
  TEST_MANIFESTS="${TEST_MANIFESTS}/overlays/cache-disabled"
elif $USE_PROXY; then
  TEST_MANIFESTS="${TEST_MANIFESTS}/overlays/proxy"
elif [ "${PIPELINES_STORE}" == "kubernetes" ]; then
  TEST_MANIFESTS="${TEST_MANIFESTS}/overlays/kubernetes-native"
else
  TEST_MANIFESTS="${TEST_MANIFESTS}/overlays/no-proxy"
fi

echo "Deploying ${TEST_MANIFESTS}..."

kubectl apply -k "${TEST_MANIFESTS}" || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Deploy unsuccessful. Failure applying ${TEST_MANIFESTS}."
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
