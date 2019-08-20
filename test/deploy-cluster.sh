#!/bin/bash
#
# Copyright 2018 Google LLC
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

set -ex

# Specify TEST_CLUSTER env variable to use an existing cluster.
TEST_CLUSTER_PREFIX=${WORKFLOW_FILE%.*}
TEST_CLUSTER_DEFAULT=$(echo $TEST_CLUSTER_PREFIX | cut -d _ -f 1)-${PULL_PULL_SHA:0:7}-${RANDOM}
TEST_CLUSTER=${TEST_CLUSTER:-${TEST_CLUSTER_DEFAULT}}
SHOULD_CLEANUP_CLUSTER=false

function clean_up {
  echo "Clean up..."
  set +e # the following clean up commands shouldn't exit on error
  if [ $SHOULD_CLEANUP_CLUSTER == true ]; then
    # TODO: remove this
    kubectl cluster-info dump
    # --async doesn't wait for this operation to complete, so we can get test
    # results faster
    yes | gcloud container clusters delete ${TEST_CLUSTER} --async
  fi
}
trap clean_up EXIT SIGINT SIGTERM

cd ${DIR}
if gcloud container clusters describe ${TEST_CLUSTER} &>/dev/null; then
  echo "Use existing test cluster: ${TEST_CLUSTER}"
else
  echo "Creating a new test cluster: ${TEST_CLUSTER}"
  # "storage-rw" is needed to allow VMs to push to gcr.io
  # reference: https://cloud.google.com/compute/docs/access/service-accounts#accesscopesiam
  SCOPE_ARG="--scopes=storage-rw"
  # Machine type and cluster size is the same as kubeflow deployment to
  # easily compare performance. We can reduce usage later.
  NODE_POOL_CONFIG_ARG="--num-nodes=2 --machine-type=n1-standard-8 \
    --enable-autoscaling --max-nodes=8 --min-nodes=2"
  gcloud container clusters create ${TEST_CLUSTER} ${SCOPE_ARG} ${NODE_POOL_CONFIG_ARG}
  SHOULD_CLEANUP_CLUSTER=true
fi

gcloud container clusters get-credentials ${TEST_CLUSTER}

# when we reuse a cluster when debugging, clean up its kfp installation first
# this does nothing with a new cluster
kubectl delete namespace ${NAMESPACE} --wait || echo "No need to delete ${NAMESPACE} namespace. It doesn't exist."
kubectl delete namespace argo --wait || echo "No need to delete argo namespace. It doesn't exist."

kubectl create namespace ${NAMESPACE} --dry-run -o yaml | kubectl apply -f -

if [ -z $SA_KEY_FILE ]; then
  SA_KEY_FILE=${DIR}/key.json
  # The service account key is for default VM service account.
  # ref: https://cloud.google.com/compute/docs/access/service-accounts#compute_engine_default_service_account
  # It was generated by the following command
  # `gcloud iam service-accounts keys create $SA_KEY_FILE --iam-account ${VM_SERVICE_ACCOUNT}`
  # Because there's a limit of 10 keys per service account, we are reusing the same key stored in the following bucket.
  gsutil cp "gs://ml-pipeline-test-keys/ml-pipeline-test-sa-key.json" $SA_KEY_FILE
fi
kubectl create secret -n ${NAMESPACE} generic user-gcp-sa --from-file=user-gcp-sa.json=$SA_KEY_FILE --dry-run -o yaml | kubectl apply -f -
