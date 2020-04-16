# Install Kubeflow Pipelines

This folder contains Kubeflow Pipelines Kustomize manifests for a light weight
deployment. You can follow the instruction and deploy Kubeflow Pipelines in an
existing cluster.

To install Kubeflow Pipelines, you have several options.
- Via [GCP AI Platform UI](http://console.cloud.google.com/ai-platform/pipelines).
- Via an upcoming commandline tool.
- Via Kubectl with Kustomize, it's detailed here.
  - Community maintains a repo [here](https://github.com/e2fyi/kubeflow-aws/tree/master/pipelines) for AWS.

## Install via Kustomize

Deploy latest version of Kubeflow Pipelines.

It uses following default settings.
- image: latest released images
- namespace: kubeflow
- application name: pipeline

### Option-1 Install it to any K8s cluster
```
kubectl apply -k cluster-scoped-resources/
kubectl wait crd/applications.app.k8s.io --for condition=established --timeout=60s
kubectl apply -k env/dev/
kubectl wait applications/pipeline -n kubeflow --for condition=Ready --timeout=1800s
kubectl port-forward -n kubeflow svc/ml-pipeline-ui 8080:80
```
Now you can access it via localhost:8080

### Option-2 Install it to GCP with in-cluster PersistentVolumeClaim storage
```
kubectl apply -k cluster-scoped-resources/
kubectl wait crd/applications.app.k8s.io --for condition=established --timeout=60s
kubectl apply -k env/gcp-dev/
kubectl wait applications/pipeline -n kubeflow --for condition=Ready --timeout=1800s
kubectl describe configmap inverse-proxy-config -n kubeflow | grep googleusercontent.com
```

### Option-3 Install it to GCP with CloudSQL & GCS-Minio managed storage
It's suggested for production usage as data always is not stored to the cluster but CloudSQL & GCS.

It's to be fixed soon in another PR.

### Change deploy namespace

To deploy Kubeflow Pipelines in namespace FOO,

-   Edit [dev/kustomization.yaml](env/dev/kustomization.yaml) or
    [gcp/kustomization.yaml](env/gcp/kustomization.yaml) namespace section to
    FOO
-   Then run

```
kubectl kustomize cluster-scoped-resources | kubectl apply -f -
# then
kubectl kustomize env/dev | kubectl apply -f -
# or
kubectl kustomize env/gcp | kubectl apply -f -
```

## Upgrade
Note - Do **NOT** follow these instructions if you are upgrading KFP in a
[proper Kubeflow installation](https://www.kubeflow.org/docs/started/getting-started/).

If you have already deployed a standalone KFP installation of version prior to
0.2.5 and you want to upgrade it, make sure the following resources do not
exist: `metadata-deployment`, `metadata-service`.
```
kubectl -n <KFP_NAMESPACE> get deployments | grep metadata-deployment
kubectl -n <KFP_NAMESPACE> get service | grep metadata-service
```

If they exist, you can delete them by running the following commands:
```
kubectl -n <KFP_NAMESPACE> delete deployment metadata-deployment
kubectl -n <KFP_NAMESPACE> delete service metadata-service
```

## Uninstall

```
### 1. namespace scoped
# Depends on how you installed it:
kubectl kustomize env/dev | kubectl delete -f -
# or
kubectl kustomize env/gcp-dev | kubectl delete -f -
# or
kubectl kustomize env/gcp | kubectl delete -f -
# or
kubectl delete applications/pipeline -n kubeflow

### 2. cluster scoped
kubectl kustomize cluster-scoped-resources | kubectl delete -f -
```

## Troubleshooting

### Permission error installing Kubeflow Pipelines to a cluster

Run

```
kubectl create clusterrolebinding your-binding --clusterrole=cluster-admin --user=[your-user-name]
```

### Samples requires "user-gcp-sa" secret

If sample code requires a "user-gcp-sa" secret, you could create one by

-   First download the GCE VM service account token
    [Document](https://cloud.google.com/iam/docs/creating-managing-service-account-keys#creating_service_account_keys)

```
gcloud iam service-accounts keys create application_default_credentials.json \
  --iam-account [SA-NAME]@[PROJECT-ID].iam.gserviceaccount.com
```

-   Run

```
kubectl create secret -n [your-namespace] generic user-gcp-sa --from-file=user-gcp-sa.json=application_default_credentials.json`
```
