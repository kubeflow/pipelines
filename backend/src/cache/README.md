## Namespace isolation and upgrades

This service caches legacy V1/raw-Argo steps. Standard V2 component pods bypass
this webhook cache; their cache implementation is separate.

Cache reuse is restricted to the pod's Kubernetes namespace. Admission derives
the key from the API server's request namespace and the selected Argo template
fields. On completion, the watcher recomputes the key from the observed pod's
namespace and template instead of trusting its mutable cache-key annotation.
Database reads and writes are also namespace-scoped. This does not establish an
integrity boundary between users who can modify workloads in the same namespace.

Upgrading adds a `Namespace` column and a namespace/key index automatically at
cache-server startup. Existing rows retain an empty namespace and are not reused;
their ownership cannot be safely inferred. Do not backfill them or restore a
global-lookup fallback. Normal expiry still applies to these rows.

Stop all old cache-server replicas before bringing up the fixed version, and
coordinate the webhook outage with workload submission. A mixed-version rollout
does not provide the new isolation guarantee. Allow startup migration to complete
before resuming submissions. Expect a cold cache, extra step executions, and
additional resource use; pre-upgrade pods with old-format keys do not seed the new
cache. Do not roll back to the vulnerable implementation to recover cache hits.

The migration/storage regression runs with SQLite in the normal cache test suite.
To also exercise MySQL and PostgreSQL, set `KFP_CACHE_TEST_MYSQL_DSN` and
`KFP_CACHE_TEST_POSTGRES_DSN` to empty disposable test databases and run:

```sh
go test ./backend/src/cache/storage -run TestCacheNamespaceMigrationAndStorage -count=1
```

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

3. Go to pipelines/manifests/kustomize/base/cache folder and run the following scripts:

```
kubectl apply -f cache-deployment.yaml --namespace $NAMESPACE
kubectl apply -f cache-service.yaml --namespace $NAMESPACE
```
