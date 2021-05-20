# RoboMaker Create Simulation Application Kubeflow Pipelines component

## Summary
Component to create RoboMaker Simulation Application's from a Kubeflow Pipelines workflow.
https://docs.aws.amazon.com/robomaker/latest/dg/create-simulation-application.html

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
sources | The code sources of the simulation application | No | String | | |
simulation_software_name | The simulation software used by the simulation application | No | Dict | | {} |
simulation_software_version | The simulation software version used by the simulation application | No | Dict | | {} |
robot_software_name | The robot software (ROS distribution) used by the simulation application | No | Dict | | {} |
robot_software_version | The robot software version (ROS distribution) used by the simulation application | No | Dict | | {} |
rendering_engine_name | The rendering engine for the simulation application | Yes | Dict | | {} |
rendering_engine_version | The rendering engine version for the simulation application | Yes | Dict | | {} |
tags | Key-value pairs to categorize AWS resources | Yes | Dict | | {} |

Notes:
* This component should to be ran as a precursor to the RoboMaker [`Simulation Job component`](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/simulation_job/README.md)
* The format for the [`sources`](https://docs.aws.amazon.com/robomaker/latest/dg/API_SourceConfig.html) field is:
```
[
    {
        "s3Bucket": "string",
        "s3Key": "string",
        "architecture": "string",
    }
]
```

## Output
The ARN of the created Simulation Application. This can be passed as an input to other components such as RoboMaker Simulation Job.

# Example code
Example of creating a Sim app, then a Sim job and finally deleting the Sim app : [robomaker_simulation_job_app](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/robomaker_simulation/robomaker_simulation_job_app.py)

# Resources
* [Create RoboMaker Simulation Application via Boto3](https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/robomaker.html#RoboMaker.Client.create_simulation_application)
