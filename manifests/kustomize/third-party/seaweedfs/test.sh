#!/usr/bin/env bash

set -xe

kubectl create ns kubeflow || echo "namespace kubeflow already exists"
kustomize build istio/ | kubectl apply --server-side -f -
kubectl -n kubeflow wait --for=condition=available --timeout=600s deploy/seaweedfs
kubectl -n kubeflow rollout status deployment/seaweedfs-credential-rotation --timeout=600s

echo "Port forwarding S3 endpoint on localhost:9000"
kubectl -n kubeflow port-forward svc/seaweedfs 9000:9000

function trap_handler {
 kubectl -n kubeflow logs -l app=seaweedfs --tail=100
 kustomize build istio/ | kubectl delete -f -
}

trap trap_handler EXIT
