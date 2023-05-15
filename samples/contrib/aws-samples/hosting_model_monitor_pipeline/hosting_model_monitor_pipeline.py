#!/usr/bin/env python3

import kfp
import sagemaker
import os
from kfp import components
from kfp import dsl
from datetime import datetime

sagemaker_Model_op = components.load_component_from_url(
    "https://raw.githubusercontent.com/kubeflow/pipelines/master/components/aws/sagemaker/Modelv2/component.yaml"
)

sagemaker_EndpointConfig_op = components.load_component_from_url(
    "https://raw.githubusercontent.com/kubeflow/pipelines/master/components/aws/sagemaker/EndpointConfig/component.yaml"
)

sagemaker_Endpoint_op = components.load_component_from_url(
    "https://raw.githubusercontent.com/kubeflow/pipelines/master/components/aws/sagemaker/Endpoint/component.yaml"
)

sagemaker_DataQualityJobDefinition_op = components.load_component_from_url(
    "https://raw.githubusercontent.com/kubeflow/pipelines/master/components/aws/sagemaker/DataQualityJobDefinition/component.yaml"
)

sagemaker_MonitoringSchedule_op = components.load_component_from_url(
    "https://raw.githubusercontent.com/kubeflow/pipelines/master/components/aws/sagemaker/MonitoringSchedule/component.yaml"
)

# Fetch environment variables
SAGEMAKER_EXECUTION_ROLE_ARN = os.getenv("SAGEMAKER_EXECUTION_ROLE_ARN", "")
S3_BUCKET_NAME = os.getenv("S3_BUCKET_NAME", "")
SAGEMAKER_REGION = os.getenv("SAGEMAKER_REGION", "us-east-1")

name_prefix = datetime.now().strftime("%Y-%m-%d-%H-%M-%S")

# Parameters for Model
xgboost_image = sagemaker.image_uris.retrieve(
    framework="xgboost", region=SAGEMAKER_REGION, version="0.90-1"
)
primary_container = {
    "containerHostname": "xgboost",
    "environment": {"my_env_key": "my_env_value"},
    "image": xgboost_image,
    "mode": "SingleModel",
    "modelDataURL": f"s3://{S3_BUCKET_NAME}/model-monitor/xgb-churn-prediction-model.tar.gz",
}

# Parameters for EndpointConfig
data_capture_config = {
    "enableCapture": True,
    "captureOptions": [{"captureMode": "Input"}, {"captureMode": "Output"}],
    "initialSamplingPercentage": 100,
    "destinationS3URI": f"s3://{S3_BUCKET_NAME}/model-monitor/datacapture",
}

# Parameters for DataQualityJobDefinition
model_monitor_image = sagemaker.image_uris.retrieve(
    framework="model-monitor", region=SAGEMAKER_REGION
)
data_quality_app_specification = {
    "imageURI": model_monitor_image,
}

data_quality_baseline_config = {
    "constraintsResource": {
        "s3URI": f"s3://{S3_BUCKET_NAME}/model-monitor/baselining/data_quality/constraints.json",
    },
    "statisticsResource": {
        "s3URI": f"s3://{S3_BUCKET_NAME}/model-monitor/baselining/data_quality/statistics.json",
    },
}

data_quality_job_output_config = {
    "monitoringOutputs": [
        {
            "s3Output": {
                "localPath": "/opt/ml/processing/output",
                "s3URI": f"s3://{S3_BUCKET_NAME}/model-monitor/reports/data-quality-job-definition/",
                "s3UploadMode": "Continuous",
            }
        }
    ]
}

job_resources = {
    "clusterConfig": {
        "instanceCount": 1,
        "instanceType": "ml.m5.large",
        "volumeSizeInGB": 20,
    }
}

stopping_condition = {"maxRuntimeInSeconds": 1800}


