#!/usr/bin/env python3

import kfp
from kfp import components, dsl
from datetime import datetime
import os

sagemaker_MonitoringSchedule_op = components.load_component_from_file(
    "../../MonitoringSchedule/component.yaml"  # change to your path
)

sagemaker_DataQualityJobDefinition_op = components.load_component_from_file(
    "../../DataQualityJobDefinition/component.yaml"  # change to your path
)

# Change name every time you run the pipeline
job_definition_name = "dq-" + datetime.now().strftime("%Y-%m-%d-%H-%M-%S")
monitoring_schedule_name = "schedule-dq-" + datetime.now().strftime("%Y-%m-%d-%H-%M-%S")

# S3 bucket where dataset is uploaded
S3_BUCKET_NAME = os.getenv("S3_BUCKET_NAME", "")

# Role with AmazonSageMakerFullAccess and AmazonS3FullAccess
# example arn:aws:iam::123456789012:role/service-role/AmazonSageMaker-ExecutionRole
ROLE_ARN = os.getenv("SAGEMAKER_EXECUTION_ROLE_ARN", "")

REGION = os.getenv("SAGEMAKER_REGION", "us-east-1")

ENDPOINT_NAME = os.getenv("ENDPOINT_NAME", "")

# If you are not on us-east-1 you can find an image URI here https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-algo-docker-registry-paths.html
data_quality_app_specification = {
    "imageURI": "156813124566.dkr.ecr.us-east-1.amazonaws.com/sagemaker-model-monitor-analyzer",
}

data_quality_baseline_config = {
    "constraintsResource": {
        "s3URI": f"s3://{S3_BUCKET_NAME}/model-monitor/baselining/data_quality/constraints.json",  # change to your bucket
    },
    "statisticsResource": {
        "s3URI": f"s3://{S3_BUCKET_NAME}/model-monitor/baselining/data_quality/statistics.json"  # change to your bucket
    },
}

data_quality_job_input = {
    "endpointInput": {
        "endpointName": f"ENDPOINT_NAME",  # change to your endpoint
        "localPath": "/opt/ml/processing/input/endpoint",
        "s3DataDistributionType": "FullyReplicated",
        "s3InputMode": "File",
    }
}

data_quality_job_output_config = {
    "monitoringOutputs": [
        {
            "s3Output": {
                "localPath": "/opt/ml/processing/output",
                "s3URI": f"s3://{S3_BUCKET_NAME}/model-monitor/reports/data-quality-job-definition/",  # change to your bucket
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

monitoring_schedule_config = {
    "monitoringType": "DataQuality",
    "scheduleConfig": {"scheduleExpression": "cron(0 * ? * * *)"},
    "monitoringJobDefinitionName": job_definition_name,
}


@dsl.pipeline(
    name="MonitoringSchedule", description="SageMaker MonitoringSchedule component"
)
def MonitoringSchedule(
    region=REGION,
    data_quality_app_specification=data_quality_app_specification,
    data_quality_baseline_config=data_quality_baseline_config,
    data_quality_job_input=data_quality_job_input,
    data_quality_job_output_config=data_quality_job_output_config,
    job_definition_name=job_definition_name,
    job_resources=job_resources,
    network_config=None,
    role_arn=ROLE_ARN,
    stopping_condition=stopping_condition,
    monitoring_schedule_config=monitoring_schedule_config,
    monitoring_schedule_name=monitoring_schedule_name,
):
    DataQualityJobDefinition = sagemaker_DataQualityJobDefinition_op(
        region=region,
        data_quality_app_specification=data_quality_app_specification,
        data_quality_baseline_config=data_quality_baseline_config,
        data_quality_job_input=data_quality_job_input,
        data_quality_job_output_config=data_quality_job_output_config,
        job_definition_name=job_definition_name,
        job_resources=job_resources,
        network_config=network_config,
        role_arn=role_arn,
        stopping_condition=stopping_condition,
    )

    MonitoringSchedule = sagemaker_MonitoringSchedule_op(
        region=region,
        monitoring_schedule_config=monitoring_schedule_config,
        monitoring_schedule_name=monitoring_schedule_name,
    ).after(DataQualityJobDefinition)


if __name__ == "__main__":
    # compile the pipeline, unzip it and get pipeline.yaml
    kfp.compiler.Compiler().compile(MonitoringSchedule, __file__ + ".tar.gz")

    print("#####################Pipeline compiled########################")
