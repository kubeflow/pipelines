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

"""Specification for the SageMaker - ModelQualityJobDefinition"""

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
class SageMakerModelQualityJobDefinitionInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the ModelQualityJobDefinition component."""

    job_definition_name: Input
    job_resources: Input
    model_quality_app_specification: Input
    model_quality_baseline_config: Input
    model_quality_job_input: Input
    model_quality_job_output_config: Input
    network_config: Input
    role_arn: Input
    stopping_condition: Input
    tags: Input


@dataclass
class SageMakerModelQualityJobDefinitionOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the ModelQualityJobDefinition component."""

    ack_resource_metadata: Output
    conditions: Output
    sagemaker_resource_name: Output


class SageMakerModelQualityJobDefinitionSpec(
    SageMakerComponentSpec[
        SageMakerModelQualityJobDefinitionInputs,
        SageMakerModelQualityJobDefinitionOutputs,
    ]
):
    INPUTS: SageMakerModelQualityJobDefinitionInputs = SageMakerModelQualityJobDefinitionInputs(
        job_definition_name=InputValidator(
            input_type=str,
            description="The name of the monitoring job definition.",
            required=True,
        ),
        job_resources=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Identifies the resources to deploy for a monitoring job.",
            required=True,
        ),
        model_quality_app_specification=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="The container that runs the monitoring job.",
            required=True,
        ),
        model_quality_baseline_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Specifies the constraints and baselines for the monitoring job.",
            required=False,
        ),
        model_quality_job_input=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="A list of the inputs that are monitored.",
            required=True,
        ),
        model_quality_job_output_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="The output configuration for monitoring jobs.",
            required=True,
        ),
        network_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Specifies the network configuration for the monitoring job.",
            required=False,
        ),
        role_arn=InputValidator(
            input_type=str,
            description="The Amazon Resource Name (ARN) of an IAM role that Amazon SageMaker can assume to perform tasks on your behalf.",
            required=True,
        ),
        stopping_condition=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="A time limit for how long the monitoring job is allowed to run before stopping.",
            required=False,
        ),
        tags=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            description="(Optional) An array of key-value pairs.",
            required=False,
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS = SageMakerModelQualityJobDefinitionOutputs(
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
        super().__init__(
            arguments,
            SageMakerModelQualityJobDefinitionInputs,
            SageMakerModelQualityJobDefinitionOutputs,
        )

    @property
    def inputs(self) -> SageMakerModelQualityJobDefinitionInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerModelQualityJobDefinitionOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerModelQualityJobDefinitionOutputs:
        return self._output_paths
