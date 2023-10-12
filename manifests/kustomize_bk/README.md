# Install Kubeflow Pipelines Standalone using Kustomize Manifests

This folder contains [Kubeflow Pipelines Standalone](https://www.kubeflow.org/docs/components/pipelines/installation/standalone-deployment/) 
Kustomize manifests.

Kubeflow Pipelines Standalone is one option to install Kubeflow Pipelines. You can review all other options in
[Installation Options for Kubeflow Pipelines](https://www.kubeflow.org/docs/components/pipelines/installation/overview/).

## Install options for different envs

To install Kubeflow Pipelines Standalone, follow [Kubeflow Pipelines Standalone Deployment documentation](https://www.kubeflow.org/docs/components/pipelines/installation/standalone-deployment/).

There are environment specific installation instructions not covered in the official deployment documentation, they are listed below.

### (env/platform-agnostic) install on any Kubernetes cluster

Install:

```bash
KFP_ENV=platform-agnostic
kubectl apply -k cluster-scoped-resources/
kubectl wait crd/applications.app.k8s.io --for condition=established --timeout=60s
kubectl apply -k "env/${KFP_ENV}/"
kubectl wait pods -l application-crd-id=kubeflow-pipelines -n kubeflow --for condition=Ready --timeout=1800s
kubectl port-forward -n kubeflow svc/ml-pipeline-ui 8080:80
```

Now you can access Kubeflow Pipelines UI in your browser by <http://localhost:8080>.

Customize:

There are two variations for platform-agnostic that uses different [argo workflow executors](https://argoproj.github.io/argo-workflows/workflow-executors/):

* env/platform-agnostic-emissary
* env/platform-agnostic-pns

You can install them by changing `KFP_ENV` in above instructions to the variation you want.

Data:

Application data are persisted in in-cluster PersistentVolumeClaim storage.

### (env/gcp) install on Google Cloud with Cloud Storage and Cloud SQL

Cloud Storage and Cloud SQL are better for operating a production cluster.

Refer to [Google Cloud Instructions](sample/README.md) for installation.

### (env/aws) install on AWS with S3 and RDS MySQL

S3 and RDS MySQL are better for operating a production cluster.

Refer to [AWS Instructions](env/aws/README.md) for installation.

Note: Community maintains a different opinionated installation manifests for AWS, refer to [e2fyi/kubeflow-aws](https://github.com/e2fyi/kubeflow-aws/tree/master/pipelines).

## Uninstall

If the installation is based on CloudSQL/GCS, after the uninstall, the data is still there,
reinstall a newer version can reuse the data.

```bash
### 1. namespace scoped
# Depends on how you installed it:
kubectl kustomize env/platform-agnostic | kubectl delete -f -
# or
kubectl kustomize env/dev | kubectl delete -f -
# or
kubectl kustomize env/gcp | kubectl delete -f -
# or
kubectl delete applications/pipeline -n kubeflow

### 2. cluster scoped
kubectl delete -k cluster-scoped-resources/
```

## Folder Structure

### Overview

* User facing manifest entrypoints are `cluster-scoped-resources` package and `env/<env-name>` package.
  * `cluster-scoped-resources` should collect all cluster-scoped resources.
  * `env/<env-name>` should collect env specific namespace-scoped resources.
  * Note, for multi-user envs, they already included cluster-scoped resources.
* KFP core components live in `base/<component-name>` folders.
  * If a component requires cluster-scoped resources, it should have a folder inside named `cluster-scoped` with related resources, but note that `base/<component-name>/kustomization.yaml` shouldn't include the `cluster-scoped` folder. `cluster-scoped` folders should be collected by top level `cluster-scoped-resources` folder.
* KFP core installations are in `base/installs/<install-type>`, they only include the core KFP components, not third party ones.
* Third party components live in `third-party/<component-name>` folders.

### For direct deployments

Env specific overlays live in `env/<env-name>` folders, they compose above components to get ready for directly deploying.

### For downstream consumers

Please compose `base/installs/<install-type>` and third party dependencies based on your own requirements.

### Rationale

Constraints for namespaced installation we need to comply with (that drove above structure):

* CRDs must be applied separately, because if we apply CRs in the same `kubectl apply` command, the CRD may not have been accepted by k8s api server (e.g. Application CRD).
* [A Kubeflow 1.0 constraint](https://github.com/kubeflow/pipelines/issues/2884#issuecomment-577158715) is that we should separate cluster scoped resources from namespace scoped resources, because sometimes different roles are required to deploy them. Cluster scoped resources usually need a cluster admin role, while namespaced resources can be deployed by individual teams managing a namespace.
