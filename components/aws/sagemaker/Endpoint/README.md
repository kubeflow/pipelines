# SageMaker Endpoint Kubeflow Pipelines component v2

Component to create [SageMaker Endpoints](https://docs.aws.amazon.com/sagemaker/latest/dg/deploy-model.html) in a Kubeflow Pipelines workflow.


## Overview

The Amazon SageMaker components for Kubeflow Pipelines version 1(v1.1.x or below) uses [Boto3 (AWS SDK for Python)](https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html) as the backend to create and manage resources on SageMaker. SageMaker components version 2(v2.0.0-alpha2 or above) uses the [ACK Service Controller for SageMaker](https://github.com/aws-controllers-k8s/sagemaker-controller) to do the same. AWS introduced [ACK](https://aws-controllers-k8s.github.io/community/) to facilitate a Kubernetes-native way of managing AWS Cloud resources. ACK includes a set of AWS service-specific controllers, one of which is the SageMaker controller. The SageMaker controller makes it easier for machine learning developers and data scientists who use Kubernetes as their control plane to train, tune, and deploy machine learning models in Amazon SageMaker.

Creating SageMaker resouces using the controller allows you to create and monitor the resources as part of a Kubeflow Pipelines workflow(same as version 1 of components) and additionally provides you a flexible and consistent experience to manage the SageMaker resources from other environments such as using the Kubernetes command line tool(kubectl) or the other Kubeflow applications such as Notebooks.

Endpoint is one of the three components(along with EndpointConfig and Model) you would use to create a Hosting deployment on Sagemaker.

### Kubeflow Pipelines backend compatibility
SageMaker components are currently supported with Kubeflow pipelines backend v1. This means, you will have to use KFP sdk 1.8.x to create your pipelines.

## Getting Started

Follow [this guide](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples#prerequisites) to setup the prerequisites for Endpoint depending on your deployment.

## Inputs Parameters
Find the high level component input parameters and their description in the [component's input specification](./component.yaml). The parameters with `JsonObject` or `JsonArray` type inputs have nested fields, you will have to refer to the [Endpoint CRD specification](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/endpoint/) for the respective structure and pass the input in JSON format. 

A quick way to see the converted JSON style input is to copy the [sample Endpoint spec](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/endpoint/#spec) and convert it to JSON using a YAML to JSON converter like [this website](https://jsonformatter.org/yaml-to-json).

For e.g. the `endpointConfigName` in the `Endpoint` CRD looks like:

```
endpointConfigName: string
```

The `endpointConfigName` input for the component would be:

```
endpointConfigName = "my-config"
```

You might also want to look at the [Endpoint API reference](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateEndpoint.html) for a detailed explaination of parameters.

## References
- [Inference on SageMaker](https://docs.aws.amazon.com/sagemaker/latest/dg/deploy-model.html)
- [Endpoint CRD specification](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/endpoint/)
- [Endpoint API reference](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateEndpoint.html)