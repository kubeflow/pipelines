# GCP inverse-proxy removal in 3.0

Starting with KFP 3.0, KFP no longer bundles the GCP inverse-proxy deployment or
publishes new `kfp-inverse-proxy-agent` images. This also applies to `master`
builds. Operators are responsible for configuring access to their installation.
This is a KFP packaging decision, not an announcement that Google has deprecated
the underlying proxy service.

The 2.x release branches, including 2.18, retain their existing manifests and
image publication behavior. Previously published inverse-proxy images remain
available for as long as older versioned images are retained; removal from 3.0
does not delete them. This is not a promise of indefinite retention or continued
maintenance of those images.

## Affected installations

The following Kustomize entrypoints previously included the inverse-proxy agent:

- `manifests/kustomize/env/gcp`
- `manifests/kustomize/env/dev`
- `manifests/kustomize/env/dev/postgresql`
- `manifests/kustomize/env/cert-manager/dev`
- `manifests/kustomize/sample`
- `test/manifests/dev`

These entrypoints no longer install the agent in 3.0. The Cloud SQL proxy, Istio,
OAuth2-Proxy, and KFP artifact proxies are separate integrations and are not
removed by this change.

## Replace access before removing the agent

If your installation uses an inverse-proxy URL, configure and verify replacement
access before retiring it. For local development, use the
[UI port-forward instructions](installation.md#accessing-the-kubeflow-pipelines-ui).
For shared or production access, configure an operator-managed gateway or ingress
with appropriate authentication, authorization, and TLS, or use the access
configuration provided by your Kubeflow distribution. Update bookmarks and SDK
client endpoint settings to the replacement URL.

Operators may instead manage a previously published agent image or a fork
independently, with their own deployment configuration and a compatible proxy
endpoint. Image availability alone does not guarantee that the external endpoint
is available. Existing SDK inverse-proxy connection and authentication
compatibility is retained for independently operated older deployments.

## Clean up existing resources

Applying the new manifests with `kubectl apply` does **not** automatically delete
objects omitted from them. Once replacement access works, remove the old bundled
objects if you are no longer using the agent:

```bash
# Use the namespace of your existing KFP installation instead of kubeflow if different.
kubectl delete -n kubeflow --ignore-not-found \
  deployment/proxy-agent \
  configmap/inverse-proxy-config \
  serviceaccount/proxy-agent-runner \
  role/proxy-agent-runner \
  rolebinding/proxy-agent-runner
```

Check that these objects belong to the old bundled deployment before deleting
them. If you use GitOps, remove the corresponding resources from your desired
state so reconciliation does not recreate them. If you are retaining an
independently managed agent, keep the resources it needs instead of running this
cleanup.

See the [removal decision](https://github.com/kubeflow/pipelines/issues/14613).
