## Build src image
To build the Docker image of cache server, run the following Docker command from the execution_cache directory:

```
docker build -t gcr.io/ml-pipeline-test/execution_cache:latest .
```

## Deploy cache service to an existing cluster
1. Configure kubectl to talk to your newly created cluster. Refer to [Configuring cluster access for kubectl](https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-access-for-kubectl).
2. Run following script:

```
./deployer/deploy-execution-cache.sh
```
