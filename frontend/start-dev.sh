#!/bin/bash

set -e

function clean_up() {
  kill -15 %1
  kill -15 %2
  kill -15 %3
}

echo "Preparing dev env for KFP frontend"

echo "Detecting api server pod names..."
MLMD_ENVOY_POD=$(kubectl get pods -l component=metadata-envoy -o=custom-columns=:.metadata.name --no-headers)
echo "Metadata envoy pod is $MLMD_ENVOY_POD"
BACKEND_POD=$(kubectl get pods -l app=ml-pipeline -o=custom-columns=:.metadata.name --no-headers)
echo "Ml pipeline backend api pod is $BACKEND_POD"

echo "Compiling node server..."
pushd server
npm run build
popd

echo "Starting to port forward backend apis..."
kubectl port-forward -n kubeflow $MLMD_ENVOY_POD 9090:9090 &
kubectl port-forward -n kubeflow $BACKEND_POD 3002:8888 &
ML_PIPELINE_SERVICE_PORT=3002 npm run mock:server 3001 &

echo "Starting to run webpack dev server..."
npm start
