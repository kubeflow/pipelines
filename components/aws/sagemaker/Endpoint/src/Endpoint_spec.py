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

"""Specification for the SageMaker - Endpoint"""

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
class SageMakerEndpointInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the Endpoint component."""

    deployment_config: Input
    endpoint_config_name: Input
    endpoint_name: Input
    tags: Input


@dataclass
class SageMakerEndpointOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the Endpoint component."""

    ack_resource_metadata: Output
    conditions: Output
    creation_time: Output
    endpoint_status: Output
    failure_reason: Output
    last_modified_time: Output
    pending_deployment_summary: Output
    production_variants: Output
    sagemaker_resource_name: Output


class SageMakerEndpointSpec(
    SageMakerComponentSpec[SageMakerEndpointInputs, SageMakerEndpointOutputs]
):
    INPUTS: SageMakerEndpointInputs = SageMakerEndpointInputs(
        deployment_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="The deployment configuration for an endpoint, which contains the desired deployment strategy and rollback configurations.",
            required=False,
        ),
        endpoint_config_name=InputValidator(
            input_type=str,
            description="The name of an endpoint configuration.",
            required=True,
        ),
        endpoint_name=InputValidator(
            input_type=str, description="The name of the endpoint.", required=True
        ),
        tags=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            description="An array of key-value pairs.",
            required=False,
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS = SageMakerEndpointOutputs(
        ack_resource_metadata=OutputValidator(
            description="All CRs managed by ACK have a common `Status.",
        ),
        conditions=OutputValidator(
            description="All CRS managed by ACK have a common `Status.",
        ),
        creation_time=OutputValidator(
            description="A timestamp that shows when the endpoint was created.",
        ),
        endpoint_status=OutputValidator(
            description="The status of the endpoint.",
        ),
        failure_reason=OutputValidator(
            description="If the status of the endpoint is Failed, the reason why it failed.",
        ),
        last_modified_time=OutputValidator(
            description="A timestamp that shows when the endpoint was last modified.",
        ),
        pending_deployment_summary=OutputValidator(
            description="Returns the summary of an in-progress deployment.",
        ),
        production_variants=OutputValidator(
            description="An array of ProductionVariantSummary objects, one for each model hosted behind this endpoint.",
        ),
        sagemaker_resource_name=OutputValidator(
            description="Resource name on Sagemaker",
        ),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(arguments, SageMakerEndpointInputs, SageMakerEndpointOutputs)

    @property
    def inputs(self) -> SageMakerEndpointInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerEndpointOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerEndpointOutputs:
        return self._output_paths
