# SageMaker Processing Kubeflow Pipelines component
## Summary
Component to submit SageMaker Processing jobs directly from a Kubeflow Pipelines workflow.
https://docs.aws.amazon.com/sagemaker/latest/dg/processing-job.html

# Details

## Intended Use
For running your data processing workloads, such as feature engineering, data validation, model evaluation, and model interpretation using AWS SageMaker.

## Runtime Arguments
Argument        | Description                 | Optional   | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------| :----------| :----------     | :----------|
region | The region where the cluster launches | No | String | | |
endpoint_url | The endpoint URL for the private link VPC endpoint. | Yes | String | | |
job_name | The name of the Ground Truth job. Must be unique within the same AWS account and AWS region | Yes | String | | LabelingJob-[datetime]-[random id]|
role | The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf | No | String | | |
image | The registry path of the Docker image that contains the training algorithm | Yes | String | | |
instance_type | The ML compute instance type | Yes | String | ml.m4.xlarge, ml.m4.2xlarge, ml.m4.4xlarge, ml.m4.10xlarge, ml.m4.16xlarge, ml.m5.large, ml.m5.xlarge, ml.m5.2xlarge, ml.m5.4xlarge, ml.m5.12xlarge, ml.m5.24xlarge, ml.c4.xlarge, ml.c4.2xlarge, ml.c4.4xlarge, ml.c4.8xlarge, ml.p2.xlarge, ml.p2.8xlarge, ml.p2.16xlarge, ml.p3.2xlarge, ml.p3.8xlarge, ml.p3.16xlarge, ml.c5.xlarge, ml.c5.2xlarge, ml.c5.4xlarge, ml.c5.9xlarge, ml.c5.18xlarge [and many more](https://aws.amazon.com/sagemaker/pricing/instance-types/) | ml.m4.xlarge |
instance_count | The number of ML compute instances to use in each training job | Yes | Int | ≥ 1 | 1 |
volume_size | The size of the ML storage volume that you want to provision in GB | Yes | Int | ≥ 1 | 30 |
resource_encryption_key | The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s) | Yes | String | | |
output_encryption_key | The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts | Yes | String | | |
max_run_time | The maximum run time in seconds per training job | Yes | Int | ≤ 432000 (5 days) | 86400 (1 day) |
environment | The environment variables to set in the Docker container | Yes | Yes | Dict | Maximum length of 1024. Key Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`. Value Pattern: `[\S\s]*`. Upto 16 key and values entries in the map | |
container_entrypoint | The entrypoint for the processing job. This is in the form of a list of strings that make a command | Yes | Yes | List of Dicts | | [] |
container_arguments | A list of string arguments to be passed to a processing job | Yes | Yes | List of Dicts | | [] |
input_config | Parameters that specify Amazon S3 inputs for a processing job | No | Dict | | {} |
output_config | Parameters that specify Amazon S3 outputs for a processing job | No | Dict | | {} |
vpc_security_group_ids | A comma-delimited list of security group IDs, in the form sg-xxxxxxxx | Yes | String | | |
vpc_subnets | A comma-delimited list of subnet IDs in the VPC to which you want to connect your hpo job | Yes | String | | |
network_isolation | Isolates the training container if true | No | Boolean | False, True | True |
traffic_encryption | Encrypts all communications between ML compute instances in distributed training if true | No | Boolean | False, True | False |
tags | Key-value pairs to categorize AWS resources | Yes | Dict | | {} |


## Output
Stores each of the processing outputs in the S3 buckets specified in the `output_config`.

# Example code
Simple example pipeline with only Train component : [simple_train_pipeline](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/simple_train_pipeline)
