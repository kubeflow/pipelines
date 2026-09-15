## Namespace isolation and upgrades

This service caches legacy V1/raw-Argo steps. Standard V2 component pods bypass
this webhook cache; their cache implementation is separate.

By default, cache reuse is restricted to the pod's Kubernetes namespace. Database
namespace filtering enforces isolation; including the namespace in the cache key
provides an additional safeguard. Admission derives the key from the API server's
request namespace and the selected Argo template fields. On completion, the
watcher recomputes the key from the observed pod's namespace and template instead
of trusting its mutable cache-key annotation. This does not establish an integrity
boundary between users who can modify workloads in the same namespace.

Upgrading adds a `Namespace` column and a namespace/key index automatically at
cache-server startup. Existing rows retain an empty namespace and are not reused
by default because their ownership cannot be safely inferred. Do not backfill
them. Normal expiry still applies to these rows.

Stop all old cache-server replicas before bringing up the fixed version, and
coordinate the webhook outage with workload submission. A mixed-version rollout
does not provide the new isolation guarantee. Allow startup migration to complete
before resuming submissions. Expect a cold cache, extra step executions, and
additional resource use; pre-upgrade pods with old-format keys do not seed the new
cache. Do not roll back to the vulnerable implementation to recover cache hits.

### Optional legacy-cache fallback

Administrators can set `ALLOW_LEGACY_CACHE_FALLBACK=true` for a bounded upgrade
period. The default is `false` when unset. After a namespaced cache miss, this
option permits a lookup using the original template-only hash, restricted to
legacy rows with `Namespace = ''`. Normal namespace validation, namespace-scoped
reads and writes, cache expiry, and disabled-cache behavior remain enforced.

Enabling this option accepts possible reuse of another tenant's historical
outputs, including poisoned outputs, because legacy row ownership is unknown.
The full namespace isolation guarantee applies only with the fallback disabled.
Stopping old replicas before upgrading is still required.

For a deployment in the `kubeflow` namespace (adjust `NAMESPACE` as needed):

```sh
NAMESPACE=kubeflow
kubectl set env deployment/cache-server -n "$NAMESPACE" ALLOW_LEGACY_CACHE_FALLBACK=true
```

Legacy hits do not populate the new namespaced cache, and no automatic promotion
or backfill occurs. Turning off the option can therefore still cause cold-cache
executions. Disable it after the planned upgrade period:

```sh
kubectl set env deployment/cache-server -n "$NAMESPACE" ALLOW_LEGACY_CACHE_FALLBACK=false
```

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
