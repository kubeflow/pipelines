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

set -e

# Google service Account (GSA)
SYSTEM_GSA=${SYSTEM_GSA:-$CLUSTER_NAME-kfp-system}
USER_GSA=${USER_GSA:-$CLUSTER_NAME-kfp-user}

# Kubernetes Service Account (KSA)
SYSTEM_KSA=(ml-pipeline-ui ml-pipeline-visualizationserver)
USER_KSA=(pipeline-runner kubeflow-pipelines-container-builder)

cat <<EOF

It is recommended to first review introduction to workload identity: https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity.

This script sets up Google service accounts and workload identity bindings for a Kubeflow Pipelines (KFP) standalone deployment.
You can also choose to manually set these up based on documentation: https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity.

Before you begin, please check the following list:
* gcloud is configured following steps: https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#before_you_begin.
* KFP is already deployed by standalone deployment: https://www.kubeflow.org/docs/pipelines/standalone-deployment-gcp/.
* kubectl talks to the cluster KFP is deployed to: https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-access-for-kubectl.

The following resources will be created to bind workload identity between GSAs and KSAs:
* Google service accounts (GSAs): $SYSTEM_GSA and $USER_GSA.
* Service account IAM policy bindings.
* Kubernetes service account annotations.

EOF

NAMESPACE=${NAMESPACE:-kubeflow}
function usage {
cat <<\EOF
Usage:
```
PROJECT_ID=<your-gcp-project-id> CLUSTER_NAME=<your-gke-cluster-name> NAMESPACE=<your-k8s-namespace> ./gcp-workload-identity-setup.sh
```

PROJECT_ID: GCP project ID your cluster belongs to.
CLUSTER_NAME: your GKE cluster's name.
NAMESPACE: Kubernetes namespace your Kubeflow Pipelines standalone deployment belongs to (default is kubeflow).
EOF
}
if [ -z "$PROJECT_ID" ]; then
  usage
  echo
  echo "Error: PROJECT_ID env variable is empty!"
  exit 1
fi
if [ -z "$CLUSTER_NAME" ]; then
  usage
  echo
  echo "Error: CLUSTER_NAME env variable is empty!"
  exit 1
fi
echo "Env variables set:"
echo "* PROJECT_ID=$PROJECT_ID"
echo "* CLUSTER_NAME=$CLUSTER_NAME"
echo "* NAMESPACE=$NAMESPACE"
echo

read -p "Continue? (Y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  exit 0
fi

echo "Creating Google service accounts..."
function create_gsa_if_not_present {
  local name=${1}
  local already_present=$(gcloud iam service-accounts list --filter='name:'$name'' --format='value(name)')
  if [ -n "$already_present" ]; then
    echo "Service account $name already exists"
  else
    gcloud iam service-accounts create $name
  fi
}
create_gsa_if_not_present $SYSTEM_GSA
create_gsa_if_not_present $USER_GSA

# You can optionally choose to add iam policy bindings to grant project permissions to these GSAs.
# You can also set these up later.
# gcloud projects add-iam-policy-binding $PROJECT_ID \
#   --member="serviceAccount:$SYSTEM_GSA@$PROJECT_ID.iam.gserviceaccount.com" \
#   --role="roles/editor"
# gcloud projects add-iam-policy-binding $PROJECT_ID \
#   --member="serviceAccount:$USER_GSA@$PROJECT_ID.iam.gserviceaccount.com" \
#   --role="roles/editor"

# Bind KSA to GSA through workload identity.
# Documentation: https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity
function bind_gsa_and_ksa {
  local gsa=${1}
  local ksa=${2}

  gcloud iam service-accounts add-iam-policy-binding $gsa@$PROJECT_ID.iam.gserviceaccount.com \
    --member="serviceAccount:$PROJECT_ID.svc.id.goog[$NAMESPACE/$ksa]" \
    --role="roles/iam.workloadIdentityUser" \
    > /dev/null # hide verbose output
  kubectl annotate serviceaccount \
    --namespace $NAMESPACE \
    --overwrite \
    $ksa \
    iam.gke.io/gcp-service-account=$gsa@$PROJECT_ID.iam.gserviceaccount.com
  echo "* Bound KSA $ksa to GSA $gsa"
}

echo "Binding each kfp system KSA to $SYSTEM_GSA"
for ksa in ${SYSTEM_KSA[@]}; do
  bind_gsa_and_ksa $SYSTEM_GSA $ksa
done

echo "Binding each kfp user KSA to $USER_GSA"
for ksa in ${USER_KSA[@]}; do
  bind_gsa_and_ksa $USER_GSA $ksa
done
