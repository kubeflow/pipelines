# Prerequisites

Following prerequisites needs to be performed for kubeflow 1.3 or lower.

### Modify KFServing torchserve image

Edit inferenceservice-config configmap

```kubectl edit cm inferenceservice-config -n kubeflow```

Update the following keys under `predictors->pytorch->v2` block

```
"image": "pytorch/torchserve-kfs",
"defaultImageVersion": "0.4.0",
"defaultGpuImageVersion": "0.4.0-gpu",
```

### Modify ml-pipeline-ui image

Edit deployment ml-pipeline-ui

```
kubectl edit deploy ml-pipeline-ui -n kubeflow
```

Edit the image field with below image:  

```
image:gcr.io/ml-pipeline-test/bab6577be93b15db1da3ba8d85d3bced08d111da/frontend@sha256:3c8ff77766c08da5a6f38445284aa585dfb74b3930717271f093a621886602d4
```

Save and exit the edit.

### Add Minio secret for KFServing 

Apply below secret and sa for KFServing to access minio server

minio-secret.yaml

```
apiVersion: v1
kind: Secret
metadata:
  name: mysecret
  annotations:
     serving.kubeflow.org/s3-endpoint: minio-service.kubeflow:9000 # replace with your s3 endpoint
     serving.kubeflow.org/s3-usehttps: "0" # by default 1, for testing with minio you need to set to 0
     serving.kubeflow.org/s3-region: "minio" # replace with the region the bucket is created in
     serving.kubeflow.org/s3-useanoncredential: "false" # omitting this is the same as false, if true will ignore credential provided and use anonymous credentials
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

```
Kubectl apply -f minio-secret.yaml -n kubeflow-user-example-com
```

### Disable sidecar injection

Run the following command to disable sidecar injection

```kubectl label namespace kubeflow-user-example-com istio-injection=disabled --overwrite```
