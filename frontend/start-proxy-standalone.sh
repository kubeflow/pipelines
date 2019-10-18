#!/bin/bash

set -e

NAMESPACE=${NAMESPACE:-kubeflow}

cat << EOF
===============================================================================
This script helps set up a dev env for client side UI only. It uses a real KFP standalone deployment for api requests.

What this does:
1. It detects pipeline ui pod name in a KFP standalone deployment.
2. Port forward pipeline ui pod to localhost:3001 (Pipeline UI dev env is configured to redirect all api requests to localhost:3001)
===============================================================================
EOF

echo "Detecting pipeline ui pod name..."
PIPELINE_UI_POD=($(kubectl get pods -n $NAMESPACE -l app=ml-pipeline-ui -o=custom-columns=:.metadata.name --no-headers))
if [ -z "$PIPELINE_UI_POD" ]; then
  echo "Couldn't get pipeline ui pod in namespace '$NAMESPACE', double check the cluster your kubectl talks to and your namespace is correct."
  echo "Namespace can be configured by setting env variable NAMESPACE. e.g. '$ NAMESPACE=kfp npm run start:proxy-standalone'"
  exit 1
fi
echo "Pipeline UI pod is $PIPELINE_UI_POD"

echo "Starting to port forward frontend server in a KFP standalone deployment to respond to apis..."
kubectl port-forward -n $NAMESPACE $PIPELINE_UI_POD 3001:3000
