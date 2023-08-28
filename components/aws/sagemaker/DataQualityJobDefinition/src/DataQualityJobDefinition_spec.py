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

"""Specification for the SageMaker - DataQualityJobDefinition"""

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
class SageMakerDataQualityJobDefinitionInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the DataQualityJobDefinition component."""

    data_quality_app_specification: Input
    data_quality_baseline_config: Input
    data_quality_job_input: Input
    data_quality_job_output_config: Input
    job_definition_name: Input
    job_resources: Input
    network_config: Input
    role_arn: Input
    stopping_condition: Input
    tags: Input


@dataclass
class SageMakerDataQualityJobDefinitionOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the DataQualityJobDefinition component."""

    ack_resource_metadata: Output
    conditions: Output
    sagemaker_resource_name: Output


class SageMakerDataQualityJobDefinitionSpec(
    SageMakerComponentSpec[
        SageMakerDataQualityJobDefinitionInputs,
        SageMakerDataQualityJobDefinitionOutputs,
    ]
):
    INPUTS: SageMakerDataQualityJobDefinitionInputs = SageMakerDataQualityJobDefinitionInputs(
        data_quality_app_specification=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Specifies the container that runs the monitoring job.",
            required=True,
        ),
        data_quality_baseline_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Configures the constraints and baselines for the monitoring job.",
            required=False,
        ),
        data_quality_job_input=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="A list of inputs for the monitoring job.",
            required=True,
        ),
        data_quality_job_output_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="The output configuration for monitoring jobs.",
            required=True,
        ),
        job_definition_name=InputValidator(
            input_type=str,
            description="The name for the monitoring job definition.",
            required=True,
        ),
        job_resources=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Identifies the resources to deploy for a monitoring job.",
            required=True,
        ),
        network_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Specifies networking configuration for the monitoring job.",
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

    OUTPUTS = SageMakerDataQualityJobDefinitionOutputs(
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
            SageMakerDataQualityJobDefinitionInputs,
            SageMakerDataQualityJobDefinitionOutputs,
        )

    @property
    def inputs(self) -> SageMakerDataQualityJobDefinitionInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerDataQualityJobDefinitionOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerDataQualityJobDefinitionOutputs:
        return self._output_paths