@dsl.pipeline(
    name="Hosting_Model_Monitoring", description="SageMaker Hosting and Model Monitor"
)
def Hosting_Model_Monitoring(
    region=SAGEMAKER_REGION,
    execution_role_arn=SAGEMAKER_EXECUTION_ROLE_ARN,
    primary_container=primary_container,
    data_capture_config=data_capture_config,
    data_quality_app_specification=data_quality_app_specification,
    data_quality_baseline_config=data_quality_baseline_config,
    data_quality_job_output_config=data_quality_job_output_config,
    job_resources=job_resources,
    stopping_condition=stopping_condition,
    model_name=name_prefix + "-model",
    endpoint_config_name=name_prefix + "-endpointcfg",
    endpoint_name=name_prefix + "-endpoint",
    job_definition_name=name_prefix + "-data-qual-job-defi",
    monitoring_schedule_name=name_prefix + "-monitoring-schedule",
):
    Model = sagemaker_Model_op(
        region=region,
        execution_role_arn=execution_role_arn,
        model_name=model_name,
        primary_container=primary_container,
    )

    production_variants_ = [
        {
            "initialInstanceCount": 1,
            "initialVariantWeight": 1,
            "instanceType": "ml.m5.large",
            "modelName": Model.outputs["sagemaker_resource_name"],
            "variantName": "instanceVariant",
            "volumeSizeInGB": 10,
        }
    ]

    EndpointConfig = sagemaker_EndpointConfig_op(
        region=region,
        endpoint_config_name=endpoint_config_name,
        data_capture_config=data_capture_config,
        production_variants=production_variants_,
    )

    Endpoint = sagemaker_Endpoint_op(
        region=region,
        endpoint_config_name=EndpointConfig.outputs["sagemaker_resource_name"],
        endpoint_name=endpoint_name,
    )

    data_quality_job_input = {
        "endpointInput": {
            "endpointName": Endpoint.outputs["sagemaker_resource_name"],
            "localPath": "/opt/ml/processing/input/endpoint",
            "s3DataDistributionType": "FullyReplicated",
            "s3InputMode": "File",
        }
    }

    DataQualityJobDefinition = sagemaker_DataQualityJobDefinition_op(
        region=region,
        data_quality_app_specification=data_quality_app_specification,
        data_quality_baseline_config=data_quality_baseline_config,
        data_quality_job_input=data_quality_job_input,
        data_quality_job_output_config=data_quality_job_output_config,
        job_definition_name=job_definition_name,
        job_resources=job_resources,
        role_arn=SAGEMAKER_EXECUTION_ROLE_ARN,
        stopping_condition=stopping_condition,
    )

    monitoring_schedule_config = {
        "monitoringType": "DataQuality",
        "scheduleConfig": {"scheduleExpression": "cron(0 * ? * * *)"},
        "monitoringJobDefinitionName": DataQualityJobDefinition.outputs[
            "sagemaker_resource_name"
        ],
    }

    MonitoringSchedule = sagemaker_MonitoringSchedule_op(
        region=region,
        monitoring_schedule_config=monitoring_schedule_config,
        monitoring_schedule_name=monitoring_schedule_name,
    )


kfp.compiler.Compiler().compile(Hosting_Model_Monitoring, __file__ + ".tar.gz")
print("=================Pipeline compiled=================")
print("Name prefix: ", name_prefix)
print(
    f"""To delete the resources created by this pipeline, run the following commands:
    export NAMESPACE=<xx> # Change it to your Kubeflow name space
    export NAME_PREFIX={name_prefix}
    kubectl delete MonitoringSchedule $NAME_PREFIX-monitoring-schedule -n $NAMESPACE
    kubectl delete DataQualityJobDefinition $NAME_PREFIX-data-qual-job-defi -n $NAMESPACE
    kubectl delete Endpoint $NAME_PREFIX-endpoint -n $NAMESPACE
    kubectl delete EndpointConfig $NAME_PREFIX-endpointcfg -n $NAMESPACE
    kubectl delete Model  $NAME_PREFIX-model -n $NAMESPACE"""
)
