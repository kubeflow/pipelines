# Artifact authorization and existing storage roots

KFP first authorizes the user for the requested namespace. For supported managed
storage, the object store then decides what that namespace's storage identity may
read. Artifact records describe provenance; registering another tenant's URI does
not grant access to its bytes.

## Choose the enforcement boundary

`KFP_SECURITY_ARTIFACT_AUTHORIZATION_MODE` is configured on the authenticated
shared UI server:

| Value | Object-storage authorization |
| --- | --- |
| `auto` (default) | Uses storage authorization for supported managed `minio://mlpipeline/...` artifacts after the controller establishes trusted namespace readiness. Missing readiness records and unsupported custom layouts retain prefix/metadata checks. Pending readiness or a failed readiness lookup denies the managed request with an actionable HTTP 503; it does not fall back to shared credentials. |
| `key-prefix` | Explicitly retains namespace-prefix and artifact-record checks. Use this when storage identities cannot provide the required isolation. |
| `storage` | Explicitly delegates S3, MinIO and GCS object reads to the authorized namespace's artifact service. Its restricted storage identity authorizes the object, without an artifact-record lookup or prescribed key layout. Use this for reviewed custom storage outside automatic discovery. |

Storage-authorized requests require authenticated namespace authorization and
namespace proxying. Explicit `storage` requires `ENABLE_AUTHZ=true` and
`ARTIFACTS_SERVICE_PROXY_ENABLED=true`; invalid configuration is rejected. Storage
authorization is enforcement, not an audit exception.

The setting covers object previews, downloads and directory operations. HTTP
artifacts retain the shared-UI path, base-URL/redirect checks and ownership policy.
Archived pod logs and volume artifacts retain their separate serving policies.
Standalone serving is unchanged.

Master can additionally resolve trusted namespace launcher-provider configuration
for explicit storage authorization. Browser-supplied provider selectors do not
grant access to another identity. Automatic discovery remains restricted to the
supported managed store; custom providers require explicit review.


## What changes automatically

The standard multi-user overlay enables managed artifact proxies. The controller
provisions or verifies each namespace's storage identity, verifies the credential's
IAM ownership and stock namespace policy, and publishes readiness evidence in the
installation namespace. Readiness is bound to the actual namespace UID and
credential/endpoint configuration; a tenant-created Secret or matching object-key
name is not sufficient authority.

The shared UI uses this readiness evidence before automatically selecting storage
authorization. The receiving service verifies the corresponding namespace and
configuration binding before using its credentials. The controller preserves
administrator-added storage grants; it does not grant arbitrary historical roots
or move objects to make a check pass. Objects already authorized by that identity
can remain at their existing roots without rewriting references or MLMD lookup.

SeaweedFS can remain central. Namespace isolation comes from per-namespace KFP
artifact services and restricted storage identities, not separate SeaweedFS
instances. Stock policies cover `private-artifacts/<namespace>/` and
`private/<namespace>/`, plus intended shared `artifacts/` and `shared/` roots.
Additional historical roots remain usable only where existing authorized grants
cover them. Distinct Secrets containing the same administrator key are not
isolation.

Managed profiles also receive a standard Kubernetes NetworkPolicy allowing proxy
port 3000 only from shared-UI Pods in the installation namespace. The policy applies
regardless of the authorization mode. The cluster must enforce NetworkPolicy;
Calico is not required. Other policies are additive and must not reopen tenant
access. Direct proxy consumers must use the authenticated shared UI.

## Upgrade managed installations

Enabling managed proxies adds one artifact receiver Pod per
`pipelines.kubeflow.org/enabled=true` namespace that did not already have one.
The default container requests **10m CPU and 70 MiB memory**, with limits of
**100m CPU and 500 MiB memory**. These are per namespace, before mesh sidecars or
custom overrides. Large fleets should count affected namespaces and check cluster
capacity, namespace quotas and temporary rolling-update surge replicas. Insufficient
capacity keeps receivers pending and can delay artifact readiness; it does not
require a particular UI/controller upgrade order.

Apply the normal multi-user deployment update. There is no manual shared-UI versus
profile-controller ordering and no storage-mode opt-in for supported managed
storage. The controller reconciles existing profiles and creates or updates their
artifact services, policies and readiness evidence. For verified managed storage
identities, it retries missing stock bucket-policy setup without replacing existing
credentials or requiring Secret deletion. It preserves administrator statements;
conflicting managed statements stay pending for operator review.

Updated UIs use a versioned receiver route that requires namespace binding. An old
receiver cannot silently accept that route without its checks. Updated receivers
retain the legacy route for old UIs during overlap: an absent binding remains
compatible there, while a supplied binding must match. The new UI never retries a
failed storage-authorized request through that legacy route or shared credentials.
This supports either component upgrade order for compatible UI/API versions;
it does not make an old UI perform new authorization or promise uninterrupted
reads during rollout. Receiver-protocol compatibility does not restore APIs
removed in 3.0: the 2.17.2 shared UI requires the V1 authorization API, which
master no longer serves. The 2.17.2 shared-UI overlap check therefore applies to
release-2.18. Master checks old-receiver rejection and automatic replacement
separately.

During proxy startup or readiness convergence, affected reads can return an
actionable HTTP 503 and need retry after reconciliation completes. Old UI replicas
may continue to deny historical roots until replaced. Persistent failures require
inspection of controller reconciliation, receiver readiness, namespace identity
and storage configuration; do not disable authorization or grant broader access.

After rollout, verify through KFP with actual user identities:

- Historical and new authorized objects, shared objects, previews, raw downloads
  and directory listings still work.
- Another tenant's private reads and listings fail, including namespace swaps,
  forged artifact associations and a Service alias targeting another receiver.
- Direct tenant access to receivers is denied through Pod and Service IPs,
  including a Pod spoofing the shared-UI label.
- A stopped disposable receiver or unavailable readiness lookup fails visibly
  without shared-credential fallback. Allow it to recover and retry the read.

Policy and readiness resources alone do not prove an installation's effective
network and storage permissions. Keep those checks in acceptance for custom mesh,
network policy and storage configurations.

## Custom storage and explicit modes

Automatic discovery is intentionally limited to the supported managed store. It
does not infer isolation for arbitrary endpoints, bucket layouts, workload
identities or custom services. Those requests retain prefix/metadata enforcement
unless the operator explicitly selects `storage` after review.

For explicit storage authorization, configure a restricted identity on each
namespace service, approve exact endpoint trust, provide the versioned receiver
protocol and equivalent ingress isolation, then test allowed historical/shared
reads and denied other-tenant reads/listings. Keep `ENABLE_AUTHZ=true` and namespace
proxying enabled on the shared UI. A missing or unavailable receiver must fail;
shared-UI credentials are not a recovery path. Custom services that cannot support
these protections must retain prefix enforcement.

Where shared credentials require the prefix policy, use `key-prefix`, configure
namespace-prefixed roots for new runs, and migrate only authorized historical
objects/references as needed. A metadata change does not move object bytes. Changing
to `key-prefix` can deny roots that worked through scoped storage identities; test
that effect before changing the mode. Authorization-mode changes do not delete
artifacts or alter retention.
