"""Defines common spec input and output classes."""
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

from typing import (
    Callable,
    List,
    NewType,
    Optional,
    Union,
)
from .spec_input_parsers import SpecInputParsers


@dataclass(frozen=True)
class SageMakerComponentInputValidator:
    """Defines the structure of a component input to be used for validation."""

    input_type: Callable
    description: str
    required: bool = False
    choices: Optional[List[object]] = None
    default: Optional[object] = None

    def to_argparse_mapping(self):
        """Maps each property to an argparse argument.

        See: https://docs.python.org/3/library/argparse.html#adding-arguments
        """
        return {
            "type": self.input_type,
            "help": self.description,
            "required": self.required,
            "choices": self.choices,
            "default": self.default,
        }


@dataclass(frozen=True)
class SageMakerComponentOutputValidator:
    """Defines the structure of a component output."""

    description: str


# The value of an input or output can be arbitrary.
# This could be replaced with a generic, as well.
SageMakerIOValue = NewType("SageMakerIOValue", object)

# Allow the component input to represent either the validator or the value itself
# This saves on having to rewrite the struct for both types.
# Could replace `object` with T if TypedDict supported generics (see above).
SageMakerComponentInput = NewType(
    "SageMakerComponentInput", Union[SageMakerComponentInputValidator, SageMakerIOValue]
)
SageMakerComponentOutput = NewType(
    "SageMakerComponentOutput",
    Union[SageMakerComponentOutputValidator, SageMakerIOValue],
)


@dataclass(frozen=True)
class SageMakerComponentBaseInputs:
    """The base class for all component inputs."""

    pass


@dataclass
class SageMakerComponentBaseOutputs:
    """A base class for all component outputs."""

    pass


@dataclass(frozen=True)
class SageMakerComponentCommonInputs(SageMakerComponentBaseInputs):
    """A set of common inputs for all components."""

    region: SageMakerComponentInput
    endpoint_url: SageMakerComponentInput
    assume_role: SageMakerComponentInput
    tags: SageMakerComponentInput


COMMON_INPUTS = SageMakerComponentCommonInputs(
    region=SageMakerComponentInputValidator(
        input_type=str,
        required=True,
        description="The region for the SageMaker resource.",
    ),
    endpoint_url=SageMakerComponentInputValidator(
        input_type=SpecInputParsers.nullable_string_argument,
        required=False,
        description="The URL to use when communicating with the SageMaker service.",
    ),
    assume_role=SageMakerComponentInputValidator(
        input_type=str,
        required=False,
        description="The ARN of an IAM role to assume when connecting to SageMaker.",
    ),
    tags=SageMakerComponentInputValidator(
        input_type=SpecInputParsers.yaml_or_json_dict,
        required=False,
        description="An array of key-value pairs, to categorize AWS resources.",
        default={},
    ),
)


@dataclass(frozen=True)
class SpotInstanceInputs(SageMakerComponentBaseInputs):
    """Inputs to enable spot instance support."""

    spot_instance: SageMakerComponentInput
    max_wait_time: SageMakerComponentInput
    max_run_time: SageMakerComponentInput
    checkpoint_config: SageMakerComponentInput


SPOT_INSTANCE_INPUTS = SpotInstanceInputs(
    spot_instance=SageMakerComponentInputValidator(
        input_type=SpecInputParsers.str_to_bool,
        description="Use managed spot training.",
        default=False,
    ),
    max_wait_time=SageMakerComponentInputValidator(
        input_type=int,
        description="The maximum time in seconds you are willing to wait for a managed spot training job to complete.",
        default=86400,
    ),
    max_run_time=SageMakerComponentInputValidator(
        input_type=int,
        description="The maximum run time in seconds for the training job.",
        default=86400,
    ),
    checkpoint_config=SageMakerComponentInputValidator(
        input_type=SpecInputParsers.yaml_or_json_dict,
        description="Dictionary of information about the output location for managed spot training checkpoint data.",
        default={},
    ),
)
