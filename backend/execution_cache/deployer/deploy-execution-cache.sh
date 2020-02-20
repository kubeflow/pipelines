#!/bin/bash

# This script is for deploying execution cache service to an existing cluster.
# Prerequisite: config kubectl to talk to your cluster. See ref below: 
# https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-access-for-kubectl

set -ex

echo "Start deploying execution cache to existing cluster:"

NAMESPACE=${NAMESPACE_TO_WATCH:default}
export CA_FILE="ca.cert"
rm -f ${CA_FILE}
touch ${CA_FILE}

# Generate signed certificate for cache server.
chmod +x ./webhook-create-signed-cert.sh
./webhook-create-signed-cert.sh --namespace "${NAMESPACE}"
echo "Signed certificate generated for cache server"

# Patch CA_BUNDLE for MutatingWebhookConfiguration
chmod +x ./webhook-patch-ca-bundle.sh
NAMESPACE="$NAMESPACE" ./webhook-patch-ca-bundle.sh <./execution-cache-configmap.yaml.template >./execution-cache-configmap-ca-bundle.yaml
echo "CA_BUNDLE patched successfully"

# Create MutatingWebhookConfiguration
cat ./execution-cache-configmap-ca-bundle.yaml
kubectl apply -f ./execution-cache-configmap-ca-bundle.yaml --namespace "${NAMESPACE}"
