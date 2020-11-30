# Kubeflow Pipelines for GKE Marketplace

## <a name="using-install-platform-console"></a>Using the Google Cloud Platform Marketplace

Get up and running with a few clicks! Install this Kubeflow Pipelines app to a
Google Kubernetes Engine cluster using Google Cloud Marketplace. Follow the
[on-screen instructions](https://console.cloud.google.com/marketplace/details/google-cloud-ai-platform/kubeflow-pipelines) and [Google Cloud AI Platform Pipelines documentation](https://cloud.google.com/ai-platform/pipelines/docs).

## <a name="using-install-command-line"></a>Using the command line

For users, please refer to [Kubeflow Pipelines Standalone](https://www.kubeflow.org/docs/pipelines/installation/standalone-deployment/) on how to install via commandline using kustomize manifests. The installation is almost the same as AI Platform Pipelines.

Refer to the [Installation Options for Kubeflow Pipelines](https://www.kubeflow.org/docs/pipelines/installation/overview/) doc for all installation options and their feature comparison.

## Developement guide

Only for developing AI Platform Pipelines purpose, we have a [CLI installation guide for AI Platform Pipelines](cli.md). It's not suitable for end users. The tool "mpdev" is for Kubeflow Pipeline developers.

This section details how to test your changes before submit codes.

1. Code changes and locally committed

2. Build

```
gcloud builds submit --config=.cloudbuild.yaml --substitutions=COMMIT_SHA="$(git rev-parse HEAD)" --project=ml-pipeline-test
```

`gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/` contains the binaries.

3. Auto-test (Install & Uninstall)

MM_VER is major minor version parsed from VERSION file which is on major.minor.patch version format.

```
MM_VER=$(cat VERSION | sed -e "s#[^0-9]*\([0-9]*\)[.]\([0-9]*\)[.]\([0-9]*\)#\1.\2#")
gcloud builds submit --config=test/cloudbuild/mkp_verify.yaml --substitutions=COMMIT_SHA="$(git rev-parse HEAD)",_DEPLOYER_VERSION=$MM_VER --project=ml-pipeline-test
```

4. Manual-test (Install with advanced parameters and don't uninstall)

Make sure your kubectl can connect to a target test cluster.

```shell
APP_INSTANCE_NAME=<yours>
NAMESPACE=<yours> # Make sure you already created the namespace
MANAGEDSTORAGE=true # True means use CloudSQL + Minio-GCS; False means use in-cluster PVC + MySQL.
CLOUDSQL=<yours> # Format like project_id:zone:cloudsql_instance_name
PROJECTID=<yours> # This field will be removed after Marketplace can pass in the project ID
mpdev install  --deployer=gcr.io/ml-pipeline-test/hosted/$(git rev-parse HEAD)/deployer:$MM_VER \
    --parameters='{"name": "'$APP_INSTANCE_NAME'", "namespace": "'$NAMESPACE'", "managedstorage.enabled": '$MANAGEDSTORAGE', "managedstorage.cloudsqlInstanceConnectionName": "'$CLOUDSQL'"}'
```
