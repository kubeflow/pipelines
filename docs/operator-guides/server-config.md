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

`WORKFLOW_SERVICE_ACCOUNT_AUDIT` defaults to `false`. Set it to `"true"` on the
`ml-pipeline` API server deployment to observe the **additional** identity checks
without blocking on their findings:

```yaml
env:
  - name: WORKFLOW_SERVICE_ACCOUNT_AUDIT
    value: "true"
```

Audit mode still enforces the workflow's main service account, including Kubernetes'
`default` account if the main field is empty. Failures involving
other accounts, including accounts introduced by plugins or retained retry state,
produce `Workflow service account audit:` warning logs and execution continues.
These accounts can therefore run without passing the expanded policy while audit
mode is enabled. Use this option temporarily to assess compatibility, then unset
it or set it to `"false"` to enforce the checks. It applies to both V1 and V2.

Warnings include the operation, namespace, workflow name or generated-name prefix,
run ID when available, account name when known, and one of these findings:

| Finding | Meaning |
| --- | --- |
| `account_not_allowed` | The additional account is not allowed or its name is not literal. |
| `account_denied` | Kubernetes denied the caller permission to use the additional account. |
| `authorization_error` | Authorization of the additional account could not be completed. |
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
