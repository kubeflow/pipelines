#!/usr/bin/env bash

set -xe

kubectl create ns kubeflow || echo "namespace kubeflow already exists"
kustomize build istio/ | kubectl apply --server-side -f -
kubectl -n kubeflow wait --for=condition=available --timeout=600s deploy/seaweedfs
kubectl -n kubeflow exec deployments/seaweedfs -c seaweedfs -- sh -c "echo \"s3.configure -user minio -access_key minio -secret_key minio123 -actions Read,Write,List -apply\" | /usr/bin/weed shell"

kubectl -n kubeflow port-forward svc/seaweedfs 9000:9000 &
echo "S3 endpoint available on localhost:9000"

function trap_handler {
 kubectl -n kubeflow logs -l app=seaweedfs --tail=100
 kustomize build istio/ | kubectl delete -f -
}

trap trap_handler EXIT
