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
import functools
import itertools
from typing import Any, Dict, List, Mapping, Optional, Union

import yaml
from kfp.components import base_model
from kfp.components import utils
from kfp.components import v1_components
from kfp.components import v1_structures
from kfp.utils import ir_utils


class InputSpec_(base_model.BaseModel):
    """Component input definitions. (Inner class).

    Attributes:
        type: The type of the input.
        default (optional): the default value for the input.
        description: Optional: the user description of the input.
    """
    type: Union[str, dict]
    default: Union[Any, None] = None
    description: Optional[str] = None


# Hack to allow access to __init__ arguments for setting _optional value
class InputSpec(InputSpec_, base_model.BaseModel):
    """Component input definitions.

    Attributes:
        type: The type of the input.
        default (optional): the default value for the input.
        description: Optional: the user description of the input.
        _optional: Wether the input is optional. An input is optional when it has an explicit default value.
    """

    @functools.wraps(InputSpec_.__init__)
    def __init__(self, *args, **kwargs):
        if args:
            raise ValueError('InputSpec does not accept positional arguments.')
        super().__init__(*args, **kwargs)
        self._optional = 'default' in kwargs


class OutputSpec(base_model.BaseModel):
    """Component output definitions.

    Attributes:
        type: The type of the output.
        description: Optional: the user description of the output.
    """
    type: Union[str, dict]
    description: Optional[str] = None


class InputValuePlaceholder(base_model.BaseModel):
    """Class that holds input value for conditional cases.

    Attributes:
        input_name: name of the input.
    """
    input_name: str
    _aliases = {'input_name': 'inputValue'}


class InputPathPlaceholder(base_model.BaseModel):
    """Class that holds input path for conditional cases.

    Attributes:
        input_name: name of the input.
    """
    input_name: str
    _aliases = {'input_name': 'inputPath'}


class InputUriPlaceholder(base_model.BaseModel):
    """Class that holds input uri for conditional cases.

    Attributes:
        input_name: name of the input.
    """
    input_name: str
    _aliases = {'input_name': 'inputUri'}


class OutputPathPlaceholder(base_model.BaseModel):
    """Class that holds output path for conditional cases.

    Attributes:
        output_name: name of the output.
    """
    output_name: str
    _aliases = {'output_name': 'outputPath'}


class OutputUriPlaceholder(base_model.BaseModel):
    """Class that holds output uri for conditional cases.

    Attributes:
        output_name: name of the output.
    """
    output_name: str
    _aliases = {'output_name': 'outputUri'}


ValidCommandArgs = Union[str, InputValuePlaceholder, InputPathPlaceholder,
                         InputUriPlaceholder, OutputPathPlaceholder,
                         OutputUriPlaceholder, 'IfPresentPlaceholder',
                         'ConcatPlaceholder']


class ConcatPlaceholder(base_model.BaseModel):
    """Class that extends basePlaceholders for concatenation.

    Attributes:
        items: string or ValidCommandArgs for concatenation.
    """
    items: List[ValidCommandArgs]
    _aliases = {'items': 'concat'}


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


def try_to_get_dict_from_string(element: str) -> Union[dict, str]:
    try:
        res = ast.literal_eval(element)
    except (ValueError, SyntaxError):
        return element

    if not isinstance(res, dict):
        return element
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
    elif isinstance(placeholder, (OutputPathPlaceholder, OutputUriPlaceholder)):
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
                            convert_str_or_dict_to_placeholder(val)
                            for val in if_placeholder_values_then
                        ],
                        otherwise=[
                            convert_str_or_dict_to_placeholder(val)
                            for val in if_placeholder_values_else
                        ]))

            elif 'concat' in arg:

                return ConcatPlaceholder(items=[
                    convert_str_or_dict_to_placeholder(val)
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
    def load_from_component_yaml(cls, component_yaml: str) -> 'ComponentSpec':
        """Loads V1 or V2 component yaml into ComponentSpec.

        Args:
            component_yaml: the component yaml in string format.

        Returns:
            Component spec in the form of V2 ComponentSpec.
        """
        json_component = yaml.safe_load(component_yaml)
        try:
            return ComponentSpec.from_dict(json_component, by_alias=True)
        except AttributeError:
            v1_component = v1_components._load_component_spec_from_component_text(
                component_yaml)
            return cls.from_v1_component_spec(v1_component)

    def save_to_component_yaml(self, output_file: str) -> None:
        """Saves ComponentSpec into YAML file.

        Args:
            output_file: File path to store the component yaml.
        """
        ir_utils._write_ir_to_file(self.to_dict(by_alias=True), output_file)
