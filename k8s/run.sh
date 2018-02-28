#!/bin/bash

eval $(minikube docker-env)
docker build -t ml-pipeline-manager-api-server $GOPATH/src/ml/apiserver

# Deploy Minio. For more info, check
# https://github.com/minio/minio/blob/master/docs/orchestration/kubernetes-yaml/README.md#standalone-quickstart
kubectl create -f https://github.com/minio/minio/blob/master/docs/orchestration/kubernetes-yaml/minio-standalone-pvc.yaml?raw=true
kubectl create -f https://github.com/minio/minio/blob/master/docs/orchestration/kubernetes-yaml/minio-standalone-deployment.yaml?raw=true
kubectl create -f https://github.com/minio/minio/blob/master/docs/orchestration/kubernetes-yaml/minio-standalone-service.yaml?raw=true

# Deploy pipeline manager
kubectl create -f ./pipeline-manager.yaml
