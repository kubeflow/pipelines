#!/bin/bash
set -euo pipefail
echo "Installing Pipelines ..."
kubectl apply -f apps/pipeline/upstream/third-party/metacontroller/base/crd.yaml
echo "Waiting for crd/compositecontrollers.metacontroller.k8s.io to be available ..."
kubectl wait --for condition=established --timeout=30s crd/compositecontrollers.metacontroller.k8s.io
kustomize build experimental/seaweedfs/istio | kubectl apply -f -
sleep 60
kubectl wait --for=condition=Ready pods --all --all-namespaces --timeout=600s \
  --field-selector=status.phase!=Succeeded

kubectl wait --for=condition=Available deployment/ml-pipeline -n kubeflow --timeout=10s
kubectl wait --for=condition=Available deployment/ml-pipeline-ui -n kubeflow --timeout=10s
kubectl wait --for=condition=Available deployment/ml-pipeline-persistenceagent -n kubeflow --timeout=10s
kubectl wait --for=condition=Available deployment/ml-pipeline-scheduledworkflow -n kubeflow --timeout=10s
kubectl wait --for=condition=Available deployment/ml-pipeline-viewer-crd -n kubeflow --timeout=10s
kubectl wait --for=condition=Available deployment/cache-server -n kubeflow --timeout=10s
kubectl wait --for=condition=Available deployment/metadata-writer -n kubeflow --timeout=10s
kubectl wait --for=condition=Available deployment/seaweedfs -n kubeflow --timeout=10s
kubectl wait --for=condition=Available deployment/mysql -n kubeflow --timeout=10s
kubectl get deployment -n kubeflow -l app=ml-pipeline
