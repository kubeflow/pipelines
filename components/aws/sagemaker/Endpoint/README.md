# SageMaker Endpoint Kubeflow Pipelines component v2

## Overview

Endpoint is one of the three components(along with EndpointConfig and Model) you would use to create a Hosting deployment on Sagemaker.

Component to create [SageMaker Endpoints](https://docs.aws.amazon.com/sagemaker/latest/dg/deploy-model.html) in a Kubeflow Pipelines workflow.

See the SageMaker Components for Kubeflow Pipelines versions section in [SageMaker Components for Kubeflow Pipelines](https://docs.aws.amazon.com/sagemaker/latest/dg/kubernetes-sagemaker-components-for-kubeflow-pipelines.html#kubeflow-pipeline-components) to learn about the differences between the version 1 and version 2 components.


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