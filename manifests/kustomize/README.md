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
default same-namespace allowance; dev-kind's broad same-namespace allowance also
includes port 8080.
Account for the complete policy set before relying on this component-local rule.
It restricts network callers, does not add webhook authentication, and does not
provide every security control of the full distribution.

The manifest test suite checks the rendered selectors, namespace placement and
port across the four multi-user entrypoints. Rendering is not enforcement
validation: on a NetworkPolicy-enforcing test cluster, use non-sensitive
connectivity checks from intended and unintended callers, inspect all applicable
policies, and verify ordinary profile provisioning/reconciliation.

## Driver plugin ServiceAccounts

The driver executor plugin uses its own ServiceAccount for Kubernetes calls,
including reading its agent Pod and Workflow. It requests a token for the
Workflow's runtime ServiceAccount through `serviceaccounts/token` and uses that
token only for KFP API calls. The runtime ServiceAccount keeps the KFP permissions
for runs and artifacts.

Token requests are restricted by `resourceNames` to `pipeline-runner` in the
standalone plugin Role and `default-editor` in the multi-user plugin ClusterRole.
When changing `DEFAULTPIPELINERUNNERSERVICEACCOUNT` or allowing custom runtime
ServiceAccounts, add their exact names to this rule in
[`pipeline-runner-role.yaml`](base/pipeline/pipeline-runner-role.yaml) or
[`ml-pipeline-driver-agent-executor-plugin-cluster-role.yaml`](base/installs/multi-user/ml-pipeline-driver-agent-executor-plugin-cluster-role.yaml).
Keep the grant restricted to named accounts. The multi-user ClusterRole is bound
inside each profile namespace; the profile controller has permission to bind
that role without directly receiving its token-issuing permissions.

The compiler passes the run-specific KFP audience in `kfp_token_audience`; no
token is included in workflow arguments. The driver validates the request against
its agent Pod and keeps the issued token in memory, refreshing it before expiry.
RBAC restricts which ServiceAccounts the plugin can request tokens for, but not
the requested audience. Enforcing an audience restriction at the Kubernetes API
requires a separate admission policy or webhook; these manifests do not install
one.

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

Archived pod logs retrieved from workflow status also require an exact match
with a configured MinIO or AWS origin, or an entry in `ALLOWED_ARTIFACT_ENDPOINTS`.
For a Kubernetes service configured through `MINIO_HOST` and `MINIO_NAMESPACE`,
the server also trusts the exact `.svc` and `CLUSTER_DOMAIN` hostnames generated
by the Argo controller and profile repositories, using `MINIO_SSL` and
`MINIO_PORT`. This preserves stock archived logs without extra allowlist entries.
Custom profile-controller `OBJECT_STORE_HOST` and `CLUSTER_DOMAIN` settings must
match the frontend's `MINIO_HOST` and `CLUSTER_DOMAIN`, or their origins must be
listed explicitly. External hostnames do not receive generated service aliases.
The server rejects other workflow-supplied endpoints before selecting credentials
or contacting storage. If configured, the operator's archive bucket remains
available as the final log-retrieval fallback.

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
Allowlisting does not grant bucket access or configure credentials. Authenticated
multi-user artifact keys must still use `private-artifacts/<namespace>/` (or the
operator-configured namespace prefix), including through tenant proxies. Configuring
an endpoint or proxy does not make an existing custom-root object belong to a
namespace. See [artifact ownership](../../frontend/README.md#multi-user-artifact-ownership)
for migrating those object paths.

In multi-user installations, custom tenant endpoints and tenant Secret-backed
providers require namespace-isolated artifact proxies. Set
`ARTIFACTS_PROXY_ENABLED: "true"` in the installation configuration if needed.
Adding an origin alone does not enable these providers through the shared UI.
The profile controller propagates the allowlist to profile artifact proxies;
configuring `AWS_S3_ENDPOINT` directly on the shared UI does not propagate that
trust to those proxies.

After applying the configuration, restart the processes that consume these
settings as environment variables. For the standard multi-user installation
(replace `kubeflow` with your installation namespace):

```sh
kubectl -n kubeflow rollout restart deployment/ml-pipeline-ui
kubectl -n kubeflow rollout restart deployment/kubeflow-pipelines-profile-controller
kubectl -n kubeflow rollout status deployment/ml-pipeline-ui
kubectl -n kubeflow rollout status deployment/kubeflow-pipelines-profile-controller
```

After the controller rollout completes, it must reconcile the new environment
into each enabled Namespace's `ml-pipeline-ui-artifact` Deployment. The controller
watches Namespace objects; restarting its webhook does not enqueue them. Wait for
the next hourly resync or change an annotation on each affected Namespace. For
example, use a fresh timestamp on the enabled Namespace named `tenant` (replace
it with your profile namespace):

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

Archived logs have a separate credential constraint: for runs outside the UI
server's namespace, the shared UI uses its own `MINIO_ACCESS_KEY` and
`MINIO_SECRET_KEY`, rather than reading the workflow's tenant Secret. An allowed
log endpoint must accept those credentials and permit reads of the archived log
objects. Alternatively, configure the operator-owned archive fallback. Enabling
artifact proxies does not change this shared-UI pod-log credential behavior; do
not grant the shared UI broad access to tenant Secrets to work around it.

Validate both an artifact read and an archived-log read after migration. An
unlisted artifact endpoint returns HTTP 400; an unlisted workflow log endpoint
returns HTTP 500 if no configured archive fallback succeeds. Both errors identify
`ALLOWED_ARTIFACT_ENDPOINTS`. Live Kubernetes pod logs may still succeed, so verify
an archived log after the original pod is gone. Keep the exact-origin, TLS, and
namespace restrictions enabled during this verification.
