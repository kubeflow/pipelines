# Server Configuration

By default, you can use Kubeflow Pipelines deployment manifests as provided,
which aim to offer a standard configuration for most use cases. At the meantime,
customizations are available for more advanced usage.

When deploying Kubeflow Pipelines servers, you can pass various environment variables
to customize the behavior of servers.

## API Server workflow service-account checks

By default, the API server enforces the service-account policy for all identities
it finds in a workflow before creating a run or recurring run, retrying a run, or
re-enabling an embedded recurring workflow. It also checks again after plugins
modify a new run. Non-default accounts must be listed in `ALLOWEDSERVICEACCOUNTS`;
in multi-user mode, the caller must additionally have `use` permission on those
`serviceaccounts`. The configured default pipeline runner retains its existing
exemption.

### Audit mode for rollout

`KFP_SECURITY_WORKFLOW_IDENTITY_MODE` defaults to `enforce` (including empty values). Set it to `"audit"` on the
`ml-pipeline` API server deployment to observe the **additional** identity checks
without blocking on their findings:

```yaml
env:
  - name: KFP_SECURITY_WORKFLOW_IDENTITY_MODE
    value: "audit"
```

This mode does not relax the workflow's main service account, including Kubernetes'
`default` account if the main field is empty. Main-account policy is controlled
independently by `KFP_SECURITY_SERVICE_ACCOUNT_MODE`, which defaults to `enforce`. Failures involving
other accounts, including accounts introduced by plugins or retained retry state,
produce structured `security_audit control=workflow_identity mode=audit` warning logs and execution continues for policy denials and incomplete local identity inspection. Authentication failures, authorization transport errors, and SubjectAccessReview evaluation errors remain blocking, even if another policy violation was audited.
These accounts can therefore run without passing the expanded policy while audit
mode is enabled. Use this option temporarily to assess compatibility, then unset
it or set it to `"enforce"` to enforce the checks. It applies to both V1 and V2.

Fail-closed evaluation handling applies to the API server's shared multi-user
authorization checks, not only to workflow service accounts. A non-empty
SubjectAccessReview `evaluationError` is treated as an internal authorization
failure even when the response also includes an allow or deny decision. Restore
healthy authorization evaluation before retrying; ordinary denials without an
evaluation error still return permission denied.

