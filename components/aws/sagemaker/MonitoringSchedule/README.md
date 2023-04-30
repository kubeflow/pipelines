Follow [this guide](../TrainingJob/README.md#getting-started) to get started with using the SageMaker Training Job pipeline component version 2.

## Input Parameters

Find the high level component input parameters and their description in the [component's input specification](./component.yaml). The parameters with `JsonObject` or `JsonArray` type inputs have nested fields, you will have to refer to the [MonitoringSchedule CRD specification](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/monitoringschedule/) for the respective structure and pass the input in JSON format. 

A quick way to see the converted JSON style input is to copy the [sample MonitoringSchedule spec](https://aws-controllers-k8s.github.io/community/reference/sagemaker/v1alpha1/monitoringschedule/#spec) and convert it to JSON using a YAML to JSON converter like [this website](https://jsonformatter.org/yaml-to-json).

For a more detailed explaination of parameters, please refer to the [AWS SageMaker API Documentation for CreateMonitoringSchedule](https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_CreateMonitoringSchedule.html).