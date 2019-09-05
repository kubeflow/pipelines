# Overview

This directory contains the stacks to deploy on Google Cloud Marketplace.

## About Google Click to Deploy

Popular open stacks on Kubernetes packaged by Google.

# Installation

Build docker image
```
docker build --tag gcr.io/ml-pipeline/google/kfp/deployer:0.1.27 -f deployer/Dockerfile .
docker push gcr.io/ml-pipeline/google/kfp/deployer:0.1.27
```

Install application CRD in a cluster
```
kubectl apply -f "https://raw.githubusercontent.com/GoogleCloudPlatform/marketplace-k8s-app-tools/master/crd/app-crd.yaml"
```

Install mpdev
```
BIN_FILE="$HOME/bin/mpdev"
docker run gcr.io/cloud-marketplace-staging/marketplace-k8s-app-tools/k8s/dev:unreleased-pr396 cat /scripts/dev > "$BIN_FILE"
chmod +x "$BIN_FILE"
export MARKETPLACE_TOOLS_TAG=unreleased-pr396
export MARKETPLACE_TOOLS_IMAGE=gcr.io/cloud-marketplace-staging/marketplace-k8s-app-tools/k8s/dev
```

Install Kubeflow Pipelines
```
kubectl create ns test
mpdev /scripts/install --deployer=gcr.io/ml-pipeline/google/kfp/deployer:0.1.27 --parameters='{"name": "installation-1", "namespace": "test"}'
```

Install with Cloud SQL and GCS
```
mpdev /scripts/install \
  --deployer=gcr.io/ml-pipeline/google/kfp/deployer:0.1.27 \
  --parameters='{"name": "installation-1", "namespace": "test" , "managedstorage.enabled": true, "managedstorage.cloudsqlInstanceConnectionName": "your-instance-name", "managedstorage.dbPassword": "your password"}'
```

To uninstall 
```
kubectl delete applications -n test installation-1
```
