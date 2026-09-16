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

Audit mode still enforces the workflow's main service account, including Kubernetes'
`default` account if the main field is empty. Failures involving
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
instead produces an `inspection_incomplete` warning in audit mode. Existing enabled
embedded schedules are not automatically inspected; re-enabling them triggers the
check. Running workloads are not retroactively reauthorized.

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
mode as V1. V2 recurring runs remain retryable when their persisted records contain
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

### TensorBoard proxy signing secret

The frontend signs scoped TensorBoard proxy paths with
`TENSORBOARD_PROXY_SIGNING_SECRET`. If this variable is unset, each frontend
server process generates a random signing secret at startup. This default is
suitable for the standard single-replica deployment, whose `Recreate` strategy
prevents pods with different process-local secrets from serving concurrently.
The UI is briefly unavailable during an update, and existing proxy paths become
invalid whenever the frontend restarts.

Deployments with multiple frontend replicas, or deployments that need proxy
paths to survive restarts, must provide the same dedicated random secret of at
least 32 bytes to every `ml-pipeline-ui` replica. Store it in a Kubernetes
Secret and reference it from the deployment, for example:

```yaml
env:
  - name: TENSORBOARD_PROXY_SIGNING_SECRET
    valueFrom:
      secretKeyRef:
        name: ml-pipeline-ui-tensorboard-proxy
        key: signing-secret
```

The base deployment uses `Recreate` to protect the process-local default. After
configuring a shared signing secret, deployments that require uninterrupted
updates can override `spec.strategy.type` to `RollingUpdate`.

Do not reuse `MINIO_SECRET_KEY` or another application credential for this
value. The frontend refuses to start when the configured signing secret is
shorter than 32 bytes or matches `MINIO_SECRET_KEY`.

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
