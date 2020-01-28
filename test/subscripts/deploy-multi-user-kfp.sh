#!/bin/bash
#
# Copyright 2019 Google LLC
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

function download-kfctl {
  if which kfctl; then
    return 0
  fi

  local tmpDir=$(mktemp)
  mkdir -p $tmpDir
  pushd $tmpDir

  gsutil cp gs://ml-pipeline/multiuser/kfctl/kfctl_78e725e_linux.tar.gz kfctl.tar.gz
  tar -xvf kfctl.tar.gz
  cp kfctl ~/bin/
  export PATH=~/bin/:$PATH
  popd
  rm -rf $tmpDir
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

  export CONFIG_URI="https://storage.googleapis.com/ml-pipeline/multiuser/kfctl_gcp_iap.0.7.0.yaml"

  # If using Cloud IAP for authentication, create environment variables
  # from the OAuth client ID and secret that you obtained earlier:
  export CLIENT_ID=$(gsutil cat gs://ml-pipeline-test-keys/oauth_client_id)
  export CLIENT_SECRET=$(gsutil cat gs://ml-pipeline-test-keys/oauth_client_secret)

  # Set KF_NAME to the name of your Kubeflow deployment. You also use this
  # value as directory name when creating your configuration directory.
  # See the detailed description in the text below this code snippet.
  # For example, your deployment name can be 'my-kubeflow' or 'kf-test'.
  export KF_NAME="$0"

  # Set the path to the base directory where you want to store one or more
  # Kubeflow deployments. For example, /opt/.
  # Then set the Kubeflow application directory for this deployment.
  export BASE_DIR="~/kf-deployment"
  export KF_DIR=${BASE_DIR}/${KF_NAME}
}

download-kfctl
deploy-kubeflow-cli-env $TEST_CLUSTER
mkdir -p $KF_DIR
pushd $KF_DIR
kfctl apply -V -f ${CONFIG_URI}
gcloud container clusters get-credentials ${KF_NAME} --zone ${ZONE} --project ${PROJECT}
# Temporarily workarounds a GKE workload identity not working issue.
gcloud container clusters upgrade ${KF_NAME} --master --cluster-version 1.14.8-gke.33

function wait-cluster-ready {
  # wait for all deployments / stateful sets to be successful
  local deployment
  for deployment in $(kubectl get deployments -n ${NAMESPACE} -o name)
  do
    kubectl rollout status $deployment -n ${NAMESPACE}
  done
  local statefulset
  for statefulset in $(kubectl get statefulesets -n ${NAMESPACE} -o name)
  do
    kubectl rollout status $statefulset -n ${NAMESPACE}
  done
}
wait-cluster-ready
popd
