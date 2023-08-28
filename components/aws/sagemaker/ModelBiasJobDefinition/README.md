# SageMaker Model Bias Job Definition Kubeflow Pipelines component v2

## Overview
Component to create the definition for a model bias job.

The component can be used with [Monitoring Schedule component](../MonitoringSchedule) to create a monitoring schedule that regularly starts Amazon SageMaker Processing Jobs to monitor the data captured for an Amazon SageMaker Endpoint.

See the SageMaker Components for Kubeflow Pipelines versions section in [SageMaker Components for Kubeflow Pipelines](https://docs.aws.amazon.com/sagemaker/latest/dg/kubernetes-sagemaker-components-for-kubeflow-pipelines.html#kubeflow-pipeline-components) to learn about the differences between the version 1 and version 2 components.

### Kubeflow Pipelines backend compatibility
SageMaker components are currently supported with Kubeflow pipelines backend v1. This means, you will have to use KFP sdk 1.8.x to create your pipelines.

## Getting Started
Follow [this guide](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples#prerequisites) to setup the prerequisites for ModelBiasJobDefinition depending on your deployment.

## Input Parameters
Find the high level component input parameters and their description in the [component's input specification](./component.yaml). The parameters with `JsonObject` or `JsonArray` type inputs have nested fields, you will have to refer to the [ModelBiasJobDefinition CRD specification](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/modelbiasjobdefinition/) for the respective structure and pass the input in JSON format. 

A quick way to see the converted JSON style input is to copy the [sample ModelBiasJobDefinition spec](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/modelbiasjobdefinition/#spec) and convert it to JSON using a YAML to JSON converter like [this website](https://jsonformatter.org/yaml-to-json).

For example, `modelBiasJobInput` is of type `object` and has the following structure:
```
modelBiasJobInput: 
  endpointInput: 
    endTimeOffset: string
    endpointName: string
    featuresAttribute: string
    inferenceAttribute: string
    localPath: string
    probabilityAttribute: string
    probabilityThresholdAttribute: number
    s3DataDistributionType: string
    s3InputMode: string
    startTimeOffset: string
  groundTruthS3Input: 
    s3URI: string
```

The JSON style input for the above parameter would be:
```
model_bias_job_input = {
    "endpointInput": {
        "endpointName": "<endpoint-name>",  # change to your endpoint
        "localPath": "<path>",
        "s3InputMode": "File",
        "s3DataDistributionType": "FullyReplicated",
        "probabilityThresholdAttribute": 0.8,
        "startTimeOffset": "-PT1H",
        "endTimeOffset": "-PT0H",
    },
    "groundTruthS3Input": {
        "s3URI": "s3://<path-to-directory>/ground_truth_data"
    },
}
```

For a more detailed explanation of parameters, please refer to the [AWS SageMaker API Documentation for CreateModelBiasJobDefinition](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateModelBiasJobDefinition.html).

## References
- [Monitor models for data and model quality, bias, and explainability](https://docs.aws.amazon.com/sagemaker/latest/dg/model-monitor.html)
- [Model Bias Job Definition CRD specification](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/modelbiasjobdefinition/)
- [AWS SageMaker API Documentation for CreateModelBiasJobDefinition](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateModelBiasJobDefinition.html).