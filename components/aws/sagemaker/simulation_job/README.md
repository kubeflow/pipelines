# RoboMaker Simulation Job Kubeflow Pipelines component

## Summary
Component to run a RoboMaker Simulation Job from a Kubeflow Pipelines workflow.
https://docs.aws.amazon.com/robomaker/latest/dg/API_CreateSimulationJob.html

## Intended Use
For running your simulation workloads using AWS RoboMaker.

## Runtime Arguments
Argument        | Description                 | Optional   | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------| :----------| :----------     | :----------|
region | The region where the cluster launches | No | String | | |
endpoint_url | The endpoint URL for the private link VPC endpoint | Yes | String | | |
assume_role | The ARN of an IAM role to assume when connecting to SageMaker | Yes | String | | |
app_name | The name of the simulation application. Must be unique within the same AWS account and AWS region | Yes | String | | SimulationApplication-[datetime]-[random id]|
role | The Amazon Resource Name (ARN) that Amazon RoboMaker assumes to perform tasks on your behalf | No | String | | |
output_bucket | The bucket to place outputs from the simulation job | No | String | | |
output_path | The S3 key where outputs from the simulation job are placed | No | String | | |
max_run | Timeout in seconds for simulation job (default: 8 * 60 * 60) | No | String | | |
failure_behavior | The failure behavior the simulation job (Continue|Fail) | Yes | String | | |
sim_app_arn | The application ARN for the simulation application | Yes | String | | |
sim_app_version | The application version for the simulation application | Yes | String | | |
sim_app_launch_config | The launch configuration for the simulation application | Yes | String | | |
sim_app_world_config | A list of world configurations | Yes | List of Dicts | | [] |
robot_app_arn | The application ARN for the robot application | Yes | String | | |
robot_app_version | The application version for the robot application | Yes | String | | |
robot_app_launch_config | The launch configuration for the robot application | Yes | Dict | | {} |
data_sources | Specify data sources to mount read-only files from S3 into your simulation | Yes | List of Dicts | | [] |
vpc_security_group_ids | A comma-delimited list of security group IDs, in the form sg-xxxxxxxx | Yes | String | | |
vpc_subnets | A comma-delimited list of subnet IDs in the VPC to which you want to connect your simulation job | Yes | String | | |
use_public_ip | A boolean indicating whether to assign a public IP address | Yes | Bool | | False |
sim_unit_limit | The simulation unit limit | Yes | String | | |
record_ros_topics | A boolean indicating whether to record all ROS topics (Used for logging) | Yes | Bool | | False |
tags | Key-value pairs to categorize AWS resources | Yes | Dict | | {} |

Notes:
* This component can be ran in a pipeline with the Create Simulation App and Delete Simulation App components or as a standalone.
* One of sim_app_arn or robot_app_arn and any related inputs must be provided.
* The format for the [`sim_app_launch_config`](https://docs.aws.amazon.com/robomaker/latest/dg/API_LaunchConfig.html) field is:
```
{
    "packageName": "string",
    "launchFile": "string",
    "environmentVariables": {
        "string": "string",
    },
    "streamUI": "bool",
}
```
* The format for the [`sim_app_world_config`](https://docs.aws.amazon.com/robomaker/latest/dg/API_WorldConfig.html) field is:
```
{
    "world": "string"
}
```
* The format for the [`robot_app_launch_config`](https://docs.aws.amazon.com/robomaker/latest/dg/API_LaunchConfig.html) field is:
```
{
    "packageName": "string",
    "launchFile": "string",
    "environmentVariables": {
        "string": "string",
    },
    "streamUI": "bool",
}
```


## Output
The output of the simulation job is sent to the location configured via output_artifacts

# Example code
Example of creating a Sim app, then a Sim job and finally deleting the Sim app : [robomaker_simulation_job_app](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/robomaker_simulation/robomaker_simulation_job_app.py)

# Resources
* [Create RoboMaker Simulation Job via Boto3](https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/robomaker.html#RoboMaker.Client.create_simulation_job)
