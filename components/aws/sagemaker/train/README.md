# SageMaker Training Kubeflow Pipelines component
## Summary
Component to submit SageMaker Training jobs directly from a Kubeflow Pipelines workflow.
https://docs.aws.amazon.com/sagemaker/latest/dg/how-it-works-training.html

# Details

## Intended Use
For model training using AWS SageMaker.

## Runtime Arguments
Argument        | Description                 | Optional   | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------| :----------| :----------     | :----------|
region | The region where the cluster launches | No | String | | |
endpoint_url | The endpoint URL for the private link VPC endpoint. | Yes | String | | |
job_name | The name of the Ground Truth job. Must be unique within the same AWS account and AWS region | Yes | String | | LabelingJob-[datetime]-[random id]|
role | The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf | No | String | | |
image | The registry path of the Docker image that contains the training algorithm | Yes | Yes | String | | |
algorithm_name | The name of the algorithm resource to use for the hyperparameter tuning job; only specify this parameter if training image is not specified | Yes | Yes | String | | |
metric_definitions | The dictionary of name-regex pairs specify the metrics that the algorithm emits | Yes | Yes | Dict | | {} |
put_mode | The input mode that the algorithm supports | Yes | No | String | File, Pipe | File |
hyperparameters  | |  |  | | |
channels | A list of dicts specifying the input channels (at least one); refer to [documentation](https://github.com/awsdocs/amazon-sagemaker-developer-guide/blob/master/doc_source/API_Channel.md) for parameters | No | No | List of Dicts | | |
instance_type | The ML compute instance type | Yes | No | String | ml.m4.xlarge, ml.m4.2xlarge, ml.m4.4xlarge, ml.m4.10xlarge, ml.m4.16xlarge, ml.m5.large, ml.m5.xlarge, ml.m5.2xlarge, ml.m5.4xlarge, ml.m5.12xlarge, ml.m5.24xlarge, ml.c4.xlarge, ml.c4.2xlarge, ml.c4.4xlarge, ml.c4.8xlarge, ml.p2.xlarge, ml.p2.8xlarge, ml.p2.16xlarge, ml.p3.2xlarge, ml.p3.8xlarge, ml.p3.16xlarge, ml.c5.xlarge, ml.c5.2xlarge, ml.c5.4xlarge, ml.c5.9xlarge, ml.c5.18xlarge | ml.m4.xlarge |
instance_count | The number of ML compute instances to use in each training job | Yes | Yes | Int | ≥ 1 | 1 |
volume_size | The size of the ML storage volume that you want to provision in GB | Yes | Yes | Int | ≥ 1 | 1 |
resource_encryption_key | The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s) | Yes | Yes | String | | |
max_run_time | The maximum run time in seconds per training job | Yes | Yes | Int | ≤ 432000 (5 days) | 86400 (1 day) |
model_artifact_path | | No | String | | |
output_encryption_key | The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts | Yes | Yes | String | | |
vpc_security_group_ids | The VPC security group IDs, in the form sg-xxxxxxxx | Yes | Yes | String | | |
vpc_subnets | The ID of the subnets in the VPC to which you want to connect your hpo job | Yes | Yes | String | | |
network_isolation | Isolates the training container if true | Yes | No | Boolean | False, True | True |
traffic_encryption | Encrypts all communications between ML compute instances in distributed training if true | Yes | No | Boolean | False, True | False |
spot_instance | Use managed spot training if true | Yes | No | Boolean | False, True | False |
max_wait_time | The maximum time in seconds you are willing to wait for a managed spot training job to complete | Yes | Yes | Int | ≤ 432000 (5 days) | 86400 (1 day) |
checkpoint_config | Dictionary of information about the output location for managed spot training checkpoint data | Yes | Yes | Dict | | {} |
tags | Key-value pairs to categorize AWS resources | Yes | Dict | | {} |

Notes :
* The parameters, data_location_1 through 8, is intended to be used for inputting the S3 URI outputs from previous steps in the pipeline, for example, from a Ground Truth labeling job. Otherwise, the S3 data location can be specified directly in the channels parameter.


## Outputs
Name | Description
:--- | :----------


# Samples

1. Get and store data in S3 buckets
2. Prepare an IAM roles with permissions to run SageMaker jobs
3. Add 'aws-secret' to your kubeflow namespace
4. Compile the pipeline:
```bash
dsl-compile --py training-pipeline.py --output training-pipeline.tar.gz
```
5. In the Kubeflow UI, upload the compiled pipeline specification (the .tar.gz file) and create a new run. Update the role_arn and the data paths, and optionally any other run parameters.
6. Once the pipeline completes, you can see the outputs under 'Output parameters' in the HPO component's Input/Output section.

# Example code
```buildoutcfg
#!/usr/bin/env python3

import kfp
from kfp import components
from kfp import dsl
from kfp.aws import use_aws_secret

sagemaker_train_op = components.load_component_from_file('../../../../components/aws/sagemaker/train/component.yaml')

@dsl.pipeline(
    name='Training pipeline',
    description='SageMaker training job test'
)
def training(
        region='us-east-1',
        endpoint_url='delete this string',
        image='382416733822.dkr.ecr.us-east-1.amazonaws.com/kmeans:1',
        training_input_mode='File',
        hyperparameters='{"k": "10", "feature_dim": "784"}',
        channels='[                                                               \
                    {                                                             \
                      "ChannelName": "train",                                     \
                      "DataSource": {                                             \
                        "S3DataSource": {                                         \
                          "S3Uri": "s3://m-kfp-mnist/mnist_kmeans_example/data",  \
                          "S3DataType": "S3Prefix",                               \
                          "S3DataDistributionType": "FullyReplicated"             \
                        }                                                         \
                      },                                                          \
                      "ContentType": "",                                          \
                      "CompressionType": "None",                                  \
                      "RecordWrapperType": "None",                                \
                      "InputMode": "File"                                         \
                    }                                                             \
                  ]',
        instance_type='ml.p2.xlarge',
        instance_count='1',
        volume_size='50',
        max_run_time='3600',
        model_artifact_path='s3://m-kfp-mnist/mnist_kmeans_example/output',
        output_encryption_key='',
        network_isolation='True',
        traffic_encryption='False',
        spot_instance='False',
        max_wait_time='3600',
        checkpoint_config='{}',
        role=''
        ):
    training = sagemaker_train_op(
        region=region,
        endpoint_url=endpoint_url,
        image=image,
        training_input_mode=training_input_mode,
        hyperparameters=hyperparameters,
        channels=channels,
        instance_type=instance_type,
        instance_count=instance_count,
        volume_size=volume_size,
        max_run_time=max_run_time,
        model_artifact_path=model_artifact_path,
        output_encryption_key=output_encryption_key,
        network_isolation=network_isolation,
        traffic_encryption=traffic_encryption,
        spot_instance=spot_instance,
        max_wait_time=max_wait_time,
        checkpoint_config=checkpoint_config,
        role=role,
    ).apply(use_aws_secret('aws-secret', 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'))

if __name__ == '__main__':
    kfp.compiler.Compiler().compile(training, __file__ + '.zip')

```

# Resources
* [Using Amazon built-in algorithms](https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-algo-docker-registry-paths.html)