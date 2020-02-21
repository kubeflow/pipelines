#!/bin/bash

set -e

NAMESPACE=${NAMESPACE:-kubeflow}

cat << EOF
===============================================================================
This script helps set up a dev env for client side UI only. It uses a real KFP standalone deployment for api requests.

What this does:
* Port forward pipeline ui service to localhost:3001 (Pipeline UI dev env is configured to redirect all api requests to localhost:3001)
===============================================================================
EOF

echo "Starting to port forward frontend server in a KFP standalone deployment to respond to apis..."
kubectl port-forward -n $NAMESPACE svc/ml-pipeline-ui 3001:80
