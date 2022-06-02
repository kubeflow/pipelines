# Copyright 2021-2022 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Definitions for component spec."""

import ast
import collections
import dataclasses
import functools
import itertools
import re
from typing import Any, Dict, List, Mapping, Optional, Type, TypeVar, Union
import uuid

import kfp
from kfp import dsl
from kfp.compiler import compiler
from kfp.components import base_model
from kfp.components import utils
from kfp.components import v1_components
from kfp.components import v1_structures
from kfp.components.types import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2
import yaml


class InputSpec_(base_model.BaseModel):
    """Component input definitions. (Inner class).

    Attributes:
        type: The type of the input.
        default (optional): the default value for the input.
        description: Optional: the user description of the input.
    """
    type: Union[str, dict]
    default: Union[Any, None] = None


# Hack to allow access to __init__ arguments for setting _optional value
class InputSpec(InputSpec_, base_model.BaseModel):
    """Component input definitions.

    Attributes:
        type: The type of the input.
        default (optional): the default value for the input.
        _optional: Wether the input is optional. An input is optional when it has an explicit default value.
    """

    @functools.wraps(InputSpec_.__init__)
    def __init__(self, *args, **kwargs) -> None:
        """InputSpec constructor, which can access the arguments passed to the
        constructor for setting ._optional value."""
        if args:
            raise ValueError('InputSpec does not accept positional arguments.')
        super().__init__(*args, **kwargs)
        self._optional = 'default' in kwargs

    @classmethod
    def from_ir_parameter_dict(
            cls, ir_parameter_dict: Dict[str, Any]) -> 'InputSpec':
        """Creates an InputSpec from a ComponentInputsSpec message in dict
        format (pipeline_spec.components.<component-
        key>.inputDefinitions.parameters.<input-key>).

        Args:
            ir_parameter_dict (Dict[str, Any]): The ComponentInputsSpec message in dict format.

        Returns:
            InputSpec: The InputSpec object.
        """
        type_ = type_utils.IR_TYPE_TO_IN_MEMORY_SPEC_TYPE.get(
            ir_parameter_dict['parameterType'])
        if type_ is None:
            raise ValueError(
                f'Unknown type {ir_parameter_dict["parameterType"]} found in IR.'
            )
        default = ir_parameter_dict.get('defaultValue')
        return InputSpec(type=type_, default=default)

    def __eq__(self, other: Any) -> bool:
        """Equality comparison for InputSpec. Robust to different type
        representations, such that it respects the maximum amount of
        information possible to encode in IR. That is, because
        `typing.List[str]` can only be represented a `List` in IR,
        'typing.List' == 'List' in this comparison.

        Args:
            other (Any): The object to compare to InputSpec.

        Returns:
            bool: True if the objects are equal, False otherwise.
        """
        if isinstance(other, InputSpec):
            return type_utils.get_canonical_name_for_outer_generic(
                self.type) == type_utils.get_canonical_name_for_outer_generic(
                    other.type) and self.default == other.default
        else:
            return False


class OutputSpec(base_model.BaseModel):
    """Component output definitions.

    Attributes:
        type: The type of the output.
    """
    type: Union[str, dict]

    @classmethod
    def from_ir_parameter_dict(
            cls, ir_parameter_dict: Dict[str, Any]) -> 'OutputSpec':
        """Creates an OutputSpec from a ComponentOutputsSpec message in dict
        format (pipeline_spec.components.<component-
        key>.outputDefinitions.parameters|artifacts.<output-key>).

        Args:
            ir_parameter_dict (Dict[str, Any]): The ComponentOutputsSpec in dict format.

        Returns:
            OutputSpec: The OutputSpec object.
        """
        type_string = ir_parameter_dict[
            'parameterType'] if 'parameterType' in ir_parameter_dict else ir_parameter_dict[
                'artifactType']['schemaTitle']
        type_ = type_utils.IR_TYPE_TO_IN_MEMORY_SPEC_TYPE.get(type_string)
        if type_ is None:
            raise ValueError(
                f'Unknown type {ir_parameter_dict["parameterType"]} found in IR.'
            )
        return OutputSpec(type=type_)

    def __eq__(self, other: Any) -> bool:
        """Equality comparison for OutputSpec. Robust to different type
        representations, such that it respects the maximum amount of
        information possible to encode in IR. That is, because
        `typing.List[str]` can only be represented a `List` in IR,
        'typing.List' == 'List' in this comparison.

        Args:
            other (Any): The object to compare to OutputSpec.

        Returns:
            bool: True if the objects are equal, False otherwise.
        """
        if isinstance(other, OutputSpec):
            return type_utils.get_canonical_name_for_outer_generic(
                self.type) == type_utils.get_canonical_name_for_outer_generic(
                    other.type)
        else:
            return False


