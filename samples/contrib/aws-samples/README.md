# Sample AWS SageMaker Kubeflow Pipelines 

This folder contains many example pipelines which use [AWS SageMaker Components for KFP](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker). The following sections explain the setup needed to run these pipelines. Once you are done with the setup, [simple_train_pipeline](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/simple_train_pipeline) is a good place to start if you have never used these components before.



## Prerequisites 

 You need a cluster with Kubeflow Pipelines installed and permissions configured. Kubeflow Pipelines offers two installation options. Select the option that applies to your use case:

<details><summary>Full Kubeflow on AWS Deployment</summary>
<p>

1. To use other Kubeflow components in addition to Kubeflow Pipelines, install the [AWS Distribution of Kubeflow.](https://awslabs.github.io/kubeflow-manifests/docs/deployment/).

2. Configure permissions to access SageMaker services by following the guide on [Kubeflow on AWS documentation](https://awslabs.github.io/kubeflow-manifests/docs/amazon-sagemaker-integration/sagemaker-components-for-kubeflow-pipelines/)
</p>
</details>

<details><summary>Standalone Kubeflow Pipelines Deployment</summary>
<p>

1. Install Kubeflow Pipelines standalone by following the documentation on [SageMaker developer guide](https://docs.aws.amazon.com/sagemaker/latest/dg/setup.html#kubeflow-pipelines-standalone). 

2. Configure permissions to access SageMaker services by following the guide on [SageMaker developer guide](https://docs.aws.amazon.com/sagemaker/latest/dg/setup.html#configure-permissions-for-pipeline)
</p>
</details>


## Inputs to the pipeline

### SageMaker execution role
**Note:** Ignore this section if you plan to run [titanic-survival-prediction](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/titanic-survival-prediction) example

This role is used by SageMaker jobs created by the KFP to access the S3 buckets and other AWS resources.
Run these commands to create the sagemaker-execution-role.   
Note down the Role ARN. You need to give this Role ARN as input in pipeline.

```
TRUST="{ \"Version\": \"2012-10-17\", \"Statement\": [ { \"Effect\": \"Allow\", \"Principal\": { \"Service\": \"sagemaker.amazonaws.com\" }, \"Action\": \"sts:AssumeRole\" } ] }"
aws iam create-role --role-name kfp-example-sagemaker-execution-role --assume-role-policy-document "$TRUST"
aws iam attach-role-policy --role-name kfp-example-sagemaker-execution-role --policy-arn arn:aws:iam::aws:policy/AmazonSageMakerFullAccess
aws iam attach-role-policy --role-name kfp-example-sagemaker-execution-role --policy-arn arn:aws:iam::aws:policy/AmazonS3FullAccess
aws iam get-role --role-name kfp-example-sagemaker-execution-role --output text --query 'Role.Arn'

# note down the Role ARN or export to env variable.
export SAGEMAKER_EXECUTION_ROLE_ARN=$(aws iam get-role --role-name kfp-example-sagemaker-execution-role --output text --query 'Role.Arn')
echo $SAGEMAKER_EXECUTION_ROLE_ARN
```