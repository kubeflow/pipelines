# Training Job Samples

The samples in this directory demonstrate how to create and monitor Training jobs on SageMaker in a Kubeflow Pipelines workflow.             

## Prerequisites    

1. Follow the instructions in the [getting started section](../README.md#getting-started) to setup the required configuration to run the component.
2. Install the following tools on your local machine or an EC2 instance:
    - [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) – A command line tool for interacting with AWS services.
    - [python 3.8+](https://www.python.org/downloads/) - A programming language used for automated installation scripts.
    - [pip](https://pip.pypa.io/en/stable/installation/) - A package installer for python.


Next, we create an S3 bucket and IAM role for SageMaker.

### S3 Bucket
To train a model with SageMaker, we need an S3 bucket to store the dataset and artifacts from the training process. Run the following commands to create an S3 bucket. Specify the value for `SAGEMAKER_REGION` as the region you want to create your SageMaker resources. For ease of use in the samples (using the default values of the pipeline), we suggest using `us-east-1` as the region.

```
export SAGEMAKER_REGION=us-east-1
export S3_BUCKET_NAME="data-bucket-${SAGEMAKER_REGION}-$RANDOM"

if [[ $SAGEMAKER_REGION == "us-east-1" ]]; then
    aws s3api create-bucket --bucket ${S3_BUCKET_NAME} --region ${SAGEMAKER_REGION}
else
    aws s3api create-bucket --bucket ${S3_BUCKET_NAME} --region ${SAGEMAKER_REGION} \
    --create-bucket-configuration LocationConstraint=${SAGEMAKER_REGION}
fi

echo ${S3_BUCKET_NAME}
```
Note down your S3 bucket name which will be used in the samples.

### SageMaker execution IAM role
The SageMaker training job needs an IAM role to access Amazon S3 and SageMaker. Run the following commands to create a SageMaker execution IAM role that is used by SageMaker to access AWS resources:

```
export SAGEMAKER_EXECUTION_ROLE_NAME="sagemaker-execution-role-$RANDOM"

TRUST="{ \"Version\": \"2012-10-17\", \"Statement\": [ { \"Effect\": \"Allow\", \"Principal\": { \"Service\": \"sagemaker.amazonaws.com\" }, \"Action\": \"sts:AssumeRole\" } ] }"
aws iam create-role --role-name ${SAGEMAKER_EXECUTION_ROLE_NAME} --assume-role-policy-document "$TRUST"
aws iam attach-role-policy --role-name ${SAGEMAKER_EXECUTION_ROLE_NAME} --policy-arn arn:aws:iam::aws:policy/AmazonSageMakerFullAccess
aws iam attach-role-policy --role-name ${SAGEMAKER_EXECUTION_ROLE_NAME} --policy-arn arn:aws:iam::aws:policy/AmazonS3FullAccess

export SAGEMAKER_EXECUTION_ROLE_ARN=$(aws iam get-role --role-name ${SAGEMAKER_EXECUTION_ROLE_NAME} --output text --query 'Role.Arn')

echo $SAGEMAKER_EXECUTION_ROLE_ARN
```
Note down the execution role ARN to use in samples.

## Creating your first Training Job

Head over to the individual sample directories to run your training jobs. The simplest example to start with is the [K-Means MNIST training](./mnist-kmeans-training/).