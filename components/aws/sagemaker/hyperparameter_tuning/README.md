# SageMaker hyperparameter optimization Kubeflow Pipeline component
## Summary
Component to submit hyperparameter tuning jobs to SageMaker directly from a Kubeflow Pipelines workflow.

# Details

## Intended Use
For hyperparameter tuning jobs using AWS SageMaker.

## Runtime Arguments
Argument        | Description                 | Optional (in pipeline definition) | Optional (in UI) | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------                       | :----------      | :----------| :----------     | :----------|
region | The region where the cluster launches | No | No | String | | |
endpoint_url | The endpoint URL for the private link VPC endpoint | Yes | String | | |
assume_role | The ARN of an IAM role to assume when connecting to SageMaker | Yes | String | | |
job_name | The name of the tuning job. Must be unique within the same AWS account and AWS region | Yes | Yes | String | | HPOJob-[datetime]-[random id] |
role | The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf | No | No | String | | |
image | The registry path of the Docker image that contains the training algorithm | Yes | Yes | String | | |
algorithm_name | The name of the algorithm resource to use for the hyperparameter tuning job; only specify this parameter if training image is not specified | Yes | Yes | String | | |
training_input_mode | The input mode that the algorithm supports | Yes | No | String | File, Pipe | File |
metric_definitions | The dictionary of name-regex pairs specify the metrics that the algorithm emits | Yes | Yes | Dict | | {} |
strategy | How hyperparameter tuning chooses the combinations of hyperparameter values to use for the training job it launches | Yes | No | String | Bayesian, Random | Bayesian |
metric_name | The name of the metric to use for the objective metric | No | No | String | | |
metric_type | Whether to minimize or maximize the objective metric | No | No | String | Maximize, Minimize | |
early_stopping_type | Whether to minimize or maximize the objective metric | Yes | No | String | Off, Auto | Off |
static_parameters | The values of hyperparameters that do not change for the tuning job | Yes | Yes | Dict | | {} |
integer_parameters | The array of IntegerParameterRange objects that specify ranges of integer hyperparameters that you want to search | Yes | Yes | List of Dicts | | [] |
continuous_parameters | The array of ContinuousParameterRange objects that specify ranges of continuous hyperparameters that you want to search | Yes | Yes | List of Dicts | | [] |
categorical_parameters | The array of CategoricalParameterRange objects that specify ranges of categorical hyperparameters that you want to search | Yes | Yes | List of Dicts | | [] |
channels | A list of dicts specifying the input channels (at least one); refer to [documentation](https://github.com/awsdocs/amazon-sagemaker-developer-guide/blob/master/doc_source/API_Channel.md) for parameters | No | No | List of Dicts | | |
output_location | The Amazon S3 path where you want Amazon SageMaker to store the results of the transform job | No | No | String | | |
output_encryption_key | The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts | Yes | Yes | String | | |
instance_type | The ML compute instance type | Yes | No | String | ml.m4.xlarge, ml.m4.2xlarge, ml.m4.4xlarge, ml.m4.10xlarge, ml.m4.16xlarge, ml.m5.large, ml.m5.xlarge, ml.m5.2xlarge, ml.m5.4xlarge, ml.m5.12xlarge, ml.m5.24xlarge, ml.c4.xlarge, ml.c4.2xlarge, ml.c4.4xlarge, ml.c4.8xlarge, ml.p2.xlarge, ml.p2.8xlarge, ml.p2.16xlarge, ml.p3.2xlarge, ml.p3.8xlarge, ml.p3.16xlarge, ml.c5.xlarge, ml.c5.2xlarge, ml.c5.4xlarge, ml.c5.9xlarge, ml.c5.18xlarge [and many more](https://aws.amazon.com/sagemaker/pricing/instance-types/)| ml.m4.xlarge |
instance_count | The number of ML compute instances to use in each training job | Yes | Yes | Int | ≥ 1 | 1 |
volume_size | The size of the ML storage volume that you want to provision in GB | Yes | Yes | Int | ≥ 1 | 30 |
max_num_jobs | The maximum number of training jobs that a hyperparameter tuning job can launch | No | No | Int | [1, 500] | |
max_parallel_jobs | The maximum number of concurrent training jobs that a hyperparameter tuning job can launch | No | No | Int | [1, 10] | |
max_run_time | The maximum run time in seconds per training job | Yes | Yes | Int | ≤ 432000 (5 days) | 86400 (1 day) |
resource_encryption_key | The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s) | Yes | Yes | String | | |
vpc_security_group_ids | A comma-delimited list of security group IDs, in the form sg-xxxxxxxx | Yes | Yes | String | | |
vpc_subnets | A comma-delimited list of subnet IDs in the VPC to which you want to connect your hpo job | Yes | Yes | String | | |
network_isolation | Isolates the training container if true | Yes | No | Boolean | False, True | True |
traffic_encryption | Encrypts all communications between ML compute instances in distributed training if true | Yes | No | Boolean | False, True | False |
spot_instance | Use managed spot training if true | Yes | No | Boolean | False, True | False |
max_wait_time | The maximum time in seconds you are willing to wait for a managed spot training job to complete | Yes | Yes | Int | ≤ 432000 (5 days) | 86400 (1 day) |
checkpoint_config | Dictionary of information about the output location for managed spot training checkpoint data | Yes | Yes | Dict | | {} |
warm_start_type | Specifies the type of warm start used | Yes | No | String | IdenticalDataAndAlgorithm, TransferLearning | |
parent_hpo_jobs | List of previously completed or stopped hyperparameter tuning jobs to be used as a starting point | Yes | Yes | String | Yes | | |
tags | Key-value pairs to categorize AWS resources | Yes | Yes | Dict | | {} |

Notes:
* Specify training image OR algorithm name. Use the image parameter for Bring Your Own Container (BYOC) algorithms, and algorithm name for Amazon built-in algorithms, custom algorithm resources in SageMaker, and algorithms subscribed to from the AWS Marketplace.
* Specify VPC security group IDs AND VPC subnets to specify the VPC that you want the training jobs to connect to.
* Specify warm start type AND 1 to 5 parent HPO jobs to launch the hyperparameter tuning job with previous jobs as a starting point.

## Outputs
Name | Description
:--- | :----------
hpo_job_name | The name of the hyper parameter tuning job
model_artifact_url | URL where model artifacts were stored
best_job_name | Best hyperparameter tuning training job name
best_hyperparameters | Tuned hyperparameters
training_image | The registry path of the Docker image that contains the training algorithm

# Requirements
* [Kubeflow pipelines SDK](https://www.kubeflow.org/docs/pipelines/sdk/install-sdk/)
* [Kubeflow set-up](https://www.kubeflow.org/docs/aws/deploy/install-kubeflow/)

# Samples
## On its own
K-Means algorithm tuning on MNIST dataset: [pipeline](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/aws-samples/mnist-kmeans-sagemaker/kmeans-hpo-pipeline.py)

Follow the steps as in the [README](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/aws-samples/mnist-kmeans-sagemaker/README.md) with some modification:
1. Get and store data in S3 buckets
2. Prepare an IAM roles with permissions to run SageMaker jobs
3. Add 'aws-secret' to your kubeflow namespace
4. Compile the pipeline:
```bash
dsl-compile --py kmeans-hpo-pipeline.py --output kmeans-hpo-pipeline.tar.gz
```
5. In the Kubeflow UI, upload the compiled pipeline specification (the .tar.gz file) and create a new run. Update the role_arn and the data paths, and optionally any other run parameters.
6. Once the pipeline completes, you can see the outputs under 'Output parameters' in the HPO component's Input/Output section.

## Integrated into a pipeline
MNIST Classification using K-Means pipeline: [Pipeline](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/aws-samples/mnist-kmeans-sagemaker/mnist-classification-pipeline.py) | [Steps](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/aws-samples/mnist-kmeans-sagemaker/README.md)

# Resources
* [Using Amazon built-in algorithms](https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-algo-docker-registry-paths.html)
* [More information on request parameters](https://github.com/awsdocs/amazon-sagemaker-developer-guide/blob/master/doc_source/API_CreateHyperParameterTuningJob.md#request-parameters)
