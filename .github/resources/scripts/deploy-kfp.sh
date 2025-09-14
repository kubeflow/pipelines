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
MULTI_USER=false
STORAGE_BACKEND="seaweedfs"
AWF_VERSION=""
DEPLOY_POSTGRES=false
POSTGRES_NS="${POSTGRES_NS:-kfp-pgx-test}"

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
    --storage)
      STORAGE_BACKEND="$2"
      shift 2
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
    --deploy-postgres)
      DEPLOY_POSTGRES=true
      shift
      ;;
    --postgres-namespace)
      shift
      if [[ -n "$1" ]]; then
        POSTGRES_NS="$1"
        shift
      else
        echo "ERROR: --postgres-namespace requires an argument"
        exit 1
      fi
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

if [ "${STORAGE_BACKEND}" != "minio" ] && [ "${STORAGE_BACKEND}" != "seaweedfs" ]; then
  echo "ERROR: Storage backend must be either 'minio' or 'seaweedfs'."
  exit 1
fi

if [ -n "${AWF_VERSION}"  ]; then
  echo "NOTE: Argo version ${AWF_VERSION} specified, updating Argo Workflow manifests..."
  echo "${AWF_VERSION}" > third_party/argo/VERSION
  make -C ./manifests/kustomize/third-party/argo update
  echo "Manifests updated for Argo version ${AWF_VERSION}."
fi

kubectl apply -k "manifests/kustomize/cluster-scoped-resources/" || EXIT_CODE=$?

kubectl wait crd/applications.app.k8s.io --for condition=established --timeout=60s || EXIT_CODE=$?
if [[ $EXIT_CODE -ne 0 ]]; then
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


# Deploy multi-user prerequisites if multi-user mode is enabled
if [ "${MULTI_USER}" == "true" ]; then
  echo "Installing Istio..."
  kubectl apply -k https://github.com/kubeflow/manifests//common/istio/istio-crds/base?ref=master
  kubectl apply -k https://github.com/kubeflow/manifests//common/istio/istio-namespace/base?ref=master
  kubectl apply -k https://github.com/kubeflow/manifests//common/istio/istio-install/base?ref=master
  echo "Waiting for all Istio Pods to become ready..."
  kubectl wait --for=condition=Ready pods --all -n istio-system --timeout=300s

  echo "Deploying Metacontroller CRD..."
  kubectl apply -f manifests/kustomize/third-party/metacontroller/base/crd.yaml
  kubectl wait --for condition=established --timeout=30s crd/compositecontrollers.metacontroller.k8s.io

  echo "Installing Profile Controller Resources..."
  kubectl apply -k https://github.com/kubeflow/manifests/applications/profiles/upstream/overlays/kubeflow?ref=master
  kubectl -n kubeflow wait --for=condition=Ready pods -l kustomize.component=profiles --timeout 180s

  echo "Creating KF Profile..."
  kubectl apply -f test/seaweedfs/test-profiles.yaml

  echo "Applying kubeflow-edit ClusterRole with proper aggregation..."
  kubectl apply -f test/seaweedfs/kubeflow-edit-clusterrole.yaml

  echo "Applying network policy to allow user namespace access to kubeflow services..."
  kubectl apply -f test/seaweedfs/allow-user-namespace-access.yaml
fi

# Manifests will be deployed according to the flag provided
if $CACHE_DISABLED; then
  TEST_MANIFESTS="${TEST_MANIFESTS}/overlays/cache-disabled"
elif $USE_PROXY; then
  TEST_MANIFESTS="${TEST_MANIFESTS}/overlays/proxy"
elif [ "${PIPELINES_STORE}" == "kubernetes" ]; then
  TEST_MANIFESTS="${TEST_MANIFESTS}/overlays/kubernetes-native"
elif [ "${MULTI_USER}" == "true" ] && [ "${STORAGE_BACKEND}" == "seaweedfs" ]; then
  TEST_MANIFESTS="${TEST_MANIFESTS}/overlays/multi-user"
elif [ "${MULTI_USER}" == "true" ] && [ "${STORAGE_BACKEND}" == "minio" ]; then
  TEST_MANIFESTS="${TEST_MANIFESTS}/overlays/multi-user-minio"
elif [ "${STORAGE_BACKEND}" == "minio" ]; then
  TEST_MANIFESTS="${TEST_MANIFESTS}/overlays/no-proxy-minio"
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

# ----------------------------------------------------------------------
# Optional: Deploy Postgres for PGX CI and wait for readiness
# Encapsulates: ensure namespace -> apply (with retries) -> wait by label
# ----------------------------------------------------------------------
if [[ "${DEPLOY_POSTGRES}" == "true" ]]; then
  echo "[deploy-kfp] Deploying Postgres to namespace: ${POSTGRES_NS}"

  # Ensure namespace exists (idempotent). If we had to create it, make sure it shows up.
  if ! kubectl get ns "${POSTGRES_NS}" >/dev/null 2>&1; then
    kubectl create namespace "${POSTGRES_NS}"
    wait_for_namespace "${POSTGRES_NS}" 30 2 || {
      echo "[deploy-kfp] Namespace ${POSTGRES_NS} did not appear in time."
      exit 1
    }
  fi

  # Sanity check: ensure there is a default StorageClass before deploying PG
  if ! kubectl get storageclass | grep -q "(default)"; then
    echo "[deploy-kfp] ERROR: No default StorageClass found."
    echo "Existing StorageClasses:"
    kubectl get storageclass -o wide || true
    echo "Hint: kfp-cluster action should provision a default SC on KinD;"
    echo "      if you're running on a different environment, please ensure a default SC exists."
    exit 1
  fi


  # Apply Postgres manifests with retries via helper (kustomize dir)
  if ! deploy_with_retries -k "manifests/kustomize/third-party/postgresql/base" 5 10; then
    echo "[deploy-kfp] Failed to apply Postgres manifests after retries."
    exit 1
  fi

  # Wait until all Postgres pods are Ready (by label)
  if ! wait_for_selector_ready "${POSTGRES_NS}" "app=postgres" "300s"; then
    echo "[deploy-kfp] Postgres pods did not become Ready in namespace ${POSTGRES_NS}."
    kubectl -n "${POSTGRES_NS}" get pods -o wide || true
    kubectl -n "${POSTGRES_NS}" get events --sort-by=.metadata.creationTimestamp | tail -n 200 || true
    exit 1
  fi

  echo "[deploy-kfp] Postgres is Ready in ${POSTGRES_NS}."
fi


echo "Finished KFP deployment."
