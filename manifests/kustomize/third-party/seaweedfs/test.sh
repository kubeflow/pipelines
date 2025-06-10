#!/usr/bin/env bash

set -xe

kubectl create ns kubeflow || echo "namespace kubeflow already exists"
kubectl get -n kubeflow svc minio-service -o=jsonpath='{.metadata.annotations.kubectl\.kubernetes\.io/last-applied-configuration}' > svc-minio-service-backup.json
kustomize build istio/ | kubectl apply --server-side -f -
kubectl -n kubeflow wait --for=condition=available --timeout=600s deploy/seaweedfs
kubectl -n kubeflow exec deployments/seaweedfs -c seaweedfs -- sh -c "echo \"s3.configure -user minio -access_key minio -secret_key minio123 -actions Read,Write,List -apply\" | /usr/bin/weed shell"

kubectl -n kubeflow port-forward svc/minio-service 8333:9000
echo "S3 endpoint available on localhost:8333" &

function trap_handler {
 kubectl -n kubeflow logs -l app=seaweedfs --tail=100
 kustomize build istio/ | kubectl delete -f -
 kubectl apply -f svc-minio-service-backup.json
}

trap trap_handler EXIT
