#!/bin/bash
#
# Copyright 2020 Google LLC
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

function clean_up {
  set +e # the following clean up commands shouldn't exit on error
  set +x # no need for command history

  echo "Status of pods before clean up:"
  kubectl get pods --all-namespaces

  echo "Clean up service accounts..."
  for sa_suffix in 'admin' 'user' 'vm';
  do
      gcloud iam service-accounts delete ${TEST_CLUSTER}-${sa_suffix}@${PROJECT}.iam.gserviceaccount.com --quiet
  done

  echo "Clean up cluster..."
  # --async doesn't wait for this operation to complete, so we can get test
  # results faster
  gcloud container clusters delete ${TEST_CLUSTER} --async --quiet
}

trap clean_up EXIT SIGINT SIGTERM

function download-kfctl {
  if which kfctl; then
    return 0
  fi

  local tmpDir=$(mktemp -d)
  pushd $tmpDir

  wget -O kfctl.tar.gz https://github.com/kubeflow/kfctl/releases/download/v1.0.2/kfctl_v1.0.2-0-ga476281_linux.tar.gz
  tar -xvf kfctl.tar.gz
  cp kfctl ~/bin/
  export PATH=~/bin/:$PATH
  popd
  rm -rf $tmpDir
}

function download-kustomize {
  if ! which kustomize; then
    # Download kustomize cli tool
    TOOL_DIR=${DIR}/bin
    mkdir -p ${TOOL_DIR}
    wget --no-verbose https://github.com/kubernetes-sigs/kustomize/releases/download/v3.1.0/kustomize_3.1.0_linux_amd64 \
      -O ${TOOL_DIR}/kustomize --no-verbose
    chmod +x ${TOOL_DIR}/kustomize
    PATH=${PATH}:${TOOL_DIR}
  fi
}

function deploy-kubeflow-cli-env {
  # Set your GCP project ID and the zone where you want to create
  # the Kubeflow deployment:
  # export PROJECT=<your GCP project ID>
  # gcloud config set project ${PROJECT}
  # export ZONE=<your GCP zone>
  # gcloud config set compute/zone ${ZONE}
  # ========
  # Above is already done in test-prep.sh

  # TODO: update the path once finalized release
  export CONFIG_URI="https://raw.githubusercontent.com/Bobgy/manifests/stage2-bugbash-1/kfdef/kfctl_gcp_iap.yaml"

  # If using Cloud IAP for authentication, create environment variables
  # from the OAuth client ID and secret that you obtained earlier:
  set +x
  export CLIENT_ID=${CLIENT_ID:-$(gsutil cat gs://ml-pipeline-test-keys/oauth_client_id)}
  export CLIENT_SECRET=${CLIENT_SECRET:-$(gsutil cat gs://ml-pipeline-test-keys/oauth_client_secret)}
  set -x

  # Set KF_NAME to the name of your Kubeflow deployment. You also use this
  # value as directory name when creating your configuration directory.
  # See the detailed description in the text below this code snippet.
  # For example, your deployment name can be 'my-kubeflow' or 'kf-test'.
  export KF_NAME="$1"

  # Set the path to the base directory where you want to store one or more
  # Kubeflow deployments. For example, /opt/.
  # Then set the Kubeflow application directory for this deployment.
  BASE_DIR="$HOME/kf-deployment"
  export KF_DIR=${BASE_DIR}/${KF_NAME}
}

function wait-cluster-ready {
  # wait for all deployments / stateful sets to be successful
  local deployment
  for deployment in $(kubectl get deployments -n ${NAMESPACE} -o name)
  do
    kubectl rollout status $deployment -n ${NAMESPACE}
  done
  local statefulset
  for statefulset in $(kubectl get statefulsets -n ${NAMESPACE} -o name)
  do
    kubectl rollout status $statefulset -n ${NAMESPACE}
  done
}

download-kfctl
download-kustomize

deploy-kubeflow-cli-env $TEST_CLUSTER

mkdir -p $KF_DIR
pushd $KF_DIR

kfctl build -V -f ${CONFIG_URI}

# Deploy test images
# TODO: Patch artifact fecter and visualization server deployed by metacontroller.
pushd kustomize/api-service/base
kustomize edit set image gcr.io/ml-pipeline/api-server=${GCR_IMAGE_BASE_DIR}/api-server:${GCR_IMAGE_TAG}
popd
pushd kustomize/pipelines-ui/base
kustomize edit set image gcr.io/ml-pipeline/frontend=${GCR_IMAGE_BASE_DIR}/frontend:${GCR_IMAGE_TAG}
popd
pushd kustomize/pipelines-viewer/base
kustomize edit set image gcr.io/ml-pipeline/viewer-crd-controller=${GCR_IMAGE_BASE_DIR}/viewer-crd-controller:${GCR_IMAGE_TAG}
popd
pushd kustomize/pipeline-visualization-service/base
kustomize edit set image gcr.io/ml-pipeline/visualization-server=${GCR_IMAGE_BASE_DIR}/visualization-server:${GCR_IMAGE_TAG}
popd
pushd kustomize/persistent-agent/base
kustomize edit set image gcr.io/ml-pipeline/persistenceagent=${GCR_IMAGE_BASE_DIR}/persistenceagent:${GCR_IMAGE_TAG}
popd
pushd kustomize/scheduledworkflow/base
kustomize edit set image gcr.io/ml-pipeline/scheduledworkflow=${GCR_IMAGE_BASE_DIR}/scheduledworkflow:${GCR_IMAGE_TAG}
popd

export CONFIG_FILE=${KF_DIR}/kfctl_gcp_iap.yaml
kfctl apply -V -f ${CONFIG_FILE}

gcloud container clusters get-credentials ${KF_NAME} --zone ${ZONE} --project ${PROJECT}

wait-cluster-ready
popd
