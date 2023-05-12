# SageMaker Data Quality Job Definition Kubeflow Pipelines component v2

## Overview
Component to create a definition for a job that monitors data quality and drift.

The component can be used with [Monitoring Schedule component](../MonitoringSchedule) to create a monitoring schedule that regularly starts Amazon SageMaker Processing Jobs to monitor the data captured for an Amazon SageMaker Endpoint.

See the SageMaker Components for Kubeflow Pipelines versions section in [SageMaker Components for Kubeflow Pipelines](https://docs.aws.amazon.com/sagemaker/latest/dg/kubernetes-sagemaker-components-for-kubeflow-pipelines.html#kubeflow-pipeline-components) to learn about the differences between the version 1 and version 2 components.

### Kubeflow Pipelines backend compatibility
SageMaker components are currently supported with Kubeflow pipelines backend v1. This means, you will have to use KFP sdk 1.8.x to create your pipelines.

## Getting Started

Follow [this guide](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples#prerequisites) to setup the prerequisites for DataQualityJobDefinition depending on your deployment.

## Input Parameters
Find the high level component input parameters and their description in the [component's input specification](./component.yaml). The parameters with `JsonObject` or `JsonArray` type inputs have nested fields, you will have to refer to the [DataQualityJobDefinition CRD specification](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/dataqualityjobdefinition/) for the respective structure and pass the input in JSON format. 

A quick way to see the converted JSON style input is to copy the [sample DataQualityJobDefinition spec](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/dataqualityjobdefinition/#spec) and convert it to JSON using a YAML to JSON converter like [this website](https://jsonformatter.org/yaml-to-json).

For example, `dataQualityBaselineConfig` is of type `object` and has the following structure:
```
dataQualityBaselineConfig: 
  baseliningJobName: string
  constraintsResource: 
    s3URI: string
  statisticsResource: 
    s3URI: string
```

The JSON style input for the above parameter would be:
```
data_quality_baseline_config = {
    "constraintsResource": {
        "s3URI": f"s3://<path-to-file>/constraints.json",
    },
    "statisticsResource": {
        "s3URI": f"s3://<path-to-file>/statistics.json"
    },
}
```

For a more detailed explanation of parameters, please refer to the [AWS SageMaker API Documentation for CreateDataQualityJobDefinition](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateDataQualityJobDefinition.html).

## References
- [Monitor models for data and model quality, bias, and explainability](https://docs.aws.amazon.com/sagemaker/latest/dg/model-monitor.html)
- [Data Quality Job Definition CRD specification](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/dataqualityjobdefinition/)
- [AWS SageMaker API Documentation for CreateDataQualityJobDefinition](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateDataQualityJobDefinition.html).