P = TypeVar('P', bound='PlaceholderSerializationMixin')


class PlaceholderSerializationMixin:
    """Mixin for *Placeholder objects that handles the
    serialization/deserialization of the placeholder."""
    _FROM_PLACEHOLDER_REGEX: Union[str, type(NotImplemented)] = NotImplemented
    _TO_PLACEHOLDER_TEMPLATE_STRING: Union[
        str, type(NotImplemented)] = NotImplemented

    @classmethod
    def _is_input_placeholder(cls) -> bool:
        """Checks if the class is an InputPlaceholder (rather than an
        OutputPlaceholder).

        Returns:
            bool: True if the class is an InputPlaceholder, False otherwise.
        """
        field_names = {field.name for field in dataclasses.fields(cls)}
        return "input_name" in field_names

    @classmethod
    def is_match(cls: Type[P], placeholder_string: str) -> bool:
        """Determines if the placeholder_string matches the placeholder
        pattern.

        Args:
            placeholder_string (str): The string (often "{{$.inputs/outputs...}}") to check.

        Returns:
            bool: Determines if the placeholder_string matches the placeholder pattern.
        """
        return re.match(cls._FROM_PLACEHOLDER_REGEX,
                        placeholder_string) is not None

    @classmethod
    def from_placeholder(cls: Type[P], placeholder_string: str) -> P:
        """Converts a placeholder string into a placeholder object.

        Args:
            placeholder_string (str): The placeholder.

        Returns:
            PlaceholderSerializationMixin subclass: The placeholder object.
        """
        if cls._FROM_PLACEHOLDER_REGEX == NotImplemented:
            raise NotImplementedError(
                f'{cls.__name__} does not support placeholder parsing.')

        matches = re.search(cls._FROM_PLACEHOLDER_REGEX, placeholder_string)
        if matches is None:
            raise ValueError(
                f'Could not parse placeholder: {placeholder_string} into {cls.__name__}'
            )
        if cls._is_input_placeholder():
            return cls(input_name=matches[1])  # type: ignore
        else:
            return cls(output_name=matches[1])  # type: ignore

    def to_placeholder(self: P) -> str:
        """Converts a placeholder object into a placeholder string.

        Returns:
            str: The placeholder string.
        """
        if self._TO_PLACEHOLDER_TEMPLATE_STRING == NotImplemented:
            raise NotImplementedError(
                f'{self.__class__.__name__} does not support creating placeholder strings.'
            )
        attr_name = 'input_name' if self._is_input_placeholder(
        ) else 'output_name'
        value = getattr(self, attr_name)
        return self._TO_PLACEHOLDER_TEMPLATE_STRING.format(value)


class InputValuePlaceholder(base_model.BaseModel,
                            PlaceholderSerializationMixin):
    """Class that holds input value for conditional cases.

    Attributes:
        input_name: name of the input.
    """
    input_name: str
    _aliases = {'input_name': 'inputValue'}
    _TO_PLACEHOLDER_TEMPLATE_STRING = "{{{{$.inputs.parameters['{}']}}}}"
    _FROM_PLACEHOLDER_REGEX = r"\{\{\$\.inputs\.parameters\[(?:''|'|\")(.+?)(?:''|'|\")]\}\}"


class InputPathPlaceholder(base_model.BaseModel, PlaceholderSerializationMixin):
    """Class that holds input path for conditional cases.

    Attributes:
        input_name: name of the input.
    """
    input_name: str
    _aliases = {'input_name': 'inputPath'}
    _TO_PLACEHOLDER_TEMPLATE_STRING = "{{{{$.inputs.artifacts['{}'].path}}}}"
    _FROM_PLACEHOLDER_REGEX = r"\{\{\$\.inputs\.artifacts\[(?:''|'|\")(.+?)(?:''|'|\")]\.path\}\}"


class InputUriPlaceholder(base_model.BaseModel, PlaceholderSerializationMixin):
    """Class that holds input uri for conditional cases.

    Attributes:
        input_name: name of the input.
    """
    input_name: str
    _aliases = {'input_name': 'inputUri'}
    _TO_PLACEHOLDER_TEMPLATE_STRING = "{{{{$.inputs.artifacts['{}'].uri}}}}"
    _FROM_PLACEHOLDER_REGEX = r"\{\{\$\.inputs\.artifacts\[(?:''|'|\")(.+?)(?:''|'|\")]\.uri\}\}"


