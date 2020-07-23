"""SageMakerComponentSpec for defining inputs/outputs for SageMakerComponents"""
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

from typing import Dict, Any

from .spec_validators import SpecValidators

class SageMakerComponentSpec(object):
    """Defines the set of inputs and outputs as expected for a SageMakerComponent.

    This class represents the inputs and outputs that are required to be provided
    to run a given SageMakerComponent. The component uses this to validate the 
    format of the input arguments as given by the pipeline at runtime. Components
    should have a corresponding ComponentSpec inheriting from this
    class and must override all private members:

        - INPUTS (as a dict of string keys)
        - OUTPUTS (also as a dict of string keys)

    Typical usage example:

        class MySageMakerComponentSpec(SageMakerComponentSpec):
            // TODO: Finish this example
            INPUTS = {}
            OUTPUTS = {}
    """

    INPUTS = {
        "region": dict(
            type=str, required=True, help="The region where the training job launches."
        ),
        "endpoint_url": dict(
            type=SpecValidators.nullable_string_argument,
            required=False,
            help="The URL to use when communicating with the SageMaker service.",
        ),
    }
    OUTPUTS = {}

    def __init__(self, arguments: Dict[str, str]):
        parsed_args = self._parse_arguments(arguments)

        # Split results into inputs and outputs
        self._inputs = {
            key: value
            for key, value in parsed_args.items()
            if key in self.INPUTS.keys()
        }
        self._outputs = {
            key: value
            for key, value in parsed_args.items()
            if key in self.OUTPUTS.keys()
        }

    @property
    def _parser(self):
        """Builds an argument parser to handle the set of defined inputs and outputs
    
        Returns:
            An argument parser that fits the set of static inputs and outputs.
        """
        parser = argparse.ArgumentParser()

        # Add each input and output to the parser
        for key, props in self.INPUTS.items():
            parser.add_argument(f"--{key}", **props)
        for key, props in self.OUTPUTS.items():
            # Outputs are appended with _file_path to differentiate them programatically
            parser.add_argument(f"--{key}_file_path", default=f"/tmp/{key}", type=str, **props)

        return parser

    def _parse_arguments(self, arguments: Dict[str, str]) -> Dict[str, Any]:
        """Passes the set of arguments through the parser to form the inputs and outputs
    
        Args:
            arguments: A dict mapping input names to corresponding string values.  

        Returns:
            A dict of input name to parsed value types.
        """
        args = self._parser.parse_args(arguments)
        return vars(args)

    @property
    def inputs(self) -> Dict[str, Any]:
        return self._inputs

    @property
    def outputs(self) -> Dict[str, str]:
        return self._outputs

