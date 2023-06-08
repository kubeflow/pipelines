#!/usr/bin/env python3


import ast
import json
import random
import kfp
from kfp import components
from kfp import dsl

sagemaker_hpo_op = components.load_component_from_file(
    "../../../../components/aws/sagemaker/hyperparameter_tuning/component.yaml"
)
sagemaker_process_op = components.load_component_from_file(
    "../../../../components/aws/sagemaker/process/component.yaml"
)
sagemaker_train_op = components.load_component_from_file(
    "../../../../components/aws/sagemaker/TrainingJob/component.yaml"
)
sagemaker_Model_op = components.load_component_from_file("../../../../components/aws/sagemaker/Modelv2/component.yaml")
sagemaker_EndpointConfig_op = components.load_component_from_file(
    "../../../../components/aws/sagemaker/EndpointConfig/component.yaml"
)
sagemaker_Endpoint_op = components.load_component_from_file(
    "../../../../components/aws/sagemaker/Endpoint/component.yaml"
)
sagemaker_batch_transform_op = components.load_component_from_file(
    "../../../../components/aws/sagemaker/batch_transform/component.yaml"
)


def processing_input(input_name, s3_uri, local_path):
    return {
        "InputName": input_name,
        "S3Input": {
            "S3Uri": s3_uri,
            "LocalPath": local_path,
            "S3DataType": "S3Prefix",
            "S3InputMode": "File",
        },
    }


def processing_output(output_name, s3_uri, local_path):
    return {
        "OutputName": output_name,
        "S3Output": {
            "S3Uri": s3_uri,
            "LocalPath": local_path,
            "S3UploadMode": "EndOfJob",
        },
    }


def training_input(input_name, s3_uri):
    return {
        "ChannelName": input_name,
        "DataSource": {"S3DataSource": {"S3Uri": s3_uri, "S3DataType": "S3Prefix"}},
    }


def model_input(train_image, model_artifact_url):
    return {
        "containerHostname": "mnist-kmeans",
        "image": train_image,
        "mode": "SingleModel",
        "modelDataURL": model_artifact_url,
    }


def get_production_variants(model_name):
    return [
    {
        "initialInstanceCount": 1,
        "initialVariantWeight": 1,
        "instanceType": "ml.t2.medium",
        "modelName": model_name,
        "variantName": "mnist-kmeans-sample",
        "volumeSizeInGB": 5,
    }
]


