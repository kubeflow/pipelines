"""SageMakerComponentSpec for defining inputs/outputs for
SageMakerComponents."""
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

import argparse

from typing import (
    Callable,
    Generic,
    List,
    TypeVar,
    Dict,
    Any,
)

from .common_inputs import (
    SageMakerComponentBaseInputs,
    SageMakerComponentBaseOutputs,
    SageMakerComponentInputValidator,
    SageMakerComponentOutputValidator,
    SageMakerIOValue,
)

IT = TypeVar("IT", bound=SageMakerComponentBaseInputs)  # Input Type
OT = TypeVar("OT", bound=SageMakerComponentBaseOutputs)  # Output Type


class SageMakerComponentSpec(Generic[IT, OT]):
    """Defines the set of inputs and outputs as expected for a
    SageMakerComponent.

    This class represents the inputs and outputs that are required to be provided
    to run a given SageMakerComponent. The component uses this to validate the
    format of the input arguments as given by the pipeline at runtime. Components
    should have a corresponding ComponentSpec inheriting from this
    class and must override all private members:

        INPUTS (as an inherited BaseInputs type)
        OUTPUTS (as an inherited BaseOutputs type)

    Typical usage example:

        class MySageMakerComponentSpec(
            SageMakerComponentSpec[SageMakerComponentInputs, SageMakerComponentOutputs]
        ):
            INPUTS = MySageMakerComponentInputs
            OUTPUTS = MySageMakerComponentOutputs
    """

    # These inputs apply to all components
    INPUTS: IT = SageMakerComponentBaseInputs()
    OUTPUTS: OT = SageMakerComponentBaseOutputs()

    OUTPUT_ARGUMENT_SUFFIX = "_output_path"

    def __init__(
        self,
        arguments: List[str],
        input_constructor: Callable[..., IT],
        output_constructor: Callable[..., OT],
    ):
        """Instantiates the spec with given user inputs.

        Args:
            arguments: A list of command line arguments.
            input_constructor: A constructor to create an input object.
            output_constructor: A constructor to create an output object.
        """
        self._validate_spec()

        parsed_args = self._parse_arguments(arguments)

        # Split results into inputs and outputs
        if self.INPUTS:
            self._inputs: IT = input_constructor(
                **{
                    key: SageMakerIOValue(value)
                    for key, value in parsed_args.items()
                    if key in self.INPUTS.__dict__.keys()
                }
            )
        else:
            self._inputs = input_constructor()

        # Map parsed keys (including suffix) to original output key name
        if self.OUTPUTS:
            parsed_key_to_output_key = {
                f"{output_key}{SageMakerComponentSpec.OUTPUT_ARGUMENT_SUFFIX}": output_key
                for output_key in self.OUTPUTS.__dict__.keys()
            }
        else:
            parsed_key_to_output_key = {}
        # Fill outputs with original keys, but match based on parsed key name
        # Default all initial values to None so we can check for completeness
        # by the end.
        self._outputs: OT = output_constructor(
            **{
                parsed_key_to_output_key.get(key): None
                for key, _ in parsed_args.items()
                if key in parsed_key_to_output_key.keys()
            }
        )

        # Store the path arguments for when we write the values to files
        self._output_paths: OT = output_constructor(
            **{
                parsed_key_to_output_key.get(key): value
                for key, value in parsed_args.items()
                if key in parsed_key_to_output_key.keys()
            }
        )

    @classmethod
    def _validate_spec(cls):
        """Ensures that all of the types given as inputs and outputs are
        validators."""
        if cls.INPUTS:
            for key, val in cls.INPUTS.__dict__.items():
                if not isinstance(val, SageMakerComponentInputValidator):
                    raise ValueError(
                        f"Input {key} is not of type {SageMakerComponentInputValidator.__name__}"
                    )

        if cls.OUTPUTS:
            for key, val in cls.OUTPUTS.__dict__.items():
                if not isinstance(val, SageMakerComponentOutputValidator):
                    raise ValueError(
                        f"Output {key} is not of type {SageMakerComponentOutputValidator.__name__}"
                    )
        pass

    @property
    def _parser(self):
        """Builds an argument parser to handle the set of defined inputs and
        outputs.

        Returns:
            An argument parser that fits the set of static inputs and outputs.
        """
        parser = argparse.ArgumentParser()

        # Add each input and output to the parser
        if self.INPUTS:
            for key, props in self.INPUTS.__dict__.items():
                parser.add_argument(f"--{key}", **props.to_argparse_mapping())
        if self.OUTPUTS:
            for key, props in self.OUTPUTS.__dict__.items():
                # Outputs are appended with _output_path to differentiate them programatically
                parser.add_argument(
                    f"--{key}{SageMakerComponentSpec.OUTPUT_ARGUMENT_SUFFIX}",
                    default=f"/tmp/{key}",
                    type=str,
                    help=props.description,
                )

        return parser

    def _parse_arguments(self, arguments: List[str]) -> Dict[str, Any]:
        """Passes the set of arguments through the parser to form the inputs
        and outputs.

        Args:
            arguments: A list of command line input arguments.

        Returns:
            A dict of input name to parsed value types.
        """
        args = self._parser.parse_args(arguments)
        return vars(args)

    @property
    def inputs(self) -> IT:
        return self._inputs

    @property
    def outputs(self) -> OT:
        return self._outputs

    @property
    def output_paths(self) -> OT:
        return self._output_paths
