# Kubeflow Pipelines Kustomize Manifests

Kubeflow Pipelines can be installed standalone and as part of the [community distribution](https://github.com/kubeflow/community-distribution).
[Installation Options for Kubeflow Pipelines](https://www.kubeflow.org/docs/components/pipelines/operator-guides/installation/).

## Multi-user profile-controller ingress

KFP's multi-user manifests include a component-scoped NetworkPolicy for the
profile controller. Its ingress rule permits pods labeled `app=metacontroller`
in the controller's namespace to reach TCP port 8080. The controller and its
normal profile reconciliation behavior are unchanged.

The CNI must enforce Kubernetes NetworkPolicy. If a custom installation changes
metacontroller's labels or namespace, adjust the policy's peer selectors to match;
a different namespace requires an explicit namespace selector alongside the pod
selector. Keep these selectors aligned when upgrading custom installations.

NetworkPolicy allowances are additive: this policy cannot narrow access granted
by another policy. Full Kubeflow Community Distribution already supplies a
default same-namespace allowance; the 2.18 dev-kind overlays additionally allow
TCP port 8080 from `0.0.0.0/0`.
Account for the complete policy set before relying on this component-local rule.
It restricts network callers, does not add webhook authentication, and does not
provide every security control of the full distribution.

The manifest test suite checks the rendered selectors, namespace placement and
port across the four multi-user entrypoints. Rendering is not enforcement
validation: on a NetworkPolicy-enforcing test cluster, use non-sensitive
connectivity checks from intended and unintended callers, inspect all applicable
policies, and verify ordinary profile provisioning/reconciliation.

## Artifact download responses

Artifact download routes return S3 and MinIO objects without extracting archive
contents and force the browser to treat every response as an attachment. Archive
filenames are preserved when available. Preview routes may still decompress an
archive and show its first entry, but they use the same download-only response
hardening; clients should consume the response body instead of relying on browser
inline rendering.

## Custom artifact-store endpoints

The UI server only accepts secret-backed S3-compatible `bucketProviders` whose
HTTP(S) origin matches the operator-configured MinIO or AWS endpoint. Add any
additional origins to `ALLOWED_ARTIFACT_ENDPOINTS` in `pipeline-install-config`.
Entries are comma-separated absolute origins, including the scheme and optional
port, for example `https://objects.example.com:9443`; paths and credentials are
not accepted. HTTP origins must be listed as HTTP and should only be used for
trusted in-cluster stores.

Upgrades from releases that allowed arbitrary provider endpoints must configure
this allowlist before users can read artifacts from a custom store. Rejected
requests return HTTP 400 and identify `ALLOWED_ARTIFACT_ENDPOINTS` as the
required operator setting. Official regional AWS S3 service endpoints are
trusted as a group only when `AWS_S3_ENDPOINT` is explicitly configured to an
official AWS S3 service endpoint; otherwise list each required origin.
Profile-created artifact proxies intentionally do not inherit another UI
server's object-store environment, so custom stores must also be listed in
`ALLOWED_ARTIFACT_ENDPOINTS` for those proxies. UI deployments that configure
`AWS_S3_ENDPOINT` directly may put the port in that value or set
`AWS_S3_PORT`; if both specify a port, they must agree.

Authenticated archived-log reads use the operator-selected archive store and
bucket after API `readLog` authorization. Workflow endpoint hints must match that
store or its exact configured Kubernetes service aliases (`.svc` and
`CLUSTER_DOMAIN`, using `MINIO_SSL` and `MINIO_PORT`). General
`ALLOWED_ARTIFACT_ENDPOINTS` entries do not authorize another log store or bucket.
Configure the selected store itself for custom storage. Native namespace/workflow
path validation also applies; see [pod-log configuration](../../frontend/README.md#pod-log-access).
Standalone artifact/log endpoint handling otherwise retains its existing behavior.

### Upgrade example: custom S3 storage

Merge the additional destinations into your installation's
`pipeline-install-config` ConfigMap before upgrading, preserving its other keys:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: pipeline-install-config
  namespace: kubeflow
data:
  ALLOWED_ARTIFACT_ENDPOINTS: "https://objects.example.com:9443,http://store.storage.svc:9000,http://store.storage.svc.cluster.local:9000"
```

List only destinations approved to receive storage credentials. A DNS alias is a
separate origin: the two `store` entries above permit both hostname spellings,
without trusting other hosts in the namespace. The provider's TLS setting must
agree with the scheme (for example, `disableSSL: 'false'` for the HTTPS origin).
Allowlisting does not grant bucket access or configure credentials.

KFP 2.18 retains the 2.17 artifact ownership policy. Complete ownership remediation
is deferred to 3.0; this upgrade does not require a new key layout, external-storage
approval policy, controller readiness record or automatic storage adoption.
Existing untracked-object prefix/strict settings remain. Missing or unavailable
ownership evidence retains the older permissive behavior, so the artifact-ownership
security issue remains unresolved for affected deployments. See the
[upgrade guide](../../docs/operator-guides/upgrade-2.18.md#artifact-compatibility-and-retained-exposure).

In multi-user installations, custom tenant endpoints and tenant Secret-backed
providers require namespace-isolated artifact proxies. Keep
`ARTIFACTS_PROXY_ENABLED` enabled in installations that use namespace proxying; this release does not introduce a new default or enable new receivers.
Adding an origin alone does not enable these providers through the shared UI.
The profile controller propagates the allowlist to profile artifact proxies;
configuring `AWS_S3_ENDPOINT` directly on the shared UI does not propagate that
trust to those proxies.

For changes to `ALLOWED_ARTIFACT_ENDPOINTS`, restart the processes that consume
that setting as an environment variable. These restarts have no required relative
order. For the standard multi-user installation
(replace `kubeflow` with your installation namespace):

```sh
kubectl -n kubeflow rollout restart deployment/ml-pipeline-ui
kubectl -n kubeflow rollout restart deployment/kubeflow-pipelines-profile-controller
kubectl -n kubeflow rollout status deployment/ml-pipeline-ui
kubectl -n kubeflow rollout status deployment/kubeflow-pipelines-profile-controller
```

After the controller rollout completes, it must reconcile the new environment
into each enabled Namespace's `ml-pipeline-ui-artifact` Deployment. Restarting its
webhook does not itself enqueue Namespace reconciliation for these environment
changes. Wait for the next hourly resync or change an annotation on each affected
Namespace. For example, use a fresh timestamp on the enabled Namespace named
`tenant` (replace it with your profile namespace):

```sh
kubectl annotate namespace tenant \
  pipelines.kubeflow.org/reconcile-at="$(date -u +%Y-%m-%dT%H:%M:%SZ)" --overwrite
```

Confirm each generated pod template contains the expected
`ALLOWED_ARTIFACT_ENDPOINTS` value before checking that Deployment's rollout status;
otherwise, the status may describe the previous completed rollout. Wait for the
updated rollout to complete before testing a custom artifact preview or download.
Standalone installations only need the UI restart. Preserve these settings in the
manifests used for future upgrades.

HTTP/HTTPS artifact reads also retain namespace receiver routing when namespace
proxying is enabled. Keep the necessary HTTP authentication and network access on
those receivers; directly served reads use the UI's configuration. Custom HTTP
credentials and Secret mounts must be preserved by your provisioning configuration
through profile reconciliation; the stock controller does not provision arbitrary
tenant HTTP credentials. Updated receivers enforce the retained destination and
redirect protections. Old receivers keep their older behavior until rollout
converges, so verify both the deployed images and actual previews/downloads.

Authenticated archived logs use the operator-selected UI storage credentials,
not a Workflow-supplied tenant Secret. Keep `ARGO_ARCHIVE_LOGS=true`, the archive
bucket, and `KFP_LOG_ARCHIVE_KEY_FORMAT` consistent with the configured store and
the actual namespace/workflow-partitioned objects. The shipped layout is supported
without moving old logs. Enabling artifact proxies does not change this shared-UI
log credential behavior; do not grant broad access to tenant Secrets as a workaround.

Validate both an artifact read and an archived-log read after migration. An
unlisted artifact endpoint returns HTTP 400. An unauthorized archived-log
location returns HTTP 403 without an unchecked storage fallback; namespace
Viewer access or endpoint allowlisting alone cannot authorize it. Verify an
archived native run after its Pod is gone, as live logs do not exercise this path.
See [pod-log configuration](../../frontend/README.md#pod-log-access) for imported
histories, custom layouts, and the coordinated UI/API rollout.
