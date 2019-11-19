#!/bin/bash

set -e

NAMESPACE=${NAMESPACE:-kubeflow}

function clean_up() {
  set +e

  echo "Stopping background jobs..."
  # jobs -l
  kill -15 %1
  kill -15 %2
}
trap clean_up EXIT SIGINT SIGTERM

echo "Preparing dev env for KFP frontend"

echo "Detecting api server pod names..."
METADATA_ENVOY_POD=($(kubectl get pods -n $NAMESPACE -l component=metadata-envoy -o=custom-columns=:.metadata.name --no-headers))
if [ -z "$METADATA_ENVOY_POD" ]; then
  echo "Couldn't get metadata envoy pod in namespace $NAMESPACE, double check the cluster your kubectl talks to."
  exit 1
fi
echo "Metadata envoy pod is $METADATA_ENVOY_POD"

PIPELINE_API_POD=($(kubectl get pods -n $NAMESPACE -l app=ml-pipeline -o=custom-columns=:.metadata.name --no-headers))
if [ -z "$PIPELINE_API_POD" ]; then
  echo "Couldn't get pipeline api pod in namespace $NAMESPACE, double check the cluster your kubectl talks to."
  exit 1
fi
echo "Ml pipeline PIPELINE_API api pod is $PIPELINE_API_POD"

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
kubectl port-forward -n $NAMESPACE $METADATA_ENVOY_POD 9090:9090 &
kubectl port-forward -n $NAMESPACE $PIPELINE_API_POD 3002:8888 &
ML_PIPELINE_SERVICE_PORT=3002 npm run mock:server 3001
