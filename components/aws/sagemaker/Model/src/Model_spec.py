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

"""Specification for the SageMaker - Model"""

from dataclasses import dataclass

from typing import List
from commonv2.sagemaker_component_spec import (
    SageMakerComponentSpec,
    SageMakerComponentBaseOutputs,
)
from commonv2.spec_input_parsers import SpecInputParsers
from commonv2.common_inputs import (
    COMMON_INPUTS,
    SageMakerComponentCommonInputs,
    SageMakerComponentInput as Input,
    SageMakerComponentOutput as Output,
    SageMakerComponentInputValidator as InputValidator,
    SageMakerComponentOutputValidator as OutputValidator,
)


@dataclass(frozen=False)
class SageMakerModelInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the Model component."""

    containers: Input
    enable_network_isolation: Input
    execution_role_arn: Input
    inference_execution_config: Input
    model_name: Input
    primary_container: Input
    tags: Input
    vpc_config: Input


@dataclass
class SageMakerModelOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the Model component."""

    ack_resource_metadata: Output
    conditions: Output
    sagemaker_resource_name: Output


class SageMakerModelSpec(
    SageMakerComponentSpec[SageMakerModelInputs, SageMakerModelOutputs]
):
    INPUTS: SageMakerModelInputs = SageMakerModelInputs(
        containers=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            description="Specifies the containers in the inference pipeline.",
            required=False,
        ),
        enable_network_isolation=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            description="Isolates the model container.",
            required=False,
        ),
        execution_role_arn=InputValidator(
            input_type=str,
            description="The Amazon Resource Name (ARN) of the IAM role that SageMaker can assume to access model artifacts and docker image for deployment on ML compute instances or for batch transform jobs.",
            required=True,
        ),
        inference_execution_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Specifies details of how containers in a multi-container endpoint are called.",
            required=False,
        ),
        model_name=InputValidator(
            input_type=str, description="The name of the new model.", required=True
        ),
        primary_container=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="The location of the primary docker image containing inference code, associated artifacts, and custom environment map that the inference code uses when the model is deployed for predictions.",
            required=False,
        ),
        tags=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            description="An array of key-value pairs.",
            required=False,
        ),
        vpc_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="A VpcConfig object that specifies the VPC that you want your model to connect to.",
            required=False,
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS = SageMakerModelOutputs(
        ack_resource_metadata=OutputValidator(
            description="All CRs managed by ACK have a common `Status.",
        ),
        conditions=OutputValidator(
            description="All CRS managed by ACK have a common `Status.",
        ),
        sagemaker_resource_name=OutputValidator(
            description="Resource name on Sagemaker",
        ),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(arguments, SageMakerModelInputs, SageMakerModelOutputs)

    @property
    def inputs(self) -> SageMakerModelInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerModelOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerModelOutputs:
        return self._output_paths
