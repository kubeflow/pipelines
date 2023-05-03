# SageMaker Model Kubeflow Pipelines component v2

## Overview

Model is one of the three components(along with Endpoint and EndpointConfig) you would use to create a Hosting deployment on Sagemaker.

Component to create [SageMaker Models](https://docs.aws.amazon.com/sagemaker/latest/dg/deploy-model.html) in a Kubeflow Pipelines workflow.

See the SageMaker Components for Kubeflow Pipelines versions section in [SageMaker Components for Kubeflow Pipelines](https://docs.aws.amazon.com/sagemaker/latest/dg/kubernetes-sagemaker-components-for-kubeflow-pipelines.html#kubeflow-pipeline-components) to learn about the differences between the version 1 and version 2 components.

### Kubeflow Pipelines backend compatibility
SageMaker components are currently supported with Kubeflow pipelines backend v1. This means, you will have to use KFP sdk 1.8.x to create your pipelines.

## Getting Started

Follow [this guide](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples#prerequisites) to setup the prerequisites for Model depending on your deployment.

## Inputs Parameters
Find the high level component input parameters and their description in the [component's input specification](./component.yaml). The parameters with `JsonObject` or `JsonArray` type inputs have nested fields, you will have to refer to the [TrainingJob CRD specification](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/model/) for the respective structure and pass the input in JSON format. 

A quick way to see the converted JSON style input is to copy the [sample Model spec](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/model/#spec) and convert it to JSON using a YAML to JSON converter like [this website](https://jsonformatter.org/yaml-to-json).

For e.g. the `primaryContainer` in the `Model` CRD looks like:

```
primaryContainer: 
  containerHostname: string
  environment: {}
  image: string
  imageConfig: 
    repositoryAccessMode: string
    repositoryAuthConfig: 
      repositoryCredentialsProviderARN: string
  inferenceSpecificationName: string
  mode: string
  modelDataURL: string
  modelPackageName: string
  multiModelConfig: 
    modelCacheSetting: string
```

The `primaryContainer` input for the component would be (not all parameters are included):

```
primaryContainer = {
    "containerHostname": "xgboost",
    "environment": {"my_env_key": "my_env_value"},
    "image": "257758044811.dkr.ecr.us-east-2.amazonaws.com/sagemaker-xgboost:0.90-1-cpu-py3",
    "modelDataURL": "s3://<path to model>",
    "modelPackageName": "SingleModel",
}
```

You might also want to look at the [Model API reference](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateModel.html) for a detailed explaination of parameters.

## References
- [Inference on SageMaker](https://docs.aws.amazon.com/sagemaker/latest/dg/deploy-model.html)
- [Model CRD specification](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/model/)
- [Model API reference](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateModel.html)