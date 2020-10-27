"""Specification for the SageMaker create model component."""
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
class SageMakerCreateModelInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the create model component."""

    region: Input
    model_name: Input
    role: Input
    container_host_name: Input
    image: Input
    model_artifact_url: Input
    environment: Input
    model_package: Input
    secondary_containers: Input
    vpc_security_group_ids: Input
    vpc_subnets: Input
    network_isolation: Input


@dataclass
class SageMakerCreateModelOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the create model component."""

    model_name: Output


class SageMakerCreateModelSpec(
    SageMakerComponentSpec[SageMakerCreateModelInputs, SageMakerCreateModelOutputs]
):
    INPUTS: SageMakerCreateModelInputs = SageMakerCreateModelInputs(
        model_name=InputValidator(
            input_type=str, required=True, description="The name of the new model."
        ),
        role=InputValidator(
            input_type=str,
            required=True,
            description="The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.",
        ),
        container_host_name=InputValidator(
            input_type=str,
            required=False,
            description="When a ContainerDefinition is part of an inference pipeline, this value uniquely identifies the container for the purposes of logging and metrics.",
            default="",
        ),
        image=InputValidator(
            input_type=str,
            required=False,
            description="The Amazon EC2 Container Registry (Amazon ECR) path where inference code is stored.",
            default="",
        ),
        model_artifact_url=InputValidator(
            input_type=str,
            required=False,
            description="S3 path where Amazon SageMaker to store the model artifacts.",
            default="",
        ),
        environment=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            required=False,
            description="The dictionary of the environment variables to set in the Docker container. Up to 16 key-value entries in the map.",
            default={},
        ),
        model_package=InputValidator(
            input_type=str,
            required=False,
            description="The name or Amazon Resource Name (ARN) of the model package to use to create the model.",
            default="",
        ),
        secondary_containers=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=False,
            description="A list of dicts that specifies the additional containers in the inference pipeline.",
            default=[],
        ),
        vpc_security_group_ids=InputValidator(
            input_type=str,
            required=False,
            description="The VPC security group IDs, in the form sg-xxxxxxxx.",
            default="",
        ),
        vpc_subnets=InputValidator(
            input_type=str,
            required=False,
            description="The ID of the subnets in the VPC to which you want to connect your hpo job.",
            default="",
        ),
        network_isolation=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            required=False,
            description="Isolates the training container.",
            default=True,
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS = SageMakerCreateModelOutputs(
        model_name=OutputValidator(
            description="The name of the model created by SageMaker."
        ),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(
            arguments, SageMakerCreateModelInputs, SageMakerCreateModelOutputs
        )

    @property
    def inputs(self) -> SageMakerCreateModelInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerCreateModelOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerCreateModelOutputs:
        return self._output_paths