class OutputParameterPlaceholder(base_model.BaseModel,
                                 PlaceholderSerializationMixin):
    """Class that holds output path for conditional cases.

    Attributes:
        output_name: name of the output.
    """
    output_name: str
    _aliases = {'output_name': 'outputPath'}
    _TO_PLACEHOLDER_TEMPLATE_STRING = "{{{{$.outputs.parameters['{}'].output_file}}}}"
    _FROM_PLACEHOLDER_REGEX = r"\{\{\$\.outputs\.parameters\[(?:''|'|\")(.+?)(?:''|'|\")]\.output_file\}\}"


class OutputPathPlaceholder(base_model.BaseModel,
                            PlaceholderSerializationMixin):
    """Class that holds output path for conditional cases.

    Attributes:
        output_name: name of the output.
    """
    output_name: str
    _aliases = {'output_name': 'outputPath'}
    _TO_PLACEHOLDER_TEMPLATE_STRING = "{{{{$.outputs.artifacts['{}'].path}}}}"
    _FROM_PLACEHOLDER_REGEX = r"\{\{\$\.outputs\.artifacts\[(?:''|'|\")(.+?)(?:''|'|\")]\.path\}\}"


class OutputUriPlaceholder(base_model.BaseModel, PlaceholderSerializationMixin):
    """Class that holds output uri for conditional cases.

    Attributes:
        output_name: name of the output.
    """
    output_name: str
    _aliases = {'output_name': 'outputUri'}
    _TO_PLACEHOLDER_TEMPLATE_STRING = "{{{{$.outputs.artifacts['{}'].uri}}}}"
    _FROM_PLACEHOLDER_REGEX = r"\{\{\$\.outputs\.artifacts\[(?:''|'|\")(.+?)(?:''|'|\")]\.uri\}\}"


ValidCommandArgs = Union[str, InputValuePlaceholder, InputPathPlaceholder,
                         InputUriPlaceholder, OutputPathPlaceholder,
                         OutputUriPlaceholder, OutputParameterPlaceholder,
                         'IfPresentPlaceholder', 'ConcatPlaceholder']


class ConcatPlaceholder(base_model.BaseModel):
    """Class that extends basePlaceholders for concatenation.

    Attributes:
        items: string or ValidCommandArgs for concatenation.
    """
    items: List[ValidCommandArgs]
    _aliases = {'items': 'concat'}

    @classmethod
    def from_concat_string(cls, concat_string: str) -> 'ConcatPlaceholder':
        """Creates a concat placeholder from an IR string indicating
        concatenation.

        Args:
            concat_string (str): The IR string (e.g. {{$.inputs.parameters['input1']}}+{{$.inputs.parameters['input2']}})

        Returns:
            ConcatPlaceholder: The ConcatPlaceholder instance.
        """
        items = []
        for a in concat_string.split('+'):
            items.extend([maybe_convert_command_arg_to_placeholder(a), '+'])
        del items[-1]
        return ConcatPlaceholder(items=items)


class IfPresentPlaceholderStructure(base_model.BaseModel):
    """Class that holds structure for conditional cases.

    Attributes:
        input_name: name of the input/output.
        then: If the input/output specified in name is present,
            the command-line argument will be replaced at run-time by the
            expanded value of then.
        otherwise: If the input/output specified in name is not present,
            the command-line argument will be replaced at run-time by the
            expanded value of otherwise.
    """
    input_name: str
    then: List[ValidCommandArgs]
    otherwise: Optional[List[ValidCommandArgs]] = None
    _aliases = {'input_name': 'inputName', 'otherwise': 'else'}

    def transform_otherwise(self) -> None:
        """Use None instead of empty list for optional."""
        self.otherwise = None if self.otherwise == [] else self.otherwise


class IfPresentPlaceholder(base_model.BaseModel):
    """Class that extends basePlaceholders for conditional cases.

    Attributes:
        if_present (ifPresent): holds structure for conditional cases.
    """
    if_structure: IfPresentPlaceholderStructure
    _aliases = {'if_structure': 'ifPresent'}


class ResourceSpec(base_model.BaseModel):
    """The resource requirements of a container execution.

    Attributes:
        cpu_limit (optional): the limit of the number of vCPU cores.
        memory_limit (optional): the memory limit in GB.
        accelerator_type (optional): the type of accelerators attached to the
            container.
        accelerator_count (optional): the number of accelerators attached.
    """
    cpu_limit: Optional[float] = None
    memory_limit: Optional[float] = None
    accelerator_type: Optional[str] = None
    accelerator_count: Optional[int] = None


