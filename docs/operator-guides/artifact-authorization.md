# Artifact authorization and existing storage roots

## Release scope and first adoption

This guide describes the proposed master/3.0 implementation in
[#14883](https://github.com/kubeflow/pipelines/pull/14883). Its storage modes,
external approvals, readiness records and versioned receiver protocol are not part
of the 2.18 release plan. Stronger artifact ownership enforcement is deferred from
2.18; the selective compatibility rollback is tracked in
[#15044](https://github.com/kubeflow/pipelines/pull/15044), with release-specific
instructions in [the 2.18 upgrade guide PR](https://github.com/kubeflow/pipelines/pull/14703).
Until that rollback merges, the release branch still has the stronger policy.
The unreleased 2.18 ownership enforce/audit setting is being removed, not shipped
as a migration option. This deferral does not fully remediate artifact ownership
when storage credentials can read other tenants' objects.

A 2.18-to-3.0 upgrade must support first adoption: it cannot assume these approvals,
readiness records or receiver capabilities already exist, or require operators to
move historical roots during an intermediate 2.18 upgrade. The backend authority
and migration contract remain under discussion in
[#14031](https://github.com/kubeflow/pipelines/issues/14031). The controls and
limitations below describe the candidate implementation, not a finalized 3.0
migration design. This release decision does not roll back master enforcement.

KFP first authorizes the user for the requested namespace. For supported managed
storage, the object store then decides what that namespace's storage identity may
read. Artifact records describe provenance; registering another tenant's URI does
not grant access to its bytes.

## Choose the enforcement boundary

`KFP_SECURITY_ARTIFACT_AUTHORIZATION_MODE` is configured on the authenticated
shared UI server:

| Value | Object-storage authorization |
| --- | --- |
| `auto` (default) | Uses storage authorization after controller readiness for managed `minio://mlpipeline/...` artifacts or an administrator-approved external S3 credential-file broker. The legacy bundled `artifacts/` root and unapproved custom layouts retain prefix/metadata checks. Pending or revoked approval and failed readiness lookup return HTTP 503; a failed storage read never retries with shared credentials. |
| `key-prefix` | Explicitly retains namespace-prefix and artifact-record checks. Use this when storage identities cannot provide the required isolation. |
| `storage` | Explicitly delegates S3, MinIO and GCS object reads to the authorized namespace's artifact service. Its restricted storage identity authorizes the object, without an artifact-record lookup or prescribed key layout. The legacy `minio://mlpipeline/artifacts` root and descendants still require ownership/prefix checks. Use this for reviewed custom storage outside automatic discovery. |

An absent readiness record retains prefix/metadata validation and the existing
namespace artifact-service route. It does not send the object request directly to
the shared UI's storage credentials. If the namespace service is not available yet,
reads can fail until its rollout completes. A failed or invalid readiness lookup is different from
an absent record and fails closed.

Storage-authorized requests require authenticated namespace authorization and
namespace proxying. Explicit `storage` requires `ENABLE_AUTHZ=true` and
`ARTIFACTS_SERVICE_PROXY_ENABLED=true`; invalid configuration is rejected. Storage
authorization is enforcement, not an audit exception.

The setting covers object previews, downloads and directory operations. HTTP
artifacts retain the shared-UI path, base-URL/redirect checks and ownership policy.
Archived pod logs and volume artifacts retain their separate serving policies.
Standalone serving is unchanged.

Explicit storage authorization can resolve trusted namespace launcher-provider
configuration. Browser-supplied provider selectors do not grant access to another
identity. The external credential-file adapter below uses
its administrator-approved destination directly; it does not infer approval from
a tenant launcher configuration.


## What changes automatically

The standard multi-user overlay enables managed artifact proxies. For the stock SeaweedFS store, the controller
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
can remain at their existing roots without rewriting references or MLMD lookup,
except for the legacy bundled `artifacts/` root described below.

SeaweedFS can remain central. Namespace isolation comes from per-namespace KFP
artifact services and restricted storage identities, not separate SeaweedFS
instances. Stock policies cover `private-artifacts/<namespace>/` and
`private/<namespace>/`, plus shared grants on `artifacts/` and `shared/`. The
`artifacts/` grant also covers older tenant outputs and is not proof they are shared.
Additional historical roots remain usable only where existing authorized grants
cover them. Distinct Secrets containing the same administrator key are not
isolation.

Managed profiles also receive a standard Kubernetes NetworkPolicy allowing proxy
port 3000 only from shared-UI Pods in the installation namespace. The policy applies
regardless of the authorization mode. The cluster must enforce NetworkPolicy.
Other policies are additive and must not reopen tenant access. Direct proxy
consumers must use the authenticated shared UI.

### Review existing grants before upgrading

**Storage authorization makes the namespace identity's read/list permissions
available to every user authorized to view artifacts in that namespace.** A
viewer need not be able to launch a Pod, read a Secret or own the run that produced
an object. This can expand UI access without changing any storage policy.

A managed `ready` record verifies the namespace identity, required stock policy
components and receiver configuration. It does **not** certify that all effective
permissions are narrowly scoped. Administrator additions remain effective: for
example, an extra `mlpipeline/*` or another tenant's private-root grant can permit
that namespace's viewers to read those objects through the UI. Listing also
depends on effective bucket permissions. KFP does not automatically revoke grants
that existing workloads may need.

The stock `shared/` grant intentionally permits every namespace identity to read,
write and delete there. Managed `auto` exposes those shared objects to namespace
viewers without a run-ownership check. This is not an administrator-curated or
immutable area. KFP does not choose it as the default pipeline root, but a user
can select it explicitly; such outputs are shared, not tenant-private.

Before rollout:

1. Identify namespace viewers and review the storage identity's **effective**
   read/list grants, including applicable inline, attached, group and bucket
   policies. Checking only the stock KFP statement or a `ready` record is
   insufficient. Do not collect credential values in the assessment.
2. Inventory current pipeline roots and historical run references under `shared/`
   or administrator-added roots. Confirm their data may be visible to every
   namespace viewer with a storage grant, and that shared data may be modified by
   other namespaces. Move confidential data and its persisted references to
   isolated roots through a reviewed migration if necessary.
3. Narrow unintended grants before enabling storage authorization. If the workload
   identity legitimately has broader access than namespace viewers should have,
   or that review is incomplete, select
   `KFP_SECURITY_ARTIFACT_AUTHORIZATION_MODE=key-prefix` on the shared UI **before
   upgrading**. This retains ownership enforcement; it does not change direct
   workload storage permissions and can deny custom roots.
4. Verify previews, downloads and listings as a read-only namespace viewer:
   allowed own/shared/custom roots and denied private roots in other namespaces.
   Test denied access to the foreign namespace itself too. Treat an unexpected
   success as a policy finding, not proof that readiness guarantees isolation.
   A directory GET can return an empty marker on some stores; HTTP 200 alone
   does not prove listing access. Require actual returned entries, or record a
   separately authorized namespace-identity storage-listing probe independently
   from the viewer's UI reads.

### Legacy `artifacts/` contains tenant history

Before the [SeaweedFS switch (#11965)](https://github.com/kubeflow/pipelines/pull/11965),
V1 workflows wrote outputs under `artifacts/<workflow>/...`, without a namespace
component. The stock policy still grants every namespace access to `artifacts/*`;
that grant does not establish that older objects were intentionally shared.

Both `auto` and explicit `storage` retain the existing ownership/prefix checks
for `minio://mlpipeline/artifacts` and its descendants. This covers previews, raw
downloads and directory operations, including when the managed record is ready.
A namespace viewer cannot use the stock broad grant to bypass those checks by
changing modes. The guard also applies to custom MinIO endpoints using that exact
URI layout. Approved external S3 roots, GCS and other buckets retain their reviewed
storage delegation; their keys are not restricted merely because they start with
`artifacts/`. Approved external bucket names may contain dots.

Inventory old run references as well as current pipeline roots. Historical
namespace-less outputs can remain denied even to their owning users: workflow
names and object existence alone do not prove namespace ownership. Migrate only
authorized objects and their persisted references to isolated namespace roots as
needed. Changing the default root does not repair existing references. This is a
limitation of the proposed 3.0 policy, not a mandatory 2.18 migration step. There is
no artifact audit option in this candidate to recover these reads. Selecting
`storage` does not disable this legacy-root check. Review broad storage grants
separately; this guard does not make arbitrary administrator-added permissions safe.

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
If an old receiver responds with its application page, the shared UI discards that
body and sends a plain namespace-service-upgrade-required HTTP 503 with
`Retry-After: 5`. Wait for automatic rollout; if it persists, inspect the profile
controller and namespace artifact-service deployment.
This supports either component upgrade order for compatible UI/API versions;
it does not make an old UI perform new authorization or promise uninterrupted
reads during rollout. Receiver-protocol compatibility does not restore APIs
removed in 3.0: the 2.17.2 shared UI requires the V1 authorization API, which
master no longer serves. The deferred [#14884 prototype](https://github.com/kubeflow/pipelines/pull/14884)
tested that old UI against a new receiver on a 2.18 base. That is historical
prototype evidence, not a shipping 2.18 feature or proof of a complete 2.18-to-3.0
upgrade. Master checks old-receiver rejection and automatic replacement separately;
first adoption and full version-to-version compatibility still need qualification.

During proxy startup or readiness convergence, affected reads can return an
actionable HTTP 503 and need retry after reconciliation completes. Old UI replicas
may continue to deny historical roots until replaced. Persistent failures require
inspection of controller reconciliation, receiver readiness, namespace identity
and storage configuration; do not disable authorization or grant broader access.
A retry alone cannot fix missing Secret-read permissions or malformed storage
configuration. An empty, missing-key or malformed authorization payload and
Secret-read permission failures return HTTP 503 with corrective guidance and no
`Retry-After`; pending readiness and transient API failures remain retryable.
Correct the controller's record/configuration or restore the intended scoped
access.

A managed-store IAM verification error can move a previously ready namespace back
to pending. The controller requests another reconciliation after 10 seconds and
restores readiness once identity, policy and receiver verification succeed, without
replacing existing credentials. This interval is a retry request, not an outage
bound: API latency, controller queues and receiver rollout can delay recovery.
Previously verified readiness is not retained while current authorization is
unknown, and pending does not automatically switch to prefix authorization.

Malformed namespace launcher configuration retains the existing
prefix-authorized proxy path; the receiver may still reject it with a configuration
error. It is not evidence that background reconciliation will make the read work.

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

## External storage with rotating broker credentials

`auto` also supports an administrator-approved external S3-compatible store whose
namespace broker writes temporary credentials to a named shared-credentials file
profile. Existing roots such as `s3://external-artifacts/model/team-a-training/`
can remain unchanged when that identity already has permission to read them.
This proposed master/3.0 path does not require MLMD or an audit exception. It is
not included in the 2.18 release plan.

This is a supported credential-file adapter, not automatic discovery of every
IRSA, STS or broker integration. The administrator must approve the broker's
namespace-to-principal mapping and effective storage policy. A principal string,
ServiceAccount annotation, successful identity lookup or artifact record alone
does not prove isolation. The controller verifies the approved receiver
configuration and readiness; it does not independently prove an external IAM
policy or grant access to historical objects.

Approved external endpoint URLs accept IPv4, DNS names and bracketed IPv6
literals, for example `http://[fd00::10]:8333`. The destination approval and binding
retain that canonical URL. If generated receiver configuration is malformed,
startup names `KFP_ARTIFACT_EXTERNAL_STORAGE_CONFIG` without printing its contents.
Correct the central approval/template or generated receiver configuration; retrying
an unchanged invalid configuration will not repair it.

### Automatically enroll namespaces from a trusted provisioner

For a fleet using the same storage integration, the installation administrator can
configure one adoption policy instead of maintaining an explicit approval for each
namespace. GitOps, a controller or an adapter for an external identity service
publishes namespace-to-principal bindings. KFP combines those bindings with the
policy's exact storage destination and an approved receiver template, then uses the
same receiver reconciliation and readiness checks as an explicit approval.

Enrollment configures artifact reads and receiver rollout. The trusted provisioner
still supplies the external identity, broker, ServiceAccount and any pipeline
launcher/root configuration needed for workloads to write artifacts. This policy
does not provision storage-write permissions or change pipeline roots.

The policy lives in `adoption-policies.json` in `kfp-artifact-storage-config`.
Referenced binding and template ConfigMaps must live in the KFP installation
namespace and have the label
`pipelines.kubeflow.org/artifact-storage-policy-source: "true"`. This label enables
observation; it does not grant trust by itself. The policy must name the exact
ConfigMap and key. References outside the installation namespace are rejected,
and the policy cannot enroll the installation namespace itself. A provisioner running
elsewhere can publish its integration records into the installation namespace.
No new Kubernetes client permissions or Secret reads are required by the profile
controller.

Protect the policy, binding source and receiver template from tenant writes.
For provisioning automation, grant update/patch only on its pre-created, named
binding ConfigMap rather than permission to edit the policy or templates.
The administrator is trusting that provisioner's namespace-to-principal mapping
and effective storage grants. Neither a matching namespace label nor a successful
artifact read proves those grants are appropriately restricted.

The following integration example keeps namespace bindings separate from the
reusable broker template. Replace the namespace UID, destination, principal and
broker configuration with values maintained by the trusted provisioning system.
The broker must already authenticate its own namespace and produce the named
credential file described below; this example does not install an identity
provider or grant storage permissions.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kfp-artifact-storage-config
  namespace: kubeflow
data:
  adoption-policies.json: |
    {
      "version": 1,
      "policies": [{
        "name": "external-artifact-storage",
        "namespaceSelector": {
          "matchLabels": {"storage.example.com/integration": "artifact-broker"}
        },
        "bindingsSource": {
          "namespace": "kubeflow",
          "configMap": "namespace-storage-identities",
          "key": "bindings.json"
        },
        "allowedStorage": {
          "source": "s3",
          "endpoint": "https://objects.example.com",
          "bucket": "external-artifacts",
          "region": "us-east-1"
        },
        "receiverTemplateRef": {
          "namespace": "kubeflow",
          "configMap": "approved-artifact-receiver",
          "key": "template.json"
        },
        "adoptExistingNamespaces": true,
        "enrollNewNamespaces": true
      }]
    }
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: namespace-storage-identities
  namespace: kubeflow
  labels:
    pipelines.kubeflow.org/artifact-storage-policy-source: "true"
data:
  bindings.json: |
    {
      "team-a": {
        "namespaceUID": "REPLACE-WITH-CURRENT-NAMESPACE-UID",
        "principal": "broker:team-a-artifact-reader"
      }
    }
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: approved-artifact-receiver
  namespace: kubeflow
  labels:
    pipelines.kubeflow.org/artifact-storage-policy-source: "true"
data:
  template.json: |
    {
      "identity": {
        "kind": "shared-credentials-file",
        "credentialFile": "/var/run/artifact-broker/credentials",
        "profile": "artifact-reader"
      },
      "podTemplate": {
        "spec": {
          "serviceAccountName": "artifact-reader",
          "containers": [{
            "name": "ml-pipeline-ui-artifact",
            "volumeMounts": [{
              "name": "artifact-credentials",
              "mountPath": "/var/run/artifact-broker",
              "readOnly": true
            }]
          }, {
            "name": "credential-broker",
            "image": "registry.example.com/your-approved-broker:replace-me",
            "volumeMounts": [{
              "name": "artifact-credentials",
              "mountPath": "/var/run/artifact-broker"
            }]
          }],
          "volumes": [{"name": "artifact-credentials", "emptyDir": {}}]
        }
      }
    }
```

Preserve unrelated data and policies when updating the central ConfigMap. The
binding inventory shown for `team-a` is output from trusted provisioning
configuration, not a new per-namespace approval an operator must hand-write.
KFP reads the actual Namespace UID and requires the published UID to match.
Recreating a namespace with the same name requires the provisioner to publish a
new binding. KFP does not infer a principal from a tenant annotation, ServiceAccount
name, object path or credential value. The template is reused as supplied; KFP
does not evaluate substitution expressions. Brokers can use standard Kubernetes
Downward API fields where their approved integration requires namespace identity.

Both enrollment flags default to `true`. If they differ, specify
`enrollmentCutoff` as an explicit UTC timestamp such as `2026-10-01T00:00:00Z`.
A namespace created at or before that timestamp is existing; one created later is
new. This classification does not change when the controller restarts or the
ConfigMap is recreated. Changing the cutoff intentionally changes eligibility.
Setting both flags to `false` disables enrollment under that policy. A matching
namespace excluded by the flags or cutoff remains pending, including one that was
previously approved; its `auto` reads return HTTP 503 after reconciliation.
Only `matchLabels` selectors are supported. A valid selector that does not match
the namespace excludes that policy from its approval evaluation; malformed
matching policy fields and records are rejected rather than silently ignored.
Likewise, an invalid explicit approval value for another namespace does not affect
this namespace. Missing or invalid protected policy sources affect only namespaces
selected by policies that reference them.

Invalid shared JSON (including duplicate keys at any nesting), document structure,
version or selectors can prevent the controller
from determining which namespaces are targeted. These errors temporarily put all
enabled managed namespaces into pending, including namespaces with no intended
external approval. Correct the shared configuration promptly. Records created by
this controller distinguish a configuration error before any external approval
from an adopted external identity: after correction or ConfigMap removal, a
never-approved namespace automatically returns to managed reconciliation without
Secret deletion. Once a valid external approval has been resolved, withdrawing it
keeps that namespace pending, even if its receiver was never ready. Restore a
reviewed valid approval to recover that external path.

Kubernetes does not validate the embedded policy JSON when saving a ConfigMap;
`kubectl --dry-run=server` checks its Kubernetes envelope, not this contract. There
is no artifact-policy admission validator. Review shared JSON and selector changes
before publishing them, and verify controller reconciliation afterward. A generic
JSON syntax check alone does not reject duplicate keys or prove valid policy or
storage isolation. Master continues reconciling the driver-agent CA Secret
while authorization is pending and preserves an already-populated ServiceAccount
token Secret; a shared configuration fault does not freeze certificate rotation.

Version-2 pending records from earlier unreleased builds lack this distinction.
They remain conservatively blocked when approval is absent; inspect their
approval and receiver history before deliberate repair. Do not delete readiness
records in bulk to force stock-credential fallback.

A matching policy needs a valid binding and template before it can authorize
storage. Missing sources, missing discovery labels, stale namespace UIDs and
invalid configuration keep the namespace pending. Multiple matching
policies, or an explicit `approvals.json` entry overlapping a matching policy,
are ambiguous and also remain pending; there is no first-match override. To move
an explicit approval to a policy, remove that namespace's explicit entry and add
the policy in the same central ConfigMap update, retaining other entries.

Policy and source changes reconcile automatically. Removing a binding, policy,
template or policy-selection label from an already approved namespace leaves a
pending record and blocks `auto` reads with retryable HTTP 503. Restoring valid
configuration recovers without broader credential fallback. As with explicit
approval, this takes effect after reconciliation and does not revoke credentials
at the external store. Keep `pipelines.kubeflow.org/enabled=true` while revoking
through the policy; removing KFP's controller-enrollment label stops its ordinary
reconciliation and is not a storage-access revocation mechanism.

Verify enrollment of an existing namespace and a newly provisioned namespace,
source/template removal and recovery, namespace UID mismatch, and denied tenant
attempts to change the central configuration. Also run the historical-access,
credential-renewal and cross-tenant checks below. This mechanism generalizes how
approvals are supplied; the current external credential adapter still requires a
broker-written named shared-credentials file.

### Explicitly approve an individual namespace

For exceptions or installations without a trusted provisioning source, the
installation administrator can maintain explicit approvals in ConfigMap
`kfp-artifact-storage-config` in the KFP installation namespace, normally
`kubeflow`. Its `approvals.json` maps
each namespace to its current namespace UID, one exact S3 bucket and endpoint,
stable broker principal, credential file/profile and approved receiver
`podTemplate`. Tenant-owned ConfigMaps, annotations and Secrets cannot establish
this approval. Restrict write access to the installation configuration accordingly.

The following integration template uses an already provisioned ServiceAccount
`artifact-reader`. Replace the namespace UID, endpoint, principal, broker image
and broker-specific configuration before applying it. Obtain the current UID with
`kubectl get namespace team-a -o jsonpath='{.metadata.uid}'`. The broker must
atomically replace `/var/run/artifact-broker/credentials` with the profile shown
below, and its own authentication and policy must restrict issued credentials to
team A. KFP does not install or configure that external broker for you.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kfp-artifact-storage-config
  namespace: kubeflow
data:
  approvals.json: |
    {
      "team-a": {
        "namespaceUID": "REPLACE-WITH-CURRENT-NAMESPACE-UID",
        "storage": {
          "source": "s3",
          "bucket": "external-artifacts",
          "endpoint": "https://objects.example.com",
          "region": "us-east-1"
        },
        "identity": {
          "kind": "shared-credentials-file",
          "principal": "broker:team-a-artifact-reader",
          "credentialFile": "/var/run/artifact-broker/credentials",
          "profile": "artifact-reader"
        },
        "podTemplate": {
          "spec": {
            "serviceAccountName": "artifact-reader",
            "containers": [
              {
                "name": "ml-pipeline-ui-artifact",
                "resources": {
                  "requests": {"cpu": "10m", "memory": "70Mi"},
                  "limits": {"cpu": "100m", "memory": "500Mi"}
                },
                "volumeMounts": [{
                  "name": "artifact-credentials",
                  "mountPath": "/var/run/artifact-broker",
                  "readOnly": true
                }]
              },
              {
                "name": "credential-broker",
                "image": "registry.example.com/your-approved-broker:replace-me",
                "volumeMounts": [{
                  "name": "artifact-credentials",
                  "mountPath": "/var/run/artifact-broker"
                }]
              }
            ],
            "volumes": [{"name": "artifact-credentials", "emptyDir": {}}]
          }
        }
      }
    }
```

The receiver container name must be `ml-pipeline-ui-artifact`. KFP supplies its
frontend image, protected serving configuration and readiness probe, and manages
the receiver Deployment, Service and ingress NetworkPolicy. Preserve approved
broker configuration, required volumes and ServiceAccount wiring in `podTemplate`;
include resources for the broker and any other sidecars. A receiver owned by
another controller is not automatically taken over. Resolve that ownership with
the deployment owner before using this managed path.

The broker-produced file contains the selected profile and temporary credentials:

```ini
[artifact-reader]
aws_access_key_id=REDACTED
aws_secret_access_key=REDACTED
aws_session_token=REDACTED
expiration=2030-01-01T00:00:00Z
```

Access key, secret key and a session token are required. The session token may use
`aws_session_token` or its legacy alias `aws_security_token`, but not both; duplicate
fields are rejected even if their values match. Optional `region` metadata is
ignored: the approved region remains authoritative. `expiration` is optional and,
when present, must be valid and in the future. The file must be regular and at most
64 KiB. `role_arn`, `source_profile`, `credential_process` and other default-chain
instructions are not accepted in the selected profile. KFP reads that profile
again for each artifact request. It never substitutes environment, instance-role,
shared-UI or another profile's credentials if the approved file is unavailable.
Use an atomic file replacement so readers do not observe a partially written
renewal. Have the broker refresh before expiry with enough time for an in-flight
storage request; KFP rejects credentials at their declared expiry, without an
additional early-expiry margin. Never place credential values in the approval
ConfigMap or reports.

### Preserve history before changing the storage destination

**Approving external storage replaces this namespace's bundled-store receiver.**
Earlier runs whose artifacts still use
`minio://mlpipeline/private-artifacts/<namespace>/...` will no longer open through
that receiver after cutover, even if their objects and metadata remain intact.
The approval covers only its declared S3 bucket and endpoint. Prefix mode or a
broader grant on that external identity does not restore the old MinIO route.
Existing roots are preserved only within the approved store.

Before adopting a namespace that has bundled-store history:

1. Inventory artifact URIs in existing runs, including previews and directory
   listings. Identify references outside the proposed external source and bucket.
2. Keep the managed receiver until the migration is ready. Copy only authorized
   historical objects into the approved external store and update their persisted
   artifact references through a reviewed migration process. KFP does not perform
   either step; changing pipeline-root defaults affects new artifacts only.
3. Verify representative historical run views with the owning users, then apply
   the external approval and verify both authorized access and tenant isolation.
   Retain old copies for the planned rollback. If historical references cannot be
   migrated safely, postpone external adoption for that namespace.

This adapter serves one approved store per namespace; it does not provide
simultaneous old-MinIO and new-S3 access or an automatic reverse migration.

### Rollout, renewal and revocation

Keep the shared UI in `auto` with authenticated namespace proxying enabled. The
controller watches the installation approval and reconciles the receiver without
a separate namespace annotation or prescribed shared-UI/controller order. It
publishes version 2 readiness in `kfp-artifact-storage-<namespace>` only after the
approved receiver, Service and NetworkPolicy converge. It derives
`configurationRevision` from the approved Pod template and binds it together with
the namespace UID, destination and credential-source configuration. Operators do
not calculate or publish that revision themselves.

Ordinary credential renewal leaves that binding and Pod template unchanged, so it
does not restart the receiver. Changing the approved principal, source, endpoint
or Pod template requires matching new readiness; namespace recreation requires
approval of the new UID. Missing or invalid approvals for a previously approved
namespace retain a pending record. They do not silently restore stock credentials
or prefix fallback. In `auto`, pending readiness and transient lookup failures
return retryable HTTP 503; invalid records and permission errors require correction instead.
Removing approval is not an immediate revocation of external storage credentials. Revoke those at the broker/store when access itself must end.
Explicit `storage` and older UIs do not use automatic readiness as their policy.

The approved source is `s3` and the approval covers one bucket and exact HTTP(S)
origin, including its port. Other sources/buckets are denied for that approved
receiver. Browser `providerInfo` cannot redirect it or select a Secret. Native
artifact URI query options are rejected rather than silently applied to a
different destination. Legacy receiver routes used by older UIs also use the
approved destination and file; they cannot fall through to ambient credentials.
The shared UI still performs its own namespace authorization. NetworkPolicy must
prevent direct tenant access to the receiver, including the legacy route.

Before relying on this path, verify actual historical previews/downloads and
directory access, repeated broker renewals without a receiver restart, expired
or revoked credentials, and denied reads/listings of another tenant's objects.
Verify a missing/partial credential file fails even when broad ambient credentials
are available. After restoring the broker, reads should recover without a Pod
restart. Test the real ingress and any custom network policies as well; controller
readiness is not a substitute for those checks.

## Custom storage and explicit modes

Outside the managed store and approved credential-file adapter, `auto` does not
infer isolation for arbitrary endpoints, workload identities or custom services.
Those requests retain prefix/metadata enforcement unless the operator explicitly
selects `storage` after review. An existing namespace storage grant may already be
correct even when automatic recognition is unavailable; distinguish that from a
missing permission before proposing object migration. Review whether a supported
storage mode can use the existing isolated identity; this candidate has no artifact
audit fallback.

### Stores without the managed IAM API

S3-compatible object access does not imply support for the controller's IAM
verification API. Custom stores such as MinIO may have pre-provisioned namespace
credential Secrets but no compatible `ListAccessKeys`/`GetUserPolicy` operations.
The default `auto` path now verifies existing credentials too. Unsupported IAM
operations, transport failures and permission errors keep the record pending, so
`minio://mlpipeline/...` reads return HTTP 503. Successful object reads alone do
not certify isolation, and retries cannot add missing API support.

Before upgrading such an installation, choose
`KFP_SECURITY_ARTIFACT_AUTHORIZATION_MODE=key-prefix` on the authenticated shared
UI to retain the ownership/prefix path, and verify the namespace receiver's
existing endpoint and credentials still work. This avoids managed readiness
selection, not authorization; arbitrary historical roots may still require
migration. Alternatively, use reviewed explicit `storage` only with genuinely
restricted namespace identities and the required receiver protocol and ingress
protection, or enroll a supported approved external broker. Keep existing Secrets
and the custom provisioner for new namespaces: neither mode adds IAM support or
makes the controller create users/keys on an unsupported store. Legacy MinIO installations can have the
same broad key copied into different Secrets, so distinct Secret names are not
proof of tenant isolation.

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
