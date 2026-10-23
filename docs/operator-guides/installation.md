# Installation

:::{note}
Upgrading an existing installation to 2.18? The
[2.18 upgrade guide](upgrade-2.18.md) explains automatic schedule adoption and UI
signing-key initialization during the normal rollout, compatibility checks for
custom configurations, and troubleshooting.
:::

As an alternative to deploying Kubeflow Pipelines (KFP) as part of the [Community distribution](https://www.kubeflow.org/docs/started/installing-kubeflow),
you can also choose to deploy only Kubeflow Pipelines standalone.

You should be familiar with [Kubernetes](https://kubernetes.io/docs/home/),
[kubectl](https://kubernetes.io/docs/reference/kubectl/overview/), and [kustomize](https://kustomize.io/).

> **Note:** Replace `2.17.0` with the Kubeflow Pipelines release you want to
> install. See the [releases page](https://github.com/kubeflow/pipelines/releases)
> for available versions.

## Upgrading to 2.18: HTTP artifact access

With namespace artifact proxying enabled, HTTP and HTTPS artifact reads retain
namespace receiver routing and its credential placement. Keep `HTTP_BASE_URL`,
HTTP authentication and network access on the process that fetches the object;
direct-serving installations use the serving UI. This upgrade does not require
moving tenant HTTP credentials into the shared UI.

Updated receivers apply the retained destination and redirect protections. Old
receivers keep their older behavior until the normal rollout replaces them.
Verify each affected receiver's image and a real preview/download after rollout.
Custom HTTP authentication and mounts must survive profile reconciliation through
the installation's provisioning configuration; stock profile generation does not
provision arbitrary tenant HTTP credentials. See
[HTTP artifact configuration](server-config.md#http-artifact-migration-for-218)
and [artifact compatibility](upgrade-2.18.md#artifact-compatibility-and-retained-exposure).

## Deploying Kubeflow Pipelines

### 1. Deploy the Kubeflow Pipelines development flavor standalone and non-production for first experiments:

```bash
export PIPELINE_VERSION=2.17.0
kubectl apply -k "github.com/kubeflow/pipelines/manifests/kustomize/cluster-scoped-resources?ref=$PIPELINE_VERSION"
kubectl wait --for condition=established --timeout=60s crd/applications.app.k8s.io
kubectl apply -k "github.com/kubeflow/pipelines/manifests/kustomize/env/dev?ref=$PIPELINE_VERSION"
```

### 2. Deploy Kubeflow Pipelines as part of the community distribution for production ready environments:

This includes proper RBAC and security settings for multi-tenant enterprise environments.

- [Install just KFP](https://github.com/kubeflow/manifests#kubeflow-pipelines)
- [Reproducible GHA workflow](https://github.com/kubeflow/manifests/blob/master/.github/workflows/pipeline_test.yaml)
- [Install with a single command](https://github.com/kubeflow/manifests#install-with-a-single-command)

### 3. Deploying Kubeflow Pipelines in Kubernetes Native API Mode for non-production experiments:

Kubeflow Pipelines can be deployed in Kubernetes native API mode, which stores pipeline definitions as Kubernetes Custom Resources instead of using external database storage. This mode provides better integration with Kubernetes native tooling and GitOps workflows.

```bash
export PIPELINE_VERSION=2.17.0
kubectl apply -k "github.com/kubeflow/pipelines/manifests/kustomize/cluster-scoped-resources?ref=$PIPELINE_VERSION"
kubectl wait --for condition=established --timeout=60s crd/applications.app.k8s.io
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.18.2/cert-manager.yaml
kubectl wait --for=condition=Ready pod -l app.kubernetes.io/instance=cert-manager -n cert-manager --timeout=300s
kubectl apply -k "github.com/kubeflow/pipelines/manifests/kustomize/env/cert-manager/platform-agnostic-k8s-native?ref=$PIPELINE_VERSION"
# Alternatively, for multi-user environments with multiple teams or users requiring isolation and RBAC controls on who can access which pipelines (still not production-ready like the community distribution), you can use the multi-user Kubernetes native mode (requires Istio to be installed, so we strongly recommend using the community distribution instead):
# kubectl apply -k "github.com/kubeflow/pipelines/manifests/kustomize/env/cert-manager/platform-agnostic-multi-user-k8s-native?ref=$PIPELINE_VERSION"
```

> 💡 **Migration Note**: If you are upgrading from a previous version not deployed in Kubernetes native API mode, consider leveraging the [migration script](https://github.com/kubeflow/pipelines/tree/master/tools/k8s-native) to export all existing pipelines and pipeline versions as Kubernetes manifests which can be applied after upgrading Kubeflow Pipelines.

## Deploying Kubeflow Pipelines with Pod-to-Pod TLS Enabled

Kubeflow Pipelines can be deployed with pod-to-pod TLS enabled. The API server serves over TLS, and all connecting deployments are mounted with CA certificates. This mode provides enhanced security.

Deploy KFP on a KinD cluster with pod-to-pod TLS enabled using the Makefile target [here](https://github.com/kubeflow/pipelines/blob/master/backend/Makefile). The corresponding manifests can be manually accessed [here](https://github.com/kubeflow/pipelines/tree/master/manifests/kustomize/env/cert-manager/platform-agnostic-standalone-tls).

## Accessing the Kubeflow Pipelines UI

The Kubeflow Pipelines deployment requires approximately 3 minutes to complete.

### Standalone / non-production

```bash
kubectl port-forward -n kubeflow svc/ml-pipeline-ui 8080:80
```

Open http://localhost:8080 on your browser to see the Kubeflow Pipelines UI.

### Community Distribution

[Port-Forward / LoadBalancer / Ingress](https://github.com/kubeflow/manifests#port-forward)