class ContainerSpec(base_model.BaseModel):
    """Container implementation definition.

    Attributes:
        image: The container image.
        command (optional): the container entrypoint.
        args (optional): the arguments to the container entrypoint.
        env (optional): the environment variables to be passed to the container.
        resources (optional): the specification on the resource requirements.
    """
    image: str
    command: Optional[List[ValidCommandArgs]] = None
    args: Optional[List[ValidCommandArgs]] = None
    env: Optional[Mapping[str, ValidCommandArgs]] = None
    resources: Optional[ResourceSpec] = None

    def transform_command(self) -> None:
        """Use None instead of empty list for command."""
        self.command = None if self.command == [] else self.command

    def transform_args(self) -> None:
        """Use None instead of empty list for args."""
        self.args = None if self.args == [] else self.args

    def transform_env(self) -> None:
        """Use None instead of empty dict for env."""
        self.env = None if self.env == {} else self.env

    @classmethod
    def from_container_dict(cls, container_dict: Dict[str,
                                                      Any]) -> 'ContainerSpec':
        """Creates a ContainerSpec from a PipelineContainerSpec message in dict
        format (pipeline_spec.deploymentSpec.executors.<executor-
        key>.container).

        Args:
            container_dict (Dict[str, Any]): PipelineContainerSpec message in dict format.

        Returns:
            ContainerSpec: The ContainerSpec instance.
        """

        args = container_dict.get('args')
        if args is not None:
            args = [
                maybe_convert_command_arg_to_placeholder(arg) for arg in args
            ]
        command = container_dict.get('command')
        if command is not None:
            command = [
                maybe_convert_command_arg_to_placeholder(c) for c in command
            ]
        return ContainerSpec(
            image=container_dict['image'],
            command=command,
            args=args,
            env=None,  # can only be set on tasks
            resources=None)  # can only be set on tasks


def maybe_convert_command_arg_to_placeholder(arg: str) -> ValidCommandArgs:
    """Infers if a command is a placeholder and converts it to the correct
    Placeholder object.

    Args:
        arg (str): The arg or command to possibly convert.

    Returns:
        ValidCommandArgs: The converted command or original string.
    """
    # short path to avoid checking all regexs
    if not arg.startswith('{{$'):
        return arg

    # handle concat placeholder
    # TODO: change when support for concat is added to IR
    if '}}+{{' in arg:
        return ConcatPlaceholder.from_concat_string(arg)

    placeholders = {
        InputValuePlaceholder,
        InputPathPlaceholder,
        InputUriPlaceholder,
        OutputPathPlaceholder,
        OutputUriPlaceholder,
        OutputParameterPlaceholder,
    }
    for placeholder_struct in placeholders:
        if placeholder_struct.is_match(arg):
            return placeholder_struct.from_placeholder(arg)

    return arg


class TaskSpec(base_model.BaseModel):
    """The spec of a pipeline task.

    Attributes:
        name: The name of the task.
        inputs: The sources of task inputs. Constant values or PipelineParams.
        dependent_tasks: The list of upstream tasks.
        component_ref: The name of a component spec this task is based on.
        trigger_condition (optional): an expression which will be evaluated into
            a boolean value. True to trigger the task to run.
        trigger_strategy (optional): when the task will be ready to be triggered.
            Valid values include: "TRIGGER_STRATEGY_UNSPECIFIED",
            "ALL_UPSTREAM_TASKS_SUCCEEDED", and "ALL_UPSTREAM_TASKS_COMPLETED".
        iterator_items (optional): the items to iterate on. A constant value or
            a PipelineParam.
        iterator_item_input (optional): the name of the input which has the item
            from the [items][] collection.
        enable_caching (optional): whether or not to enable caching for the task.
            Default is True.
        display_name (optional): the display name of the task. If not specified,
            the task name will be used as the display name.
    """
    name: str
    inputs: Mapping[str, Any]
    dependent_tasks: List[str]
    component_ref: str
    trigger_condition: Optional[str] = None
    trigger_strategy: Optional[str] = None
    iterator_items: Optional[Any] = None
    iterator_item_input: Optional[str] = None
    enable_caching: bool = True
    display_name: Optional[str] = None


class DagSpec(base_model.BaseModel):
    """DAG(graph) implementation definition.

    Attributes:
        tasks: The tasks inside the DAG.
        outputs: Defines how the outputs of the dag are linked to the sub tasks.
    """
    tasks: Mapping[str, TaskSpec]
    outputs: Mapping[str, Any]


class ImporterSpec(base_model.BaseModel):
    """ImporterSpec definition.

    Attributes:
        artifact_uri: The URI of the artifact.
        type_schema: The type of the artifact.
        reimport: Whether or not import an artifact regardless it has been
         imported before.
        metadata (optional): the properties of the artifact.
    """
    artifact_uri: str
    type_schema: str
    reimport: bool
    metadata: Optional[Mapping[str, Any]] = None


