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

TEST_CLUSTER_PREFIX=${WORKFLOW_FILE%.*}
TEST_CLUSTER=$(echo $TEST_CLUSTER_PREFIX | cut -d _ -f 1)-${PULL_PULL_SHA:0:7}-${RANDOM}

# Install ksonnet
KS_VERSION="0.13.0"

curl -LO https://github.com/ksonnet/ksonnet/releases/download/v${KS_VERSION}/ks_${KS_VERSION}_linux_amd64.tar.gz
tar -xzf ks_${KS_VERSION}_linux_amd64.tar.gz
chmod +x ./ks_${KS_VERSION}_linux_amd64/ks
# Add ks to PATH
PATH=$PATH:`pwd`/ks_${KS_VERSION}_linux_amd64

## Download latest kubeflow release source code
KUBEFLOW_SRC=${DIR}/kubeflow_latest_release
mkdir ${KUBEFLOW_SRC}
cd ${KUBEFLOW_SRC}
export KUBEFLOW_TAG=pipelines
curl https://raw.githubusercontent.com/kubeflow/kubeflow/${KUBEFLOW_TAG}/scripts/download.sh | bash

export CLIENT_ID=${RANDOM}
export CLIENT_SECRET=${RANDOM}
KFAPP=${TEST_CLUSTER}

function clean_up {
  echo "Clean up..."
  cd ${DIR}/${KFAPP}
  ${KUBEFLOW_SRC}/scripts/kfctl.sh delete all
  # delete the storage
  gcloud deployment-manager --project=${PROJECT} deployments delete ${KFAPP}-storage --quiet
}
trap clean_up EXIT SIGINT SIGTERM

cd ${DIR}
${KUBEFLOW_SRC}/scripts/kfctl.sh init ${KFAPP} --platform ${PLATFORM} --project ${PROJECT} --skipInitProject

cd ${KFAPP}
${KUBEFLOW_SRC}/scripts/kfctl.sh generate platform
${KUBEFLOW_SRC}/scripts/kfctl.sh apply platform
${KUBEFLOW_SRC}/scripts/kfctl.sh generate k8s
${KUBEFLOW_SRC}/scripts/kfctl.sh apply k8s

gcloud container clusters get-credentials ${TEST_CLUSTER}
