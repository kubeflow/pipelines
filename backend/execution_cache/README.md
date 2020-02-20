## Build src image
To build the Docker image of cache server, run the following Docker command from the execution_cache directory:

```
docker build -t gcr.io/ml-pipeline-test/execution_cache:latest .
```

## Deploy cache service to an existing cluster
1. Configure kubectl to talk to your newly created cluster. Refer to [Configuring cluster access for kubectl](https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-access-for-kubectl).
2. Run deploy shell script to generate certificates and create MutatingWebhookConfiguration:

```
export NAMESPACE=kubeflow
./deployer/deploy-execution-cache.sh
```

3. Go to pipelines/manifests/kustomize/metadata folder and run following scripts:

```
kubectl apply -f execution-cache-deployment.yaml --namespace $NAMESPACE
kubectl apply -f execution-cache-service.yaml --namespace $NAMESPACE
```
