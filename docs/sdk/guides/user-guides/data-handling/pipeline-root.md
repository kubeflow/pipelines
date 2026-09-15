# Pipeline Root Configuration Guide

Starting from [Kubeflow Pipelines SDK v2](https://kubeflow-pipelines.readthedocs.io/en/stable/) and Kubeflow Pipelines v2, Kubeflow Pipelines supports a new intermediate artifact repository feature: pipeline root.

## Before you start
This guide goes over the basic of how to use Kubeflow Pipelines [pipeline roots](../../concepts/pipeline-root.md).
This guide assumes that you already have Kubeflow Pipelines installed (see [Kubeflow Pipelines deployment
guide](../../operator-guides/installation/index.md) to deploy Kubeflow Pipelines) on a Kubernetes cluster.

## What is pipeline root?

[Pipeline root](../../concepts/pipeline-root.md) represents the path within an object store bucket where Kubeflow Pipelines stores a pipeline's artifacts.
This feature supports SeaweedFS, S3, GCS natively using [Go CDK](https://github.com/google/go-cloud).

Artifacts can be more accessible in S3 and GCS when integrating Kubeflow Pipelines with other systems.

## How to configure pipeline root authentication
### SeaweedFS
You don't need to pass the authentication for SeaweedFS.
Kubeflow Pipelines is configured with the authentication of the SeaweedFS instance deployed with itself.

#### GCS
If you want to specify the `pipeline root` to GCS, check [authentication-pipelines](https://googlecloudplatform.github.io/kubeflow-gke-docs/docs/pipelines/authentication-pipelines/)

#### S3
If you want to specify the `pipeline root` to S3, please choose one of the following options:

* Via [AWS IRSA](https://aws.amazon.com/blogs/containers/cross-account-iam-roles-for-kubernetes-service-accounts/)

* Via kfp sdk:
`dsl.get_pipeline_conf().add_op_transformer(aws.use_aws_secret('xxx', ‘xxx’, ‘xxx’))`

<!-- TODO: this function appears to be outdated -- update -->

**references**:
* [add-op-transformer](https://kubeflow-pipelines.readthedocs.io/en/stable/source/dsl.html#kfp.dsl.PipelineConf.add_op_transformer)
<!-- TODO: add_op_transformer no longer exists in this doc reference -->
* [use-aws-secret](https://kubeflow-pipelines.readthedocs.io/en/stable/source/kfp.extensions.html#kfp.aws.use_aws_secret)

## How to configure pipeline root

### Via ConfigMaps
The default Pipeline root at the Kubeflow pipeline deployment level can be changed by configuring the KFP Launcher configmap.

Instructions can be found [here](../../operator-guides/configure-object-store.md#kfp-launcher-object-store-configuration).

####  Via Building Pipelines
You can configure a pipeline root through the `kfp.dsl.pipeline` annotation when [building pipelines](https://www.kubeflow.org/docs/components/pipelines/legacy-v1/sdk/build-pipeline#build-your-pipeline).

####  Via Submitting a Pipeline through SDK
You can configure pipeline root via `pipeline_root` argument when you submit a Pipeline using one of the following:
* [create_run_from_pipeline_func](https://kubeflow-pipelines.readthedocs.io/en/stable/source/client.html#kfp.Client.create_run_from_pipeline_func)
* [create_run_from_pipeline_package](https://kubeflow-pipelines.readthedocs.io/en/stable/source/client.html#kfp.Client.create_run_from_pipeline_package)
* [run_pipeline](https://kubeflow-pipelines.readthedocs.io/en/stable/source/client.html#kfp.Client.run_pipeline).

####  Via Submitting a Pipeline Run through UI
You can configure a pipeline root via the `pipeline_root` run parameters when you submit a pipeline run in the UI
:::{image} ../../images/pipelines/v1/v2-compatible/pipelines-ui-pipelineroot.png
:alt: Configure pipeline root on the pipelines UI
:class: mt-3 mb-3 border rounded
:::
