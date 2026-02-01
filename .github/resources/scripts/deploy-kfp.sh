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

TEST_MANIFESTS=".github/resources/manifests"
PIPELINES_STORE="database"
USE_PROXY=false
CACHE_DISABLED=false
ARTIFACT_PROXY_ENABLED=false
MULTI_USER=false
AWF_VERSION=""
POD_TO_POD_TLS_ENABLED=false
SEAWEEDFS_INIT_TIMEOUT=300s

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
    --multi-user)
      MULTI_USER=true
      shift
      ;;
    --artifact-proxy)
      ARTIFACT_PROXY_ENABLED=true
      shift
      ;;
    --argo-version)
      shift
      if [[ -n "$1" ]]; then
        AWF_VERSION="$1"
        shift
      else
        echo "ERROR: --argo-version requires an argument"
        exit 1
      fi
      ;;
    --tls-enabled)
      POD_TO_POD_TLS_ENABLED=true
      shift
      ;;
  esac
done

if [ "${USE_PROXY}" == "true" ] && [ "${PIPELINES_STORE}" == "kubernetes" ]; then
  echo "ERROR: Kubernetes Pipeline store cannot be deployed with proxy support."
  exit 1
fi

if [ "${MULTI_USER}" == "true" ] && [ "${USE_PROXY}" == "true" ]; then
  echo "ERROR: Multi-user mode cannot be deployed with proxy support."
  exit 1
fi

if [ -n "${AWF_VERSION}"  ]; then
  echo "NOTE: Argo version ${AWF_VERSION} specified, updating Argo Workflow manifests..."
  echo "${AWF_VERSION}" > third_party/argo/VERSION
  make -C ./third_party/argo update_manifests
  echo "Manifests updated for Argo version ${AWF_VERSION}."
fi

kubectl apply -k "manifests/kustomize/cluster-scoped-resources/" || EXIT_CODE=$?

kubectl wait crd/applications.app.k8s.io --for condition=established --timeout=60s || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]; then
  echo "Failed to deploy cluster-scoped resources."
  exit $EXIT_CODE
fi

# If pipelines store is set to 'kubernetes' or pod-to-pod TLS is set to 'true', cert-manager must be deployed
if [ "${PIPELINES_STORE}" == "kubernetes" ] || [ "${POD_TO_POD_TLS_ENABLED}" == "true" ]; then
  #Install cert-manager
  make -C ./backend install-cert-manager || EXIT_CODE=$?
  if [[ $EXIT_CODE -ne 0 ]]
  then
    echo "Failed to deploy cert-manager."
    exit $EXIT_CODE
  fi
fi


# Deploy multi-user prerequisites if multi-user mode is enabled
if [ "${MULTI_USER}" == "true" ]; then
  echo "Installing Istio..."
  kubectl apply -k https://github.com/kubeflow/manifests/common/istio/istio-crds/base?ref=master
  kubectl apply -k https://github.com/kubeflow/manifests/common/istio/istio-namespace/base?ref=master
  kubectl apply -k https://github.com/kubeflow/manifests/common/istio/istio-install/base?ref=master
  echo "Waiting for all Istio Pods to become ready..."
  kubectl wait --for=condition=Ready pods --all -n istio-system --timeout=300s

  echo "Deploying Metacontroller CRD..."
  kubectl apply -f manifests/kustomize/third-party/metacontroller/base/crd.yaml
  kubectl wait --for condition=established --timeout=30s crd/compositecontrollers.metacontroller.k8s.io

  echo "Installing Profile Controller Resources..."
  kubectl apply -k https://github.com/kubeflow/manifests/applications/profiles/upstream/overlays/kubeflow?ref=master
  kubectl -n kubeflow wait --for=condition=Ready pods -l kustomize.component=profiles --timeout 180s
fi

# Manifests will be deployed according to the flag provided
if [ "${MULTI_USER}" == "false" ] && [ "${PIPELINES_STORE}" != "kubernetes" ]; then
  TEST_MANIFESTS="${TEST_MANIFESTS}/standalone"
  if $CACHE_DISABLED && $USE_PROXY; then
    TEST_MANIFESTS="${TEST_MANIFESTS}/cache-disabled-proxy"
  elif $CACHE_DISABLED; then
    TEST_MANIFESTS="${TEST_MANIFESTS}/cache-disabled"
  elif $USE_PROXY; then
    TEST_MANIFESTS="${TEST_MANIFESTS}/proxy"
  elif $POD_TO_POD_TLS_ENABLED; then
    TEST_MANIFESTS="${TEST_MANIFESTS}/tls-enabled"
  else
    TEST_MANIFESTS="${TEST_MANIFESTS}/default"
  fi
elif [ "${MULTI_USER}" == "false" ] && [ "${PIPELINES_STORE}" == "kubernetes" ]; then
  TEST_MANIFESTS="${TEST_MANIFESTS}/kubernetes-native"
  if $CACHE_DISABLED; then
    TEST_MANIFESTS="${TEST_MANIFESTS}/cache-disabled"
  else
    TEST_MANIFESTS="${TEST_MANIFESTS}/default"
  fi
elif [ "${MULTI_USER}" == "true" ]; then
  TEST_MANIFESTS="${TEST_MANIFESTS}/multiuser"
  if $ARTIFACT_PROXY_ENABLED; then
    TEST_MANIFESTS="${TEST_MANIFESTS}/artifact-proxy"
  elif $CACHE_DISABLED; then
    TEST_MANIFESTS="${TEST_MANIFESTS}/cache-disabled"
  else
    TEST_MANIFESTS="${TEST_MANIFESTS}/default"
  fi
fi


echo "Deploying ${TEST_MANIFESTS}..."

kubectl kustomize --load-restrictor LoadRestrictionsNone "${TEST_MANIFESTS}" | kubectl apply -f - || EXIT_CODE=$?
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

# Ensure SeaweedFS S3 auth is configured before proceeding
wait_for_seaweedfs_init kubeflow "${SEAWEEDFS_INIT_TIMEOUT}" || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]
then
  echo "SeaweedFS init job did not complete successfully."
  exit 1
fi
echo "SeaweedFS init job completed successfully."

if [ "${MULTI_USER}" == "true" ]; then
  echo "Creating KF Profile..."
  kubectl apply -f test_data/kubernetes/seaweedfs/test-profiles.yaml
  sleep 30 # Let the profile controler reconcile the namespace

  echo "Applying kubeflow-edit ClusterRole with proper aggregation..."
  kubectl apply -f test_data/kubernetes/seaweedfs/kubeflow-edit-clusterrole.yaml

  echo "Applying network policy to allow user namespace access to kubeflow services..."
  kubectl apply -f test_data/kubernetes/seaweedfs/allow-user-namespace-access.yaml
fi

# Verify pipeline integration for multi-user mode
if [ "${MULTI_USER}" == "true" ]; then
  echo "Verifying Pipeline Integration..."
  KF_PROFILE=kubeflow-user-example-com
  if ! kubectl get secret mlpipeline-minio-artifact -n $KF_PROFILE > /dev/null 2>&1; then
    echo "Error: Secret mlpipeline-minio-artifact not found in namespace $KF_PROFILE"
  fi
  kubectl get secret mlpipeline-minio-artifact -n "$KF_PROFILE" -o json | jq -r '.data | keys[] as $k | "\($k): \(. | .[$k] | @base64d)"' | tr '\n' ' '
fi

collect_artifacts kubeflow

echo "Finished KFP deployment."
