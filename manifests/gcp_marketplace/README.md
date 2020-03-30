# Kubeflow Pipelines for GKE Marketplace

Kubeflow Pipelines can be installed using either of the following approaches:

* [Using the Google Cloud Platform Console](#using-install-platform-console)

* [Using the command line](#using-install-command-line)

## <a name="using-install-platform-console"></a>Using the Google Cloud Platform Marketplace

Get up and running with a few clicks! Install this Kubeflow Pipelines app to a
Google Kubernetes Engine cluster using Google Cloud Marketplace. Follow the
[on-screen instructions](https://console.cloud.google.com/marketplace/details/google-cloud-ai-platform/kubeflow-pipelines) and [guide](https://github.com/kubeflow/pipelines/blob/master/manifests/gcp_marketplace/guide.md).


## <a name="using-install-command-line"></a>Using the command line

We prefer you use Google Cloud Platform Marketplace UI to deploy the application.
If you want to know how , please follow the [guide](https://github.com/kubeflow/pipelines/blob/master/manifests/gcp_marketplace/cli.md). It's not target for production usage. The tool "mpdev" is for Kubeflow Pipeline developers. We will provide better command line experiences in 2020 Q2/Q3. Please check [Standalone CLI](https://www.kubeflow.org/docs/pipelines/installation/standalone-deployment/) for now on how to install via commandline.

## Developement guide

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

```

```