@dsl.pipeline(
    name="MNIST Classification pipeline",
    description="MNIST Classification using KMEANS in SageMaker",
)
def mnist_classification(role_arn="", bucket_name=""):
    # Common component inputs
    region = "us-east-1"
    instance_type = "ml.m5.2xlarge"
    train_image = "382416733822.dkr.ecr.us-east-1.amazonaws.com/kmeans:1"

    # Training input and output location based on bucket name
    hpo_channels = [
        training_input("train", f"s3://{bucket_name}/mnist_kmeans_example/train_data"),
        training_input("test", f"s3://{bucket_name}/mnist_kmeans_example/test_data"),
    ]
    train_output_location = f"s3://{bucket_name}/mnist_kmeans_example/output"

    process = sagemaker_process_op(
        role=role_arn,
        region=region,
        image="763104351884.dkr.ecr.us-east-1.amazonaws.com/pytorch-training:1.5.0-cpu-py36-ubuntu16.04",
        instance_type=instance_type,
        container_entrypoint=[
            "python",
            "/opt/ml/processing/code/kmeans_preprocessing.py",
        ],
        input_config=[
            processing_input(
                "mnist_tar",
                "s3://sagemaker-sample-files/datasets/image/MNIST/mnist.pkl.gz",
                "/opt/ml/processing/input",
            ),
            processing_input(
                "source_code",
                f"s3://{bucket_name}/mnist_kmeans_example/processing_code/kmeans_preprocessing.py",
                "/opt/ml/processing/code",
            ),
        ],
        output_config=[
            processing_output(
                "train_data",
                f"s3://{bucket_name}/mnist_kmeans_example/",
                "/opt/ml/processing/output_train/",
            ),
            processing_output(
                "test_data",
                f"s3://{bucket_name}/mnist_kmeans_example/",
                "/opt/ml/processing/output_test/",
            ),
            processing_output(
                "valid_data",
                f"s3://{bucket_name}/mnist_kmeans_example/input/",
                "/opt/ml/processing/output_valid/",
            ),
        ],
    )

    hpo = sagemaker_hpo_op(
        region=region,
        image=train_image,
        metric_name="test:msd",
        metric_type="Minimize",
        static_parameters={"k": "10", "feature_dim": "784"},
        integer_parameters=[
            {"Name": "mini_batch_size", "MinValue": "500", "MaxValue": "600"},
            {"Name": "extra_center_factor", "MinValue": "10", "MaxValue": "20"},
        ],
        categorical_parameters=[
            {"Name": "init_method", "Values": ["random", "kmeans++"]}
        ],
        channels=hpo_channels,
        output_location=train_output_location,
        instance_type=instance_type,
        max_num_jobs=3,
        max_parallel_jobs=2,
        role=role_arn,
    ).after(process)

    trainingJobName = "sample-mnist-v2-trainingjob" + str(random.randint(0, 999999))

    training = sagemaker_train_op(
        region=region,
        algorithm_specification={
            "trainingImage": train_image,
            "trainingInputMode": "File",
        },
        hyper_parameters=hpo.outputs["best_hyperparameters"],
        input_data_config=[
            {
                "channelName": "train",
                "dataSource": {
                    "s3DataSource": {
                        "s3DataType": "S3Prefix",
                        "s3URI": f"s3://{bucket_name}/mnist_kmeans_example/train_data",
                        "s3DataDistributionType": "FullyReplicated",
                    },
                },
                "compressionType": "None",
                "RecordWrapperType": "None",
                "InputMode": "File",
            }
        ],
        output_data_config={"s3OutputPath": f"s3://{bucket_name}"},
        resource_config={
            "instanceCount": 1,
            "instanceType": "ml.m4.xlarge",
            "volumeSizeInGB": 5,
        },
        role_arn=role_arn,
        training_job_name=trainingJobName,
        stopping_condition={"maxRuntimeInSeconds": 3600},
    )

    def get_s3_model_artifact(model_artifacts) -> str:
        import ast

        model_artifacts = ast.literal_eval(model_artifacts)
        return model_artifacts["s3ModelArtifacts"]

    get_s3_model_artifact_op = kfp.components.create_component_from_func(
        get_s3_model_artifact, output_component_file="get_s3_model_artifact.yaml"
    )
    model_artifact_url = get_s3_model_artifact_op(
        training.outputs["model_artifacts"]
    ).output

    Model = sagemaker_Model_op(
        region=region,
        execution_role_arn=role_arn,
        primary_container=model_input(train_image, model_artifact_url),
    )
    model_name =  Model.outputs["sagemaker_resource_name"]
    EndpointConfig = sagemaker_EndpointConfig_op(
        region=region,
        production_variants=get_production_variants(model_name),
    )

    endpoint_config_name = EndpointConfig.outputs["sagemaker_resource_name"]

    Endpoint = sagemaker_Endpoint_op(
        region=region,
        endpoint_config_name=endpoint_config_name,
    )


    sagemaker_batch_transform_op(
        region=region,
        model_name=model_name,
        instance_type=instance_type,
        batch_strategy="MultiRecord",
        input_location=f"s3://{bucket_name}/mnist_kmeans_example/input",
        content_type="text/csv",
        split_type="Line",
        output_location=f"s3://{bucket_name}/mnist_kmeans_example/output",
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(mnist_classification, __file__ + ".tar.gz")
    print("#####################Pipeline compiled########################")
