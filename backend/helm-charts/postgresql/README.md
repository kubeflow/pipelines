# PostgreSQL Helm Deployment for Kubeflow Pipelines

This directory contains the Helm values file to deploy a PostgreSQL instance for Kubeflow Pipelines using the Bitnami PostgreSQL Chart.

## Prerequisites

- **Helm**: version 3.x installed (`helm version` should show v3).
- **Kubernetes** cluster with the `kubeflow` namespace created (or adjust commands to match your target namespace).

## Installation Steps

1. **Add the Bitnami chart repository**

   ```bash
   helm repo add bitnami https://charts.bitnami.com/bitnami
   ```

2. **Update local chart repository cache**

   ```bash
   helm repo update
   ```

3. **Install or upgrade the PostgreSQL release**

   ```bash
   helm upgrade --install kfp-postgres bitnami/postgresql \
     --namespace kubeflow \
     -f helm-charts/postgresql/values.yaml
   ```

   - `kfp-postgres` is the release name.
   - `bitnami/postgresql` refers to the community chart.
   - `-f values.yaml` applies your custom parameters.

## values.yaml Highlights

Customize the following key sections in `values.yaml` as needed:

```yaml
global:
  postgresql:
    postgresqlPassword: <YOUR_PASSWORD> # PostgreSQL superuser password
    postgresqlDatabase: kubeflow # Default database name
    postgresqlUsername: kubeflow # Default database user
persistence:
  enabled: true
  size: 10Gi # PVC size
  storageClass: standard # StorageClass for persistent volume
service:
  port: 5432 # PostgreSQL service port
```

## Verifying Deployment

1. **Check resources**

   ```bash
   kubectl get pods,svc,pvc -n kubeflow
   ```

2. **Port-forward and connect with psql**
   ```bash
   kubectl port-forward svc/kfp-postgres 5432:5432 -n kubeflow
   psql postgres://kubeflow:<YOUR_PASSWORD>@localhost:5432/kubeflow
   ```

## Upgrading & Uninstalling

- **Upgrade with new values**

  ```bash
  helm upgrade kfp-postgres bitnami/postgresql \
    --namespace kubeflow \
    -f helm-charts/postgresql/values.yaml
  ```

- **Uninstall release**
  ```bash
  helm uninstall kfp-postgres --namespace kubeflow
  ```

## Notes

- You can override individual settings via `--set` flags instead of editing `values.yaml`.
- To use a different namespace, modify the `--namespace` flag accordingly.
