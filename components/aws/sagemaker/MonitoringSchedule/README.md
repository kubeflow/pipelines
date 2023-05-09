# SageMaker Monitoring Schedule Kubeflow Pipelines component v2

Component to create a monitoring schedule that regularly starts Amazon SageMaker Processing Jobs to monitor the data captured for an Amazon SageMaker Endpoint.

The component can be used alone or with one of 4 job definition components:
- [Data Quality Job Definition](../DataQualityJobDefinition)
- [Model Bias Job Definition](../ModelBiasJobDefinition)
- [Model Explainability Job Definition](../ModelExplainabilityJobDefinition)
- [Model Quality Job Definition](../ModelQualityJobDefinition)

### SageMaker Kubeflow Pipeline component versioning
See the SageMaker Components for Kubeflow Pipelines versions section in [SageMaker Components for Kubeflow Pipelines](https://docs.aws.amazon.com/sagemaker/latest/dg/kubernetes-sagemaker-components-for-kubeflow-pipelines.html#kubeflow-pipeline-components) to learn about the differences between the version 1 and version 2 components.

### Kubeflow Pipelines backend compatibility
SageMaker components are currently supported with Kubeflow pipelines backend v1. This means, you will have to use KFP sdk 1.8.x to create your pipelines.

## Getting Started
Follow [this guide](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples#prerequisites) to setup the prerequisites for MonitoringSchedule depending on your deployment.

## Input Parameters
Find the high level component input parameters and their description in the [component's input specification](./component.yaml). The parameters with `JsonObject` or `JsonArray` type inputs have nested fields, you will have to refer to the [MonitoringSchedule CRD specification](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/monitoringschedule/) for the respective structure and pass the input in JSON format. 

A quick way to see the converted JSON style input is to copy the [sample MonitoringSchedule spec](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/monitoringschedule/#spec) and convert it to JSON using a YAML to JSON converter like [this website](https://jsonformatter.org/yaml-to-json).

For example, `monitoringScheduleConfig` is of type `object` and has the following structure:
```
monitoringScheduleConfig: 
    monitoringType: DataQuality
    scheduleConfig: 
        scheduleExpression: cron(0 * ? * * *)
    monitoringJobDefinitionName: <job_definition_name>
```

The JSON style input for the above parameter would be:
```
monitoring_schedule_config = {
    "monitoringType": "DataQuality",
    "scheduleConfig": {"scheduleExpression": "cron(0 * ? * * *)"},
    "monitoringJobDefinitionName": <job_definition_name>,
}
```

For a more detailed explanation of parameters, please refer to the [AWS SageMaker API Documentation for CreateMonitoringSchedule](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateMonitoringSchedule.html).

## References
- [Monitor models for data and model quality, bias, and explainability](https://docs.aws.amazon.com/sagemaker/latest/dg/model-monitor.html)
- [Monitoring Schedule CRD specification](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/monitoringschedule/)
- [AWS SageMaker API Documentation for CreateMonitoringSchedule](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateMonitoringSchedule.html)