class Implementation(base_model.BaseModel):
    """Implementation definition.

    Attributes:
        container: container implementation details.
        graph: graph implementation details.
        importer: importer implementation details.
    """
    container: Optional[ContainerSpec] = None
    graph: Optional[DagSpec] = None
    importer: Optional[ImporterSpec] = None

    @classmethod
    def from_deployment_spec_dict(cls, deployment_spec_dict: Dict[str, Any],
                                  component_name: str) -> 'Implementation':
        """Creates an Implmentation object from a deployment spec message in
        dict format (pipeline_spec.deploymentSpec).

        Args:
            deployment_spec_dict (Dict[str, Any]): PipelineDeploymentConfig message in dict format.
            component_name (str): The name of the component.

        Returns:
            Implementation: An implementation object.
        """
        executor_key = utils._EXECUTOR_LABEL_PREFIX + component_name
        container = deployment_spec_dict['executors'][executor_key]['container']
        container_spec = ContainerSpec.from_container_dict(container)
        return Implementation(container=container_spec)


def try_to_get_dict_from_string(string: str) -> Union[dict, str]:
    """Tries to parse a dictionary from a string if possible, else returns the
    string.

    Args:
        string (str): The string to parse.

    Returns:
        Union[dict, str]: The dictionary if one was successfully parsed, else the original string.
    """
    try:
        res = ast.literal_eval(string)
    except (ValueError, SyntaxError):
        return string

    if not isinstance(res, dict):
        return string
    return res


def convert_str_or_dict_to_placeholder(
    element: Union[str, dict,
                   ValidCommandArgs]) -> Union[str, ValidCommandArgs]:
    """Converts command and args elements to a placholder type based on value
    of the key of the placeholder string, else returns the input.

    Args:
        element (Union[str, dict, ValidCommandArgs]): A ContainerSpec.command or ContainerSpec.args element.

    Raises:
        TypeError: If `element` is invalid.

    Returns:
        Union[str, ValidCommandArgs]: Possibly converted placeholder or original input.
    """

    if not isinstance(element, (dict, str)):
        return element

    elif isinstance(element, str):
        res = try_to_get_dict_from_string(element)
        if not isinstance(res, dict):
            return element

    elif isinstance(element, dict):
        res = element
    else:
        raise TypeError(
            f'Invalid type for arg: {type(element)}. Expected str or dict.')

    has_one_entry = len(res) == 1

    if not has_one_entry:
        raise ValueError(
            f'Got unexpected dictionary {res}. Expected a dictionary with one entry.'
        )

    first_key = list(res.keys())[0]
    first_value = list(res.values())[0]
    if first_key == 'inputValue':
        return InputValuePlaceholder(
            input_name=utils.sanitize_input_name(first_value))

    elif first_key == 'inputPath':
        return InputPathPlaceholder(
            input_name=utils.sanitize_input_name(first_value))

    elif first_key == 'inputUri':
        return InputUriPlaceholder(
            input_name=utils.sanitize_input_name(first_value))

    elif first_key == 'outputPath':
        return OutputPathPlaceholder(
            output_name=utils.sanitize_input_name(first_value))

    elif first_key == 'outputUri':
        return OutputUriPlaceholder(
            output_name=utils.sanitize_input_name(first_value))

    elif first_key == 'ifPresent':
        structure_kwargs = res['ifPresent']
        structure_kwargs['input_name'] = structure_kwargs.pop('inputName')
        structure_kwargs['otherwise'] = structure_kwargs.pop('else')
        structure_kwargs['then'] = [
            convert_str_or_dict_to_placeholder(e)
            for e in structure_kwargs['then']
        ]
        structure_kwargs['otherwise'] = [
            convert_str_or_dict_to_placeholder(e)
            for e in structure_kwargs['otherwise']
        ]
        if_structure = IfPresentPlaceholderStructure(**structure_kwargs)

        return IfPresentPlaceholder(if_structure=if_structure)

    elif first_key == 'concat':
        return ConcatPlaceholder(items=[
            convert_str_or_dict_to_placeholder(e) for e in res['concat']
        ])

    else:
        raise TypeError(
            f'Unexpected command/argument type: "{element}" of type "{type(element)}".'
        )


