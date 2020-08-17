# SageMaker Processing Kubeflow Pipelines component

## Summary
Component to submit SageMaker Processing jobs directly from a Kubeflow Pipelines workflow.
https://docs.aws.amazon.com/sagemaker/latest/dg/processing-job.html

## Intended Use
For running your data processing workloads, such as feature engineering, data validation, model evaluation, and model interpretation using AWS SageMaker.

## Runtime Arguments
Argument        | Description                 | Optional   | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------| :----------| :----------     | :----------|
region | The region where the cluster launches | No | String | | |
endpoint_url | The endpoint URL for the private link VPC endpoint | Yes | String | | |
assume_role | The ARN of an IAM role to assume when connecting to SageMaker | Yes | String | | |
job_name | The name of the Processing job. Must be unique within the same AWS account and AWS region | Yes | String | | ProcessingJob-[datetime]-[random id]|
role | The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf | No | String | | |
image | The registry path of the Docker image that contains the processing script | Yes | String | | |
instance_type | The ML compute instance type | Yes | String | ml.m4.xlarge, ml.m4.2xlarge, ml.m4.4xlarge, ml.m4.10xlarge, ml.m4.16xlarge, ml.m5.large, ml.m5.xlarge, ml.m5.2xlarge, ml.m5.4xlarge, ml.m5.12xlarge, ml.m5.24xlarge, ml.c4.xlarge, ml.c4.2xlarge, ml.c4.4xlarge, ml.c4.8xlarge, ml.p2.xlarge, ml.p2.8xlarge, ml.p2.16xlarge, ml.p3.2xlarge, ml.p3.8xlarge, ml.p3.16xlarge, ml.c5.xlarge, ml.c5.2xlarge, ml.c5.4xlarge, ml.c5.9xlarge, ml.c5.18xlarge [and many more](https://aws.amazon.com/sagemaker/pricing/instance-types/) | ml.m4.xlarge |
instance_count | The number of ML compute instances to use in each processing job | Yes | Int | ≥ 1 | 1 |
volume_size | The size of the ML storage volume that you want to provision in GB | Yes | Int | ≥ 1 | 30 |
resource_encryption_key | The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s) | Yes | String | | |
output_encryption_key | The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts | Yes | String | | |
max_run_time | The maximum run time in seconds per processing job | Yes | Int | ≤ 432000 (5 days) | 86400 (1 day) |
environment | The environment variables to set in the Docker container | Yes | Yes | Dict | Maximum length of 1024. Key Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`. Value Pattern: `[\S\s]*`. Upto 16 key and values entries in the map | |
container_entrypoint | The entrypoint for the processing job. This is in the form of a list of strings that make a command | Yes | Yes | List of Strings | | [] |
container_arguments | A list of string arguments to be passed to a processing job | Yes | Yes | List of Strings | | [] |
input_config | Parameters that specify Amazon S3 inputs for a processing job | No | List of Dicts | | [] |
output_config | Parameters that specify Amazon S3 outputs for a processing job | No | List of Dict | | [] |
vpc_security_group_ids | A comma-delimited list of security group IDs, in the form sg-xxxxxxxx | Yes | String | | |
vpc_subnets | A comma-delimited list of subnet IDs in the VPC to which you want to connect your hpo job | Yes | String | | |
network_isolation | Isolates the processing container if true | No | Boolean | False, True | True |
traffic_encryption | Encrypts all communications between ML compute instances in distributed processing if true | No | Boolean | False, True | False |
tags | Key-value pairs to categorize AWS resources | Yes | Dict | | {} |

Notes:
* You can find more information about how container entrypoint and arguments are used at the [Build Your Own Processing Container](https://docs.aws.amazon.com/sagemaker/latest/dg/build-your-own-processing-container.html#byoc-run-image) documentation.
* Each key and value in the `environment` parameter string to string map can have length of up to 1024. SageMaker supports up to 16 entries in the map.
* The format for the [`input_config`](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_ProcessingInput.html) field is:
```
[
  {
    'InputName': 'string',
    'S3Input': {
      'S3Uri': 'string',
      'LocalPath': 'string',
      'S3DataType': 'ManifestFile'|'S3Prefix',
      'S3InputMode': 'Pipe'|'File',
      'S3DataDistributionType': 'FullyReplicated'|'ShardedByS3Key',
      'S3CompressionType': 'None'|'Gzip'
    }
  },
]
```
* The format for the [`output_config`](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_ProcessingS3Output.html) field is:
```
[
  {
    'OutputName': 'string',
    'S3Output': {
      'S3Uri': 'string',
      'LocalPath': 'string',
      'S3UploadMode': 'Continuous'|'EndOfJob'
    }
  },
]
```

## Outputs
Name | Description
:--- | :----------
job_name | Processing job name
output_artifacts | A dictionary mapping with `output_config` `OutputName` as the key and `S3Uri` as the value

## Requirements
* [Kubeflow pipelines SDK](https://www.kubeflow.org/docs/pipelines/sdk/install-sdk/)
* [Kubeflow set-up](https://www.kubeflow.org/docs/aws/deploy/install-kubeflow/)

## Resources
* [Create Processing Job API documentation](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateProcessingJob.html)
* [Boto3 API reference](https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_processing_job)