Invalid values fail startup, configuration reload, and request validation. Audit mode emits an exposure warning on startup/reload and is planned for removal in [3.0](https://github.com/kubeflow/pipelines/issues/14367). Apply the same mode to every API server replica and persist it in your deployment configuration.

Warnings include the operation, namespace, workflow name or generated-name prefix,
run ID when available, account name when known, and one of these findings:

| Finding | Meaning |
| --- | --- |
| `account_not_allowed` | The additional account is not allowed. |
| `account_denied` | Kubernetes denied the caller permission to use the additional account. |
| `inspection_incomplete` | The workflow's identities could not be fully collected, for example because a patch is dynamic or uses noncanonical account fields. |

An inspection failure stops collection, so `inspection_incomplete` is **not** a
complete account inventory and later violations may not be reported. Review that
workflow's templates, pod patches and retained execution state before enabling
enforcement. Audit warnings do not include manifests, patch contents or raw error
payloads. Logs may repeat across lifecycle operations and the post-plugin check.

Audit mode does not bypass ordinary workflow validation: fresh submissions still
reject external template references and discard caller-supplied workflow status.
On retained workflows checked during retry or re-enable, an external reference
instead produces an `inspection_incomplete` warning in audit mode. In multi-user mode, scheduled ticks pass through the API and inspect the restored
execution inputs. Re-enabling an embedded schedule also triggers inspection.
Single-user embedded execution paths that do not call the API are not retroactively
inspected. Running workloads are not retroactively reauthorized. See the
[combined mode matrix](scheduled-service-accounts.md#combining-main-account-and-workflow-identity-modes).

### V1 and V2 compatibility

**V1 / raw Argo workflows:** this is the main compatibility change. Enforcement
requires literal service-account names and canonical `serviceAccountName` or
`serviceAccount` fields in pod patches. Dynamic `podSpecPatch` expressions are
rejected even when they only affect resources. Every declared template is checked,
including unused templates and overridden settings. External templates must be
inlined. Submitted workflow status is ignored, so entrypoints must be defined in
the submitted spec. Audit mode can help identify additional-account and patch
inspection failures, but the validation and status rules still apply.

**V2 / compiler-generated workflows:** the compiler's exact
`{{inputs.parameters.pod-spec-patch}}` placeholder remains supported because the
KFP driver constructs the runtime patch without selecting a service account.
Literal accounts elsewhere in the workflow are still checked. V2 creation,
plugin processing, retries and recurring runs use the same enforcement or audit
policy. V2 recurring runs remain retryable when their persisted records contain
both the source pipeline spec and the compiled workflow manifest. Audit mode is
not normally needed solely for the compiler-generated patch.

For a run or recurring job created from a stored pipeline version, retry or
re-enablement grants this exception
only when that exact version still contains its validated V2 source. Deleting
the version or removing its saved source disables the exception; unused supplied
manifests and newer pipeline versions cannot restore it. Static workflows can
still retry or be re-enabled if their service-account checks pass. Legacy V2
recurring jobs with both source manifest fields remain re-enableable when their
authoritative source establishes V2 provenance.

## Frontend Server

When deploying frontend server called `ml-pipeline-ui`, you can pass various environment
variables to customize the server behavior for your namespace. Some examples are shown
in the [ml-pipeline-ui-deployment.yaml](https://github.com/kubeflow/pipelines/blob/b630d5c8ae7559be0011e67f01e3aec1946ef765/manifests/kustomize/base/pipeline/ml-pipeline-ui-deployment.yaml#L32-L50).

### Artifact storage endpoint allowlist

For S3-compatible storage, `ALLOWED_ARTIFACT_ENDPOINTS` lists additional exact
trusted origins; a permissive domain regex alone does not authorize a custom
storage origin. See the [custom S3 upgrade example](https://github.com/kubeflow/pipelines/blob/master/manifests/kustomize/README.md#upgrade-example-custom-s3-storage)
for ConfigMap, profile-proxy rollout, alias, and archived-log credential guidance.
HTTP artifact bases use the separate setting described below.

You can configure `ALLOWED_ARTIFACT_DOMAIN_REGEX` to allowlist object storage endpoint
that your frontend server will fetch artifacts from. If the domain that frontend server
tries to fetch does not match the regular expression defined in
`ALLOWED_ARTIFACT_DOMAIN_REGEX`, it will return error to users that the requested domain
is not allowed.

#### Standalone Kubeflow Pipelines deployment

By default, the value for `ALLOWED_ARTIFACT_DOMAIN_REGEX` is `"^.*$"`. You can customize
this value for your users, for example: `^.*.yourdomain$` in the
[ml-pipeline-ui-deployment.yaml](https://github.com/kubeflow/pipelines/blob/b630d5c8ae7559be0011e67f01e3aec1946ef765/manifests/kustomize/base/pipeline/ml-pipeline-ui-deployment.yaml#L32-L50).

#### Full fledged Kubeflow deployment

For full fledged Kubeflow, each namespace corresponds to a project with the same name.
To configure the `ALLOWED_ARTIFACT_DOMAIN_REGEX` value for user namespace, add an entry in `ml-pipeline-ui-artifact`
just like this example in [sync.py](https://github.com/kubeflow/pipelines/blob/b630d5c8ae7559be0011e67f01e3aec1946ef765/manifests/kustomize/base/installs/multi-user/pipelines-profile-controller/sync.py#L304-L310) for `ALLOWED_ARTIFACT_DOMAIN_REGEX` environment variable,
the entry is identical to the environment variable instruction in Standalone Kubeflow Pipelines
deployment.

### HTTP artifact migration for 2.18

For a standalone deployment with an existing artifact URI such as
`https://files.example:9443/reports/result.json`, configure a fully qualified
approved base on the frontend server:

```yaml
# Merge into the existing ConfigMap; preserve its other keys.
apiVersion: v1
kind: ConfigMap
metadata:
  name: pipeline-install-config
  namespace: kubeflow
data:
  HTTP_BASE_URL: "https://files.example:9443/reports/"
```

The server retrieves the original URI without appending the hostname or
`reports/` a second time. Existing artifact URIs do not need rewriting. In this
form, the request's scheme, host, and effective port must match the configured
origin, and its path must stay within the configured path boundary. For example,
`/reports-other/file` and `/private/file` are outside `/reports/`. Redirects must
remain in the same approved origin/path and also pass
`ALLOWED_ARTIFACT_DOMAIN_REGEX`. Use a common approved path prefix if your storage
server redirects into a different download directory. Do not widen the boundary
to destinations that should not receive artifact requests or configured HTTP
credentials.

A fully qualified base cannot contain credentials, a query, or a fragment.
Configure supported HTTP authentication through `HTTP_AUTHORIZATION_KEY` and
`HTTP_AUTHORIZATION_DEFAULT_VALUE` on the serving process, rather than embedding
credentials in the URL. The installation configuration propagates the base URL,
not authentication credentials. Query-bearing signed URLs are not a replacement
for the supported artifact path/authentication configuration.

The existing **scheme-less gateway form remains supported**:
`HTTP_BASE_URL=gateway.example/artifacts/`, with an HTTP artifact request for
logical bucket `dataset` and key `result.json`, still fetches
`http://gateway.example/artifacts/dataset/result.json` (or HTTPS when requested).
Keep that form if you intentionally use gateway bucket/path mapping. Adding a
scheme selects the original-URI behavior above; it is not a cosmetic change to a
gateway setting. Neither form permits fetching an arbitrary request-selected
host with an unset base.

Both forms validate the configured path before URL normalization. Use a base
without raw or percent-encoded `.`/`..` path segments, backslashes, invalid
percent encoding, or embedded ASCII control characters (literal or percent-encoded).
Correct these values in the installation configuration; they are rejected with
HTTP 400 on artifact requests, without preventing server startup.

The base is read at process startup. After applying the ConfigMap, restart
`ml-pipeline-ui`. Authenticated multi-user HTTP requests are fetched by the shared
UI even when namespace artifact proxies are enabled. Configure any HTTP
authentication on the shared UI as well. Both the configured base boundary and
the namespace ownership policy apply: artifact keys and redirects must stay under
`private-artifacts/<namespace>/` (or the configured namespace prefix). For example,
use `https://files.example:9443/private-artifacts/` as the base for namespace-scoped
URIs; the `/reports/` example above applies to standalone deployments.

Multi-user HTTP redirects cannot change origin or logical gateway bucket, and
each redirected object must remain under the authorized namespace prefix. This
object is also revalidated against the configured MLMD ownership policy: strict
`mlmd-only` mode and conflicting ownership evidence continue to deny access. The
check runs on the shared UI before contacting the redirect destination, including
during rolling upgrades with older tenant proxies. Keep HTTP credentials and
network access available to the shared UI; configuring them only on tenant
proxies is insufficient. Redirect destinations containing queries or fragments
are rejected, including redirects to query-signed download URLs. Use the supported
HTTP header authentication configuration instead.

The temporary `ARTIFACT_OWNERSHIP_ENFORCEMENT=audit` setting still permits an
initial custom-root read with matching MLMD evidence. That evidence does not
authorize another object chosen by an HTTP redirect: custom-root redirects are
denied unless the destination is the identical URL. Use the final artifact URL
directly while migrating custom roots. Standalone deployments retain approved-base
redirect behavior without namespace checks. Object-store downloads continue to
use tenant proxies and support their legacy download route during rolling upgrades.

The installation also propagates the base to profile proxies for their direct
HTTP serving configuration. Restart `kubeflow-pipelines-profile-controller` and
wait for its rollout to finish. The controller watches enabled Namespace objects;
restarting its webhook does not enqueue them. Wait for the next hourly resync or
change an annotation on each affected Namespace, as shown in the
[profile-proxy rollout example](https://github.com/kubeflow/pipelines/blob/master/manifests/kustomize/README.md#upgrade-example-custom-s3-storage).
Confirm each generated `ml-pipeline-ui-artifact` pod template contains the expected
`HTTP_BASE_URL` before checking its rollout status, which could otherwise report
the previous rollout's completion. This propagation does not change the shared
UI's authenticated HTTP serving path. Preserve the setting in your installation
manifests for later upgrades.

When upgrading from manifests that did not emit `HTTP_BASE_URL` into profile
proxy pods, the first reconciliation adds the entry even when its value is empty.
With `ARTIFACTS_PROXY_ENABLED=true`, this changes existing artifact-proxy pod
templates and causes a one-time rollout. Subsequent resyncs do not roll unchanged
templates.

Test an existing artifact preview and download after rollout. A missing base
returns HTTP 400 naming `HTTP_BASE_URL`; an invalid base, mismatched origin/path,
or out-of-base redirect also returns HTTP 400 without fetching the disallowed
destination. Domain-regex rejection remains an additional restriction. Endpoint
configuration does not bypass namespace authorization or artifact ownership
checks. Artifact responses remain attachments, and HTTP archive bytes remain
unextracted.

### TensorBoard proxy signing secret

The frontend signs scoped TensorBoard proxy paths with
`TENSORBOARD_PROXY_SIGNING_SECRET`. The default Kustomize installation initializes
a dedicated random key in the `ml-pipeline-ui-tensorboard-proxy` Secret and
injects its `signing-secret` field into every UI replica. The UI uses
`Recreate` by default so the first upgrade cannot serve requests through old and
new UI pods with incompatible signing keys. During Deployment upgrades,
[Kubernetes waits for old pods to terminate before creating replacements](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#recreate-deployment).
After adopting the shared key, operators can
[enable rolling UI updates](#enabling-rolling-ui-updates).

A separate initialization Job generates the key only if the field is absent.
It preserves existing keys, including operator-provided keys, and concurrent
initializers use Kubernetes resource versions to avoid overwriting each other.
The Job can only get and update this named Secret; the UI receives no additional
Secret permissions. There is no Deployment-to-Job ordering dependency: `Recreate`
only orders old and new UI pods. The non-optional `secretKeyRef` for
`signing-secret` is the startup gate. On a fresh installation, a UI pod may
briefly show `CreateContainerConfigError` while the Job populates the empty
Secret; kubelet retries and starts the container when the key exists. This is
expected and does not require an init container or Secret-read RBAC for the UI.
Wait for the initializer Job to complete and then for the UI rollout before
routing traffic. An existing valid shared key lets new UI pods start immediately,
without waiting for the idempotent Job.
If initialization fails, check the Job status, its get/update permissions, and
whether an existing key is valid UTF-8 of at least 32 bytes with no NUL characters.
An invalid existing key is never replaced automatically. After correcting a failed
initialization, delete the failed Job and reapply the manifests to retry. If the Secret was deleted, restore it from
backup before starting replacement UI pods. If restoration is impossible,
reapply the manifests to recreate the empty Secret, delete the existing
initializer Job, and reapply again to generate a new key. Restart all UI replicas
together after regeneration. Reapplying alone does not rerun a completed Job.

The Secret manifest intentionally omits `data` and `stringData`. Keep those
fields absent when using automatic initialization, and preserve the Secret
across upgrades and in backups. Reapplying the manifests does not rotate the key.
For GitOps, use apply-based reconciliation and keep generated key data out of
Git. Do not replace, force-recreate, or prune this Secret during upgrades. If
Argo CD reports the generated field as drift, scope `ignoreDifferences` to this
Secret's name and namespace and `/data/signing-secret`, and enable
`RespectIgnoreDifferences=true` to preserve the live value during sync. See
[Argo CD sync options](https://argo-cd.readthedocs.io/en/latest/user-guide/sync-options/#respect-ignore-differences-configs).

The Job name includes a hash of its source template, configured image, and
settings so image upgrades create a new Job without mutating a completed Job's
immutable pod template. Completed Jobs and their generated ConfigMaps can be
pruned after upgrades; do not delete the signing Secret. If an overlay changes
the Job pod template, also add or change a literal in its ConfigMap generator to
force a new Job name.

To use an externally managed key, provide a dedicated random secret of at least
32 UTF-8 bytes with no NUL characters and override the deployment's reference if
necessary:

```yaml
env:
  - name: TENSORBOARD_PROXY_SIGNING_SECRET
    valueFrom:
      secretKeyRef:
        name: my-tensorboard-signing-secret
        key: signing-secret
```

Do not reuse `MINIO_SECRET_KEY` or another application credential. The frontend
refuses to start if the configured signing secret is shorter than 32 UTF-8 bytes
or matches `MINIO_SECRET_KEY`.

**Upgrade and rotation:** the first upgrade from a storage-derived or
process-local key briefly interrupts the UI and invalidates existing TensorBoard
proxy URLs. Wait for the upgrade to complete, then reopen TensorBoard from the
UI to obtain a new URL. Subsequent UI restarts and rolling updates keep URLs
valid while the shared key remains unchanged. Deliberately replacing or losing
the Secret invalidates existing URLs again. After manual rotation, restart every
UI replica to load the new value; use a coordinated restart to avoid serving
with different keys during the transition.

Outside the default manifests, leaving `TENSORBOARD_PROXY_SIGNING_SECRET` unset
still generates a process-local key. This supports local development, but URLs
expire on process restart and replicas cannot share URLs. Configure a shared key
before enabling multiple replicas or rolling updates in custom deployments.

#### Enabling rolling UI updates

Adopt the shared key and enable rolling updates in two separate apply or GitOps
sync phases. Keep the same shared-key revision throughout both phases.

First, remove any existing rolling-update overrides and apply the shared-key
revision with `Recreate`. If `spec.strategy.rollingUpdate` was explicitly
configured, remove that field as well or replace the entire strategy with
`{type: Recreate}`. Wait for the rollout to complete for your installation's UI
Deployment and namespace; for the default `kubeflow` installation:

```bash
kubectl -n kubeflow rollout status deployment/ml-pipeline-ui
```

Only after that command succeeds and every UI pod uses the shared key, add this
override to your installation's `kustomization.yaml` and apply the same revision
again:

```yaml
patches:
  - target:
      group: apps
      version: v1
      kind: Deployment
      name: ml-pipeline-ui
    patch: |-
      - op: replace
        path: /spec/strategy
        value:
          type: RollingUpdate
          rollingUpdate:
            maxUnavailable: 0
            maxSurge: 1
```

For GitOps, complete the first sync and wait for the UI rollout before starting
the second sync with this override. Do not combine the phases or enable rolling
updates while adoption is in progress: old and shared-key UI pods must not
overlap during the first migration. Keep the override for future upgrades and
allow capacity for one additional UI pod.

## Proxy

Since KFP 2.5, you can set a server-scoped proxy configuration for the backend by setting any of the following environment variables (in uppercase) in the
API Server deployment. All variables are optional.

- `HTTP_PROXY`
- `HTTPS_PROXY`
- `NO_PROXY`

If `HTTP_PROXY` or `HTTPS_PROXY` is set and `NO_PROXY` is not set, `NO_PROXY` will automatically be set to `localhost,127.0.0.1,.svc.cluster.local,kubernetes.default.svc,metadata-grpc-service,0,1,2,3,4,5,6,7,8,9`.

### Example of an API Server deployment with `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` set

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: ml-pipeline
    application-crd-id: kubeflow-pipelines
  name: ml-pipeline
  namespace: kubeflow
spec:
  selector:
    matchLabels:
      app: ml-pipeline
      application-crd-id: kubeflow-pipelines
  template:
    metadata:
      annotations:
        cluster-autoscaler.kubernetes.io/safe-to-evict: "true"
      labels:
        app: ml-pipeline
        application-crd-id: kubeflow-pipelines
    spec:
      containers:
      - env:
        - name: HTTP_PROXY
          value: http://squid.squid.svc.cluster.local:3128
        - name: HTTPS_PROXY
          value: http://squid.squid.svc.cluster.local:3128
        - name: NO_PROXY
          value: localhost,127.0.0.1,.svc.cluster.local,kubernetes.default.svc,metadata-grpc-service,0,1,2,3,4,5,6,7,8,9
```

### Pipeline size limits

The API server enforces three independent byte ceilings. Configure these environment
variables on the `ml-pipeline` deployment and roll out all replicas consistently:

| Environment variable | What it bounds | Default | Supported range |
| --- | --- | --- | --- |
| `MAX_PIPELINE_UPLOAD_BYTES` | Pipeline file/package bytes from multipart uploads, URL imports, and bootstrap files | 33554432 (32 MiB) | 1–134217728 bytes (128 MiB) |
| `MAX_PIPELINE_SPEC_BYTES` | Extracted YAML/JSON, uncompressed pipeline files, and object-store pipeline-spec reads | 33554432 (32 MiB) | 1–134217728 bytes (128 MiB) |
| `MAX_PIPELINE_UPDATE_BODY_BYTES` | Entire HTTP PUT/PATCH body on pipeline and pipeline-version update routes | 33554432 (32 MiB) | 1–134217728 bytes (128 MiB) |

Values are decimal integers in bytes; `64MiB`, zero, negative, and out-of-range
values are invalid. Unset or empty values use the default. Invalid settings prevent
API-server startup; there is no unlimited or audit mode. The upper bound limits
administrator overrides because these paths buffer content in memory; it is not
a guarantee that every deployment can safely accept concurrent requests that large.
Raise limits only after sizing memory for concurrent uploads, decompression, parsing,
and request processing. Prefer moving large embedded artifacts, notebooks, and code
to container images or object storage.

For example, to permit 64 MiB pipeline files and extracted specifications:

```sh
kubectl set env deployment/ml-pipeline -n kubeflow \
  MAX_PIPELINE_UPLOAD_BYTES=67108864 MAX_PIPELINE_SPEC_BYTES=67108864
kubectl rollout status deployment/ml-pipeline -n kubeflow
```

Persist overrides in your deployment/GitOps configuration. The upload ceiling applies
to the selected file, not total multipart framing. Raw YAML/JSON must fit both input
and spec ceilings. Compressed packages must fit the input ceiling and their extracted
specification must fit the spec ceiling. Tar scanning remains bounded by the spec
ceiling plus 1 MiB for headers, metadata, padding, and other entries encountered during
scanning; this derived budget cannot be disabled independently. Remove unnecessary
archive entries if traversal is rejected even though the selected YAML is small.

These settings do not raise ingress/proxy, gRPC, Kubernetes object, or database limits.
Configure and validate the complete request path; raising the KFP limit alone does not
ensure a large pipeline can execute. Processes using the Kubernetes pipeline-upload
client or bootstrap helpers read these same environment variables locally; keep their
configuration consistent where those paths are used.

Multipart upload and update-body size rejections return HTTP 413 with the applicable
ceiling and setting. Other pipeline APIs preserve their existing InvalidArgument error
mapping. API-server logs include `size_limit_exceeded`, the control, limit in bytes,
and configuration name, without logging the rejected payload. Bounded reads do not
measure the full rejected file, so errors report that it exceeds the ceiling rather
than claiming an exact total size. Non-size parsing and transport failures retain
sanitized upload responses.

To return to defaults, remove the overrides and roll out the deployment. Before
lowering `MAX_PIPELINE_SPEC_BYTES`, check stored pipelines accepted under the higher
limit: object-store specifications may no longer be readable for subsequent execution. This
setting does not add a new size check to the existing direct database-spec read path.

### Legacy V1 cache security mode

On the cache-server deployment, `KFP_SECURITY_LEGACY_CACHE_MODE` accepts `enforce`
(default) or `audit`. Audit permits legacy cache reuse after a scoped miss and
reports `ownership_unknown`; it weakens isolation and is planned for removal in
3.0.0. Invalid mode values are rejected at startup. See the [cache migration guide](https://github.com/kubeflow/pipelines/blob/release-2.18/backend/src/cache/README.md#temporary-cache-audit-mode)
for configuration, rollout, and return-to-enforcement instructions. Native V2
caching is unaffected.

## Recurring runs and custom service accounts

See [Service accounts for recurring runs](scheduled-service-accounts.md) for
`ALLOWEDSERVICEACCOUNTS`, scoped controller grants, multi-user upgrade requirements,
and revoking scheduled execution.

### Service-account authorization migration mode

`KFP_SECURITY_SERVICE_ACCOUNT_MODE` accepts `enforce` (default, including upgrades)
or `audit`. Audit temporarily allows service-account policy denials and logs
warnings, restoring the associated security exposure. It does not disable
existing authentication or namespace authorization. See the
[scope, rollout, and migration instructions](scheduled-service-accounts.md#temporary-audit-mode-for-migration).
Audit mode is planned for removal in 3.0.0 ([#14367](https://github.com/kubeflow/pipelines/issues/14367)).
