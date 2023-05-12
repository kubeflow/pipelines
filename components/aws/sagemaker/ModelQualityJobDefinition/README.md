# SageMaker Model Quality Job Definition Kubeflow Pipelines component v2

## Overview
Component to creates a definition for a job that monitors model quality and drift.

The component can be used with [Monitoring Schedule component](../MonitoringSchedule) to create a monitoring schedule that regularly starts Amazon SageMaker Processing Jobs to monitor the data captured for an Amazon SageMaker Endpoint.

See the SageMaker Components for Kubeflow Pipelines versions section in [SageMaker Components for Kubeflow Pipelines](https://docs.aws.amazon.com/sagemaker/latest/dg/kubernetes-sagemaker-components-for-kubeflow-pipelines.html#kubeflow-pipeline-components) to learn about the differences between the version 1 and version 2 components.

### Kubeflow Pipelines backend compatibility
SageMaker components are currently supported with Kubeflow pipelines backend v1. This means, you will have to use KFP sdk 1.8.x to create your pipelines.

## Getting Started

Follow [this guide](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples#prerequisites) to setup the prerequisites for ModelQualityJobDefinition depending on your deployment.

## Input Parameters
Find the high level component input parameters and their description in the [component's input specification](./component.yaml). The parameters with `JsonObject` or `JsonArray` type inputs have nested fields, you will have to refer to the [ModelQualityJobDefinition CRD specification](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/modelqualityjobdefinition/) for the respective structure and pass the input in JSON format. 

A quick way to see the converted JSON style input is to copy the [sample ModelQualityJobDefinition spec](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/modelqualityjobdefinition/#spec) and convert it to JSON using a YAML to JSON converter like [this website](https://jsonformatter.org/yaml-to-json).

For example, `modelQualityAppSpecification` is of type `object` and has the following structure:
```
jobDefinitionName: string
```

The JSON style input for the above parameter would be:
```
job_definition_name = "<your-job-definition-name>"
```

For a more detailed explanation of parameters, please refer to the [AWS SageMaker API Documentation for CreateModelQualityJobDefinition](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateModelQualityJobDefinition.html).

## References
- [Monitor models for data and model quality, bias, and explainability](https://docs.aws.amazon.com/sagemaker/latest/dg/model-monitor.html)
- [Model Quality Job Definition CRD specification](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/modelqualityjobdefinition/)
- [AWS SageMaker API Documentation for CreateModelQualityJobDefinition](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateModelQualityJobDefinition.html).