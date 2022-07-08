# Prerequisites

## Below are the prerequisites to be satisfied for running the samples.

For running the samples you will need a cluster with Kubeflow 1.4.xxx (or later)  installed,
refer https://github.com/kubeflow/manifests for details.

### Add Minio secret for KServe

Apply below secret and service account for KServe to access minio server

minio-secret.yaml

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mysecret
  annotations:
     serving.kserve.io/s3-endpoint: minio-service.kubeflow:9000 # replace with your s3 endpoint
     serving.kserve.io/s3-usehttps: "0" # by default 1, for testing with minio you need to set to 0
     serving.kserve.io/s3-region: "minio" # replace with the region the bucket is created in
     serving.kserve.io/s3-useanoncredential: "false" # omitting this is the same as false, if true will ignore credential provided and use anonymous credentials
type: Opaque
data:
  AWS_ACCESS_KEY_ID: <base-64-minio-access-key> # replace with your base64 encoded minio credential
  AWS_SECRET_ACCESS_KEY: <base-64-minio-secret-key> # replace with your base64 encoded minio credential
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sa
secrets:
  - name: mysecret
```

Run the following command to set the secrets

```Kubectl apply -f minio-secret.yaml -n kubeflow-user-example-com```

### Disable sidecar injection

Run the following command to disable sidecar injection

```kubectl label namespace kubeflow-user-example-com istio-injection=disabled --overwrite```

## Migrate to KServe 0.8.0

Refer: https://kserve.github.io/website/admin/migration/#migrating-from-kubeflow-based-kfserving

Note: Install KServe 0.8.0

### Modify KServe predictor image

Edit inferenceservice-config configmap

```kubectl edit cm inferenceservice-config -n kubeflow```

Update the following keys under `predictors -> pytorch` block

```yaml
"image": "pytorch/torchserve-kfs",
"defaultImageVersion": "0.5.1",
"defaultGpuImageVersion": "0.5.1-gpu",
```