def _check_valid_placeholder_reference(valid_inputs: List[str],
                                       valid_outputs: List[str],
                                       placeholder: ValidCommandArgs) -> None:
    """Validates input/output placeholders refer to an existing input/output.

    Args:
        valid_inputs: The existing input names.
        valid_outputs: The existing output names.
        arg: The placeholder argument for checking.

    Raises:
        ValueError: if any placeholder references a non-existing input or
            output.
        TypeError: if any argument is neither a str nor a placeholder
            instance.
    """
    if isinstance(
            placeholder,
        (InputValuePlaceholder, InputPathPlaceholder, InputUriPlaceholder)):
        if placeholder.input_name not in valid_inputs:
            raise ValueError(
                f'Argument "{placeholder}" references non-existing input.')
    elif isinstance(placeholder, (OutputParameterPlaceholder,
                                  OutputPathPlaceholder, OutputUriPlaceholder)):
        if placeholder.output_name not in valid_outputs:
            raise ValueError(
                f'Argument "{placeholder}" references non-existing output.')
    elif isinstance(placeholder, IfPresentPlaceholder):
        if placeholder.if_structure.input_name not in valid_inputs:
            raise ValueError(
                f'Argument "{placeholder}" references non-existing input.')
        for placeholder in itertools.chain(
                placeholder.if_structure.then or [],
                placeholder.if_structure.otherwise or []):
            _check_valid_placeholder_reference(valid_inputs, valid_outputs,
                                               placeholder)
    elif isinstance(placeholder, ConcatPlaceholder):
        for placeholder in placeholder.items:
            _check_valid_placeholder_reference(valid_inputs, valid_outputs,
                                               placeholder)
    elif not isinstance(placeholder, str):
        raise TypeError(
            f'Unexpected argument "{placeholder}" of type {type(placeholder)}.')


ValidCommandArgTypes = (str, InputValuePlaceholder, InputPathPlaceholder,
                        InputUriPlaceholder, OutputPathPlaceholder,
                        OutputUriPlaceholder, IfPresentPlaceholder,
                        ConcatPlaceholder)


