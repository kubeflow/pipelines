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

sagemaker_model_op = components.load_component_from_file("../../model/component.yaml")


@dsl.pipeline(
    name="Create Model in SageMaker", description="SageMaker model component test"
)
def create_model_pipeline(
    region="",
    endpoint_url="",
    image="",
    model_name="",
    model_artifact_url="",
    network_isolation="",
    role="",
):
    sagemaker_model_op(
        region=region,
        endpoint_url=endpoint_url,
        model_name=model_name,
        image=image,
        model_artifact_url=model_artifact_url,
        network_isolation=network_isolation,
        role=role,
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        create_model_pipeline, "SageMaker_create_model_pipeline" + ".yaml"
    )
