# Server Configuration

By default, you can use Kubeflow Pipelines deployment manifests as provided,
which aim to offer a standard configuration for most use cases. At the meantime,
customizations are available for more advanced usage.

When deploying Kubeflow Pipelines servers, you can pass various environment variables
to customize the behavior of servers.

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
`RollingUpdate` with `maxUnavailable: 0`, so upgrades can keep serving requests
while a replacement pod becomes ready. Allow capacity for one additional UI pod.

A separate initialization Job generates the key only if the field is absent.
It preserves existing keys, including operator-provided keys, and concurrent
initializers use Kubernetes resource versions to avoid overwriting each other.
The Job can only get and update this named Secret; the UI receives no additional
Secret permissions. UI pods wait for the required Secret field before starting.
If initialization fails, check the Job status, its get/update permissions, and
whether an existing key is too short. An invalid existing key is never replaced
automatically. After correcting a failed initialization, delete the failed Job
and reapply the manifests to retry. If the Secret was deleted, restore it from
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
32 bytes and override the deployment's reference if necessary:

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
process-local key invalidates existing TensorBoard proxy URLs. Reopen TensorBoard
from the UI to obtain a new URL. Subsequent UI restarts and rolling updates keep
URLs valid while the shared key remains unchanged. Deliberately replacing or
losing the Secret invalidates existing URLs again. After manual rotation,
restart every UI replica to load the new value; use a coordinated restart to
avoid serving with different keys during the transition.

Outside the default manifests, leaving `TENSORBOARD_PROXY_SIGNING_SECRET` unset
still generates a process-local key. This supports local development, but URLs
expire on process restart and replicas cannot share URLs. Configure a shared key
before enabling multiple replicas or rolling updates in custom deployments.

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