class ComponentSpec(base_model.BaseModel):
    """The definition of a component.

    Attributes:
        name: The name of the component.
        description (optional): the description of the component.
        inputs (optional): the input definitions of the component.
        outputs (optional): the output definitions of the component.
        implementation: The implementation of the component. Either an executor
            (container, importer) or a DAG consists of other components.
    """
    name: str
    implementation: Implementation
    description: Optional[str] = None
    inputs: Optional[Dict[str, InputSpec]] = None
    outputs: Optional[Dict[str, OutputSpec]] = None

    def transform_name(self) -> None:
        """Converts the name to a valid name."""
        self.name = utils.maybe_rename_for_k8s(self.name)

    def transform_inputs(self) -> None:
        """Use None instead of empty list for inputs."""
        self.inputs = None if self.inputs == {} else self.inputs

    def transform_outputs(self) -> None:
        """Use None instead of empty list for outputs."""
        self.outputs = None if self.outputs == {} else self.outputs

    def transform_command_input_placeholders(self) -> None:
        """Converts command and args elements to a placholder type where
        applicable."""
        if self.implementation.container is not None:

            if self.implementation.container.command is not None:
                self.implementation.container.command = [
                    convert_str_or_dict_to_placeholder(e)
                    for e in self.implementation.container.command
                ]

            if self.implementation.container.args is not None:
                self.implementation.container.args = [
                    convert_str_or_dict_to_placeholder(e)
                    for e in self.implementation.container.args
                ]

    def validate_placeholders(self):
        """Validates that input/output placeholders refer to an existing
        input/output."""
        implementation = self.implementation
        if getattr(implementation, 'container', None) is None:
            return

        containerSpec: ContainerSpec = implementation.container

        valid_inputs = [] if self.inputs is None else list(self.inputs.keys())
        valid_outputs = [] if self.outputs is None else list(
            self.outputs.keys())

        for arg in itertools.chain((containerSpec.command or []),
                                   (containerSpec.args or [])):
            _check_valid_placeholder_reference(valid_inputs, valid_outputs, arg)

    @classmethod
    def from_v1_component_spec(
            cls,
            v1_component_spec: v1_structures.ComponentSpec) -> 'ComponentSpec':
        """Converts V1 ComponentSpec to V2 ComponentSpec.

        Args:
            v1_component_spec: The V1 ComponentSpec.

        Returns:
            Component spec in the form of V2 ComponentSpec.

        Raises:
            ValueError: If implementation is not found.
            TypeError: if any argument is neither a str nor Dict.
        """
        component_dict = v1_component_spec.to_dict()
        if component_dict.get('implementation') is None:
            raise ValueError('Implementation field not found')

        if 'container' not in component_dict.get(
                'implementation'):  # type: ignore
            raise NotImplementedError

        def convert_v1_if_present_placholder_to_v2(
                arg: Dict[str, Any]) -> Union[Dict[str, Any], ValidCommandArgs]:
            if isinstance(arg, str):
                arg = try_to_get_dict_from_string(arg)
            if not isinstance(arg, dict):
                return arg

            if 'if' in arg:
                if_placeholder_values = arg['if']
                if_placeholder_values_then = list(if_placeholder_values['then'])
                try:
                    if_placeholder_values_else = list(
                        if_placeholder_values['else'])
                except KeyError:
                    if_placeholder_values_else = []
                return IfPresentPlaceholder(
                    if_structure=IfPresentPlaceholderStructure(
                        input_name=utils.sanitize_input_name(
                            if_placeholder_values['cond']['isPresent']),
                        then=[
                            convert_str_or_dict_to_placeholder(
                                convert_v1_if_present_placholder_to_v2(val))
                            for val in if_placeholder_values_then
                        ],
                        otherwise=[
                            convert_str_or_dict_to_placeholder(
                                convert_v1_if_present_placholder_to_v2(val))
                            for val in if_placeholder_values_else
                        ]))

            elif 'concat' in arg:
                return ConcatPlaceholder(items=[
                    convert_str_or_dict_to_placeholder(
                        convert_v1_if_present_placholder_to_v2(val))
                    for val in arg['concat']
                ])
            elif isinstance(arg, (ValidCommandArgTypes, dict)):
                return arg
            else:
                raise TypeError(
                    f'Unexpected argument {arg} of type {type(arg)}.')

        implementation = component_dict['implementation']['container']
        implementation['command'] = [
            convert_v1_if_present_placholder_to_v2(command)
            for command in implementation.pop('command', [])
        ]
        implementation['args'] = [
            convert_v1_if_present_placholder_to_v2(command)
            for command in implementation.pop('args', [])
        ]
        implementation['env'] = {
            key: convert_v1_if_present_placholder_to_v2(command)
            for key, command in implementation.pop('env', {}).items()
        }
        container_spec = ContainerSpec(image=implementation['image'])

        # Must assign these after the constructor call, otherwise it won't work.
        if implementation['command']:
            container_spec.command = implementation['command']
        if implementation['args']:
            container_spec.args = implementation['args']
        if implementation['env']:
            container_spec.env = implementation['env']

        return ComponentSpec(
            name=component_dict.get('name', 'name'),
            description=component_dict.get('description'),
            implementation=Implementation(container=container_spec),
            inputs={
                utils.sanitize_input_name(spec['name']): InputSpec(
                    type=spec.get('type', 'Artifact'),
                    default=spec.get('default', None))
                for spec in component_dict.get('inputs', [])
            },
            outputs={
                utils.sanitize_input_name(spec['name']):
                OutputSpec(type=spec.get('type', 'Artifact'))
                for spec in component_dict.get('outputs', [])
            })

    @classmethod
    def from_pipeline_spec_dict(
            cls, pipeline_spec_dict: Dict[str, Any]) -> 'ComponentSpec':
        raw_name = pipeline_spec_dict['pipelineInfo']['name']

        implementation = Implementation.from_deployment_spec_dict(
            pipeline_spec_dict['deploymentSpec'], raw_name)

        def inputs_dict_from_components_dict(
                components_dict: Dict[str, Any],
                component_name: str) -> Dict[str, InputSpec]:
            component_key = utils._COMPONENT_NAME_PREFIX + component_name
            parameters = components_dict[component_key].get(
                'inputDefinitions', {}).get('parameters', {})
            return {
                name: InputSpec.from_ir_parameter_dict(parameter_dict)
                for name, parameter_dict in parameters.items()
            }

        def outputs_dict_from_components_dict(
                components_dict: Dict[str, Any],
                component_name: str) -> Dict[str, OutputSpec]:
            component_key = utils._COMPONENT_NAME_PREFIX + component_name
            parameters = components_dict[component_key].get(
                'outputDefinitions', {}).get('parameters', {})
            artifacts = components_dict[component_key].get(
                'outputDefinitions', {}).get('artifacts', {})
            all_outputs = {**parameters, **artifacts}
            return {
                name: OutputSpec.from_ir_parameter_dict(parameter_dict)
                for name, parameter_dict in all_outputs.items()
            }

        def extract_description_from_command(commands: List[str]) -> str:
            for command in commands:
                if isinstance(command, str) and 'import kfp' in command:
                    for node in ast.walk(ast.parse(command)):
                        if isinstance(
                                node,
                            (ast.FunctionDef, ast.ClassDef, ast.Module)):
                            docstring = ast.get_docstring(node)
                            if docstring:
                                return docstring
            return ''

        inputs = inputs_dict_from_components_dict(
            pipeline_spec_dict['components'], raw_name)
        outputs = outputs_dict_from_components_dict(
            pipeline_spec_dict['components'], raw_name)

        description = extract_description_from_command(
            implementation.container.command) or None
        return ComponentSpec(
            name=raw_name,
            implementation=implementation,
            description=description,
            inputs=inputs,
            outputs=outputs)

    @classmethod
    def from_pipeline_spec_yaml(cls,
                                pipeline_spec_yaml: str) -> 'ComponentSpec':
        """Creates a ComponentSpec from a pipeline spec in YAML format.

        Args:
            component_yaml (str): Component spec in YAML format.

        Returns:
            ComponentSpec: The component spec object.
        """
        return ComponentSpec.from_pipeline_spec_dict(
            yaml.safe_load(pipeline_spec_yaml))

    @classmethod
    def load_from_component_yaml(cls, component_yaml: str) -> 'ComponentSpec':
        """Loads V1 or V2 component yaml into ComponentSpec.

        Args:
            component_yaml: the component yaml in string format.

        Returns:
            Component spec in the form of V2 ComponentSpec.
        """

        json_component = yaml.safe_load(component_yaml)
        is_v1 = 'implementation' in set(json_component.keys())
        if is_v1:
            v1_component = v1_components._load_component_spec_from_component_text(
                component_yaml)
            return cls.from_v1_component_spec(v1_component)
        else:
            return ComponentSpec.from_pipeline_spec_dict(json_component)

    def save_to_component_yaml(self, output_file: str) -> None:
        """Saves ComponentSpec into IR YAML file.

        Args:
            output_file: File path to store the component yaml.
        """

        pipeline_spec = self.to_pipeline_spec()
        compiler.write_pipeline_spec_to_file(pipeline_spec, output_file)

    def to_pipeline_spec(self) -> pipeline_spec_pb2.PipelineSpec:
        """Creates a pipeline instance and constructs the pipeline spec for a
        single component.

        Args:
            component_spec: The ComponentSpec to convert to PipelineSpec.

        Returns:
            A PipelineSpec proto representing the compiled component.
        """
        # import here to aviod circular module dependency
        from kfp.compiler import pipeline_spec_builder as builder
        from kfp.components import pipeline_task
        from kfp.components import tasks_group
        from kfp.components.types import type_utils

        args_dict = {}
        pipeline_inputs = self.inputs or {}

        for arg_name, input_spec in pipeline_inputs.items():
            arg_type = input_spec.type
            if not type_utils.is_parameter_type(
                    arg_type) or type_utils.is_task_final_status_type(arg_type):
                raise TypeError(
                    builder.make_invalid_input_type_error_msg(
                        arg_name, arg_type))
            args_dict[arg_name] = dsl.PipelineParameterChannel(
                name=arg_name, channel_type=arg_type)

        task = pipeline_task.PipelineTask(self, args_dict)

        # instead of constructing a pipeline with pipeline_context.Pipeline,
        # just build the single task group
        group = tasks_group.TasksGroup(
            group_type=tasks_group.TasksGroupType.PIPELINE)
        group.tasks.append(task)

        # Fill in the default values.
        args_list_with_defaults = [
            dsl.PipelineParameterChannel(
                name=input_name,
                channel_type=input_spec.type,
                value=input_spec.default,
            ) for input_name, input_spec in pipeline_inputs.items()
        ]
        group.name = uuid.uuid4().hex

        pipeline_name = self.name
        pipeline_args = args_list_with_defaults
        task_group = group

        builder.validate_pipeline_name(pipeline_name)

        pipeline_spec = pipeline_spec_pb2.PipelineSpec()
        pipeline_spec.pipeline_info.name = pipeline_name
        pipeline_spec.sdk_version = f'kfp-{kfp.__version__}'
        # Schema version 2.1.0 is required for kfp-pipeline-spec>0.1.13
        pipeline_spec.schema_version = '2.1.0'
        pipeline_spec.root.CopyFrom(
            builder.build_component_spec_for_group(
                pipeline_channels=pipeline_args,
                is_root_group=True,
            ))

        deployment_config = pipeline_spec_pb2.PipelineDeploymentConfig()
        root_group = task_group

        task_name_to_parent_groups, group_name_to_parent_groups = builder.get_parent_groups(
            root_group)

        def get_inputs(task_group: tasks_group.TasksGroup,
                       task_name_to_parent_groups):
            inputs = collections.defaultdict(set)
            if len(task_group.tasks) != 1:
                raise ValueError(
                    f'Error compiling component. Expected one task in task group, got {len(task_group.tasks)}.'
                )
            only_task = task_group.tasks[0]
            if only_task.channel_inputs:
                for group_name in task_name_to_parent_groups[only_task.name]:
                    inputs[group_name].add((only_task.channel_inputs[-1], None))
            return inputs

        inputs = get_inputs(task_group, task_name_to_parent_groups)

        builder.build_spec_by_group(
            pipeline_spec=pipeline_spec,
            deployment_config=deployment_config,
            group=root_group,
            inputs=inputs,
            dependencies={},  # no dependencies for single-component pipeline
            rootgroup_name=root_group.name,
            task_name_to_parent_groups=task_name_to_parent_groups,
            group_name_to_parent_groups=group_name_to_parent_groups,
            name_to_for_loop_group={},  # no for loop for single-component pipeline
        )

        return pipeline_spec
