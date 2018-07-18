#!/bin/bash

# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# K8s Namespace that all resources deployed to


# This file is intended to work with Kubeflow kf_deploy.sh. It's intended
# to be called after Kubeflow is deployed. For more details on how to
# deploy kubeflow, see https://github.com/kubeflow/kubeflow/pull/1159

set -xe

if [[ -z "${GITHUB_USERNAME}" ]]; then
  echo ""
  read -p "Github Username: " GITHUB_USERNAME
fi

if [[ -z "${GITHUB_TOKEN}" ]]; then
  echo ""
  read -p "Github Token: " GITHUB_TOKEN
fi

PIPELINE_REPO=${PIPELINE_REPO:-"`pwd`/pipeline_repo"}
PIPELINE_VERSION=${PIPELINE_VERSION:-"master"}
PIPELINE_DEPLOY=${PIPELINE_DEPLOY:-true}

# Namespace where ml-pipeline is deployed. Default to kubeflow namespace
K8S_NAMESPACE=${K8S_NAMESPACE:-"kubeflow"}

if [[ ! -d "${PIPELINE_REPO}" ]]; then
  if [ "${PIPELINE_VERSION}" == "master" ]; then
    TAG=${PIPELINE_VERSION}
  else
    TAG=v${PIPELINE_VERSION}
  fi
  TMPDIR=$(mktemp -d /tmp/tmp.pipeline-repo-XXXX)
  curl -sL --user "$GITHUB_USERNAME:$GITHUB_TOKEN" https://github.com/googleprivate/ml/archive/${PIPELINE_VERSION}.tar.gz > ${TMPDIR}/pipeline.tar.gz
  tar -xzvf ${TMPDIR}/pipeline.tar.gz  -C ${TMPDIR}
  # GitHub seems to strip out the v in the file name.
  SOURCE_DIR=$(find ${TMPDIR} -maxdepth 1 -type d -name "ml*")
  mv ${SOURCE_DIR} "${PIPELINE_REPO}"
fi


DEPLOYMENT_NAME=${DEPLOYMENT_NAME:-"ml-pipeline"}
KS_DIR=${KUBEFLOW_KS_DIR:-"`pwd`/${DEPLOYMENT_NAME}_app"}

if [[ ! -d "${KUBEFLOW_KS_DIR}" ]]; then
  # Create the ksonnet app
  cd $(dirname "${KS_DIR}")
  ks init $(basename "${KS_DIR}")
  cd "${KS_DIR}"
fi

ks env set default --namespace "${K8S_NAMESPACE}"
ks registry add ml-pipeline "${PIPELINE_REPO}/ml-pipeline"
ks pkg install ml-pipeline/ml-pipeline
ks generate ml-pipeline ml-pipeline

# Apply the ml pipeline component
if ${PIPELINE_DEPLOY}; then
  ks apply default -c ml-pipeline
fi