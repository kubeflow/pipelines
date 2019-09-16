#!/bin/bash

set -e

NAMESPACE=${NAMESPACE:-kubeflow}

function clean_up() {
  kill -15 %1
  kill -15 %2
  kill -15 %3
}

echo "Preparing dev env for KFP frontend"

echo "Detecting api server pod names..."
METADATA_ENVOY_POD=$(kubectl get pods -n $NAMESPACE -l component=metadata-envoy -o=custom-columns=:.metadata.name --no-headers)
echo "Metadata envoy pod is $METADATA_ENVOY_POD"
BACKEND_POD=$(kubectl get pods -n $NAMESPACE -l app=ml-pipeline -o=custom-columns=:.metadata.name --no-headers)
echo "Ml pipeline backend api pod is $BACKEND_POD"

echo "Compiling node server..."
pushd server
npm run build
popd

# Frontend dev server proxies api requests to node server listening to
# localhost:3001 (configured in frontend/package.json -> proxy field).
#
# Node server proxies requests further to localhost:3002 or localhost:9090
# based on what request it is.
#
# localhost:3002 port forwards to ml_pipeline api server pod.
# localhost:9090 port forwards to metadata_envoy pod.

echo "Starting to port forward backend apis..."
kubectl port-forward -n kubeflow $METADATA_ENVOY_POD 9090:9090 &
kubectl port-forward -n kubeflow $BACKEND_POD 3002:8888 &
ML_PIPELINE_SERVICE_PORT=3002 npm run mock:server 3001 &

echo "Starting to run webpack dev server..."
npm start
