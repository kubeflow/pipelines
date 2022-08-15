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
import functools
import itertools
import re
from typing import Any, Dict, List, Mapping, Optional, Union
import uuid

from google.protobuf import json_format
import kfp
from kfp.compiler import compiler
from kfp.components import base_model
from kfp.components import placeholders
from kfp.components import utils
from kfp.components import v1_components
from kfp.components import v1_structures
from kfp.components.container_component_artifact_channel import \
    ContainerComponentArtifactChannel
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
    """Container definition.

    This is only used for pipeline authors when constructing a containerized component
    using @container_component decorator.

    Examples:
      ::

        @container_component
        def container_with_artifact_output(
            num_epochs: int,  # built-in types are parsed as inputs
            model: Output[Model],
            model_config_path: OutputPath(str),
        ):
            return ContainerSpec(
                image='gcr.io/my-image',
                command=['sh', 'run.sh'],
                args=[
                    '--epochs',
                    num_epochs,
                    '--model_path',
                    model.uri,
                    '--model_config_path',
                    model_config_path,
                ])
    """
    image: str
    """Container image."""

    command: Optional[List[placeholders.CommandLineElement]] = None
    """Container entrypoint."""

    args: Optional[List[placeholders.CommandLineElement]] = None
    """Arguments to the container entrypoint."""


class ContainerSpecImplementation(base_model.BaseModel):
    """Container implementation definition."""
    image: str
    """Container image."""

    command: Optional[List[placeholders.CommandLineElement]] = None
    """Container entrypoint."""

    args: Optional[List[placeholders.CommandLineElement]] = None
    """Arguments to the container entrypoint."""

    env: Optional[Mapping[str, placeholders.CommandLineElement]] = None
    """Environment variables to be passed to the container."""

    resources: Optional[ResourceSpec] = None
    """Specification on the resource requirements."""

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
    def from_container_spec(
            cls,
            container_spec: ContainerSpec) -> 'ContainerSpecImplementation':
        return ContainerSpecImplementation(
            image=container_spec.image,
            command=container_spec.command,
            args=container_spec.args,
            env=None,
            resources=None)

    @classmethod
    def from_container_dict(
            cls, container_dict: Dict[str,
                                      Any]) -> 'ContainerSpecImplementation':
        """Creates a ContainerSpecImplementation from a PipelineContainerSpec
        message in dict format
        (pipeline_spec.deploymentSpec.executors.<executor- key>.container).

        Args:
            container_dict (Dict[str, Any]): PipelineContainerSpec message in dict format.

        Returns:
            ContainerSpecImplementation: The ContainerSpecImplementation instance.
        """
        args = container_dict.get('args')
        if args is not None:
            args = [
                placeholders.maybe_convert_placeholder_string_to_placeholder(
                    arg) for arg in args
            ]
        command = container_dict.get('command')
        if command is not None:
            command = [
                placeholders.maybe_convert_placeholder_string_to_placeholder(c)
                for c in command
            ]
        return ContainerSpecImplementation(
            image=container_dict['image'],
            command=command,
            args=args,
            env=None,  # can only be set on tasks
            resources=None)  # can only be set on tasks


