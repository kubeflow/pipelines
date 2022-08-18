#!/usr/bin/env python3

import kfp
from kfp import components
from kfp import dsl
import os
import random

# Training job component, path is relative from this directory.
sagemaker_training_op = components.load_component_from_file(
    "../../component.yaml"
)
# This section initializes complex data structures that will be used for the pipeline.

# S3 bucket where dataset is uploaded
S3_BUCKET_NAME = os.getenv("S3_BUCKET_NAME", "")

# Role with AmazonSageMakerFullAccess and AmazonS3FullAccess
# example arn:aws:iam::123456789012:role/service-role/AmazonSageMaker-ExecutionRole
ROLE_ARN = os.getenv("SAGEMAKER_EXECUTION_ROLE_ARN", "")

REGION = "us-east-1"

# The URL and tag of your ECR container
# If you are not on us-east-1 you can find an image URI here https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-algo-docker-registry-paths.html
TRAINING_IMAGE = "382416733822.dkr.ecr.us-east-1.amazonaws.com/kmeans:1"


def algorithm_specification(training_image):
    return {
        "trainingImage": training_image,
        "trainingInputMode": "File",
    }


resourceConfig = {
    "instanceCount": 1,
    "instanceType": "ml.m4.xlarge",
    "volumeSizeInGB": 10,
}


def training_input(s3_bucket_name):
    return [
        {
            "channelName": "train",
            "dataSource": {
                "s3DataSource": {
                    "s3DataType": "S3Prefix",
                    # change it to your input path of the train data: s3://<YOUR BUCKET>/mnist_kmeans_example/train_data
                    "s3URI": f"s3://{s3_bucket_name}/mnist_kmeans_example/train_data",
                    "s3DataDistributionType": "FullyReplicated",
                },
            },
            "compressionType": "None",
            "RecordWrapperType": "None",
            "InputMode": "File",
        }
    ]


def training_output(s3_bucket_name):
    return {"s3OutputPath": f"s3://{s3_bucket_name}"}


@dsl.pipeline(name="TrainingJob", description="SageMaker TrainingJob component")
def TrainingJob(
    s3_bucket_name=S3_BUCKET_NAME,
    sagemaker_role_arn=ROLE_ARN,
    region=REGION,
    training_image=TRAINING_IMAGE,
    hyper_parameters={"k": "10", "feature_dim": "784"},
    resource_config={
        "instanceCount": 1,
        "instanceType": "ml.m4.xlarge",
        "volumeSizeInGB": 10,
    },
):
    sagemaker_training_op(
        region=region,
        algorithm_specification=algorithm_specification(training_image),
        enable_inter_container_traffic_encryption=False,
        enable_network_isolation=False,
        hyper_parameters=hyper_parameters,
        input_data_config=training_input(s3_bucket_name),
        output_data_config=training_output(s3_bucket_name),
        resource_config=resource_config,
        role_arn=sagemaker_role_arn,
        stopping_condition={"maxRuntimeInSeconds": 3600},
    )


if __name__ == "__main__":
    # Compiling the pipeline
    kfp.compiler.Compiler().compile(TrainingJob, __file__ + ".tar.gz")

    print("#####################Pipeline compiled########################")
