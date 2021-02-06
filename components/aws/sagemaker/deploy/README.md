# SageMaker Hosting Services - Create Endpoint Kubeflow Pipeline component

## Summary
Component to deploy a model in SageMaker Hosting Service from a Kubeflow Pipelines workflow.

## Details
Deploying a model using Amazon SageMaker hosting services is a three-step process:

1. **Create a model in Amazon SageMaker** - Specify the S3 path where model artifacts are stored and Docker registry path for the image that contains the inference code 
2. **Create an endpoint configuration for an HTTPS endpoint** - Specify the name of model in production variants and the type of instance that you want Amazon SageMaker to launch to host the model.
3. **Create an HTTPS endpoint** - Launch the ML compute instances and deploy the model as specified in the endpoint configuration

This component handles Step 2 and 3. Step 1 can be done using the [create model component](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/model) for AWS SageMaker.
Endpoint created above can be updated using this component as well. When update_endpoint is set to true, endpoint is updated if it exists, if not one is created. Tags will be ignored if the endpoint is being upated.

## Intended Use
Create an endpoint in AWS SageMaker Hosting Service for model deployment.

## Runtime Arguments
Argument        | Description                 | Optional (in pipeline definition) | Optional (in UI) | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------                       | :----------      | :----------| :----------     | :----------|
region | The region where the endpoint is created | No | No | String | | |
endpoint_url | The endpoint URL for the private link VPC endpoint | Yes | String | | |
assume_role | The ARN of an IAM role to assume when connecting to SageMaker | Yes | String | | |
endpoint_config_name | The name of the endpoint configuration | Yes | Yes | String | | |
endpoint_config_tags | Key-value pairs to tag endpoint configurations in AWS | Yes | Yes | Dict | | {} |
endpoint_tags | Key-value pairs to tag the Hosting endpoint in AWS | Yes | Yes | Dict | | {} |
endpoint_name | The name of the endpoint. The name must be unique within an AWS Region in your AWS account | Yes | Yes | String | | |

In SageMaker, you can create an endpoint that can host multiple models. The set of parameters below represent a production variant. A production variant identifies a model that you want to host and the resources (e.g. instance type, initial traffic distribution etc.) to deploy for hosting it. You must specify at least one production variant to create an endpoint.

Argument        | Description                 | Optional (in pipeline definition) | Optional (in UI) | Data type  | Accepted values | Default    |
:---            | :----------                 | :----------                       | :----------      | :----------| :----------     | :----------|
model_name_[1, 3] | The name of the model that you want to host. This is the name that you specified when creating the model | No | No | String | | |
variant_name_[1, 3] | The name of the production variant | Yes | Yes | String | | variant_name_[1, 3] |
instance_type_[1, 3] | The ML compute instance type | Yes | Yes | String | ml.m4.xlarge, ml.m4.2xlarge, ml.m4.4xlarge, ml.m4.10xlarge, ml.m4.16xlarge, ml.m5.large, ml.m5.xlarge, ml.m5.2xlarge, ml.m5.4xlarge, ml.m5.12xlarge, ml.m5.24xlarge, ml.c4.xlarge, ml.c4.2xlarge, ml.c4.4xlarge, ml.c4.8xlarge, ml.p2.xlarge, ml.p2.8xlarge, ml.p2.16xlarge, ml.p3.2xlarge, ml.p3.8xlarge, ml.p3.16xlarge, ml.c5.xlarge, ml.c5.2xlarge, ml.c5.4xlarge, ml.c5.9xlarge, ml.c5.18xlarge [and many more](https://aws.amazon.com/sagemaker/pricing/instance-types/)| ml.m4.xlarge |
initial_instance_count_[1, 3] | Number of instances to launch initially | Yes | Yes | Integer | ≥ 1 | 1 |
initial_variant_weight_[1, 3] | Determines initial traffic distribution among all of the models that you specify in the endpoint configuration. The traffic to a production variant is determined by the ratio of the VariantWeight to the sum of all VariantWeight values across all ProductionVariants. | Yes | Yes | Float | Minimum value of 0 | |
accelerator_type_[1, 3] | The size of the Elastic Inference (EI) instance to use for the production variant | Yes | Yes | String| ml.eia1.medium, ml.eia1.large, ml.eia1.xlarge | |
update_endpoint | Updates the end point if it exists | Yes | Yes | Bool | | False |

Notes:
* Please use the links in the [Resources section](#Resources) for detailed information on each input parameter and SageMaker APIs used in this component
* The parameters, `model_name_1` through `3`, is intended to be output of [create model component](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/model) from previous steps in the pipeline. `model_name_[1, 3]` and other parameters for a production variant can be specified directly as well if the component is being used on its own.

## Outputs
Name | Description
:--- | :----------
endpoint_name | HTTPS Endpoint URL where client applications can send requests using [InvokeEndpoint](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_InvokeEndpoint.html) API

## Requirements
* [Kubeflow pipelines SDK](https://www.kubeflow.org/docs/pipelines/sdk/install-sdk/)
* [Kubeflow set-up](https://www.kubeflow.org/docs/aws/deploy/install-kubeflow/)

## Samples
### Integrated into a pipeline
MNIST Classification pipeline: [Pipeline](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/aws-samples/mnist-kmeans-sagemaker/mnist-classification-pipeline.py) | [Steps](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/aws-samples/mnist-kmeans-sagemaker/README.md)

## Resources
* Create Endpoint Configuration
  * [Create Endpoint Configuration API documentation](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateEndpointConfig.html)
  * [Boto3 API reference](https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_endpoint_config)
* Create Endpoint
  * [Create Endpoint API documentation](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateEndpoint.html)
  * [Boto3 API reference](https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_endpoint)
