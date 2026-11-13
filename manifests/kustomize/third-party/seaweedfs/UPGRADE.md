# Upgrade SeaweedFS

Change the image tag in the Deployment to the desired version. Available images
are listed [here](https://hub.docker.com/r/chrislusf/seaweedfs).

## Automatic administrator credential migration

The bundled deployment replaces the publicly known administrator credential
with an installation-specific credential. A Go binary in the KFP API image runs
in two modes; there is no permanently running credential controller:

1. The SeaweedFS `prepare-storage-credentials` init container prepares the
   persistent `seaweedfs-admin-credentials` Secret. It preserves an existing
   nondefault legacy credential or generates a replacement once.
2. If migration is incomplete, bootstrap creates a revision-named Kubernetes
   Job using the same image as its own Pod. Storage then starts with the required
   generated credentials, authenticating both S3 and IAM before dynamic
   identities are registered. Only the init container receives the bootstrap
   Kubernetes API token; the storage container does not.
3. The Job verifies registration, restarts storage once to reload its separate
   IAM cache, switches `mlpipeline-minio-artifact`, and rolls managed API, UI and
   profile-controller consumers. It waits for their old replicas to terminate.
4. While the published credential remains accepted, the Job waits for active
   pipeline pods and explicit shared-Secret consumers in the installation
   namespace to finish. New affected pods extend this conservative drain. There
   is no deadline that forces revocation; the vulnerability remains exploitable
   until draining finishes.
5. The Job revokes the exact public credential and requires both S3 and IAM to
   reject it. A customized secret sharing the old access-key identifier is not
   deleted merely because of that identifier. The Job records `DefaultRevoked`
   and exits successfully.

The PVC, objects, tenant keys and policies are preserved. Only the installation
namespace is reconciled; external storage and profile namespace Secrets are not
changed. A replacement API image must contain `/bin/credential-rotation`; do not
pin this component to an older release image lacking that binary.

## Status and controller health

Inspect migration state without displaying credential data:

```sh
kubectl -n kubeflow get jobs -l app=seaweedfs-credential-rotation
kubectl -n kubeflow logs -l app=seaweedfs-credential-rotation --all-containers=true --prefix
kubectl -n kubeflow get secret seaweedfs-admin-credentials \
  -o 'jsonpath={.metadata.annotations.pipelines\.kubeflow\.org/storage-credential-phase}{"\n"}'
```

- `ReplacementPrepared`: the replacement credential is durable.
- `DrainingWorkloads`: the known credential remains valid while workloads finish.
  Secret annotations `pipelines.kubeflow.org/storage-credential-drain-count` and
  `pipelines.kubeflow.org/storage-credential-drain-sample` expose the count and up
  to five pod names. Busy namespaces can extend the wait indefinitely. Operators
  can pause submissions to allow draining. Suspended schedules without active
  pods do not themselves block it.
- `DefaultRevoked`: both listeners rejected the credential at completion. The
  Job also reports Kubernetes `Complete` after its process exits.

While the Job runs, `/readyz` reports recent successful reconciliation. Normal
rollout waits and draining are healthy; API/configuration errors or stale
reconciliation make it unready. Pod readiness is not migration completion.
A completed Job consumes no running CPU or memory. The phase is a durable
historical result, not continuous monitoring of later storage changes.

## Restart and upgrade lifecycle

Interrupted Jobs retry using durable state and the same credential. Bootstrap
on a later storage restart reuses the credential and skips an already-completed
migration. If an incomplete Job needs a new image, bootstrap first deletes the
old Job with foreground propagation and a UID precondition; only after its pods
are removed does it create the replacement. This handles immutable Job templates
without parallel migrations. Failed Jobs can similarly be replaced on the next
storage bootstrap. No TTL repeatedly deletes and recreates completed Jobs.

Completion describes the verified storage state at that time. Restoring an old
PVC snapshot can restore old identities without changing the Kubernetes Secret.
After such a restore, explicitly request migration again before reopening access:

```sh
kubectl -n kubeflow annotate secret seaweedfs-admin-credentials \
  pipelines.kubeflow.org/storage-credential-phase=ReplacementPrepared --overwrite
kubectl -n kubeflow rollout restart deployment/seaweedfs
```

Completed Job records can be removed after retaining any needed diagnostics;
keep both credential Secrets. The short bootstrap path remains part of both
master and release-2.18 for fresh installation and credential preservation. The
one-time migration does not require a resident controller after completion.
There is no continuous drift repair: exclude runtime Secret data from GitOps.

## Secret ownership and custom deployments

Preserve both credential Secrets. Ordinary `kubectl apply -k` leaves the existing
legacy Secret available for adoption. GitOps must retain/orphan that Secret
before the first upgrade, particularly when it contains custom credentials;
prune or replace must not erase it before bootstrap reads it. A deleted custom
key cannot be recovered automatically. Never populate the generated Secret
manifest with defaults or assign a credential Secret to the Job as its owner.

The bootstrap Role can read/patch the two named Secrets, read its Pod to resolve
the effective image, and get/create/delete namespace Jobs. Kubernetes cannot
restrict creation by resource name. The migration Role can get/patch the two
Secrets, create the legacy Secret, list namespace Pods and get/patch storage and
three named consumer Deployments. Neither has cluster-wide access or pod-exec.
The existing NetworkPolicy permits IAM access only to the profile controller
and migration Job. NetworkPolicy enforcement remains required.

Custom deployment names and clients outside the inventory require explicit
coordination. Malformed, immutable or conflicting credential Secrets stop the
migration rather than silently overwriting operator configuration.

## Compatibility and release validation

Storage restarts for the new configuration and once more for the IAM cache;
managed services also roll. Draining protects active workloads from premature
key revocation, not from these earlier storage interruptions. Multi-user tenant
keys remain unchanged. The 2.18 launcher retries an output upload once with a
fresh Secret, whereas master's launcher does not refresh on authentication
failure. Argo upload retries can retain a captured credential. A newer binary
cannot change clients already running during upgrade.

The compatibility workflow exercises actual pinned images, fresh authentication
before registration, client/server-side manifest reapply, preserved and newly
provisioned tenants, a held pipeline pod that blocks revocation, and rejection
by both listeners after restart. A synthetic held pod tests revocation ordering;
it is not full Argo/launcher upload endurance coverage. Apply checks do not
establish safety for GitOps prune/replace or custom field ownership.

Do not roll back by restoring the published default credential. Preserve the
new Secrets and use them with any rollback manifests. Credential rotation is
not incident recovery: investigate additional identities or policies created
by a previously unauthorized administrator separately.
