import kfp
import json
import utils

from kfp import components
from kfp import dsl
from kfp.aws import use_aws_secret

sagemaker_train_op = components.load_component_from_file("../../train/component.yaml")

CHANNELS = [
    {
        "ChannelName": "train",
        "DataSource": {
            "S3DataSource": {
                "S3Uri": "s3://{}/mnist_kmeans_example/data".format(
                    utils.get_s3_data_bucket()
                ),
                "S3DataType": "S3Prefix",
                "S3DataDistributionType": "FullyReplicated",
            }
        },
        "ContentType": "",
        "CompressionType": "None",
        "RecordWrapperType": "None",
        "InputMode": "File",
    }
]


@dsl.pipeline(name="SM Training pipeline", description="SageMaker training job test")
def simple_mnist_training(
    region=utils.get_region(),
    endpoint_url="",
    image="174872318107.dkr.ecr.us-west-2.amazonaws.com/kmeans:1",
    training_input_mode="File",
    hyperparameters='{"k": "10", "feature_dim": "784"}',
    channels=json.dumps(CHANNELS),
    instance_type="ml.p2.xlarge",
    instance_count="1",
    volume_size="80",
    max_run_time="3600",
    model_artifact_path="s3://{}/mnist_kmeans_example/output".format(
        utils.get_s3_data_bucket()
    ),
    output_encryption_key="",
    network_isolation="True",
    traffic_encryption="False",
    spot_instance="False",
    max_wait_time="3600",
    checkpoint_config="{}",
    role=utils.get_role_arn(),
):
    sagemaker_train_op(
        region=region,
        endpoint_url=endpoint_url,
        image=image,
        training_input_mode=training_input_mode,
        hyperparameters=hyperparameters,
        channels=channels,
        instance_type=instance_type,
        instance_count=instance_count,
        volume_size=volume_size,
        max_run_time=max_run_time,
        model_artifact_path=model_artifact_path,
        output_encryption_key=output_encryption_key,
        network_isolation=network_isolation,
        traffic_encryption=traffic_encryption,
        spot_instance=spot_instance,
        max_wait_time=max_wait_time,
        checkpoint_config=checkpoint_config,
        role=role,
    ).apply(use_aws_secret("aws-secret", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"))


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        simple_mnist_training, "simple_mnist_training" + ".yaml"
    )
