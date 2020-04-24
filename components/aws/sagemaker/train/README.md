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
image | The registry path of the Docker image that contains the training algorithm | Yes | String | | |
algorithm_name | The name of the algorithm resource to use for the hyperparameter tuning job; only specify this parameter if training image is not specified | Yes | String | | |
metric_definitions | The dictionary of name-regex pairs specify the metrics that the algorithm emits | Yes | Dict | | {} |
put_mode | The input mode that the algorithm supports | No | String | File, Pipe | File |
hyperparameters  | Hyperparameters for the selected algorithm | No | Dict | [Depends on Algo](https://docs.aws.amazon.com/sagemaker/latest/dg/k-means-api-config.html)| |
channels | A list of dicts specifying the input channels (at least one); refer to [documentation](https://github.com/awsdocs/amazon-sagemaker-developer-guide/blob/master/doc_source/API_Channel.md) for parameters | No | No | List of Dicts | | |
instance_type | The ML compute instance type | Yes | No | String | ml.m4.xlarge, ml.m4.2xlarge, ml.m4.4xlarge, ml.m4.10xlarge, ml.m4.16xlarge, ml.m5.large, ml.m5.xlarge, ml.m5.2xlarge, ml.m5.4xlarge, ml.m5.12xlarge, ml.m5.24xlarge, ml.c4.xlarge, ml.c4.2xlarge, ml.c4.4xlarge, ml.c4.8xlarge, ml.p2.xlarge, ml.p2.8xlarge, ml.p2.16xlarge, ml.p3.2xlarge, ml.p3.8xlarge, ml.p3.16xlarge, ml.c5.xlarge, ml.c5.2xlarge, ml.c5.4xlarge, ml.c5.9xlarge, ml.c5.18xlarge | ml.m4.xlarge |
instance_count | The number of ML compute instances to use in each training job | Yes | Int | ≥ 1 | 1 |
volume_size | The size of the ML storage volume that you want to provision in GB | Yes | Int | ≥ 1 | 1 |
resource_encryption_key | The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s) | Yes | String | | |
max_run_time | The maximum run time in seconds per training job | Yes | Int | ≤ 432000 (5 days) | 86400 (1 day) |
model_artifact_path | | No | String | | |
output_encryption_key | The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts | Yes | String | | |
vpc_security_group_ids | The VPC security group IDs, in the form sg-xxxxxxxx | Yes | String | | |
vpc_subnets | The ID of the subnets in the VPC to which you want to connect your hpo job | Yes | String | | |
network_isolation | Isolates the training container if true | No | Boolean | False, True | True |
traffic_encryption | Encrypts all communications between ML compute instances in distributed training if true | No | Boolean | False, True | False |
spot_instance | Use managed spot training if true | No | Boolean | False, True | False |
max_wait_time | The maximum time in seconds you are willing to wait for a managed spot training job to complete | Yes | Int | ≤ 432000 (5 days) | 86400 (1 day) |
checkpoint_config | Dictionary of information about the output location for managed spot training checkpoint data | Yes | Dict | | {} |
tags | Key-value pairs to categorize AWS resources | Yes | Dict | | {} |


## Output
Stores the Model in the s3 bucket you specified 

# Example code
Simple example pipeline with only Train component : [simple_train_pipeline](https://github.com/kubeflow/pipelines/tree/documents/samples/contrib/aws-samples/simple_train_pipeline)

# Resources
* [Using Amazon built-in algorithms](https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-algo-docker-registry-paths.html)