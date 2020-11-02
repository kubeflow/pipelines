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

import difflib
from tempfile import NamedTemporaryFile
from typing import Callable, Dict, Type, Union, List, NamedTuple, cast
from kfp.components.structures import (
    ComponentSpec,
    InputSpec,
    OutputSpec,
    ContainerImplementation,
    InputValuePlaceholder,
    OutputPathPlaceholder,
    ContainerSpec,
)

from .sagemaker_component import SageMakerComponent
from .sagemaker_component_spec import (
    SageMakerComponentSpec,
    SageMakerComponentInputValidator,
    SageMakerComponentOutputValidator,
)
from .spec_input_parsers import SpecInputParsers

CommandlineArgumentType = Union[
    str, InputValuePlaceholder, OutputPathPlaceholder,
]


class IOArgs(NamedTuple):
    """Represents a structure that fills in a YAML spec inputs/outputs and
    provides the associated container args."""

    inputs: List[InputSpec]
    outputs: List[OutputSpec]
    args: List[CommandlineArgumentType]


class SageMakerComponentCompiler(object):
    """Compiles SageMaker components into its associated component.yaml."""

    # Maps all Python argument parser types to their associated KFP input types
    KFP_TYPE_FROM_ARGS: Dict[Callable, str] = {
        str: "String",
        int: "Integer",
        bool: "Bool",
        SpecInputParsers.nullable_string_argument: "String",
        SpecInputParsers.yaml_or_json_dict: "JsonObject",
        SpecInputParsers.yaml_or_json_list: "JsonArray",
        SpecInputParsers.str_to_bool: "Bool",
    }

    @staticmethod
    def _create_and_write_component(
        component_def: Type[SageMakerComponent],
        component_file_path: str,
        output_path: str,
        component_image_uri: str,
        component_image_tag: str,
    ):
        """Creates a component YAML specification and writes it to file.

        Args:
            component_def: The type of the SageMaker component.
            component_file_path: The path to the component definition file.
            output_path: The output path for the `component.yaml` file.
            component_image_uri: Compiled image URI.
            component_image_tag: Compiled image tag.
        """
        component_spec = SageMakerComponentCompiler._create_component_spec(
            component_def,
            component_file_path,
            component_image_uri=component_image_uri,
            component_image_tag=component_image_tag,
        )
        SageMakerComponentCompiler._write_component(component_spec, output_path)

    @staticmethod
    def _create_and_compare_component(
        component_def: Type[SageMakerComponent],
        component_file_path: str,
        compare_path: str,
        component_image_uri: str,
        component_image_tag: str,
    ):
        """Creates a component YAML specification and compares it to an
        existing file.

        Args:
            component_def: The type of the SageMaker component.
            component_file_path: The path to the component definition file.
            compare_path: The file path for the existing `component.yaml` file to compare against.
            component_image_uri: Compiled image URI.
            component_image_tag: Compiled image tag.
        """
        component_spec = SageMakerComponentCompiler._create_component_spec(
            component_def,
            component_file_path,
            component_image_uri=component_image_uri,
            component_image_tag=component_image_tag,
        )
        return SageMakerComponentCompiler._compare_component(
            component_spec, compare_path
        )

    @staticmethod
    def _create_io_from_component_spec(spec: Type[SageMakerComponentSpec]) -> IOArgs:
        """Parses the set of inputs and outputs from a component spec into the
        YAML spec form.

        Args:
            spec: A component specification definition.

        Returns:
            IOArgs: The IO arguments object filled with the fields from the
                component spec definition.
        """
        inputs = []
        outputs = []
        args = []

        # Iterate through all inputs adding them to the argument list
        for key, _input in spec.INPUTS.__dict__.items():
            # We know all of these values are validators as we have validated the spec
            input_validator: SageMakerComponentInputValidator = cast(
                SageMakerComponentInputValidator, _input
            )

            # Map from argsparser to KFP component
            input_spec = InputSpec(
                name=key,
                description=input_validator.description,
                type=SageMakerComponentCompiler.KFP_TYPE_FROM_ARGS.get(
                    input_validator.input_type, "String"
                ),
            )

            # Add optional fields
            if input_validator.default is not None:
                input_spec.__dict__["default"] = str(input_validator.default)
            elif not input_validator.required:
                # If not required and has no default, add empty string
                input_spec.__dict__["default"] = ""
            inputs.append(input_spec)

            # Add arguments to input list
            args.append(f"--{key}")
            args.append(InputValuePlaceholder(input_name=key))

        for key, _output in spec.OUTPUTS.__dict__.items():
            output_validator: SageMakerComponentOutputValidator = cast(
                SageMakerComponentOutputValidator, _output
            )
            outputs.append(
                OutputSpec(name=key, description=output_validator.description)
            )

            # Add arguments to input list
            args.append(f"--{key}{SageMakerComponentSpec.OUTPUT_ARGUMENT_SUFFIX}")
            args.append(OutputPathPlaceholder(output_name=key))

        return IOArgs(inputs=inputs, outputs=outputs, args=args)

    @staticmethod
    def _create_component_spec(
        component_def: Type[SageMakerComponent],
        component_file_path: str,
        component_image_uri: str,
        component_image_tag: str,
    ) -> ComponentSpec:
        """Create a component YAML specification object based on a component.

        Args:
            component_def: The type of the SageMaker component.
            component_file_path: The path to the component definition file.
            component_image_uri: Compiled image URI.
            component_image_tag: Compiled image tag.

        Returns:
            ComponentSpec: A `component.yaml` specification object.
        """
        io_args = SageMakerComponentCompiler._create_io_from_component_spec(
            component_def.COMPONENT_SPEC
        )

        return ComponentSpec(
            name=component_def.COMPONENT_NAME,
            description=component_def.COMPONENT_DESCRIPTION,
            inputs=io_args.inputs,
            outputs=io_args.outputs,
            implementation=ContainerImplementation(
                container=ContainerSpec(
                    image=f"{component_image_uri}:{component_image_tag}",
                    command=["python3"],
                    args=[component_file_path,] + io_args.args,  # type: ignore
                )
            ),
        )

    @staticmethod
    def _write_component(component_spec: ComponentSpec, output_path: str):
        """Write a component YAML specification to a file.

        Args:
            component_spec: A `component.yaml` specification object.
            output_path: The path to write the specification.
        """
        component_spec.save(output_path)

    @staticmethod
    def _compare_component(component_spec: ComponentSpec, compare_path: str):
        """Compare an existing specification file to a new specification.

        Args:
            component_spec: A `component.yaml` specification object.
            compare_path: The path of the existing specification file.
        """
        # Write new spec into a temporary file
        temp_spec_file = NamedTemporaryFile(mode="w", delete=False)
        component_spec.save(temp_spec_file.name)

        ignore_image: Callable[[str], bool] = lambda line: not line.lstrip().startswith(
            "image:"
        )

        with open(temp_spec_file.name, mode="r") as temp_file:
            with open(compare_path, mode="r") as existing_file:
                temp_lines = list(filter(ignore_image, temp_file.readlines()))
                existing_lines = list(filter(ignore_image, existing_file.readlines()))

                # Cast to list to read through generator
                diff_results = list(
                    difflib.unified_diff(
                        temp_lines,
                        existing_lines,
                        fromfile=temp_spec_file.name,
                        tofile=compare_path,
                    )
                )

        if len(diff_results) == 0:
            return False
        return "\n".join(diff_results)

    @staticmethod
    def compile(
        component_def: Type[SageMakerComponent],
        component_file_path: str,
        output_path: str,
        component_image_uri: str,
        component_image_tag: str,
    ):
        """Compiles a defined component into its component YAML specification.

        Args:
            component_def: The type of the SageMaker component.
            component_file_path: The path to the component definition file.
            output_path: The output path for the `component.yaml` file.
            component_image_uri: Compiled image URI.
            component_image_tag: Compiled image tag.
        """
        SageMakerComponentCompiler._create_and_write_component(
            component_def,
            component_file_path,
            output_path,
            component_image_uri,
            component_image_tag,
        )

    @staticmethod
    def check(
        component_def: Type[SageMakerComponent],
        component_file_path: str,
        compare_path: str,
        component_image_uri: str,
        component_image_tag: str,
    ):
        """Compiles a defined component into its component YAML specification
        and compares it against an existing YAML file.

        Args:
            component_def: The type of the SageMaker component.
            component_file_path: The path to the component definition file.
            compare_path: The file path for the `component.yaml` file to compare against.
            component_image_uri: Compiled image URI.
            component_image_tag: Compiled image tag.
        """
        return SageMakerComponentCompiler._create_and_compare_component(
            component_def,
            component_file_path,
            compare_path,
            component_image_uri,
            component_image_tag,
        )
