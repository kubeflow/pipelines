#!/usr/bin/env python3


import kfp
import json
import copy
from kfp import components
from kfp import dsl

sagemaker_hpo_op = components.load_component_from_file("../../../../components/aws/sagemaker/hyperparameter_tuning/component.yaml")
sagemaker_process_op = components.load_component_from_file("../../../../components/aws/sagemaker/process/component.yaml")
sagemaker_train_op = components.load_component_from_file("../../../../components/aws/sagemaker/train/component.yaml")
sagemaker_model_op = components.load_component_from_file("../../../../components/aws/sagemaker/model/component.yaml")
sagemaker_deploy_op = components.load_component_from_file("../../../../components/aws/sagemaker/deploy/component.yaml")
sagemaker_batch_transform_op = components.load_component_from_file("../../../../components/aws/sagemaker/batch_transform/component.yaml")

# Update this to match the name of your bucket
my_bucket_name = "my-bucket"


def processing_input(input_name, s3_uri, local_path):
    return {
        "InputName": input_name,
        "S3Input": {
            "S3Uri": s3_uri,
            "LocalPath": local_path,
            "S3DataType": "S3Prefix",
            "S3InputMode": "File",
            "S3CompressionType": "None",
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
        "DataSource": {
            "S3DataSource": {
                "S3Uri": s3_uri,
                "S3DataType": "S3Prefix",
                "S3DataDistributionType": "FullyReplicated",
            }
        },
        "CompressionType": "None",
        "RecordWrapperType": "None",
        "InputMode": "File",
    }


hpoChannels = [
    training_input(
        "train", f"s3://{my_bucket_name}/mnist_kmeans_example/input/train_data"
    ),
    training_input(
        "test", f"s3://{my_bucket_name}/mnist_kmeans_example/input/test_data"
    ),
]
trainChannels = [
    training_input(
        "train", f"s3://{my_bucket_name}/mnist_kmeans_example/input/train_data"
    )
]


@dsl.pipeline(
    name="MNIST Classification pipeline",
    description="MNIST Classification using KMEANS in SageMaker",
)
def mnist_classification(
    region="us-east-1",
    # General component inputs
    output_encryption_key="",
    instance_type="ml.m5.2xlarge",
    instance_count=1,
    volume_size=50,
    max_run_time=3600,
    endpoint_url="",
    network_isolation=True,
    traffic_encryption=False,
    role_arn="",
    # Pre-processing inputs
    process_instance_type="ml.m5.large",
    process_instance_count=1,
    process_entrypoint=["python", "/opt/ml/processing/code/kmeans_preprocessing.py"],
    process_image="763104351884.dkr.ecr.us-east-1.amazonaws.com/pytorch-training:1.5.0-cpu-py36-ubuntu16.04",
    process_input_config=[
        processing_input(
            "mnist_tar",
            "s3://sagemaker-sample-data-us-east-1/algorithms/kmeans/mnist/mnist.pkl.gz",
            "/opt/ml/processing/input",
        ),
        processing_input(
            "source_code",
            f"s3://{my_bucket_name}/mnist_kmeans_example/processing_code/kmeans_preprocessing.py",
            "/opt/ml/processing/code",
        ),
    ],
    process_output_config=[
        processing_output(
            "train_data",
            f"s3://{my_bucket_name}/mnist_kmeans_example/input/",
            "/opt/ml/processing/output_train/",
        ),
        processing_output(
            "test_data",
            f"s3://{my_bucket_name}/mnist_kmeans_example/input/",
            "/opt/ml/processing/output_test/",
        ),
        processing_output(
            "valid_data",
            f"s3://{my_bucket_name}/mnist_kmeans_example/input/",
            "/opt/ml/processing/output_valid/",
        ),
    ],
    # HyperParameter Tuning inputs
    hpo_strategy="Bayesian",
    hpo_metric_name="test:msd",
    hpo_metric_type="Minimize",
    hpo_early_stopping_type="Off",
    hpo_static_parameters={"k": "10", "feature_dim": "784"},
    hpo_integer_parameters=[
        {"Name": "mini_batch_size", "MinValue": "500", "MaxValue": "600"},
        {"Name": "extra_center_factor", "MinValue": "10", "MaxValue": "20"},
    ],
    hpo_continuous_parameters=[],
    hpo_categorical_parameters=[
        {"Name": "init_method", "Values": ["random", "kmeans++"]}
    ],
    hpo_channels=hpoChannels,
    hpo_spot_instance=False,
    hpo_max_wait_time=3600,
    hpo_checkpoint_config={},
    hpo_max_num_jobs=9,
    hpo_max_parallel_jobs=3,
    # Training inputs
    train_image="382416733822.dkr.ecr.us-east-1.amazonaws.com/kmeans:1",
    train_input_mode="File",
    train_output_location=f"s3://{my_bucket_name}/mnist_kmeans_example/output",
    train_channels=trainChannels,
    train_spot_instance=False,
    train_max_wait_time=3600,
    train_checkpoint_config={},
    # Batch transform inputs
    batch_transform_instance_type="ml.m4.xlarge",
    batch_transform_input=f"s3://{my_bucket_name}/mnist_kmeans_example/input",
    batch_transform_data_type="S3Prefix",
    batch_transform_content_type="text/csv",
    batch_transform_compression_type="None",
    batch_transform_ouput=f"s3://{my_bucket_name}/mnist_kmeans_example/output",
    batch_transform_max_concurrent=4,
    batch_transform_max_payload=6,
    batch_strategy="MultiRecord",
    batch_transform_split_type="Line",
):
    process = sagemaker_process_op(
        role=role_arn,
        region=region,
        image=process_image,
        instance_type=instance_type,
        instance_count=process_instance_count,
        volume_size=volume_size,
        max_run_time=max_run_time,
        container_entrypoint=process_entrypoint,
        input_config=process_input_config,
        output_config=process_output_config
    )

    hpo = sagemaker_hpo_op(
        region=region,
        endpoint_url=endpoint_url,
        image=train_image,
        training_input_mode=train_input_mode,
        strategy=hpo_strategy,
        metric_name=hpo_metric_name,
        metric_type=hpo_metric_type,
        early_stopping_type=hpo_early_stopping_type,
        static_parameters=hpo_static_parameters,
        integer_parameters=hpo_integer_parameters,
        continuous_parameters=hpo_continuous_parameters,
        categorical_parameters=hpo_categorical_parameters,
        channels=hpo_channels,
        output_location=train_output_location,
        output_encryption_key=output_encryption_key,
        instance_type=instance_type,
        instance_count=instance_count,
        volume_size=volume_size,
        max_num_jobs=hpo_max_num_jobs,
        max_parallel_jobs=hpo_max_parallel_jobs,
        max_run_time=max_run_time,
        network_isolation=network_isolation,
        traffic_encryption=traffic_encryption,
        spot_instance=hpo_spot_instance,
        max_wait_time=hpo_max_wait_time,
        checkpoint_config=hpo_checkpoint_config,
        role=role_arn,
    ).after(process)

    training = sagemaker_train_op(
        region=region,
        endpoint_url=endpoint_url,
        image=train_image,
        training_input_mode=train_input_mode,
        hyperparameters=hpo.outputs["best_hyperparameters"],
        channels=train_channels,
        instance_type=instance_type,
        instance_count=instance_count,
        volume_size=volume_size,
        max_run_time=max_run_time,
        model_artifact_path=train_output_location,
        output_encryption_key=output_encryption_key,
        network_isolation=network_isolation,
        traffic_encryption=traffic_encryption,
        spot_instance=train_spot_instance,
        max_wait_time=train_max_wait_time,
        checkpoint_config=train_checkpoint_config,
        role=role_arn,
    )

    create_model = sagemaker_model_op(
        region=region,
        endpoint_url=endpoint_url,
        model_name=training.outputs["job_name"],
        image=training.outputs["training_image"],
        model_artifact_url=training.outputs["model_artifact_url"],
        network_isolation=network_isolation,
        role=role_arn,
    )

    prediction = sagemaker_deploy_op(
        region=region, endpoint_url=endpoint_url, model_name_1=create_model.output,
    )

    batch_transform = sagemaker_batch_transform_op(
        region=region,
        endpoint_url=endpoint_url,
        model_name=create_model.output,
        instance_type=batch_transform_instance_type,
        instance_count=instance_count,
        max_concurrent=batch_transform_max_concurrent,
        max_payload=batch_transform_max_payload,
        batch_strategy=batch_strategy,
        input_location=batch_transform_input,
        data_type=batch_transform_data_type,
        content_type=batch_transform_content_type,
        split_type=batch_transform_split_type,
        compression_type=batch_transform_compression_type,
        output_location=batch_transform_ouput,
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(mnist_classification, __file__ + ".zip")

