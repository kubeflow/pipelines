# SageMaker Ground Truth Kubeflow Pipelines component
## Summary
Component to submit SageMaker Ground Truth labeling jobs directly from a Kubeflow Pipelines workflow.

# Details

## Intended Use
For Ground Truth jobs using AWS SageMaker.

## Runtime Arguments
Argument        | Description                 | Optional   | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------| :----------| :----------     | :----------|
region | The region where the cluster launches | No | String | | |
endpoint_url | The endpoint URL for the private link VPC endpoint | Yes | String | | |
assume_role | The ARN of an IAM role to assume when connecting to SageMaker | Yes | String | | |
role | The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf | No | String | | |
job_name | The name of the Ground Truth job. Must be unique within the same AWS account and AWS region | Yes | String | | LabelingJob-[datetime]-[random id]|
label_attribute_name | The attribute name to use for the label in the output manifest file | Yes | String | | job_name |
manifest_location | The Amazon S3 location of the manifest file that describes the input data objects | No | String | | |
output_location | The Amazon S3 path where you want Amazon SageMaker to store the results of the transform job | No | String | | |
output_encryption_key | The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts | Yes | String | | |
task_type | Built in image classification, bounding box, text classification, or semantic segmentation, or custom; If custom, please provide pre- and post-labeling task lambda functions | No | String | Image Classification, Bounding Box, Text Classification, Semantic Segmentation, Custom | |
worker_type | The workteam for data labeling | No | String | Public, Private, Vendor | |
workteam_arn | The ARN of the work team assigned to complete the tasks; specify if worker type is private or vendor | Yes | String | | |
no_adult_content | If data is free of adult content; specify if worker type is public | Yes | Boolean | False, True | False |
no_ppi | If data is free of personally identifiable information; specify if worker type is public | Yes | Boolean | False, True | False |
label_category_config | The S3 URL of the JSON structured file that defines the categories used to label the data objects | Yes | String | | |
max_human_labeled_objects | The maximum number of objects that can be labeled by human workers | Yes | Int | ≥ 1 | all objects |
max_percent_objects | The maximum percentage of input data objects that should be labeled | Yes | Int | [1, 100] | 100 |
enable_auto_labeling | Enables auto-labeling; only for bounding box, text classification, and image classification | Yes | Boolean | False, True | False |
initial_model_arn | The ARN of the final model used for a previous auto-labeling job | Yes | String | | |
resource_encryption_key | The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s) | Yes | String | | |
ui_template | The Amazon S3 bucket location of the UI template | No | String | | |
pre_human_task_function | The ARN of a Lambda function that is run before a data object is sent to a human worker | Yes | String | | |
post_human_task_function | The ARN of a Lambda function implements the logic for annotation consolidation | Yes | String | | |
task_keywords | Keywords used to describe the task so that workers on Amazon Mechanical Turk can discover the task | Yes | String | | |
title | A title for the task for your human workers | No | String | | |
description | A description of the task for your human workers | No | String | | |
num_workers_per_object | The number of human workers that will label an object | No | Int | [1, 9] | |
time_limit | The maximum run time in seconds per training job | No | Int | [30, 28800] | |
task_availibility | The length of time that a task remains available for labeling by human workers | Yes | Int | Public workforce: [1, 43200], other: [1, 864000] | |
max_concurrent_tasks | The maximum number of data objects that can be labeled by human workers at the same time | Yes | Int | [1, 1000] | |
workforce_task_price | The price that you pay for each task performed by a public worker in USD; Specify to the tenth fractions of a cent; Format as "0.000" | Yes | Float | 0.000 |
tags | Key-value pairs to categorize AWS resources | Yes | Dict | | {} |

## Outputs
Name | Description
:--- | :----------
output_manifest_location | URL where labeling results were stored
active_learning_model_arn | ARN of the resulting active learning model

# Requirements
* [Kubeflow pipelines SDK](https://www.kubeflow.org/docs/pipelines/sdk/install-sdk/)
* [Kubeflow set-up on AWS](https://www.kubeflow.org/docs/aws/deploy/install-kubeflow/)

# Samples
## Used in a pipeline with workteam creation and training
Mini image classification demo: [Demo](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/aws-samples/ground_truth_pipeline_demo/)

# References
* [Ground Truth documentation](https://docs.aws.amazon.com/sagemaker/latest/dg/sms.html)
* [Building a custom data labeling workflow](https://aws.amazon.com/blogs/machine-learning/build-a-custom-data-labeling-workflow-with-amazon-sagemaker-ground-truth/)
* [Sample UI template for Bounding Box](https://github.com/awslabs/amazon-sagemaker-examples/blob/master/ground_truth_labeling_jobs/ground_truth_object_detection_tutorial/object_detection_tutorial.ipynb)
* [Sample UI template for Image Classification](https://github.com/awslabs/amazon-sagemaker-examples/tree/master/ground_truth_labeling_jobs/from_unlabeled_data_to_deployed_machine_learning_model_ground_truth_demo_image_classification)
* [Using Ground Truth results in training jobs](https://github.com/awslabs/amazon-sagemaker-examples/blob/master/ground_truth_labeling_jobs/object_detection_augmented_manifest_training/object_detection_augmented_manifest_training.ipynb)
