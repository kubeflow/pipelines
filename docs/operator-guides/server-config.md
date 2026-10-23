# Server Configuration

By default, you can use Kubeflow Pipelines deployment manifests as provided,
which aim to offer a standard configuration for most use cases. At the meantime,
customizations are available for more advanced usage.

When deploying Kubeflow Pipelines servers, you can pass various environment variables
to customize the behavior of servers.

## Run retention and index preparation

Retention is disabled unless `RUNS_RETENTION_TIME` (archive) or
`ARCHIVED_RUNS_RETENTION_TIME` (delete) is configured. When enabled, the API server
prepares the required database indexes in the background before collection;
API startup does not wait for these builds. If retention was already configured
but paused because indexes were missing, upgrading can now start preparation and
subsequent archival/deletion automatically. Use prepare-only across all replicas
to review that backlog before allowing catch-up.

| Setting | Default | Behavior |
| --- | --- | --- |
| `RUNS_GC_INDEX_BUILD_TIMEOUT` | `12h` | Positive Go duration for each sequential online build attempt; metadata-lock waits stay bounded at five seconds. Server cleanup may continue after cancellation. |
| `RUNS_GC_INDEX_PREPARE_ONLY` | `false` | When true, prepare indexes but never archive/delete runs. Works with retention disabled so indexes can be prepared ahead of activation. |
| `RUNS_GC_INTERVAL` | `6h` | Collection interval; a new leader, completed build or recovery waits a full interval before catch-up. |
| `RUNS_GC_BATCH_SIZE` | `100` | Maximum rows per existing collection batch; catch-up may drain many batches. |

Use prepare-only consistently across all API replicas and installations sharing
the database. It cannot pause an old leader before its configuration is updated.
For first adoption, leave retention disabled, prepare the indexes, then enable
the desired windows and clear prepare-only when ready for catch-up. Invalid
prepare-only/timeout values fail configuration validation.

