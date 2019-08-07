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
SHOULD_DELETE_CLUSTER=0

SA_KEY_FILE=${DIR}/key-${RANDOM}.json
KEY_ID=
if [ -z ${VM_SERVICE_ACCOUNT} ]; then
  echo "VM_SERVICE_ACCOUNT env variable is needed by deploy-cluster.sh"
  exit 1
fi

function clean_up {
  echo "Clean up..."
  set +e # the following clean up commands shouldn't exit on error
  if [ $SHOULD_DELETE_CLUSTER == 1 ]; then
    yes | gcloud container clusters delete ${TEST_CLUSTER}
  fi
  rm -f ${SA_KEY_FILE}

  # There's a max limit of keys downloaded for a single service account. So we
  # need to delete unused keys during clean_up.
  if [ ! -z $KEY_ID ]; then
    yes | gcloud iam service-accounts keys delete $KEY_ID --iam-account ${VM_SERVICE_ACCOUNT}
  fi
}
trap clean_up EXIT SIGINT SIGTERM

cd ${DIR}
if gcloud container clusters describe ${TEST_CLUSTER}; then
  echo "Use existing test cluster: ${TEST_CLUSTER}"
else
  echo "Creating a new test cluster: ${TEST_CLUSTER}"
  # storage-rw is needed to allow VMs to push to gcr.io
  # reference: https://cloud.google.com/compute/docs/access/service-accounts#accesscopesiam
  gcloud container clusters create ${TEST_CLUSTER} --scopes=storage-rw
  SHOULD_DELETE_CLUSTER=1
fi

gcloud container clusters get-credentials ${TEST_CLUSTER}

# when we reuse a cluster when debugging, clean up its kfp installation first
# this does nothing with a new cluster
kubectl delete namespace ${NAMESPACE} --wait || echo "No need to delete ${NAMESPACE} namespace. It doesn't exist."

kubectl create namespace ${NAMESPACE} --dry-run -o yaml | kubectl apply -f -
KEY_ID=$(gcloud iam service-accounts keys create $SA_KEY_FILE --iam-account ${VM_SERVICE_ACCOUNT} 2>&1 >/dev/null | grep -oP "(?<=created key \[)\w+");
echo "KEY_ID=${KEY_ID}"
kubectl create secret -n ${NAMESPACE} generic user-gcp-sa --from-file=user-gcp-sa.json=$SA_KEY_FILE --dry-run -o yaml | kubectl apply -f -
