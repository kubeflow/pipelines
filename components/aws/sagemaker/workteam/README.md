# SageMaker create private workteam Kubeflow Pipelines component
## Summary
Component to submit SageMaker create private workteam jobs directly from a Kubeflow Pipelines workflow.

# Details

## Intended Use
For creating a private workteam from pre-existing Amazon Cognito user groups using AWS SageMaker.

## Runtime Arguments
Argument        | Description                 | Optional   | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------| :----------| :----------     | :----------|
region | The region where the cluster launches | No | String | | |
endpoint_url | The endpoint URL for the private link VPC endpoint | Yes | String | | |
assume_role | The ARN of an IAM role to assume when connecting to SageMaker | Yes | String | | |
team_name | The name of your work team | No | String | | |
description | A description of the work team | No | String | | |
user_pool | An identifier for a user pool, which must be in the same region as the service that you are calling | No | String | | |
user_groups | An identifier for user groups separated by commas | No | String | | |
client_id | An identifier for an application client, which you must create using Amazon Cognito | No | String | | |
sns_topic | The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts | Yes | String | | |
tags | Key-value pairs to categorize AWS resources | Yes | Dict | | {} |

Notes:
* Workers in private workteams per account and region may come from only one Amazon Cognito user pool. However, you may have several user groups within the user pool to define different workteams.
* Your Amazon Cognito user pool must be in the same region that you are creating the Ground Truth job in.

## Outputs
Name | Description
:--- | :----------
workteam_arn | ARN of the workteam

# Requirements
* [Kubeflow pipelines SDK](https://www.kubeflow.org/docs/pipelines/sdk/install-sdk/)
* [Kubeflow set-up on AWS](https://www.kubeflow.org/docs/aws/deploy/install-kubeflow/)
* User pool, user groups, and app client ID set up on Amazon Cognito
  1. Create a user pool in Amazon Cognito. Configure the user pool as needed, and make sure to create an app client. The Pool ID will be found under General settings.
  2. After creating the user pool, go to the Users and Groups section and create a group. Create users for the team, and add those users to the group.
  3. Under App integration > Domain name, create an Amazon Cognito domain for the user pool.

# Samples
## In a pipeline with Ground Truth and training
Mini image classification: [Demo](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/ground_truth_pipeline_demo)

# References
* [Managing a private workforce](https://docs.aws.amazon.com/sagemaker/latest/dg/sms-workforce-management-private.html)