class RetryPolicy(base_model.BaseModel):
    """The retry policy of a container execution.

    Attributes:
        num_retries (int): Number of times to retry on failure.
        backoff_duration (int): The the number of seconds to wait before triggering a retry.
        backoff_factor (float): The exponential backoff factor applied to backoff_duration. For example, if backoff_duration="60" (60 seconds) and backoff_factor=2, the first retry will happen after 60 seconds, then after 120, 240, and so on.
        backoff_max_duration (int): The maximum duration during which the task will be retried.
    """
    max_retry_count: Optional[int] = None
    backoff_duration: Optional[str] = None
    backoff_factor: Optional[float] = None
    backoff_max_duration: Optional[str] = None

    def to_proto(self) -> pipeline_spec_pb2.PipelineTaskSpec.RetryPolicy:
        # include defaults so that IR is more reflective of runtime behavior
        max_retry_count = self.max_retry_count or 0
        backoff_duration = self.backoff_duration or '0s'
        backoff_factor = self.backoff_factor or 2.0
        backoff_max_duration = self.backoff_max_duration or '3600s'

        # include max duration seconds cap so that IR is more reflective of runtime behavior
        backoff_duration_seconds = f'{convert_duration_to_seconds(backoff_duration)}s'
        backoff_max_duration_seconds = f'{min(convert_duration_to_seconds(backoff_max_duration), 3600)}s'

        return json_format.ParseDict(
            {
                'max_retry_count': max_retry_count,
                'backoff_duration': backoff_duration_seconds,
                'backoff_factor': backoff_factor,
                'backoff_max_duration': backoff_max_duration_seconds,
            }, pipeline_spec_pb2.PipelineTaskSpec.RetryPolicy())

    @classmethod
    def from_proto(
        cls, retry_policy_proto: pipeline_spec_pb2.PipelineTaskSpec.RetryPolicy
    ) -> 'RetryPolicy':
        return cls.from_dict(
            json_format.MessageToDict(
                retry_policy_proto, preserving_proto_field_name=True))


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
    retry_policy: Optional[RetryPolicy] = None


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
    container: Optional[ContainerSpecImplementation] = None
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
        container_spec = ContainerSpecImplementation.from_container_dict(
            container)
        return Implementation(container=container_spec)


def _check_valid_placeholder_reference(
        valid_inputs: List[str], valid_outputs: List[str],
        placeholder: placeholders.CommandLineElement) -> None:
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
    if isinstance(placeholder, ContainerComponentArtifactChannel):
        raise ValueError(
            'Cannot access artifact by itself in the container definition. Please use .uri or .path instead to access the artifact.'
        )
    elif isinstance(
            placeholder,
        (placeholders.InputValuePlaceholder, placeholders.InputPathPlaceholder,
         placeholders.InputUriPlaceholder)):
        if placeholder.input_name not in valid_inputs:
            raise ValueError(
                f'Argument "{placeholder}" references non-existing input.')
    elif isinstance(placeholder, (placeholders.OutputParameterPlaceholder,
                                  placeholders.OutputPathPlaceholder,
                                  placeholders.OutputUriPlaceholder)):
        if placeholder.output_name not in valid_outputs:
            raise ValueError(
                f'Argument "{placeholder}" references non-existing output.')
    elif isinstance(placeholder, placeholders.IfPresentPlaceholder):
        if placeholder.input_name not in valid_inputs:
            raise ValueError(
                f'Argument "{placeholder}" references non-existing input.')
        for placeholder in itertools.chain(placeholder.then or [],
                                           placeholder.else_ or []):
            _check_valid_placeholder_reference(valid_inputs, valid_outputs,
                                               placeholder)
    elif isinstance(placeholder, placeholders.ConcatPlaceholder):
        for placeholder in placeholder.items:
            _check_valid_placeholder_reference(valid_inputs, valid_outputs,
                                               placeholder)
    elif not isinstance(placeholder, placeholders.ExecutorInputPlaceholder
                       ) and not isinstance(placeholder, str):
        raise TypeError(
            f'Unexpected argument "{placeholder}" of type {type(placeholder)}.')


