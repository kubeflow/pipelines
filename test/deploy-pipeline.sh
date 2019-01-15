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

set -x


usage()
{
    echo "usage: deploy.sh
    [--platform             the deployment platform. Valid values are: [gcp, minikube]]
    [--project 				the project name.]
    [--test_cluster]		the name of the testing cluster]
    [--gcr_image_base_dir   the gcr image base directory including images such as apiImage and persistenceAgentImage]
    [--gcr_image_tag   		the tags for images such as apiImage and persistenceAgentImage]
    [-h help]"
}
GCR_IMAGE_TAG=latest

while [ "$1" != "" ]; do
    case $1 in
             --platform )             shift
                                      PLATFORM=$1
                                      ;;
             --project )              shift
                                      PROJECT=$1
                                      ;;
             --test_cluster )         shift
                                      TEST_CLUSTER=$1
                                      ;;
             --gcr_image_base_dir )   shift
                                      GCR_IMAGE_BASE_DIR=$1
                                      ;;
             --gcr_image_tag )   	  shift
                                      GCR_IMAGE_TAG=$1
                                      ;;                                      
             -h | --help )            usage
                                      exit
                                      ;;
             * )                      usage
                                      exit 1
    esac
    shift
done

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"

# Install ksonnet
KS_VERSION="0.11.0"
curl -LO https://github.com/ksonnet/ksonnet/releases/download/v${KS_VERSION}/ks_${KS_VERSION}_linux_amd64.tar.gz
tar -xzf ks_${KS_VERSION}_linux_amd64.tar.gz
chmod +x ./ks_${KS_VERSION}_linux_amd64/ks
mv ./ks_${KS_VERSION}_linux_amd64/ks /usr/local/bin/

# Download kubeflow master
KUBEFLOW_MASTER=${DIR}/kubeflow_master
git clone https://github.com/kubeflow/kubeflow.git ${KUBEFLOW_MASTER}

## Download latest kubeflow release source code
KUBEFLOW_SRC=${DIR}/kubeflow_latest_release
mkdir ${KUBEFLOW_SRC}
cd ${KUBEFLOW_SRC}
export KUBEFLOW_TAG=v0.3.1
curl https://raw.githubusercontent.com/kubeflow/kubeflow/${KUBEFLOW_TAG}/scripts/download.sh | bash

## Override the pipeline config with code from master
cp -r ${KUBEFLOW_MASTER}/kubeflow/pipeline ${KUBEFLOW_SRC}/kubeflow/pipeline
cp -r ${KUBEFLOW_MASTER}/kubeflow/argo ${KUBEFLOW_SRC}/kubeflow/argo

# TODO temporarily set KUBEFLOW_SRC as KUBEFLOW_MASTER. This should be deleted when latest release have the pipeline entry
KUBEFLOW_SRC=${KUBEFLOW_MASTER}

export CLIENT_ID=${RANDOM}
export CLIENT_SECRET=${RANDOM}
KFAPP=${TEST_CLUSTER}

function clean_up {
  echo "Clean up..."
  cd ${KFAPP}
  ${KUBEFLOW_SRC}/scripts/kfctl.sh delete all
}
trap clean_up EXIT

${KUBEFLOW_SRC}/scripts/kfctl.sh init ${KFAPP} --platform ${PLATFORM} --project ${PROJECT} --skipInitProject

cd ${KFAPP}
${KUBEFLOW_SRC}/scripts/kfctl.sh generate platform
${KUBEFLOW_SRC}/scripts/kfctl.sh apply platform
${KUBEFLOW_SRC}/scripts/kfctl.sh generate k8s

## Update pipeline component image
pushd ks_app
ks param set pipeline apiImage ${GCR_IMAGE_BASE_DIR}/api:${GCR_IMAGE_TAG}
ks param set pipeline persistenceAgentImage ${GCR_IMAGE_BASE_DIR}/persistenceagent:${GCR_IMAGE_TAG}
ks param set pipeline scheduledWorkflowImage ${GCR_IMAGE_BASE_DIR}/scheduledworkflow:${GCR_IMAGE_TAG}
ks param set pipeline uiImage ${GCR_IMAGE_BASE_DIR}/frontend:${GCR_IMAGE_TAG}
popd

${KUBEFLOW_SRC}/scripts/kfctl.sh apply k8s