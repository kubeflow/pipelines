# RoboMaker Simulation Job Batch Kubeflow Pipelines component

## Summary
Component to run a RoboMaker Simulation Job Batch from a Kubeflow Pipelines workflow.
https://docs.aws.amazon.com/robomaker/latest/dg/API_StartSimulationJobBatch.html

## Intended Use
For running your simulation workloads using AWS RoboMaker.

    max_concurrency: Input
    simulation_job_requests: Input
    sim_app_arn: Input

## Runtime Arguments
Argument        | Description                 | Optional   | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------| :----------| :----------     | :----------|
region | The region where the cluster launches | No | String | | |
endpoint_url | The endpoint URL for the private link VPC endpoint | Yes | String | | |
assume_role | The ARN of an IAM role to assume when connecting to SageMaker | Yes | String | | |
app_name | The name of the simulation application. Must be unique within the same AWS account and AWS region | Yes | String | | SimulationApplication-[datetime]-[random id]|
role | The Amazon Resource Name (ARN) that Amazon RoboMaker assumes to perform tasks on your behalf | No | String | | |
timeout_in_secs | The amount of time, in seconds, to wait for the batch to complete | Yes | String | | |
max_concurrency | The number of active simulation jobs create as part of the batch that can be in an active state at the same time | Yes | Int | | |
simulation_job_requests | A list of simulation job requests to create in the batch | No | List of Dicts | | [] |
sim_app_arn | The application ARN for the simulation application | Yes | String | | |
tags | Key-value pairs to categorize AWS resources | Yes | Dict | | {} |

Notes:
* This component can be ran in a pipeline with the Create Simulation App and Delete Simulation App components or as a standalone.
* One of sim_app_arn can be provided as an input, or can be embedded as the 'application' value for any of the simulation_job_requests.
* The format for the [`simulation_job_requests`](https://docs.aws.amazon.com/robomaker/latest/dg/API_SimulationJobRequest.html) field is:
```
[
    {
        "outputLocation": {
            "s3Bucket": "string",
            "s3Prefix": "string",
        },
        "loggingConfig": {"recordAllRosTopics": "bool"},
        "maxJobDurationInSeconds": "int",
        "iamRole": "string",
        "failureBehavior": "string",
        "simulationApplications": [
            {
                "application": "string",
                "launchConfig": {
                    "packageName": "string",
                    "launchFile": "string",
                    "environmentVariables": {
                        "string": "string",
                    },
                    "streamUI": "bool",
                },
            }
        ],
        "vpcConfig": {
            "subnets": "list",
            "securityGroups": "list",
            "assignPublicIp": "bool",
        },
    }
]
```

## Output
The ARN and ID of the batch job.

# Example code
Example of creating a Sim app, then a Sim job batch and finally deleting the Sim app : [robomaker_simulation_job_batch_app](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/robomaker_simulation/robomaker_simulation_job_batch_app.py)

# Resources
* [Create RoboMaker Simulation Job Batch via Boto3](https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/robomaker.html#RoboMaker.Client.start_simulation_job_batch)