ValidCommandArgTypes = (str, placeholders.InputValuePlaceholder,
                        placeholders.InputPathPlaceholder,
                        placeholders.InputUriPlaceholder,
                        placeholders.OutputPathPlaceholder,
                        placeholders.OutputUriPlaceholder,
                        placeholders.IfPresentPlaceholder,
                        placeholders.ConcatPlaceholder)


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

    def validate_placeholders(self):
        """Validates that input/output placeholders refer to an existing
        input/output."""
        implementation = self.implementation
        if getattr(implementation, 'container', None) is None:
            return

        containerSpecImplementation: ContainerSpecImplementation = implementation.container

        valid_inputs = [] if self.inputs is None else list(self.inputs.keys())
        valid_outputs = [] if self.outputs is None else list(
            self.outputs.keys())

        for arg in itertools.chain((containerSpecImplementation.command or []),
                                   (containerSpecImplementation.args or [])):
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
            TypeError: If any argument is neither a str nor Dict.
        """
        component_dict = v1_component_spec.to_dict()
        if component_dict.get('implementation') is None:
            raise ValueError('Implementation field not found')

        if 'container' not in component_dict.get(
                'implementation'):  # type: ignore
            raise NotImplementedError('Container implementation not found.')

        container = component_dict['implementation']['container']
        container['command'] = [
            placeholders
            .maybe_convert_v1_yaml_placeholder_to_v2_placeholder_str(
                command, component_dict=component_dict)
            for command in container.get('command', [])
        ]
        container['args'] = [
            placeholders
            .maybe_convert_v1_yaml_placeholder_to_v2_placeholder_str(
                command, component_dict=component_dict)
            for command in container.get('args', [])
        ]
        container['env'] = {
            key: placeholders
            .maybe_convert_v1_yaml_placeholder_to_v2_placeholder_str(
                command, component_dict=component_dict)
            for key, command in container.get('env', {}).items()
        }
        container_spec = ContainerSpecImplementation.from_container_dict({
            'image': container['image'],
            'command': container['command'],
            'args': container['args'],
            'env': container['env']
        })

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

        def extract_description_from_command(
                commands: List[str]) -> Union[str, None]:
            for command in commands:
                if isinstance(command, str) and 'import kfp' in command:
                    for node in ast.walk(ast.parse(command)):
                        if isinstance(
                                node,
                            (ast.FunctionDef, ast.ClassDef, ast.Module)):
                            docstring = ast.get_docstring(node)
                            if docstring:
                                return docstring
            return None

        inputs = inputs_dict_from_components_dict(
            pipeline_spec_dict['components'], raw_name)
        outputs = outputs_dict_from_components_dict(
            pipeline_spec_dict['components'], raw_name)

        description = extract_description_from_command(
            implementation.container.command or [])
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
        from kfp.components import pipeline_channel
        from kfp.components import pipeline_task
        from kfp.components import tasks_group
        from kfp.components.types import type_utils

        args_dict = {}
        pipeline_inputs = self.inputs or {}

        for arg_name, input_spec in pipeline_inputs.items():
            arg_type = input_spec.type
            args_dict[arg_name] = pipeline_channel.create_pipeline_channel(
                name=arg_name, channel_type=arg_type)

        task = pipeline_task.PipelineTask(self, args_dict)

        # instead of constructing a pipeline with pipeline_context.Pipeline,
        # just build the single task group
        group = tasks_group.TasksGroup(
            group_type=tasks_group.TasksGroupType.PIPELINE)
        group.tasks.append(task)

        # Fill in the default values.
        args_list_with_defaults = [
            pipeline_channel.create_pipeline_channel(
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


def normalize_time_string(duration: str) -> str:
    """Normalizes a time string.
        Examples:
            - '1 hour' -> '1h'
            - '2 hours' -> '2h'
            - '2hours' -> '2h'
            - '2 w' -> '2w'
            - '2w' -> '2w'
    Args:
        duration (str): The unnormalized duration string.
    Returns:
        str: The normalized duration string.
    """
    no_ws_duration = duration.replace(' ', '')
    duration_split = [el for el in re.split(r'(\D+)', no_ws_duration) if el]

    if len(duration_split) != 2:
        raise ValueError(
            f"Invalid duration string: '{duration}'. Expected one value (as integer in string) and one unit, such as '1 hour'."
        )

    value = duration_split[0]
    unit = duration_split[1]

    first_letter_of_unit = unit[0]
    return value + first_letter_of_unit


def convert_duration_to_seconds(duration: str) -> int:
    """Converts a duration string to seconds.

    Args:
        duration (str): The unnormalized duration string. (e.g. '1h', '1 hour', '2
            hours', '2w', '2 weeks', '2d', etc.)
    Raises:
        ValueError: If the time unit is not one of seconds, minutes, hours, days,
            or weeks.
    Returns:
        int: The number of seconds in the duration.
    """
    duration = normalize_time_string(duration)
    seconds_per_unit = {'s': 1, 'm': 60, 'h': 3_600, 'd': 86_400, 'w': 604_800}
    if duration[-1] not in seconds_per_unit.keys():
        raise ValueError(
            f"Unsupported duration unit: '{duration[-1]}' for '{duration}'.")
    return int(duration[:-1]) * seconds_per_unit[duration[-1]]
