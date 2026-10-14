# Artifact authorization and existing storage roots

In storage authorization mode, KFP authorizes the user for the requested namespace.
Object storage then decides
what that namespace's storage identity may read. Artifact metadata records are
provenance; registering another tenant's URI does not grant access to its bytes.

## Choose the enforcement boundary

`KFP_SECURITY_ARTIFACT_AUTHORIZATION_MODE` is configured on the authenticated
shared UI server:

| Value | Authorization for S3, MinIO and GCS artifact reads |
| --- | --- |
| `key-prefix` (default) | Retains the existing namespace-prefix and artifact-record checks. Use this with shared storage credentials. |
| `storage` | After user/namespace authorization and path validation, delegate to the namespace artifact service. Its storage credentials authorize the object. No artifact-record lookup or prescribed object-key layout is required. |

`storage` requires both `ENABLE_AUTHZ=true` and
`ARTIFACTS_SERVICE_PROXY_ENABLED=true`. Invalid modes or missing prerequisites fail
configuration validation. This is full enforcement through a different authority,
not an audit mode. The existing default is unchanged.

The setting covers artifact previews, downloads and directory operations for the
object-storage sources above. HTTP artifacts retain the existing shared-UI path,
base-URL/redirect checks and ownership policy. Archived pod logs use a separate
path and are not enabled by this setting. Volume artifacts retain their existing
namespace-proxy requirement. Do not infer coverage of one route from another.

## Storage isolation is a prerequisite

A central object store is compatible with namespace isolation. Separate SeaweedFS
instances are not required. The relevant boundary is the permission attached to
the credentials used for each request:

- Use a distinct, restricted storage identity for each namespace artifact service.
  Separate Secrets containing the same administrator key are not isolation.
- Verify that the identity can read intended historical roots and shared objects,
  and cannot read another namespace's private objects. Namespace-specific bucket
  names, custom prefixes and compound names are supported when the storage policy
  grants them. A directory name alone does not grant access.
- Keep endpoint trust and credential configuration restrictive. The mode does not
  relax endpoint allowlists, path validation or namespace user authorization.
- The shared UI must route the request to the authorized namespace's service.
  A missing or unavailable proxy must not fall back to shared UI credentials.
- Restrict direct access to artifact services. The profile controller generates a
  NetworkPolicy allowing port 3000 only from shared-UI Pods in the installation
  namespace. Set `KFP_NAMESPACE` correctly on custom controller deployments;
  stock manifests use the controller's namespace via the downward API. The CNI
  must enforce NetworkPolicy. Other policies are additive: remove or narrow rules
  that also allow cross-tenant access to these Pods. Custom deployments must provide
  equivalent ingress isolation before activation. The generated policy applies to
  namespace artifact services in both modes; clients that previously called them
  directly must use the authenticated shared UI.

Stock SeaweedFS profile provisioning grants a namespace access to its
`private-artifacts/<namespace>/` and `private/<namespace>/` roots plus the intended
shared `artifacts/` and `shared/` roots. It does not automatically grant arbitrary
historical roots. Existing credential Secrets are reused, so inspect the effective
storage permissions rather than assuming an upgraded controller changed them.
The mode does not alter storage IAM policies or migrate object bytes.

## Adopt without moving existing artifacts

1. Inventory the object stores, namespace artifact services, effective credentials
   and historical roots. Identify all serving replicas and alternate access paths.
2. Deploy the updated profile controller and reconcile affected namespaces. Verify
   the generated ingress policy and its actual enforcement; existence alone is not
   proof. Custom installations must provide equivalent protection. Confirm the
   namespace services use the intended scoped credentials and trusted destinations.
3. Test storage policy before changing the UI: namespace A must read an authorized
   historical object and fail to read namespace B's private object. Also test
   prefix/directory listing so object names cannot leak across the boundary.
4. Set `KFP_SECURITY_ARTIFACT_AUTHORIZATION_MODE=storage` on the authenticated
   shared UI, keeping its namespace proxy enabled. Roll out all UI replicas.
   During mixed configuration, replicas still using `key-prefix` may deny custom
   roots; retry after the rollout finishes. Older images may ignore the new
   setting; do not treat enforcement as complete until every serving UI replica
   is updated.
5. Test previews and raw downloads through KFP with actual user identities. Check
   an existing arbitrary root, a shared object, a denied other-tenant URI and a
   namespace parameter swap. An importer or metadata association must not turn a
   denied object into an allowed one. Stop the namespace proxy and confirm that
   retrieval fails instead of falling back to central credentials.

For example, `s3://results/customer-17/model.bin` can remain where it is. Grant the
namespace's storage identity the intended access, then use the storage mode; no
`private-artifacts/<namespace>/` rewrite or metadata-record presence is required.
Do not grant broader storage access merely to make an upgrade check pass.

To roll back the authorization mode, restore `key-prefix` and roll out the UI.
Historical custom roots may again be denied. Keep scoped storage credentials and
proxy ingress protections in place. The mode change does not delete artifacts or
change retention settings.
