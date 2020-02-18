#!/bin/bash

# This script is for deploying execution cache service to an existing cluster.
# Prerequisite: config kubectl to talk to your cluster. See ref below: 
# https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-access-for-kubectl

set -ex

echo "Start deploying execution cache to existing cluster:"
# kubectl config current-context

# Generate signed certificate for cache server.
chmod +x ./webhook-create-signed-cert.sh
./webhook-create-signed-cert.sh
echo "Signed certificate generated for cache server"

# Patch CA_BUNDLE for MutatingWebhookConfiguration
chmod +x ./webhook-patch-ca-bundle.sh
cat ./execution-cache-configmap.yaml.template | ./webhook-patch-ca-bundle.sh > ./execution-cache-configmap-ca-bundle.yaml
echo "CA_BUNDLE patched successfully"

# Create Deployment
kubectl create -f ./execution-cache-deployment.yaml
# Create Service
kubectl create -f ./execution-cache-service.yaml
# Create MutatingWebhookConfiguration
kubectl create -f ./execution-cache-configmap-ca-bundle.yaml
