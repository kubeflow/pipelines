import yaml
from typing import Type, Optional, Union, List, NamedTuple
from mypy_extensions import TypedDict

from .sagemaker_component import SageMakerComponent
from .sagemaker_component_spec import SageMakerComponentSpec
from .spec_validators import SpecValidators


class ComponentSpec(TypedDict):
    class InputSpec(TypedDict, total=False):
        name: str
        description: str
        type: str
        default: Optional[str]
        required: Optional[bool]

    class OutputSpec(TypedDict):
        name: str
        description: str

    class ImplementationSpec(TypedDict):
        class ContainerSpec(TypedDict):
            class ArgumentValueSpec(TypedDict):
                inputValue: str

            image: str
            command: List[str]
            args: List[Union[str, Type[ArgumentValueSpec]]]

        container: Type[ContainerSpec]

    name: str
    description: str
    inputs: List[Type[InputSpec]]
    outputs: List[Type[OutputSpec]]
    implementation: Type[ImplementationSpec]


class IOArgs(NamedTuple):
    """Represents a structure that fills in a YAML spec inputs/outputs and provides the associated container args"""

    inputs: List[ComponentSpec.InputSpec]
    outputs: List[ComponentSpec.OutputSpec]
    args: List[
        Union[
            str, Type[ComponentSpec.ImplementationSpec.ContainerSpec.ArgumentValueSpec]
        ]
    ]


class SageMakerComponentCompiler(object):
    KFP_TYPE_FROM_ARGS = {
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
    ):
        """Creates a component YAML specification and writes it to file"""
        component_spec = SageMakerComponentCompiler._create_component_spec(
            component_def, component_file_path
        )
        SageMakerComponentCompiler._write_component(component_spec, output_path)

    @staticmethod
    def _create_io_from_component_spec(spec: Type[SageMakerComponentSpec]) -> IOArgs:
        inputs = []
        outputs = []
        args = []

        # Iterate through all inputs adding them to the argument list
        for key, _input in spec.INPUTS.items():
            # Map from argsparser to KFP component
            input_spec = ComponentSpec.InputSpec(
                name=key,
                description=_input.get("help"),
                type=SageMakerComponentCompiler.KFP_TYPE_FROM_ARGS.get(
                    _input.get("type")
                ),
            )

            # Add optional fields
            if "default" in _input:
                input_spec["default"] = _input.get("default")
            if "required" in _input:
                input_spec["required"] = _input.get("required")
            inputs.append(input_spec)

            # Add arguments to input list
            args.append(f"--{key}")
            args.append(
                ComponentSpec.ImplementationSpec.ContainerSpec.ArgumentValueSpec(
                    inputValue=key
                )
            )

        for key, _output in spec.OUTPUTS.items():
            outputs.append(
                ComponentSpec.OutputSpec(name=key, description=_output.get("help"))
            )

            # Add arguments to input list
            args.append(f"--{key}_file_path")
            args.append(
                ComponentSpec.ImplementationSpec.ContainerSpec.ArgumentValueSpec(
                    inputValue=key
                )
            )

        return IOArgs(inputs=inputs, outputs=outputs, args=args)

    @staticmethod
    def _create_component_spec(
        component_def: Type[SageMakerComponent], component_file_path: str
    ) -> ComponentSpec:
        """Create a component YAML specification object based on a component"""
        io_args = SageMakerComponentCompiler._create_io_from_component_spec(
            component_def.COMPONENT_SPEC
        )

        return ComponentSpec(
            name=component_def.COMPONENT_NAME,
            description=component_def.COMPONENT_DESCRIPTION,
            inputs=io_args.inputs,
            outputs=io_args.outputs,
            implementation=ComponentSpec.ImplementationSpec(
                container=ComponentSpec.ImplementationSpec.ContainerSpec(
                    image="kfp-component:latest",
                    command=["python3"],
                    args=[component_file_path,] + io_args.args,
                )
            ),
        )

    @staticmethod
    def _write_component(component_spec: ComponentSpec, output_path: str):
        """Write a component YAML specification to a file"""
        with open(output_path, "w") as stream:
            yaml.dump(component_spec, stream)

    @staticmethod
    def compile(
        component_def: Type[SageMakerComponent],
        component_file_path: str,
        output_path: str,
    ):
        """Compiles a defined component into its component YAML specification"""
        SageMakerComponentCompiler._create_and_write_component(
            component_def, component_file_path, output_path
        )

