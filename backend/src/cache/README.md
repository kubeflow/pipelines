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

### Temporary cache audit mode

Set `KFP_SECURITY_LEGACY_CACHE_MODE` on the **cache-server deployment**:

| Value | Behavior |
| --- | --- |
| Unset, empty, or `enforce` (default) | Reuse only namespace-scoped entries. Legacy entries with unknown ownership are not reused. |
| `audit` | After a genuine namespaced cache miss, permit a lookup using the original template-only hash, restricted to legacy rows with `Namespace = ''`. Log each legacy reuse as `ownership_unknown`. |

There is no `legacy` or `off` mode. Invalid values prevent startup. Audit mode
accepts possible reuse of another tenant's historical outputs, including poisoned
outputs. It does not establish that the entry belongs to the requesting namespace.
The full namespace isolation guarantee requires `enforce`.

Audit preserves namespace validation, scoped-hit precedence, namespace-scoped
writes, expiry limits, and disabled-cache behavior. A database error never enables
legacy fallback: the task executes normally without a cache hit, as before.
Known entries in another namespace are never a fallback source. This setting does
not affect native V2 caching or change authentication/authorization elsewhere.

The server warns at startup when audit is active. Each legacy reuse emits a
`security_audit` event with `control=legacy_cache`, `mode=audit`,
`reason=ownership_unknown`, `operation=cache_lookup`, disposition, namespace, pod,
and cache entry ID. Events do not include cached outputs, credentials, or
pipeline inputs. These are findings about an unknown owner, not successful
ownership checks. No background scan or automatic cache rebuild runs at startup.

For a deployment in the `kubeflow` namespace (adjust `NAMESPACE` as needed):

```sh
NAMESPACE=kubeflow
kubectl set env deployment/cache-server -n "$NAMESPACE" \
  KFP_SECURITY_LEGACY_CACHE_MODE=audit
kubectl rollout status deployment/cache-server -n "$NAMESPACE"
```

Keep all cache-server replicas configured consistently and persist the setting in
your deployment/GitOps configuration. A restart/rollout is required for environment
changes. The coordinated initial upgrade described above still applies: stop old
cache-server replicas before starting namespace-aware replicas.

Legacy hits do not populate the new namespaced cache, and no promotion or backfill
occurs. Return to enforcement after the migration period:

```sh
kubectl set env deployment/cache-server -n "$NAMESPACE" \
  KFP_SECURITY_LEGACY_CACHE_MODE=enforce
kubectl rollout status deployment/cache-server -n "$NAMESPACE"
```

Tasks incur cold-cache executions as matching pipelines next run, not as a startup
replay of historical runs. Successful executions warm the namespace-scoped cache.
Plan for extra runtime/compute and possible queueing; historical run/artifact data
is not deleted by disabling audit. We plan to remove audit mode in **3.0.0**, tracked in
[#14367](https://github.com/kubeflow/pipelines/issues/14367).

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
