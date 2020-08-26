# SageMaker Hosting Services - Create Model Kubeflow Pipeline component

## Summary
Component to create a model in SageMaker from a Kubeflow Pipelines workflow.

## Details
Deploying a model using Amazon SageMaker hosting services is a three-step process:

1. **Create a model in Amazon SageMaker** - Specify the S3 path where model artifacts are stored and Docker registry path for the image that contains the inference code 
2. **Create an endpoint configuration for an HTTPS endpoint** - Specify the name of model in production variants and the type of instance that you want Amazon SageMaker to launch to host the model.
3. **Create an HTTPS endpoint** - Launch the ML compute instances and deploy the model as specified in the endpoint configuration

This component handles Step 1. Step 2 and 3 can be done using the [deploy component](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/deploy) for AWS SageMaker.

## Intended Use
Create a model in Amazon SageMaker to be used for [creating an endpoint](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/deploy) in hosting services or [run a batch transform job](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/batch_transform).

## Runtime Arguments
Argument        | Description                 | Optional (in pipeline definition) | Optional (in UI) | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------                       | :----------      | :----------| :----------     | :----------|
region | The region where the model is created | No | No | String | | |
endpoint_url | The endpoint URL for the private link VPC endpoint | Yes | String | | |
assume_role | The ARN of an IAM role to assume when connecting to SageMaker | Yes | String | | |
tags | Key-value pairs to tag the model created in AWS | Yes | Yes | Dict | | {} |
role | The ARN of the IAM role that Amazon SageMaker can assume to access model artifacts and docker image for deployment on ML compute instances or for batch transform jobs | No | No | String | | |
network_isolation | Isolates the model container. No inbound or outbound network calls can be made to or from the model container | Yes | Yes | Boolean | | True |
model_name | The name of the new model | No | No | String | | |
vpc_subnets | The ID of the subnets in the VPC to which you want to connect your training job or model | No if `vpc_security_group_ids` is specified | No if `vpc_security_group_ids` is specified | Array of Strings | | |
vpc_security_group_ids | The security groups for the VPC that is specified in the vpc_subnets field | No if `vpc_subnets` is specified | No if `vpc_subnets` is specified  | Array of Strings | | |

The docker image containing inference code, associated artifacts, and environment map that the inference code uses when the model is deployed for predictions make up the [`ContainerDefinition`](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_ContainerDefinition.html) object in CreateModel API. The following parameters(except secondary_containers) describes the container, as part of model definition:

Argument        | Description                 | Optional (in pipeline definition) | Optional (in UI) | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------                       | :----------      | :----------| :----------     | :----------|
container_host_name | When a ContainerDefinition is part of an [inference pipeline](https://docs.aws.amazon.com/sagemaker/latest/dg/inference-pipelines.html), the value of the parameter uniquely identifies the container for the purposes of [logging and metrics](https://docs.aws.amazon.com/sagemaker/latest/dg/inference-pipeline-logs-metrics.html) | Yes | Yes | String | Length Constraints: Maximum length of 63. Pattern: `^[a-zA-Z0-9](-*[a-zA-Z0-9])*` | |
environment | The environment variables to set in the Docker container | Yes | Yes | Dict | Maximum length of 1024. Key Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`. Value Pattern: `[\S\s]*` | {} |
image | The Amazon EC2 Container Registry (Amazon ECR) path where inference code is stored | Yes | Yes | String | | |
model_artifact_url | The S3 path where the model artifacts are stored. This path must point to a single gzip compressed tar archive (.tar.gz suffix) | Yes | Yes | String | | |
model_package | The name or Amazon Resource Name (ARN) of the model package to use to create the model | Yes | Yes | String | | |
secondary_containers | List of ContainerDefinition dictionaries in form of string (see Notes below) | Yes | Yes | String| Maximum number of 5 items | |

Notes:
* Please use the links in the [Resources section](#Resources) for detailed information on each input parameter and SageMaker APIs used in this component
* If you don't specify a value for `container_host_name` parameter for a `ContainerDefinition` that is part of an [inference pipeline](https://docs.aws.amazon.com/sagemaker/latest/dg/inference-pipelines.html), a unique name is automatically assigned based on the position of the `ContainerDefinition` in the pipeline. If you specify a value for the `ContainerHostName` for any `ContainerDefinition` that is part of an inference pipeline, you must specify a value for the `ContainerHostName` parameter of every `ContainerDefinition` in that pipeline.
* Each key and value in the `Environment` parameter string to string map can have length of up to 1024. SageMaker supports up to 16 entries in the map.
* Input format to specify `secondary_containers` is:
```
[ 
      { 
         "ContainerHostname": "string",
         "Environment": { 
            "string" : "string" 
         },
         "Image": "string",
         "ModelDataUrl": "string",
         "ModelPackageName": "string"
      }
   ]
```
* Specify either an `image` and `model_artifact_url`, OR a `model_package` in the `ContainerDefinition`.
* If you have a single container to define the model, use the parameters `container_host_name`, `environment`, `image`, `model_artifact_url`, `model_package` directly to define the primary container.

## Outputs
Name | Description
:--- | :----------
model_name | The name of the model created in Amazon SageMaker

## Requirements
* [Kubeflow pipelines SDK](https://www.kubeflow.org/docs/pipelines/sdk/install-sdk/)
* [Kubeflow set-up](https://www.kubeflow.org/docs/aws/deploy/install-kubeflow/)

## Samples
### Integrated into a pipeline
MNIST Classification pipeline: [Pipeline](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/aws-samples/mnist-kmeans-sagemaker/mnist-classification-pipeline.py) | [Steps](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/aws-samples/mnist-kmeans-sagemaker/README.md)

## Resources
* [Create Model API documentation](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateModel.html)
* [Boto3 API reference](https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_model)
