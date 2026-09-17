# Kubeflow Pipelines Kustomize Manifests

Kubeflow Pipelines can be installed standalone and as part of the [community distribution](https://github.com/kubeflow/community-distribution).
[Installation Options for Kubeflow Pipelines](https://www.kubeflow.org/docs/components/pipelines/operator-guides/installation/).

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
Allowlisting does not grant bucket access or configure credentials.

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

The controller must then reconcile the new environment into each enabled
profile's `ml-pipeline-ui-artifact` Deployment. Confirm those pod templates contain
the updated allowlist and their rollouts complete before testing a custom
artifact preview or download. Standalone installations only need the UI restart.
Preserve these settings in the manifests used for future upgrades.

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
