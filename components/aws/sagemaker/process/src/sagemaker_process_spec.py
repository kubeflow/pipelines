"""Specification for the SageMaker processing component."""
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
    SageMakerComponentInput as Input,
    SageMakerComponentOutput as Output,
    SageMakerComponentInputValidator as InputValidator,
    SageMakerComponentOutputValidator as OutputValidator,
)


@dataclass(frozen=True)
class SageMakerProcessInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the process component."""

    job_name: Input
    role: Input
    image: Input
    instance_type: Input
    instance_count: Input
    volume_size: Input
    resource_encryption_key: Input
    output_encryption_key: Input
    max_run_time: Input
    environment: Input
    container_entrypoint: Input
    container_arguments: Input
    output_config: Input
    input_config: Input
    vpc_security_group_ids: Input
    vpc_subnets: Input
    network_isolation: Input
    traffic_encryption: Input


@dataclass
class SageMakerProcessOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the process component."""

    job_name: Output
    output_artifacts: Output


class SageMakerProcessSpec(
    SageMakerComponentSpec[SageMakerProcessInputs, SageMakerProcessOutputs]
):
    INPUTS: SageMakerProcessInputs = SageMakerProcessInputs(
        job_name=InputValidator(
            input_type=str,
            required=False,
            description="The name of the processing job.",
            default="",
        ),
        role=InputValidator(
            input_type=str,
            required=True,
            description="The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.",
        ),
        image=InputValidator(
            input_type=str,
            required=True,
            description="The registry path of the Docker image that contains the processing container.",
            default="",
        ),
        instance_type=InputValidator(
            required=True,
            input_type=str,
            description="The ML compute instance type.",
            default="ml.m4.xlarge",
        ),
        instance_count=InputValidator(
            required=True,
            input_type=int,
            description="The number of ML compute instances to use in each processing job.",
            default=1,
        ),
        volume_size=InputValidator(
            input_type=int,
            required=False,
            description="The size of the ML storage volume that you want to provision.",
            default=30,
        ),
        resource_encryption_key=InputValidator(
            input_type=str,
            required=False,
            description="The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).",
            default="",
        ),
        output_encryption_key=InputValidator(
            input_type=str,
            required=False,
            description="The AWS KMS key that Amazon SageMaker uses to encrypt the processing artifacts.",
            default="",
        ),
        max_run_time=InputValidator(
            input_type=int,
            required=False,
            description="The maximum run time in seconds for the processing job.",
            default=86400,
        ),
        environment=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            required=False,
            description="The dictionary of the environment variables to set in the Docker container. Up to 16 key-value entries in the map.",
            default={},
        ),
        container_entrypoint=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=False,
            description="The entrypoint for the processing job. This is in the form of a list of strings that make a command.",
            default=[],
        ),
        container_arguments=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=False,
            description="A list of string arguments to be passed to a processing job.",
            default=[],
        ),
        input_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=False,
            description="Parameters that specify Amazon S3 inputs for a processing job.",
            default=[],
        ),
        output_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=True,
            description="Parameters that specify Amazon S3 outputs for a processing job.",
            default=[],
        ),
        vpc_security_group_ids=InputValidator(
            input_type=str,
            required=False,
            description="The VPC security group IDs, in the form sg-xxxxxxxx.",
        ),
        vpc_subnets=InputValidator(
            input_type=str,
            required=False,
            description="The ID of the subnets in the VPC to which you want to connect your hpo job.",
        ),
        network_isolation=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            required=False,
            description="Isolates the processing container.",
            default=True,
        ),
        traffic_encryption=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            required=False,
            description="Encrypts all communications between ML compute instances in distributed training.",
            default=False,
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS = SageMakerProcessOutputs(
        job_name=OutputValidator(description="Processing job name."),
        output_artifacts=OutputValidator(
            description="A dictionary containing the output S3 artifacts."
        ),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(arguments, SageMakerProcessInputs, SageMakerProcessOutputs)

    @property
    def inputs(self) -> SageMakerProcessInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerProcessOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerProcessOutputs:
        return self._output_paths