Build failures pause GC, not API availability, and retry with bounded backoff.
Interrupted PostgreSQL builds with the exact managed index definition recover
automatically using concurrent drop/rebuild. Active DDL defers recovery; conflicting
definitions require inspection. Coordinate manual changes to managed indexes by
stopping preparation across all replicas first.
Online DDL can still consume substantial resources and briefly delay writes at
metadata locks. Review the [preparation and recovery procedure](https://github.com/kubeflow/pipelines/blob/release-2.18/docs/agents/development.md#run-garbage-collection-database-migration)
for capacity planning, PostgreSQL invalid-index recovery and staged activation.

## Automatic artifact preview generation

Automatic artifact preview generation is **enabled by default** for supported
Python-component Dataset and Model outputs. It is controlled by pipeline authors,
not by deployment configuration. Use
`@dsl.pipeline(pipeline_config=dsl.PipelineConfig(artifact_previews=False))`
to disable generation throughout a pipeline, or
`@dsl.component(artifact_previews=False)` for a particular component. A pipeline
opt-out also applies to nested pipelines and exit handlers and cannot be
overridden by enabling previews on a component.

These settings are compiled into the pipeline. Recompile and use the updated
pipeline definition when changing them. No preview-specific ConfigMap, mounted
configuration file, profile reconciliation, or API-server/driver setting is
required. Because generation is on by default, set pipeline or component
opt-outs before upgrading sensitive workloads to a supporting SDK.

Dataset previews can contain up to ten rows and twenty columns. Review row
samples and model configuration for sensitive data; pipeline authors can keep
only dataset schema with `@dsl.component(artifact_preview_rows=False)`.
Disabling generation does not remove or hide previously generated, cached,
imported, or manually written previews, and does not change ordinary artifact
production, downloads, or access controls.

See [Dataset and Model previews](../user-guides/data-handling/artifacts.md#dataset-and-model-previews)
for supported formats, pipeline/component privacy settings, and the file contract.

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

### Artifact request limits

Each UI server instance permits 300 artifact requests per minute per user in
authenticated deployments, using the same trusted upstream identity header as
authorization. Standalone and unauthenticated requests share a budget by client
IP instead; caller-supplied identity and forwarding headers do not override it.
The ingress must continue to strip untrusted identity headers before supplying
the authenticated user.

Previews and downloads share this budget across namespaces and base-path aliases.
Requests beyond the limit receive HTTP 429 with a `Retry-After` delay before any
artifact lookup, authorization call, or storage read. The budget is local to each
server process, not a distributed deployment-wide quota.

When the `pipeline-install-config` key `ARTIFACTS_PROXY_ENABLED` is `true`, the
shared UI applies each user's budget once, before forwarding a request to the
namespace's `ml-pipeline-ui-artifact` server. The profile controller sets
`ARTIFACTS_RATE_LIMIT_DISABLED=true` on those generated servers. They run with
authorization disabled and see every forwarded request as coming from the shared
UI, so a second limit would combine all of a namespace's users into one budget.
If a namespace server returns HTTP 429, the shared UI preserves that status and
its delay-seconds `Retry-After` value for previews and downloads.

`ARTIFACTS_RATE_LIMIT_DISABLED` accepts `true` or `1`; any other value keeps the
limit. Generated namespace servers rely on the shared UI for this limit, as they
do for authorization. Do not set the variable on the shared UI or in standalone
deployments, which have no upstream limit. After upgrading, the profile
controller adds the variable to each existing generated Deployment at its next
reconciliation, which rolls that Deployment once.

### Artifact storage endpoint allowlist

For S3-compatible storage, `ALLOWED_ARTIFACT_ENDPOINTS` lists additional exact
trusted origins; a permissive domain regex alone does not authorize a custom
storage origin. See the [custom S3 upgrade example](https://github.com/kubeflow/pipelines/blob/release-2.18/manifests/kustomize/README.md#upgrade-example-custom-s3-storage)
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
not authentication credentials. Query-bearing signed URLs used as initial artifact
URIs are not a replacement for the supported path/authentication configuration;
redirect queries follow the destination policy described below.

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

The base is read at process startup. Preserve HTTP authentication and network
access on the process that already serves the artifacts. With namespace artifact
proxying enabled, the shared UI authorizes the caller and forwards HTTP requests
to that namespace's receiver. Otherwise the shared UI serves the request. The
2.18 upgrade does not require moving namespace-local HTTP credentials into the
shared UI.

Updated serving processes retain configured destination checks on every redirect.
A fully qualified base confines redirects to its origin and path. The scheme-less
gateway form retains redirects to allowed destinations, including another path or
origin accepted by `ALLOWED_ARTIFACT_DOMAIN_REGEX`; authentication headers are
removed when the origin changes. Redirect queries, including signed download
queries, are allowed within the applicable destination policy. Redirect limits,
URL/path validation and credential protections remain enforced.

The stronger artifact ownership requirement is deferred to 3.0. Initial requests
retain 2.17's MLMD-based ownership behavior, including known namespace-mismatch
rejection and the existing prefix fallback for untracked artifacts. Neither an
initial tracked custom root nor its redirect acquires a new mandatory namespace
prefix. Missing namespace evidence and unavailable MLMD retain their historical
permissive ownership behavior; caller authentication and namespace permission
checks still fail closed. This is a compatibility decision, not complete
remediation of the cross-tenant artifact ownership advisory. The unreleased
`KFP_SECURITY_ARTIFACT_OWNERSHIP_MODE` enforce/audit control is removed.

The installation propagates the configured base and allowed artifact endpoints
to managed namespace receivers, not their HTTP authentication credentials. Keep
custom receiver credential configuration in its existing deployment source and
verify that reconciliation preserves it. The profile controller applies its
configured frontend image and environment through ordinary Deployment rollouts.
Restarting its webhook does not enqueue every namespace: existing namespaces
reconcile on the normal hourly resync or when their watched configuration changes.
The [profile-proxy rollout example](https://github.com/kubeflow/pipelines/blob/release-2.18/manifests/kustomize/README.md#upgrade-example-custom-s3-storage)
shows how to request an earlier reconciliation when needed.

Either shared-UI/receiver update order retains the established proxy path. During
overlap, old receivers retain their old redirect behavior; the shared UI upgrade
alone does not install the newer HTTP protections in them. Before considering the
rollout complete, verify the serving receiver image and `HTTP_BASE_URL` in each
pod template, its completed rollout, and that old receiver pods have exited.
Custom/unmanaged receivers use their existing deployment mechanism. No new
artifact approval or readiness protocol is required.

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

### Artifact ownership controls in 2.18

The stronger ownership requirement is deferred to 3.0. The existing 2.17 lookup
controls below remain available; they do not establish complete object ownership.
Configure them on the authenticated serving UI, restart that process after changes,
and keep its replicas consistent.

| Setting | Default | Purpose |
| --- | --- | --- |
| `ARTIFACT_NAMESPACE_OWNERSHIP_MODE` | `mlmd-then-prefix` | Controls a successful MLMD lookup with no artifact: `mlmd-then-prefix` checks the leading namespace prefix; `mlmd-only` denies untracked artifacts. Neither value changes the retained permissive behavior for unavailable MLMD or missing tracked namespace evidence. |
| `ARTIFACT_NAMESPACE_KEY_PREFIX` | `private-artifacts` | One literal leading object-key prefix in `<prefix>/<namespace>/...` for the no-record fallback. It does not move objects or impose a new prefix on tracked custom roots. |

The unreleased `KFP_SECURITY_ARTIFACT_OWNERSHIP_MODE` and earlier
`ARTIFACT_OWNERSHIP_ENFORCEMENT` settings are removed; neither enables artifact
audit. The proposed `auto`/`storage` authorization modes and external-storage
adoption policies do not ship in 2.18. See
[artifact compatibility and retained exposure](upgrade-2.18.md#artifact-compatibility-and-retained-exposure)
for the preserved access behavior and the separate 3.0 security work.

### TensorBoard proxy signing secret

#### Standard manifests: automatic initialization

The standard deployment initializes the signing key automatically during
installation or upgrade. Kubernetes starts the UI once the key is available;
no manual key generation or Secret editing is needed. The
{ref}`2.18 UI signing-key guidance <prepare-the-ui-signing-key>` gives the
normal upgrade steps and first-adoption impact. The custom configuration and
recovery sections below are not additional steps for every installation.

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
Kubernetes handles UI startup during the rollout. An existing valid shared key
lets new UI pods start immediately, without waiting for the idempotent Job.

#### Custom overlays and GitOps

Custom overlays must retain the initializer Job and its RBAC, the named Secret,
the UI's required Secret reference, and `Recreate` for first adoption. Updating
only the UI image does not install these resources. If you already override the
deployment with an externally managed key, see the configuration below.

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

#### Externally managed signing keys

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

#### Key rotation and URL lifetime

The first upgrade from a storage-derived or process-local key briefly interrupts
the UI and invalidates existing TensorBoard
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

#### Signing-key startup and recovery

If UI startup remains blocked, inspect the initializer Job's status and logs and
its get/update permissions on `ml-pipeline-ui-tensorboard-proxy`. The Job name
includes a generated hash; use its name from the rendered manifests or installed
Jobs rather than assuming a fixed suffix. A brief `CreateContainerConfigError`
while the key is absent is normal; a failed Job or persistent error is not.
Do not delete or regenerate the key as the first troubleshooting step.

Check whether an existing key is valid UTF-8 of at least 32 bytes with no NUL
characters, without printing its value in logs or support reports. An invalid
existing key is never replaced automatically. After correcting a failed
initialization, delete the failed Job and reapply the manifests to retry.

If the Secret was deleted, restore it from backup before starting replacement UI
pods. If restoration is impossible, reapply the manifests to recreate the empty
Secret, delete the existing initializer Job, and reapply again to generate a new
key. Restart all UI replicas together after regeneration; existing TensorBoard
proxy URLs are invalidated. Reapplying alone does not rerun a completed Job.

#### Enabling rolling UI updates

Rolling UI updates are an optional customization; the standard upgrade retains
`Recreate`. If you choose to enable them, adopt the shared key and change the
strategy in two separate apply or GitOps sync phases. Keep the same shared-key
revision throughout both phases.

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

### Legacy metrics file limit

`MAX_METRICS_FILE_BYTES` bounds each uncompressed legacy V1 metrics file. Its default
is 1048576 bytes (1 MiB); set a positive decimal integer on the process collecting
legacy workflow metrics, normally the persistence agent. The collector also bounds
the archive response and traversal. This is independent of the API's pipeline-size
settings and does not configure native V2 metrics collection.

Unset values use the default. Invalid, zero or negative values fall back to the
default with a warning, rather than preventing startup. Review memory/concurrency
before raising the limit, persist the setting in your deployment configuration and
roll out the collector. There is no audit or unlimited mode.

An oversized or malformed metrics file can leave an otherwise completed task without
metrics. Review the collector's limit error, reduce the file or deliberately raise
`MAX_METRICS_FILE_BYTES` for future collection. Completed-node permanent failures
are not automatically replayed when the setting changes; verify a new execution
rather than assuming old metrics will be recovered. The pre-existing metric-count
and single-file archive rules still apply.

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
