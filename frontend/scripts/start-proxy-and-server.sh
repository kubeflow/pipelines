#!/bin/bash

set -e

NAMESPACE=${NAMESPACE:-kubeflow}

function clean_up() {
  set +e

  echo "Stopping background jobs..."
  # jobs -l
  kill -15 %1
  kill -15 %2
  kill -15 %3
}
trap clean_up EXIT SIGINT SIGTERM

echo "Preparing dev env for KFP frontend"

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
kubectl port-forward -n $NAMESPACE svc/metadata-envoy-service 9090:9090 &
kubectl port-forward -n $NAMESPACE svc/ml-pipeline 3002:8888 &
kubectl port-forward -n $NAMESPACE svc/minio-service 9000:9000 &
export MINIO_HOST=localhost
export MINIO_NAMESPACE=
if [ "$1" == "--inspect" ]; then
  ML_PIPELINE_SERVICE_PORT=3002 npm run mock:server:inspect 3001
else
  ML_PIPELINE_SERVICE_PORT=3002 npm run mock:server 3001
fi
