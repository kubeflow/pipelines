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

"""Specification for the SageMaker - EndpointConfig"""

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
class SageMakerEndpointConfigInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the EndpointConfig component."""

    async_inference_config: Input
    data_capture_config: Input
    endpoint_config_name: Input
    kms_key_id: Input
    production_variants: Input
    tags: Input


@dataclass
class SageMakerEndpointConfigOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the EndpointConfig component."""

    ack_resource_metadata: Output
    conditions: Output
    sagemaker_resource_name: Output


class SageMakerEndpointConfigSpec(
    SageMakerComponentSpec[
        SageMakerEndpointConfigInputs, SageMakerEndpointConfigOutputs
    ]
):
    INPUTS: SageMakerEndpointConfigInputs = SageMakerEndpointConfigInputs(
        async_inference_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Specifies configuration for how an endpoint performs asynchronous inference.",
            required=False,
        ),
        data_capture_config=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            description="Configuration to control how SageMaker captures inference data.",
            required=False,
        ),
        endpoint_config_name=InputValidator(
            input_type=str,
            description="The name of the endpoint configuration.",
            required=True,
        ),
        kms_key_id=InputValidator(
            input_type=str,
            description="The Amazon Resource Name (ARN) of a Amazon Web Services Key Management Service key that SageMaker uses to encrypt data on the storage volume attached to the ML compute instance that hosts the endpoint",
            required=False,
        ),
        production_variants=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            description="An array of ProductionVariant objects, one for each model that you want to host at this endpoint.",
            required=True,
        ),
        tags=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            description="An array of key-value pairs.",
            required=False,
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS = SageMakerEndpointConfigOutputs(
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
            arguments, SageMakerEndpointConfigInputs, SageMakerEndpointConfigOutputs
        )

    @property
    def inputs(self) -> SageMakerEndpointConfigInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerEndpointConfigOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerEndpointConfigOutputs:
        return self._output_paths
