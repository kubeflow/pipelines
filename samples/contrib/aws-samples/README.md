# Sample AWS SageMaker Kubeflow Pipelines 

This folder contains many example pipelines which use [AWS SageMaker Components for KFP](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker). The following sections explain the setup needed to run these pipelines. Once you are done with the setup, [simple_train_pipeline](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/simple_train_pipeline) is a good place to start if you have never used these components before.



## Prerequisites 

 You need a cluster with Kubeflow Pipelines installed and permissions configured, Kubeflow Pipelines offers two installation options.

<details><summary>Full Kubeflow on AWS Deployment</summary>
<p>

1. Install Kubeflow Pipelines along with the rest of the Kubeflow Components. [Install Kubeflow on AWS cluster](https://awslabs.github.io/kubeflow-manifests/docs/deployment/).

2. Configure Kubeflow Pipelines permissions on [AWS Deployment](https://awslabs.github.io/kubeflow-manifests/docs/amazon-sagemaker-integration/sagemaker-components-for-kubeflow-pipelines/)
</p>
</details>

<details><summary>Standalone Kubeflow Pipelines Deployment</summary>
<p>

1. [Install only Kubeflow Pipelines](https://docs.aws.amazon.com/sagemaker/latest/dg/setup.html#kubeflow-pipelines-standalone). 

2. Configure Kubeflow Pipelines permissions on [Standalone Deployment](https://docs.aws.amazon.com/sagemaker/latest/dg/setup.html#configure-permissions-for-pipeline)
</p>
</details>
