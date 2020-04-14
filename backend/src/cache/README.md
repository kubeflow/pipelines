## Build src image
To build the Docker image of cache server, run the following Docker command from the pipelines directory:

```
docker build -t gcr.io/ml-pipeline/cache-server:latest -f backend/Dockerfile.cacheserver .
```

## Deploy cache service to an existing KFP deployment
1. Configure kubectl to talk to your newly created cluster. Refer to [Configuring cluster access for kubectl](https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-access-for-kubectl).
2. Run deploy shell script to generate certificates and create MutatingWebhookConfiguration:

```
# Assume KFP is deployed in the namespace kubeflow
export NAMESPACE=kubeflow
./deployer/deploy-cache-service.sh
```

3. Go to pipelines/manifests/kustomize/base/cache folder and run following scripts:

```
kubectl apply -f cache-deployment.yaml --namespace $NAMESPACE
kubectl apply -f cache-service.yaml --namespace $NAMESPACE
```
