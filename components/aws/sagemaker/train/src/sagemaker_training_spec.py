"""Specification for the SageMaker training component."""
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from dataclasses import dataclass

from typing import List
from common.sagemaker_component_spec import (
    SageMakerComponentSpec,
    SageMakerComponentBaseOutputs,
)
from common.spec_input_parsers import SpecInputParsers
from common.common_inputs import (
    COMMON_INPUTS,
    SageMakerComponentCommonInputs,
    SpotInstanceInputs,
    SPOT_INSTANCE_INPUTS,
    SageMakerComponentInput as Input,
    SageMakerComponentOutput as Output,
    SageMakerComponentInputValidator as InputValidator,
    SageMakerComponentOutputValidator as OutputValidator,
)


@dataclass(frozen=True)
class SageMakerTrainingInputs(SageMakerComponentCommonInputs, SpotInstanceInputs):
    """Defines the set of inputs for the training component."""

    job_name: Input
    role: Input
    image: Input
    algorithm_name: Input
    metric_definitions: Input
    training_input_mode: Input
    hyperparameters: Input
    channels: Input
    instance_type: Input
    instance_count: Input
    volume_size: Input
    resource_encryption_key: Input
    max_run_time: Input
    model_artifact_path: Input
    output_encryption_key: Input
    vpc_security_group_ids: Input
    vpc_subnets: Input
    network_isolation: Input
    traffic_encryption: Input
    debug_hook_config: Input
    debug_rule_config: Input


@dataclass
class SageMakerTrainingOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the training component."""

    model_artifact_url: Output
    job_name: Output
    training_image: Output


class SageMakerTrainingSpec(
    SageMakerComponentSpec[SageMakerTrainingInputs, SageMakerTrainingOutputs]
):
    INPUTS: SageMakerTrainingInputs = SageMakerTrainingInputs(
        job_name=InputValidator(
            input_type=str, description="The name of the training job.", default="",
        ),
        role=InputValidator(
            input_type=str,
            required=True,
            description="The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.",
        ),
        image=InputValidator(
            input_type=str,
            description="The registry path of the Docker image that contains the training algorithm.",
            default="",
        ),
        algorithm_name=InputValidator(
            input_type=str,
            description="The name of the resource algorithm to use for the training job. Do not specify a value for this if using training image.",
            default="",
        ),
        metric_definitions=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="The dictionary of name-regex pairs specify the metrics that the algorithm emits.",
            default={},
        ),
        training_input_mode=InputValidator(
            choices=["File", "Pipe"],
            input_type=str,
            description="The input mode that the algorithm supports. File or Pipe.",
            default="File",
        ),
        hyperparameters=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Dictionary of hyperparameters for the the algorithm.",
            default={},
        ),
        channels=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=True,
            description="A list of dicts specifying the input channels. Must have at least one.",
        ),
        instance_type=InputValidator(
            input_type=str,
            description="The ML compute instance type.",
            default="ml.m4.xlarge",
        ),
        instance_count=InputValidator(
            required=True,
            input_type=int,
            description="The number of ML compute instances to use in the training job.",
            default=1,
        ),
        volume_size=InputValidator(
            input_type=int,
            required=True,
            description="The size of the ML storage volume that you want to provision.",
            default=30,
        ),
        resource_encryption_key=InputValidator(
            input_type=str,
            description="The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).",
            default="",
        ),
        model_artifact_path=InputValidator(
            input_type=str,
            required=True,
            description="Identifies the S3 path where you want Amazon SageMaker to store the model artifacts.",
        ),
        output_encryption_key=InputValidator(
            input_type=str,
            description="The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts.",
            default="",
        ),
        vpc_security_group_ids=InputValidator(
            input_type=str,
            description="The VPC security group IDs, in the form sg-xxxxxxxx.",
        ),
        vpc_subnets=InputValidator(
            input_type=str,
            description="The ID of the subnets in the VPC to which you want to connect your hpo job.",
        ),
        network_isolation=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            description="Isolates the training container.",
            default=True,
        ),
        traffic_encryption=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            description="Encrypts all communications between ML compute instances in distributed training.",
            default=False,
        ),
        debug_hook_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Configuration information for the debug hook parameters, collection configuration, and storage paths.",
            default={},
        ),
        debug_rule_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            description="Configuration information for debugging rules.",
            default=[],
        ),
        **vars(COMMON_INPUTS),
        **vars(SPOT_INSTANCE_INPUTS)
    )

    OUTPUTS = SageMakerTrainingOutputs(
        model_artifact_url=OutputValidator(description="The model artifacts URL."),
        job_name=OutputValidator(description="The training job name."),
        training_image=OutputValidator(
            description="The registry path of the Docker image that contains the training algorithm."
        ),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(arguments, SageMakerTrainingInputs, SageMakerTrainingOutputs)

    @property
    def inputs(self) -> SageMakerTrainingInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerTrainingOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerTrainingOutputs:
        return self._output_paths
