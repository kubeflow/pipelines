"""Compiler for SageMaker component files into YAML."""
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

import yaml
from typing import Dict, Mapping, Type, Optional, Union, List, NamedTuple, cast
from mypy_extensions import TypedDict

from .sagemaker_component import SageMakerComponent
from .sagemaker_component_spec import SageMakerComponentSpec, SageMakerComponentInputValidator, SageMakerComponentOutputValidator
from .spec_validators import SpecValidators


class InputSpec(TypedDict, total=False):
    name: str
    description: str
    type: str
    default: Optional[str]
    required: Optional[bool]


class OutputSpec(TypedDict):
    name: str
    description: str



class ArgumentValueSpec(TypedDict):
    inputValue: str

class ContainerSpec(TypedDict):

    image: str
    command: List[str]
    args: List[Union[str, ArgumentValueSpec]]

class ImplementationSpec(TypedDict):

    container: ContainerSpec

class ComponentSpec(TypedDict):
    """Defines the structure of a KFP `component.yaml` file."""

    name: str
    description: str
    inputs: List[InputSpec]
    outputs: List[OutputSpec]
    implementation: ImplementationSpec


class IOArgs(NamedTuple):
    """Represents a structure that fills in a YAML spec inputs/outputs and
    provides the associated container args."""

    inputs: List[InputSpec]
    outputs: List[OutputSpec]
    args: List[
        Union[
            str, ArgumentValueSpec
        ]
    ]


class SageMakerComponentCompiler(object):
    # Maps all Python argument parser types to their associated KFP input types
    KFP_TYPE_FROM_ARGS: Dict[object, str] = {
        str: "String",
        int: "Integer",
        bool: "Bool",
        SpecValidators.nullable_string_argument: "String",
        SpecValidators.yaml_or_json_dict: "JsonObject",
        SpecValidators.yaml_or_json_list: "JsonArray",
        SpecValidators.str_to_bool: "Bool",
    }

    @staticmethod
    def _create_and_write_component(
        component_def: Type[SageMakerComponent],
        component_file_path: str,
        output_path: str,
        component_image_uri: str,
        component_image_tag: str,
    ):
        """Creates a component YAML specification and writes it to file."""
        component_spec = SageMakerComponentCompiler._create_component_spec(
            component_def,
            component_file_path,
            component_image_uri=component_image_uri,
            component_image_tag=component_image_tag,
        )
        SageMakerComponentCompiler._write_component(component_spec, output_path)

    @staticmethod
    def _create_io_from_component_spec(spec: Type[SageMakerComponentSpec]) -> IOArgs:
        """Parses the set of inputs and outputs from a component spec into the
        YAML spec form."""
        inputs = []
        outputs = []
        args = []

        # Iterate through all inputs adding them to the argument list
        for key, _input in spec.INPUTS.__dict__.items():
            # We know all of these values are validators as we have validated the spec
            input_validator: SageMakerComponentInputValidator = cast(SageMakerComponentInputValidator, _input)
            # Map from argsparser to KFP component
            input_spec = InputSpec(
                name=key,
                description=input_validator.description,
                type=SageMakerComponentCompiler.KFP_TYPE_FROM_ARGS.get(
                    input_validator.input_type, "String"
                ),
            )

            # Add optional fields
            if input_validator.default:
                input_spec["default"] = str(input_validator.default)
            # if input_validator.required:
            #     input_spec["required"] = input_validator.required
            inputs.append(input_spec)

            # Add arguments to input list
            args.append(f"--{key}")
            args.append(
                ArgumentValueSpec(
                    inputValue=key
                )
            )

        for key, _output in spec.OUTPUTS.__dict__.items():
            output_validator: SageMakerComponentOutputValidator = cast(SageMakerComponentOutputValidator, _output)
            outputs.append(
                OutputSpec(name=key, description=output_validator.description)
            )

            # Add arguments to input list
            args.append(f"--{key}{SageMakerComponentSpec.OUTPUT_ARGUMENT_SUFFIX}")
            args.append(
                ArgumentValueSpec(
                    inputValue=key
                )
            )

        return IOArgs(inputs=inputs, outputs=outputs, args=args)

    @staticmethod
    def _create_component_spec(
        component_def: Type[SageMakerComponent],
        component_file_path: str,
        component_image_uri: str,
        component_image_tag: str,
    ) -> ComponentSpec:
        """Create a component YAML specification object based on a
        component."""
        io_args = SageMakerComponentCompiler._create_io_from_component_spec(
            component_def.COMPONENT_SPEC
        )

        return ComponentSpec(
            name=component_def.COMPONENT_NAME,
            description=component_def.COMPONENT_DESCRIPTION,
            inputs=io_args.inputs,
            outputs=io_args.outputs,
            implementation=ImplementationSpec(
                container=ContainerSpec(
                    image=f"{component_image_uri}:{component_image_tag}",
                    command=["python3"],
                    args=[component_file_path,] + io_args.args, # type: ignore
                )
            ),
        )

    @staticmethod
    def _write_component(component_spec: ComponentSpec, output_path: str):
        """Write a component YAML specification to a file."""
        with open(output_path, "w") as stream:
            yaml.dump(component_spec, stream)

    @staticmethod
    def compile(
        component_def: Type[SageMakerComponent],
        component_file_path: str,
        output_path: str,
        component_image_uri: str,
        component_image_tag: str,
    ):
        """Compiles a defined component into its component YAML
        specification."""
        SageMakerComponentCompiler._create_and_write_component(
            component_def,
            component_file_path,
            output_path,
            component_image_uri,
            component_image_tag,
        )
