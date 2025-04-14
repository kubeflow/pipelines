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

USE_PROXY=false

while getopts ":p-:" OPT; do
    case $OPT in
        -) [ "$OPTARG" = "proxy" ] && USE_PROXY=true || { echo "Unknown option --$OPTARG"; exit 1; };;
        \?) echo "Invalid option: -$OPTARG" >&2; exit 1;;
    esac
done

shift $((OPTIND-1))

kubectl apply -k "manifests/kustomize/cluster-scoped-resources/"
kubectl apply -k "manifests/kustomize/base/crds"
kubectl wait crd/applications.app.k8s.io --for condition=established --timeout=60s || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Failed to deploy cluster-scoped resources."
  exit $EXIT_CODE
fi

#Install cert-manager
make -C ./backend install-cert-manager || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Failed to deploy cert-manager."
  exit $EXIT_CODE
fi

# Deploy manifest
TEST_MANIFESTS=".github/resources/manifests/argo"

if [[ "$PIPELINE_STORE" == "kubernetes" ]]; then
  TEST_MANIFESTS=".github/resources/manifests/kubernetes-native"
fi

if $USE_PROXY; then
  TEST_MANIFESTS="${TEST_MANIFESTS}/overlays/proxy"
else
  TEST_MANIFESTS="${TEST_MANIFESTS}/overlays/no-proxy"
fi

# Check if image patch file exists, indicating we need to update the kustomization
if [ -f "/tmp/kfp-patches/image-patch.yaml" ]; then
  echo "Found image patch file. Updating kustomization.yaml to use locally loaded images."
  
  # Create a backup of the original kustomization.yaml
  KUSTOMIZATION_FILE="${TEST_MANIFESTS}/kustomization.yaml"
  cp "${KUSTOMIZATION_FILE}" "${KUSTOMIZATION_FILE}.bak"
  
  # Update the image references in the kustomization.yaml file
  sed -i 's|newName: kind-registry:5000/apiserver|newName: docker.io/library/apiserver|g' "${KUSTOMIZATION_FILE}"
  sed -i 's|newName: kind-registry:5000/persistenceagent|newName: docker.io/library/persistenceagent|g' "${KUSTOMIZATION_FILE}"
  sed -i 's|newName: kind-registry:5000/scheduledworkflow|newName: docker.io/library/scheduledworkflow|g' "${KUSTOMIZATION_FILE}"
  
  # Check if driver and launcher are present, add them if they're not
  if ! grep -q "kfp-driver" "${KUSTOMIZATION_FILE}"; then
    sed -i '/images:/a\- name: ghcr.io/kubeflow/kfp-driver\n  newName: docker.io/library/driver\n  newTag: latest' "${KUSTOMIZATION_FILE}"
  else
    sed -i 's|newName: kind-registry:5000/driver|newName: docker.io/library/driver|g' "${KUSTOMIZATION_FILE}"
  fi
  
  if ! grep -q "kfp-launcher" "${KUSTOMIZATION_FILE}"; then
    sed -i '/images:/a\- name: ghcr.io/kubeflow/kfp-launcher\n  newName: docker.io/library/launcher\n  newTag: latest' "${KUSTOMIZATION_FILE}"
  else
    sed -i 's|newName: kind-registry:5000/launcher|newName: docker.io/library/launcher|g' "${KUSTOMIZATION_FILE}"
  fi
  
  echo "Updated kustomization.yaml:"
  cat "${KUSTOMIZATION_FILE}"
fi

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

