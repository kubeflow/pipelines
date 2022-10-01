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
from commonv2.spec_input_parsers import SpecInputParsers


@dataclass(frozen=False)
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


@dataclass(frozen=False)
class SageMakerComponentBaseInputs:
    """The base class for all component inputs."""

    pass


@dataclass
class SageMakerComponentBaseOutputs:
    """A base class for all component outputs."""

    pass


@dataclass(frozen=False)
class SageMakerComponentCommonInputs(SageMakerComponentBaseInputs):
    """A set of common inputs for all components."""

    region: SageMakerComponentInput


COMMON_INPUTS = SageMakerComponentCommonInputs(
    region=SageMakerComponentInputValidator(
        input_type=str,
        required=True,
        description="The region for the SageMaker resource.",
    ),
)
