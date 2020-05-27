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

sagemaker_hpo_op = components.load_component_from_file(
    "../../hyperparameter_tuning/component.yaml"
)


@dsl.pipeline(
    name="SageMaker HyperParameter Tuning", description="SageMaker HPO job test"
)
def hpo_pipeline(
    region="",
    algorithm_name="",
    training_input_mode="",
    static_parameters="",
    integer_parameters="",
    channels="",
    categorical_parameters="",
    early_stopping_type="",
    max_parallel_jobs="",
    max_num_jobs="",
    metric_name="",
    metric_type="",
    hpo_strategy="",
    instance_type="",
    instance_count="",
    volume_size="",
    max_run_time="",
    output_location="",
    network_isolation="",
    max_wait_time="",
    role="",
):
    sagemaker_hpo_op(
        region=region,
        algorithm_name=algorithm_name,
        training_input_mode=training_input_mode,
        static_parameters=static_parameters,
        integer_parameters=integer_parameters,
        channels=channels,
        categorical_parameters=categorical_parameters,
        early_stopping_type=early_stopping_type,
        max_parallel_jobs=max_parallel_jobs,
        max_num_jobs=max_num_jobs,
        metric_name=metric_name,
        metric_type=metric_type,
        strategy=hpo_strategy,
        instance_type=instance_type,
        instance_count=instance_count,
        volume_size=volume_size,
        max_run_time=max_run_time,
        output_location=output_location,
        network_isolation=network_isolation,
        max_wait_time=max_wait_time,
        role=role,
    )


if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        hpo_pipeline, "SageMaker_hyperparameter_tuning_pipeline" + ".yaml"
    )
