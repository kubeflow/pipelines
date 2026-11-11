# Upgrade SeaweedFS

Change the image tag in the Deployment to the desired version. You can find the available images [here](https://hub.docker.com/r/chrislusf/seaweedfs).

## Automatic administrator credential migration

The bundled deployment replaces the publicly known administrator credential
with a generated, persistent credential. The `seaweedfs-credential-rotation`
Deployment performs the migration automatically:

1. Preserve an existing nondefault `mlpipeline-minio-artifact` credential, or
   generate a replacement in `seaweedfs-admin-credentials`.
2. Wait for the updated SeaweedFS pod to register the replacement. The pod
   requires both generated Secret keys before startup. An explicit empty S3
   configuration enables authentication during initial registration while
   retaining dynamically managed tenant identities.
3. Update the legacy `mlpipeline-minio-artifact` Secret, preserving its name for
   existing workflow references. Roll the API server, UI and profile controller
   if they consume that Secret. Wait for their old pods to terminate.
4. Test the publicly shipped credential. Revoke it only if that exact credential
   still works, then verify rejection. An operator's customized credential with
   the same access-key identifier is not deleted merely because of its name.

The migration preserves the PVC, objects, tenant credentials and tenant IAM
policies. It can resume after interruption and reuses the same generated key.
Only the installation namespace is reconciled; profile namespace Secrets are
not changed. An external object store is outside this bundled migration.

Inspect progress without displaying credentials:

```sh
kubectl -n kubeflow logs deployment/seaweedfs-credential-rotation
kubectl -n kubeflow get secret seaweedfs-admin-credentials \
  -o 'jsonpath={.metadata.annotations.pipelines\.kubeflow\.org/storage-credential-phase}{"\n"}'
kubectl -n kubeflow rollout status deployment/seaweedfs-credential-rotation
```

`ReplacementPrepared` means the replacement is durable; logs identify storage
or consumer rollout waits. `DefaultRevoked` means the managed rollout completed
and the shipped credential was rejected at the last successful reconciliation.
A subsequent storage/API failure makes the reconciler unready, even if the
annotation records an earlier successful reconciliation. Repair the reported
failure and let the reconciler retry. Never log or paste the Secret data.

### Secret ownership and custom deployments

Preserve **both** credential Secrets across upgrades. The legacy Secret is no
longer a declarative manifest resource. Ordinary `kubectl apply -k` leaves the
existing Secret available for adoption; on a new installation the reconciler
creates it. GitOps users must retain/orphan the existing legacy Secret during
the first migration, especially if it contains custom credentials. Do not let
pruning delete it before adoption. Exclude runtime Secret data from subsequent
declarative reconciliation. No automated controller can recover a custom key
that GitOps deletes before the controller reads it.

Do not populate the generated Secret manifest with default values. The
reconciler detects stale default restoration in the legacy Secret and rolls
managed consumers again. Conflicting custom credentials, immutable Secrets,
or unsupported credential characters stop migration rather than silently
overwriting an operator's configuration. Custom deployment names and consumers
outside the three managed deployments require explicit rollout coordination.

The reconciler can get/patch only the two named Secrets and three managed
consumer Deployments. Kubernetes cannot scope Secret creation by name, so its
namespace Role also permits Secret creation for first-install bootstrap. It
has no cluster role or pod-exec permission. The storage NetworkPolicy adds only
this reconciler to the IAM port's allowed callers; it does not expose filer or
master interfaces to tenants. NetworkPolicy enforcement remains required.

### Compatibility and release validation

Managed services roll during migration; zero downtime is not guaranteed.
Multi-user workloads retain their per-namespace credentials. Standalone and
custom clients using the shared administrator credential can cache the old key.
The 2.18 launcher rereads credentials when retrying output uploads, but an Argo
artifact/log upload can retain its driver credential through retries. Do not
assume active uploads are disruption-free merely because the Secret changed.

Before releasing this change, validate actual SeaweedFS 4.50/master and
4.51/release-2.18 persistence, interruption at each migration phase, repeated
manifest apply, tenant access, bootstrap authentication and active standalone
Argo/launcher uploads. The third-party compatibility workflow applies the actual
baseline and candidate manifests, preserves a pre-upgrade tenant key and data,
and checks old-key rejection before and after a storage restart. It does not
replace the active-upload validation gate.

Do not roll back by restoring the published default credential. Preserve the
new Secrets and use them with any rollback manifests; old startup manifests
can re-register the vulnerable credential. Credential rotation is not incident
recovery: investigate any additional identities or policies created by a
previously unauthorized administrator separately.
