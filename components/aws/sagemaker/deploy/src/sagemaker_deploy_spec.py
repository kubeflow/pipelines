"""Specification for the SageMaker deploy component."""
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
class SageMakerDeployInputs(SageMakerComponentCommonInputs):
    """Defines the set of inputs for the deploy component."""

    endpoint_name: Input
    endpoint_config_name: Input
    endpoint_config_tags: Input
    resource_encryption_key: Input

    update_endpoint: Input

    variant_name_1: Input
    model_name_1: Input
    initial_instance_count_1: Input
    instance_type_1: Input
    initial_variant_weight_1: Input
    accelerator_type_1: Input

    variant_name_2: Input
    model_name_2: Input
    initial_instance_count_2: Input
    instance_type_2: Input
    initial_variant_weight_2: Input
    accelerator_type_2: Input

    variant_name_3: Input
    model_name_3: Input
    initial_instance_count_3: Input
    instance_type_3: Input
    initial_variant_weight_3: Input
    accelerator_type_3: Input


@dataclass
class SageMakerDeployOutputs(SageMakerComponentBaseOutputs):
    """Defines the set of outputs for the deploy component."""

    endpoint_name: Output


class SageMakerDeploySpec(
    SageMakerComponentSpec[SageMakerDeployInputs, SageMakerDeployOutputs]
):
    INPUTS: SageMakerDeployInputs = SageMakerDeployInputs(
        endpoint_config_name=InputValidator(
            input_type=str,
            required=False,
            description="The name of the endpoint configuration. If an existing endpoint is being updated, a suffix is automatically added if this config name exists.",
            default="",
        ),
        update_endpoint=InputValidator(
            input_type=SpecInputParsers.str_to_bool,
            required=False,
            description="Update endpoint if it exists.",
            default=False,
        ),
        variant_name_1=InputValidator(
            input_type=str,
            required=False,
            description="The name of the production variant.",
            default="variant-name-1",
        ),
        model_name_1=InputValidator(
            input_type=str,
            required=True,
            description="The model name used for endpoint deployment.",
        ),
        initial_instance_count_1=InputValidator(
            input_type=int,
            required=False,
            description="Number of instances to launch initially.",
            default=1,
        ),
        instance_type_1=InputValidator(
            input_type=str,
            required=False,
            description="The ML compute instance type.",
            default="ml.m4.xlarge",
        ),
        initial_variant_weight_1=InputValidator(
            input_type=float,
            required=False,
            description="Determines initial traffic distribution among all of the models that you specify in the endpoint configuration.",
            default=1.0,
        ),
        accelerator_type_1=InputValidator(
            choices=["ml.eia1.medium", "ml.eia1.large", "ml.eia1.xlarge", ""],
            input_type=str,
            required=False,
            description="The size of the Elastic Inference (EI) instance to use for the production variant.",
            default="",
        ),
        variant_name_2=InputValidator(
            input_type=str,
            required=False,
            description="The name of the production variant.",
            default="variant-name-2",
        ),
        model_name_2=InputValidator(
            input_type=str,
            required=False,
            description="The model name used for endpoint deployment.",
            default="",
        ),
        initial_instance_count_2=InputValidator(
            input_type=int,
            required=False,
            description="Number of instances to launch initially.",
            default=1,
        ),
        instance_type_2=InputValidator(
            input_type=str,
            required=False,
            description="The ML compute instance type.",
            default="ml.m4.xlarge",
        ),
        initial_variant_weight_2=InputValidator(
            input_type=float,
            required=False,
            description="Determines initial traffic distribution among all of the models that you specify in the endpoint configuration.",
            default=1.0,
        ),
        accelerator_type_2=InputValidator(
            choices=["ml.eia1.medium", "ml.eia1.large", "ml.eia1.xlarge", ""],
            input_type=str,
            required=False,
            description="The size of the Elastic Inference (EI) instance to use for the production variant.",
            default="",
        ),
        variant_name_3=InputValidator(
            input_type=str,
            required=False,
            description="The name of the production variant.",
            default="variant-name-3",
        ),
        model_name_3=InputValidator(
            input_type=str,
            required=False,
            description="The model name used for endpoint deployment.",
            default="",
        ),
        initial_instance_count_3=InputValidator(
            input_type=int,
            required=False,
            description="Number of instances to launch initially.",
            default=1,
        ),
        instance_type_3=InputValidator(
            input_type=str,
            required=False,
            description="The ML compute instance type.",
            default="ml.m4.xlarge",
        ),
        initial_variant_weight_3=InputValidator(
            input_type=float,
            required=False,
            description="Determines initial traffic distribution among all of the models that you specify in the endpoint configuration.",
            default=1.0,
        ),
        accelerator_type_3=InputValidator(
            choices=["ml.eia1.medium", "ml.eia1.large", "ml.eia1.xlarge", ""],
            input_type=str,
            required=False,
            description="The size of the Elastic Inference (EI) instance to use for the production variant.",
            default="",
        ),
        resource_encryption_key=InputValidator(
            input_type=str,
            required=False,
            description="The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).",
            default="",
        ),
        endpoint_config_tags=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            required=False,
            description="An array of key-value pairs, to categorize AWS resources.",
            default={},
        ),
        endpoint_name=InputValidator(
            input_type=str,
            required=False,
            description="The name of the endpoint.",
            default="",
        ),
        **vars(COMMON_INPUTS),
    )

    OUTPUTS = SageMakerDeployOutputs(
        endpoint_name=OutputValidator(description="The created endpoint name."),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(arguments, SageMakerDeployInputs, SageMakerDeployOutputs)

    @property
    def inputs(self) -> SageMakerDeployInputs:
        return self._inputs

    @property
    def outputs(self) -> SageMakerDeployOutputs:
        return self._outputs

    @property
    def output_paths(self) -> SageMakerDeployOutputs:
        return self._output_paths
