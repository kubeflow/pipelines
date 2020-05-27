# Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You
# may not use this file except in compliance with the License. A copy of
# the License is located at
#
# http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is
# distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF
# ANY KIND, either express or implied. See the License for the specific
# language governing permissions and limitations under the License.

import kfp
from kfp import components
from kfp import dsl
from kfp.aws import use_aws_secret

sagemaker_train_op = components.load_component_from_file("../../train/component.yaml")


@dsl.pipeline(name="SageMaker Training", description="SageMaker training job test")
def training_pipeline(
    region="",
    endpoint_url="",
    image="",
    training_input_mode="",
    hyperparameters="",
    channels="",
    instance_type="",
    instance_count="",
    volume_size="",
    max_run_time="",
    model_artifact_path="",
    output_encryption_key="",
    network_isolation="",
    traffic_encryption="",
    spot_instance="",
    max_wait_time="",
    checkpoint_config="{}",
    role="",
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
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        training_pipeline, "SageMaker_training_pipeline" + ".yaml"
    )
