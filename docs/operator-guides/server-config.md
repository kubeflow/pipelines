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
Secret permissions. UI pods wait for the required Secret field before starting.
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
