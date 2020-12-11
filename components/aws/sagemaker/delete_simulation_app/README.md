# RoboMaker Delete Simulation Application Kubeflow Pipelines component

## Summary
Component to delete RoboMaker Simulation Application's from a Kubeflow Pipelines workflow.
https://docs.aws.amazon.com/robomaker/latest/dg/API_DeleteSimulationApplication.html

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
arn | The Amazon Resource Name (ARN) of the simulation application | No | String | | |
version | The version of the simulation application | Yes | String | | |

Notes:
* This component can be used to clean up any simulation apps that were created by other components such as the Create Simulation App component.
* This component should to be ran as after the RoboMaker [`Simulation Job component`](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/simulation_job/README.md)
* The format for the [`sources`](https://docs.aws.amazon.com/robomaker/latest/dg/API_SourceConfig.html) field is:


## Output
The ARN of the deleted Simulation Application.

# Example code
Example of creating a Sim app, then a Sim job and finally deleting the Sim app : [robomaker_simulation_job_app](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/robomaker_simulation/robomaker_simulation_job_app.py)

# Resources
* [Delete RoboMaker Simulation Application via Boto3](https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/robomaker.html#RoboMaker.Client.delete_simulation_application)
