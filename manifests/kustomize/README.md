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
including reading its agent Pod, Workflow, and WorkflowTaskSet. It requests a token for the
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

Argo 4.1.2 also needs a runtime token Secret for each custom ServiceAccount.
The agent disables ordinary token automounting and falls back to the exact name
`<service-account>.service-account-token`. The stock manifests provide this for
`pipeline-runner` and profile-created `default-editor` accounts only. In the
workflow's namespace, create a Secret annotated for the custom account, for
example:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: custom-runner.service-account-token
  namespace: your-pipeline-namespace
  annotations:
    kubernetes.io/service-account.name: custom-runner
type: kubernetes.io/service-account-token
```

Create `ServiceAccount/custom-runner` first, retain its normal workflow/agent
RBAC bindings, and wait for Kubernetes to populate the Secret's token before
starting runs. The token-creation allowlist alone is not sufficient. This Secret
is a long-lived Kubernetes credential, not the short-lived run-scoped KFP token;
restrict access and remove it when retiring the account. Do not put a token value
in source control. Do not confuse it with the plugin's separate ServiceAccount
and token Secret.

The compiler passes the run-specific KFP audience in `kfp_token_audience`; no
token is included in workflow arguments. The driver validates the request against
its agent Pod and keeps the issued token in memory, refreshing it before expiry.
RBAC restricts which ServiceAccounts the plugin can request tokens for, but not
the requested audience. Enforcing an audience restriction at the Kubernetes API
requires a separate admission policy or webhook; these manifests do not install
one.

## Driver execution and transport retries

The driver returns Argo `Running`/`requeue` while execution continues, rather than
holding an RPC open beyond Argo's 30-second timeout. Every poll authenticates the
agent Pod and resolves exactly one active task in its WorkflowTaskSet, verifying
the Workflow owner UID. Missing, ambiguous, or temporarily inconsistent taskset
state returns HTTP 503 for retry, never starts guessed work. Plugin RBAC therefore
requires namespace-scoped `workflowtasksets/get` in addition to `workflows/get`.
The default standalone and profile-bound plugin roles include these permissions.

Concurrent polls and lost HTTP responses share one execution and replay its
terminal outputs for that taskset node. A deliberate Argo retry has a new node ID
and executes separately; inherited workflow retry policies are unchanged. Results
remain in memory for the agent process lifetime (memory grows with executed
driver tasks). This is **not durable exactly-once execution**: restarting the
agent/plugin loses the cache and can repeat task or external-plugin side effects.
Driver work is canceled when the service shuts down and has a 30-minute context
deadline; individual request cancellation does not cancel it. Calls must honor
context cancellation. Workflow termination relies on Argo terminating the agent,
not on an additional Workflow watcher in the plugin.

## TLS agent trust projection

The cert-manager TLS overlay uses an explicit controller `podSpecPatch` to
project **only `ca.crt`** from the reserved Secret into workflow agents. The
serving Secret also contains `tls.key`; neither the driver nor Argo agent
containers receive that key through the filtered volume. The API server keeps
its separate serving-key mount. Keep the trust patch when customizing the TLS
overlay or its workflow defaults.

This key projection avoids a second Secret-copy controller and follows ordinary
Kubernetes Secret-volume updates on certificate rotation. CA mounts are directory
mounts, not `subPath` mounts; new driver invocations build their TLS client from
the current CA file. Verify the rendered agent volume's `items` allowlist before
rollout and after custom patches. This is mount minimization, not protection
against an identity separately authorized to read the whole Secret.

## Driver custom CA migration

`CABUNDLE_SECRET_NAME` and `CABUNDLE_CONFIGMAP_NAME` still configure launcher
trust, but no longer mount certificates into the driver. Before upgrading an
existing custom-CA TLS installation, configure agent trust separately: Argo only
mounts `Secret/argo-workflows-agent-ca-certificates` from the **workflow's
namespace**. Copy the approved CA bundle into its `ca.crt` key in every workflow
namespace, including profile namespaces, and keep it synchronized on rotation.
Do not copy the API server's private key. The existing TLS driver sidecar patch
mounts that Secret at `/kfp/certs` and sets `CA_CERT_PATH=/kfp/certs/ca.crt`.

For example, create `env/custom-ca/company-ca.crt` containing your public CA
bundle and `env/custom-ca/kustomization.yaml` under this directory:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: kubeflow
resources:
  - ../platform-agnostic
patches:
  - path: ../cert-manager/platform-agnostic-standalone-tls/patches/ml-pipeline-driver-plugin-cm.yaml
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: ml-pipeline
      spec:
        template:
          spec:
            containers:
              - name: ml-pipeline-api-server
                env:
                  - name: CABUNDLE_CONFIGMAP_NAME
                    value: company-ca
configMapGenerator:
  - name: company-ca
    files:
      - ca.crt=company-ca.crt
secretGenerator:
  - name: argo-workflows-agent-ca-certificates
    files:
      - ca.crt=company-ca.crt
generatorOptions:
  disableNameSuffixHash: true
```

