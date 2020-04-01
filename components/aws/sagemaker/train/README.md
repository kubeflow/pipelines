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
Example pipeline with only Train component :
```buildoutcfg

```
Example inputs to this pipeline :
```buildoutcfg
region : us-east-1
endpoint_url : leave this empty
image : 382416733822.dkr.ecr.us-east-1.amazonaws.com/kmeans:1
training_input_mode : File
hyperparameters : {"k": "10", "feature_dim": "784"}
channels : Along with other parametes, you need to pass the S3 Uri where you have data
you can get sample data using this code >>> aws s3 cp s3://m-kfp-mnist/mnist_kmeans_example/data s3://<your_bucket_name>/mnist_kmeans_example/data

                [
                  {
                    "ChannelName": "train",
                    "DataSource": {
                      "S3DataSource": {
                        "S3Uri": "<your_s3_bucket_where_you_have_data>",
                        "S3DataType": "S3Prefix",
                        "S3DataDistributionType": "FullyReplicated"
                      }
                    },
                    "ContentType": "",
                    "CompressionType": "None",
                    "RecordWrapperType": "None",
                    "InputMode": "File"
                  }
                ]

instance_type : ml.p2.xlarge
instance_count : 1
volume_size : 50
max_run_time : 3600
model_artifact_path : <some_s3_bucket_where_output_will_be_stored>
output_encryption_key : leave this empty
network_isolation : True
traffic_encryption : False
spot_instance : False
max_wait_time : 3600
checkpoint_config : {}
role : You need a IAM role with Full sagemaker permissions and S3 access 
If you already have that then use that role ARN
Else you can create a new role 
Go to you AWS account -> IAM -> Roles -> Create Role -> select sagemaker service -> Next -> Next -> Next -> Give some name -> Create Role -> Click on the role that you created -> Attach policy -> AmazonS3FullAccess -> Attach Policy -> Copy role ARN
Example role input : arn:aws:iam::999999999999:role/test_temp_role
```


# Resources
* [Using Amazon built-in algorithms](https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-algo-docker-registry-paths.html)