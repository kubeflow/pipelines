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

# This script is for deploying cache service to an existing cluster.
# Prerequisite: config kubectl to talk to your cluster. See ref below: 
# https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-access-for-kubectl

set -ex

echo "Start deploying cache service to existing cluster:"

NAMESPACE=${NAMESPACE_TO_WATCH:-kubeflow}
MUTATING_WEBHOOK_CONFIGURATION_NAME="cache-webhook"

# This should fail if there are connectivity problems
# Gotcha: Listing all objects requires list permission,
# but when listing a single oblect kubecttl will fail if it's not found
# unless --ignore-not-found is specified.
kubectl get mutatingwebhookconfigurations "${MUTATING_WEBHOOK_CONFIGURATION_NAME}" --namespace "${NAMESPACE}" --ignore-not-found >webhooks.txt

if grep "${MUTATING_WEBHOOK_CONFIGURATION_NAME}" --word-regexp <webhooks.txt; then
    echo "Webhook is already installed. Sleeping forever."
    sleep infinity
fi


export CA_FILE="ca_cert"
rm -f ${CA_FILE}
touch ${CA_FILE}

# Generate signed certificate for cache server.
./webhook-create-signed-cert.sh --namespace "${NAMESPACE}" --cert_output_path "${CA_FILE}"
echo "Signed certificate generated for cache server"

# Patch CA_BUNDLE for MutatingWebhookConfiguration
NAMESPACE="$NAMESPACE" ./webhook-patch-ca-bundle.sh --cert_input_path "${CA_FILE}" <./cache-configmap.yaml.template >./cache-configmap-ca-bundle.yaml
echo "CA_BUNDLE patched successfully"

# Create MutatingWebhookConfiguration
cat ./cache-configmap-ca-bundle.yaml
kubectl apply -f ./cache-configmap-ca-bundle.yaml --namespace "${NAMESPACE}"

# TODO: Check whether we really need to check for the existence of the webhook
# Usually the Kubernetes objects appear immediately. 
while true; do 
    # Should fail if there are connectivity problems
    kubectl get mutatingwebhookconfigurations "${MUTATING_WEBHOOK_CONFIGURATION_NAME}" --namespace "${NAMESPACE}" --ignore-not-found >webhooks.txt

    if grep "${MUTATING_WEBHOOK_CONFIGURATION_NAME}" --word-regexp <webhooks.txt; then
        echo "Webhook has been installed. Sleeping forever."
        sleep infinity
    else
        echo "Webhook is not visible yet. Waiting a bit."
        sleep 10s
    fi    
done