Render with `kustomize build --load-restrictor=LoadRestrictionsNone env/custom-ca`.
This example configures **client trust only**, not an API server TLS listener or
serving certificate. Replace `../platform-agnostic` with your existing TLS-enabled
installation resources and preserve the serving-certificate and other client TLS
configuration. If you already generate `company-ca`, reuse that resource rather
than defining it twice. No cert-manager installation is needed to reuse the driver
patch; do not include the entire cert-manager overlay. The embedded
`sidecar.container` is a string: Kustomize cannot deep-patch its env or mounts.
Use the complete TLS sidecar patch and preserve your image override if applicable.
Create the reserved Secret separately in every other workflow namespace before
starting runs. Verify a driver KFP API call and a launcher task after migration;
manifest rendering alone does not verify TLS trust or live Argo mounts.

## Driver log artifact credentials

Driver log upload reads credentials from the configured object-store provider,
not `LOG_ACCESS_KEY`/`LOG_SECRET_KEY` environment variables. External S3/GCS
installations do not need the stock MinIO Secret merely to start the plugin.
The default driver roles allow only the stock artifact and MLflow Secret names.
When a provider specifies a custom credential Secret, grant the **plugin** account
`get` for that exact Secret in the workflow namespace; permissions on the runtime
ServiceAccount alone do not cover driver log upload. For example, apply this
alongside `Secret/artifact-store-credentials` in the workflow namespace (replace
`your-pipeline-namespace` and the credential name with your configured values):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: driver-custom-artifact-credentials
  namespace: your-pipeline-namespace
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    resourceNames: ["artifact-store-credentials"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: driver-custom-artifact-credentials
  namespace: your-pipeline-namespace
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: driver-custom-artifact-credentials
subjects:
  - kind: ServiceAccount
    name: ml-pipeline-driver-agent-executor-plugin
    namespace: your-pipeline-namespace
```

Repeat the namespace-local grant for each enabled profile that uses custom
credentials. Do not grant wildcard Secret access or bind this Role cluster-wide.
After applying, verify the plugin identity can get the configured Secret and
cannot get an unrelated Secret, then run a pipeline and check its System Logs
artifact. The manifest tests check the exact-name authorization contract, not
live Kubernetes authorization or object-store access.

## Driver Pod metadata migration

The executor-plugin driver no longer creates per-task driver Pods.
`DRIVER_POD_LABELS` and `DRIVER_POD_ANNOTATIONS` are therefore deprecated and
nonempty values cause API server startup or configuration reload to fail. Remove
these keys from `config.json`, the API server environment, and the ConfigMaps
that feed it before upgrading. Blank, `null`, and empty-object values are
accepted as unconfigured; invalid or nonempty values are rejected with migration
guidance.

To retain driver-only metadata, use an administrator-managed mutating admission
policy or webhook that matches Pods labeled
`workflows.argoproj.io/component: agent` in the intended pipeline namespaces.
Apply required mesh annotations and monitoring labels at creation time; changing
an already running Pod does not retroactively inject a sidecar. Keep Argo's
ownership, workflow, and run-identity labels intact, and verify both agent
startup and component execution after the change.

If the metadata is deliberately intended for **all** workflow Pods, configure
Argo controller `workflowDefaults.spec.podMetadata` instead. This includes
component Pods and is not an automatic translation of the old driver-only
settings. Check template-level metadata overrides and mesh behavior before
rollout. KFP does not install a metadata-mutating admission policy for you.

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
