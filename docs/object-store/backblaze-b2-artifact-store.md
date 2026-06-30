# Backblaze B2 as the Kubeflow Pipelines artifact store

This guide shows how to point Kubeflow Pipelines (KFP) at a [Backblaze B2](https://www.backblaze.com/cloud-storage)
bucket as the artifact store, using B2's S3-compatible API. It reuses the same configuration knobs KFP already exposes
for its default object store (endpoint, region, SSL, path-style addressing, and the credentials secret). No code
changes are required: B2 is reached through KFP's existing S3 client paths.

> Scope note: the umbrella `kubeflow/kubeflow` repository is a meta/governance repo and contains no storage code. The
> artifact-store integration surface lives here, in `kubeflow/pipelines`. Configure B2 in this repo's manifests and
> runtime config, not in `kubeflow/kubeflow`.

## What "artifact store" means in KFP

KFP touches object storage from two distinct components, and both must point at the same B2 bucket:

1. **The API server** (`ml-pipeline`) stores uploaded pipeline definitions and reads/writes artifacts and archived
   logs. Its object-store settings come from `ObjectStoreConfig.*` (see
   [`backend/src/apiserver/config/config.json`](../../backend/src/apiserver/config/config.json) and the
   `OBJECTSTORECONFIG_*` environment variables in
   [`manifests/kustomize/base/pipeline/ml-pipeline-apiserver-deployment.yaml`](../../manifests/kustomize/base/pipeline/ml-pipeline-apiserver-deployment.yaml)).
2. **The driver and launcher** (run inside each pipeline pod) upload step outputs and download step inputs. Their
   object-store settings come from the per-namespace `kfp-launcher` ConfigMap `providers` block, parsed by
   [`backend/src/v2/config/s3.go`](../../backend/src/v2/config/s3.go) and applied by
   [`backend/src/v2/objectstore/object_store.go`](../../backend/src/v2/objectstore/object_store.go).
3. **The Argo workflow controller** stores the workflow's own archived logs and artifacts via its
   `artifactRepository.s3` block (see
   [`manifests/kustomize/third-party/argo/base/workflow-controller-configmap-patch.yaml`](../../manifests/kustomize/third-party/argo/base/workflow-controller-configmap-patch.yaml)).

All three read the same Kubernetes secret for credentials, named `mlpipeline-minio-artifact` by default
([`backend/src/v2/config/env.go`](../../backend/src/v2/config/env.go), constant `minioArtifactSecretName`). That secret
holds the B2 application key id under `accesskey` and the B2 application key under `secretkey`.

## B2 settings you will need

Read these from the Backblaze B2 console (or `b2 account get`) and keep them as environment values. Never hardcode the
application key.

| B2 convention env var      | Example value                                | Maps onto KFP setting                                  |
| -------------------------- | -------------------------------------------- | ------------------------------------------------------ |
| `B2_ENDPOINT`              | `https://s3.us-east-005.backblazeb2.com`     | `endpoint` / `ObjectStoreConfig.Host`                  |
| `B2_REGION`                | `us-east-005`                                | `region` / `ObjectStoreConfig.Region`                  |
| `B2_BUCKET_NAME`           | `your-kfp-artifacts`                         | `bucketName` / `ObjectStoreConfig.BucketName`          |
| `B2_APPLICATION_KEY_ID`    | (key id)                                     | secret key `accesskey` / `ObjectStoreConfig.AccessKey` |
| `B2_APPLICATION_KEY`       | (application key, secret)                    | secret key `secretkey` / `ObjectStoreConfig.SecretAccessKey` |

Use a B2 application key scoped to the single bucket KFP will use. The bucket should be private.

## Step 1: credentials secret

KFP and Argo both read credentials from the `mlpipeline-minio-artifact` secret. Set `accesskey` to your B2 application
key id and `secretkey` to your B2 application key. The default secret lives at
[`manifests/kustomize/third-party/seaweedfs/base/seaweedfs/mlpipeline-minio-artifact-secret.yaml`](../../manifests/kustomize/third-party/seaweedfs/base/seaweedfs/mlpipeline-minio-artifact-secret.yaml);
replace its placeholder values:

```yaml
kind: Secret
apiVersion: v1
metadata:
  name: mlpipeline-minio-artifact
stringData:
  accesskey: "${B2_APPLICATION_KEY_ID}"
  secretkey: "${B2_APPLICATION_KEY}"
```

The secret key names `accesskey` and `secretkey` are fixed: the driver/launcher path reads them as
`minioArtifactAccessKeyKey` and `minioArtifactSecretKeyKey` in
[`backend/src/v2/config/env.go`](../../backend/src/v2/config/env.go), the API server binds them via the
`OBJECTSTORECONFIG_ACCESSKEY` and `OBJECTSTORECONFIG_SECRETACCESSKEY` env vars in the API server deployment, and Argo
reads them through `accessKeySecret`/`secretKeySecret` in the workflow controller ConfigMap.

## Step 2: API server object-store config

The API server reads `ObjectStoreConfig.*`. The defaults live in
[`backend/src/apiserver/config/config.json`](../../backend/src/apiserver/config/config.json):

```json
"ObjectStoreConfig": {
  "AccessKey": "minio",
  "SecretAccessKey": "minio123",
  "BucketName": "mlpipeline",
  "PipelinePath": "pipelines"
}
```

In the deployed manifests these are overridden by `OBJECTSTORECONFIG_*` environment variables on the API server
container ([`manifests/kustomize/base/pipeline/ml-pipeline-apiserver-deployment.yaml`](../../manifests/kustomize/base/pipeline/ml-pipeline-apiserver-deployment.yaml)).
Viper maps an env var to a config key by replacing `.` with `_` and matching case-insensitively
([`backend/src/apiserver/main.go`](../../backend/src/apiserver/main.go): `SetEnvKeyReplacer(strings.NewReplacer(".", "_"))`
plus `AutomaticEnv()`), so `OBJECTSTORECONFIG_HOST` sets `ObjectStoreConfig.Host`, and so on.

Point the API server at B2 by setting these on the `ml-pipeline-api-server` container:

```yaml
env:
  - name: OBJECTSTORECONFIG_HOST
    value: "s3.us-east-005.backblazeb2.com"   # B2_ENDPOINT host, no scheme
  - name: OBJECTSTORECONFIG_PORT
    value: ""                                  # leave empty for the default HTTPS port
  - name: OBJECTSTORECONFIG_REGION
    value: "us-east-005"                       # B2_REGION
  - name: OBJECTSTORECONFIG_SECURE
    value: "true"                              # B2 requires TLS; default in config.json-less deploys is false
  - name: OBJECTSTORECONFIG_BUCKETNAME
    valueFrom:
      configMapKeyRef:
        name: pipeline-install-config
        key: bucketName                        # set to B2_BUCKET_NAME in pipeline-install-config
  - name: OBJECTSTORECONFIG_ACCESSKEY
    valueFrom:
      secretKeyRef:
        name: mlpipeline-minio-artifact
        key: accesskey
  - name: OBJECTSTORECONFIG_SECRETACCESSKEY
    valueFrom:
      secretKeyRef:
        name: mlpipeline-minio-artifact
        key: secretkey
```

How these flow into the S3 client (see `buildConfigFromEnvVars` and `newS3BucketClient` in
[`backend/src/apiserver/client_manager/client_manager.go`](../../backend/src/apiserver/client_manager/client_manager.go)):

- `Host` and `Port` are joined into the endpoint; the protocol prefix (`https://` vs `http://`) is chosen from
  `Secure`. Set `OBJECTSTORECONFIG_SECURE: "true"` so the client talks TLS to B2.
- `Region` defaults to `us-east-1` if empty, so set it explicitly to your B2 region.
- The client always uses path-style addressing: `newS3BucketClient` sets `o.UsePathStyle = true`. This is what B2
  expects, so no extra flag is needed.
- `AccessKey`/`SecretAccessKey` are applied as static credentials. Leaving both empty falls back to the default AWS
  credential chain; for B2 you set both from the secret.

Set the bucket name in
[`manifests/kustomize/base/installs/generic/pipeline-install-config.yaml`](../../manifests/kustomize/base/installs/generic/pipeline-install-config.yaml)
by changing `bucketName: mlpipeline` to your B2 bucket name. You can also set `defaultPipelineRoot` there to
`s3://your-kfp-artifacts/v2/artifacts` so v2 pipelines default to the B2 bucket via the `s3://` scheme (the comments in
that file already list `s3://your-bucket/path/to/artifacts` as a valid value).

## Step 3: driver/launcher provider config (`kfp-launcher`)

When a pipeline runs, the driver and launcher resolve credentials and endpoint from the `kfp-launcher` ConfigMap's
`providers` block, not from the API server env vars. The base ConfigMap is
[`manifests/kustomize/base/pipeline/kfp-launcher-configmap.yaml`](../../manifests/kustomize/base/pipeline/kfp-launcher-configmap.yaml);
add a `providers` key. The schema is defined by `S3ProviderConfig` / `S3ProviderDefault` / `S3Credentials` /
`S3SecretRef` in [`backend/src/v2/config/s3.go`](../../backend/src/v2/config/s3.go), keyed under `s3` by
`BucketProviders` in [`backend/src/v2/config/env.go`](../../backend/src/v2/config/env.go).

Because a pipeline root with the `s3://` scheme selects the `s3` provider (`ParseProviderFromPath` in
[`backend/src/v2/objectstore/config.go`](../../backend/src/v2/objectstore/config.go)), configure B2 under `s3.default`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kfp-launcher
data:
  defaultPipelineRoot: "s3://your-kfp-artifacts/v2/artifacts"
  providers: |
    s3:
      default:
        endpoint: "s3.us-east-005.backblazeb2.com"   # B2_ENDPOINT host
        region: "us-east-005"                          # B2_REGION
        disableSSL: false                              # keep TLS on for B2
        forcePathStyle: true                           # B2 uses path-style addressing
        credentials:
          fromEnv: false
          secretRef:
            secretName: mlpipeline-minio-artifact
            accessKeyKey: accesskey
            secretKeyKey: secretkey
```

Field-by-field, grounded in `S3ProviderDefault.ProvideSessionInfo` ([`backend/src/v2/config/s3.go`](../../backend/src/v2/config/s3.go)):

- `endpoint`: copied straight into the session params (`params["endpoint"]`). Path-style requests are sent to this host.
- `region`: copied into `params["region"]`. Required for B2; the comment in the struct marks it optional only for AWS.
- `disableSSL`: defaults to `false` when omitted. Leave it `false` (or omit) so the client uses HTTPS to B2.
- `forcePathStyle`: defaults to `true` when omitted (see both `s3.go` and `StructuredS3Params` in
  [`backend/src/v2/objectstore/config.go`](../../backend/src/v2/objectstore/config.go)). B2 needs path-style, so the
  default already works; setting it `true` explicitly documents intent.
- `credentials.secretRef`: names the secret and the two keys to read. These resolve to the same
  `mlpipeline-minio-artifact` secret from Step 1.
- `maxRetries`: optional, defaults to `5`.

These params are consumed by `createS3BucketSession` in
[`backend/src/v2/objectstore/object_store.go`](../../backend/src/v2/objectstore/object_store.go), which builds an AWS
SDK v2 S3 client with `config.WithRegion(params.Region)`, `o.UsePathStyle = params.ForcePathStyle`,
`o.EndpointOptions.DisableHTTPS = params.DisableSSL`, and `o.BaseEndpoint = <endpoint>` (the code only skips
`BaseEndpoint` for `s3.amazonaws.com` hosts, so a B2 endpoint is always applied). For B2 set the application key id and
application key on this same secret.

In multi-user mode each profile namespace gets its own `kfp-launcher` ConfigMap, so apply the `providers` block to every
namespace that runs pipelines, or set it as the default via `defaultPipelineRoot` in `pipeline-install-config`.

## Step 4: Argo workflow controller artifact repository

Argo writes its own archived logs and step artifacts using the `artifactRepository.s3` block in
[`manifests/kustomize/third-party/argo/base/workflow-controller-configmap-patch.yaml`](../../manifests/kustomize/third-party/argo/base/workflow-controller-configmap-patch.yaml).
Point it at B2:

```yaml
artifactRepository: |
  archiveLogs: true
  s3:
    endpoint: "s3.us-east-005.backblazeb2.com"   # B2_ENDPOINT host
    bucket: "your-kfp-artifacts"                   # B2_BUCKET_NAME
    insecure: false                                # B2 requires TLS
    keyFormat: "private-artifacts/{{workflow.namespace}}/{{workflow.name}}/{{workflow.creationTimestamp.Y}}/{{workflow.creationTimestamp.m}}/{{workflow.creationTimestamp.d}}/{{pod.name}}"
    accessKeySecret:
      name: mlpipeline-minio-artifact
      key: accesskey
    secretKeySecret:
      name: mlpipeline-minio-artifact
      key: secretkey
```

The shipped patch uses `insecure: true` against an in-cluster store; set `insecure: false` for B2 since B2's S3
endpoint is HTTPS-only. Argo's S3 client uses path-style by default, which matches B2. The `artifactRepository.s3`
schema also accepts an optional `region` field, owned by Argo rather than this repo; Argo derives a region for
non-AWS endpoints, so it can be omitted for B2.

## Step 5: apply and validate

After editing the manifests, rebuild and apply the Kustomize overlay you deploy from (see
[`manifests/kustomize/README.md`](../../manifests/kustomize/README.md)). Then restart the affected deployments so the new
config and env are picked up:

```bash
kubectl rollout restart deployment/ml-pipeline -n kubeflow
kubectl rollout restart deployment/workflow-controller -n kubeflow
```

Run a small pipeline and confirm:

- New objects appear under your B2 bucket (for example under the `v2/artifacts` and `private-artifacts/` prefixes).
- The KFP UI can open run artifacts (this exercises the API server -> B2 read path).
- Step pods complete without object-store errors in the driver/launcher logs (this exercises the `kfp-launcher`
  provider path).

### Verifying the S3 knobs out of cluster

Before deploying, you can sanity-check that B2 accepts the exact knobs KFP uses (endpoint, region, path-style, static
credentials) with a short S3 client check that mirrors `createS3BucketSession`:

```python
import os, boto3
from botocore.config import Config

s3 = boto3.client(
    "s3",
    endpoint_url=os.environ["B2_ENDPOINT"],            # https://s3.us-east-005.backblazeb2.com
    region_name=os.environ["B2_REGION"],               # us-east-005
    aws_access_key_id=os.environ["B2_APPLICATION_KEY_ID"],
    aws_secret_access_key=os.environ["B2_APPLICATION_KEY"],
    config=Config(s3={"addressing_style": "path"}),    # forcePathStyle / UsePathStyle = true
)

key = "kubeflow-pipelines-artifacts/healthcheck.txt"
s3.put_object(Bucket=os.environ["B2_BUCKET_NAME"], Key=key, Body=b"ok")
assert s3.get_object(Bucket=os.environ["B2_BUCKET_NAME"], Key=key)["Body"].read() == b"ok"
s3.delete_object(Bucket=os.environ["B2_BUCKET_NAME"], Key=key)
```

A successful put/get/delete confirms your endpoint, region, path-style, and key scope are correct before you wire them
into the cluster.

## Companion note: KServe model serving on B2

If you serve models with KServe from the same B2 bucket, KServe uses the storage-initializer with an `s3://` URI plus S3
endpoint and credentials supplied as environment variables on the storage-initializer container. This is a separate
component from KFP (it lives in `kserve/kserve`), but the credential and endpoint pattern is the same B2 S3 surface.

```yaml
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: my-model
spec:
  predictor:
    model:
      modelFormat:
        name: sklearn
      storageUri: "s3://your-kfp-artifacts/models/my-model"
```

The storage-initializer reads the S3 endpoint and credentials from environment. Provide them on the service account /
secret that KServe attaches to the storage-initializer, for example:

```yaml
S3_ENDPOINT: "s3.us-east-005.backblazeb2.com"   # B2_ENDPOINT host
S3_REGION: "us-east-005"                          # B2_REGION
S3_USE_HTTPS: "1"                                 # B2 requires TLS
S3_USE_VIRTUAL_BUCKET: "0"                        # path-style addressing for B2
AWS_ACCESS_KEY_ID: "${B2_APPLICATION_KEY_ID}"
AWS_SECRET_ACCESS_KEY: "${B2_APPLICATION_KEY}"
```

`AWS_ACCESS_KEY_ID` maps to your B2 application key id and `AWS_SECRET_ACCESS_KEY` to your B2 application key. With these
set, `storageUri: s3://...` resolves against B2. Consult the KServe docs for the exact secret/service-account wiring,
since that contract is owned by `kserve/kserve`, not by `kubeflow/pipelines`.

## Reference: source of each documented knob

| Knob                                          | Source file                                                                                          |
| --------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `ObjectStoreConfig.{AccessKey,SecretAccessKey,BucketName}` defaults | `backend/src/apiserver/config/config.json`                                      |
| `OBJECTSTORECONFIG_*` env binding             | `manifests/kustomize/base/pipeline/ml-pipeline-apiserver-deployment.yaml`, `backend/src/apiserver/main.go` |
| `ObjectStoreConfig.{Host,Port,Secure,Region}` -> S3 client, `UsePathStyle = true` | `backend/src/apiserver/client_manager/client_manager.go` (`buildConfigFromEnvVars`, `newS3BucketClient`) |
| `kfp-launcher` `providers.s3.default` schema (`endpoint`, `region`, `disableSSL`, `forcePathStyle`, `maxRetries`, `credentials.secretRef`) | `backend/src/v2/config/s3.go`, `backend/src/v2/config/env.go` |
| Session params (`forcePathStyle` default `true`) | `backend/src/v2/objectstore/config.go` (`StructuredS3Params`)                                     |
| S3 client build (`BaseEndpoint`, `WithRegion`, `UsePathStyle`, `DisableHTTPS`) | `backend/src/v2/objectstore/object_store.go` (`createS3BucketSession`)              |
| Credentials secret name + keys (`mlpipeline-minio-artifact`, `accesskey`, `secretkey`) | `backend/src/v2/config/env.go`, `manifests/kustomize/third-party/seaweedfs/base/seaweedfs/mlpipeline-minio-artifact-secret.yaml` |
| Argo `artifactRepository.s3` (`endpoint`, `bucket`, `insecure`, `accessKeySecret`, `secretKeySecret`) | `manifests/kustomize/third-party/argo/base/workflow-controller-configmap-patch.yaml` |
| `defaultPipelineRoot` accepts `s3://`         | `manifests/kustomize/base/installs/generic/pipeline-install-config.yaml`, `backend/src/v2/config/env.go` |
