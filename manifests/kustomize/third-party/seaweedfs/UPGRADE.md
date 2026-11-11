# Upgrade SeaweedFS

Change the image tag in the Deployment to the desired version. You can find the available images [here](https://hub.docker.com/r/chrislusf/seaweedfs).

## Automatic administrator credential migration

The bundled deployment replaces the publicly known administrator credential
with a generated, persistent credential. The `seaweedfs-credential-rotation`
Deployment performs the migration automatically:

1. Preserve an existing nondefault `mlpipeline-minio-artifact` credential, or
   generate a replacement in `seaweedfs-admin-credentials`.
2. Wait for the updated SeaweedFS pod to register the replacement, then restart
   it once automatically so the separate IAM listener reloads its credential
   cache. Consumer credentials are switched only after that restart. The pod
   requires both generated Secret keys before startup. The generated credential
   bootstraps both S3 and the separate IAM listener through required environment
   references, enabling authentication before dynamic identities are registered.
   Dynamically managed tenant identities remain supported.
3. Update the legacy `mlpipeline-minio-artifact` Secret, preserving its name for
   existing workflow references. Roll the API server, UI and profile controller
   if they consume that Secret. Wait for their old pods to terminate.
4. If the published credential remains accepted, wait automatically for active
   pipeline pods and other pods referencing the shared legacy Secret in the
   installation namespace to finish. Rolled managed services do not block.
   New affected pods extend this conservative drain; there is no forced timeout.
5. Test the publicly shipped credential. Revoke it only if that exact credential
   still works, then verify rejection by both S3 and IAM. An operator's customized credential with
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
# Controller health only; this does not wait for credential revocation.
kubectl -n kubeflow rollout status deployment/seaweedfs-credential-rotation
```

`ReplacementPrepared` means the replacement is durable; logs identify storage
or consumer rollout waits. `DrainingWorkloads` means the old credential is still
accepted while workloads finish. The Secret annotations
`pipelines.kubeflow.org/storage-credential-drain-count` and
`pipelines.kubeflow.org/storage-credential-drain-sample` show the blocking count
and up to five pod names. A busy namespace can delay revocation indefinitely;
operators can pause submissions to let it become idle. Suspending a schedule
with no active pods does not itself block the drain. `DefaultRevoked` means the managed rollout completed
and the shipped credential was rejected at the last successful reconciliation.
Controller readiness is independent of migration completion: successful
reconciliation remains ready during normal rollout waits and `DrainingWorkloads`.
API failures, malformed state and reconciliation errors make it unready. A
healthy Deployment therefore does not mean the published credential has been
revoked; check the phase and drain annotations separately. A failure can leave
the phase recording an earlier successful reconciliation. Repair the reported
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

The reconciler can get/patch only the two named Secrets, the storage Deployment and three managed
consumer Deployments. Kubernetes cannot scope Secret creation by name, so its
namespace Role also permits Secret creation for first-install bootstrap. It
has no cluster role or pod-exec permission. The storage NetworkPolicy adds only
this reconciler to the IAM port's allowed callers; it does not expose filer or
master interfaces to tenants. NetworkPolicy enforcement remains required.

### Compatibility and release validation

Storage restarts once to register the new configuration and once more to reload
the separate IAM authentication cache. Managed services then roll during
migration; zero downtime is not guaranteed.
Multi-user workloads retain their per-namespace credentials. Standalone and
custom clients using the shared administrator credential can cache the old key.
The 2.18 launcher rereads credentials on its one retry of output uploads; that
retry may still fail during storage downtime. Master's launcher refreshes only
after transient transport errors, so an already-cached key rejected after
revocation does not trigger a refresh. An Argo artifact/log upload initializes
its credential once and retries with that value on both branches. Revoking a
key during such an upload can fail the task. A component still computing may
instead load the new key when its Argo upload starts.

A newer launcher image cannot change binaries already running during upgrade.
The automatic drain prevents revocation while potentially affected pods remain;
it does not prevent the earlier storage restarts from interrupting requests.
The known credential remains exploitable until the drain completes. There is
no automatic deadline that trades workload safety for immediate revocation.
Clients outside the installation namespace or without a pipeline run label or
visible Secret reference require explicit coordination.

Before releasing this change, validate actual SeaweedFS 4.50/master and
4.51/release-2.18 persistence, interruption at each migration phase, repeated
manifest apply, tenant access, bootstrap authentication and active standalone
Argo/launcher uploads. The third-party compatibility workflow applies the actual
baseline and candidate manifests, preserves a pre-upgrade tenant key and data,
and checks old-key rejection before and after a storage restart. It also probes
fresh S3/IAM authentication before dynamic registration, completes a fresh
installation, and checks that client-side and server-side manifest apply retain
runtime Secret data and identity. This does not validate GitOps prune, replace,
or application-specific field ownership. A held synthetic pipeline pod verifies
that both listeners accept the old key during draining and migration completes
after the pod terminates; it is not a full Argo/launcher upload endurance test.

Do not roll back by restoring the published default credential. Preserve the
new Secrets and use them with any rollback manifests; old startup manifests
can re-register the vulnerable credential. Credential rotation is not incident
recovery: investigate any additional identities or policies created by a
previously unauthorized administrator separately.

### Controller lifecycle

This Deployment is installed on both master and release-2.18. It remains active
after migration to provision credentials on fresh installations, preserve them
across restarts and detect restoration of the old declarative defaults. It uses
the existing profile-controller Python/botocore runtime pattern. This is a
long-lived maintained component, not a release-2.18-only upgrade Job. Removing
it later requires retaining those bootstrap and reconciliation responsibilities
elsewhere; the legacy-key migration path can be retired separately when the
supported upgrade policy permits it.
