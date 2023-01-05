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
import itertools
import re
from typing import Any, Dict, List, Mapping, Optional, Union
import uuid

from google.protobuf import json_format
import kfp
from kfp.components import placeholders
from kfp.components import utils
from kfp.components import v1_components
from kfp.components import v1_structures
from kfp.components.container_component_artifact_channel import \
    ContainerComponentArtifactChannel
from kfp.components.types import artifact_types
from kfp.components.types import type_annotations
from kfp.components.types import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2
import yaml


@dataclasses.dataclass
class InputSpec:
    """Component input definitions.

    Attributes:
        type: The type of the input.
        default (optional): the default value for the input.
        optional: Wether the input is optional. An input is optional when it has an explicit default value.
        is_artifact_list: True if `type` represents a list of the artifact type. Only applies when `type` is an artifact.
    """
    type: Union[str, dict]
    default: Optional[Any] = None
    optional: bool = False
    # This special flag for lists of artifacts allows type to be used the same way for list of artifacts and single artifacts. This is aligned with how IR represents lists of artifacts (same as for single artifacts), as well as simplifies downstream type handling/checking operations in the SDK since we don't need to parse the string `type` to determine if single artifact or list.
    is_artifact_list: bool = False

    def __post_init__(self) -> None:
        self._validate_type()
        self._validate_usage_of_optional()

    @classmethod
    def from_ir_component_inputs_dict(
            cls, ir_component_inputs_dict: Dict[str, Any]) -> 'InputSpec':
        """Creates an InputSpec from a ComponentInputsSpec message in dict
        format (pipeline_spec.components.<component-
        key>.inputDefinitions.parameters.<input-key>).

        Args:
            ir_component_inputs_dict (Dict[str, Any]): The ComponentInputsSpec
                message in dict format.

        Returns:
            InputSpec: The InputSpec object.
        """
        if 'parameterType' in ir_component_inputs_dict:
            type_string = ir_component_inputs_dict['parameterType']
            type_ = type_utils.IR_TYPE_TO_IN_MEMORY_SPEC_TYPE.get(type_string)
            if type_ is None:
                raise ValueError(f'Unknown type {type_string} found in IR.')
            default_value = ir_component_inputs_dict.get('defaultValue')
            # fallback to checking if the parameter has a default value,
            # since some IR compiled with kfp<=2.0.0b8 will have defaults
            # without isOptional=True
            optional = ir_component_inputs_dict.get(
                'isOptional', 'defaultValue' in ir_component_inputs_dict)
            return InputSpec(
                type=type_, default=default_value, optional=optional)

        else:
            type_ = ir_component_inputs_dict['artifactType']['schemaTitle']
            schema_version = ir_component_inputs_dict['artifactType'][
                'schemaVersion']
            # TODO: would be better to extract these fields from the proto
            # message, as False default would be preserved
            optional = ir_component_inputs_dict.get('isOptional', False)
            return InputSpec(
                type=type_utils.create_bundled_artifact_type(
                    type_, schema_version),
                optional=optional)

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

    def _validate_type(self) -> None:
        """Type should either be a parameter or a valid bundled artifact type
        by the time it gets to InputSpec.

        This allows us to perform fewer checks downstream.
        """
        # TODO: add transformation logic so that we don't have to transform inputs at every place they are used, including v1 back compat support
        if not spec_type_is_parameter(self.type):
            type_utils.validate_bundled_artifact_type(self.type)

    def _validate_usage_of_optional(self) -> None:
        """Validates that the optional and default properties are in consistent
        states."""
        # Because None can be the default value, None cannot be used to to indicate no default. This is why we need the optional field. This check prevents users of InputSpec from setting these two values to an inconsistent state, forcing users of InputSpec to be explicit about optionality.
        if self.optional is False and self.default is not None:
            raise ValueError(
                f'`optional` argument to {self.__class__.__name__} must be True if `default` is not None.'
            )


@dataclasses.dataclass
class OutputSpec:
    """Component output definitions.

    Attributes:
        type: The type of the output.
        is_artifact_list: True if `type` represents a list of the artifact type. Only applies when `type` is an artifact.
    """
    type: Union[str, dict]
    # This special flag for lists of artifacts allows type to be used the same way for list of artifacts and single artifacts. This is aligned with how IR represents lists of artifacts (same as for single artifacts), as well as simplifies downstream type handling/checking operations in the SDK since we don't need to parse the string `type` to determine if single artifact or list.
    is_artifact_list: bool = False

    def __post_init__(self) -> None:
        self._validate_type()
        # TODO: remove this method when we support output lists of artifacts
        self._prevent_using_output_lists_of_artifacts()

    @classmethod
    def from_ir_component_outputs_dict(
            cls, ir_component_outputs_dict: Dict[str, Any]) -> 'OutputSpec':
        """Creates an OutputSpec from a ComponentOutputsSpec message in dict
        format (pipeline_spec.components.<component-
        key>.outputDefinitions.parameters|artifacts.<output-key>).

        Args:
            ir_component_outputs_dict (Dict[str, Any]): The ComponentOutputsSpec
                in dict format.

        Returns:
            OutputSpec: The OutputSpec object.
        """
        if 'parameterType' in ir_component_outputs_dict:
            type_string = ir_component_outputs_dict['parameterType']
            type_ = type_utils.IR_TYPE_TO_IN_MEMORY_SPEC_TYPE.get(type_string)
            if type_ is None:
                raise ValueError(f'Unknown type {type_string} found in IR.')
            return OutputSpec(type=type_,)
        else:
            type_ = ir_component_outputs_dict['artifactType']['schemaTitle']
            schema_version = ir_component_outputs_dict['artifactType'][
                'schemaVersion']
            return OutputSpec(
                type=type_utils.create_bundled_artifact_type(
                    type_, schema_version))

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

    def _validate_type(self):
        """Type should either be a parameter or a valid bundled artifact type
        by the time it gets to OutputSpec.

        This allows us to perform fewer checks downstream.
        """
        # TODO: add transformation logic so that we don't have to transform outputs at every place they are used, including v1 back compat support
        if not spec_type_is_parameter(self.type):
            type_utils.validate_bundled_artifact_type(self.type)

    def _prevent_using_output_lists_of_artifacts(self):
        if self.is_artifact_list:
            raise NotImplementedError(
                'Output lists of artifacts are not yet supported.')


def spec_type_is_parameter(type_: str) -> bool:
    in_memory_type = type_annotations.maybe_strip_optional_from_annotation_string(
        type_utils.get_canonical_name_for_outer_generic(type_))

    return in_memory_type in type_utils.IN_MEMORY_SPEC_TYPE_TO_IR_TYPE or in_memory_type == 'PipelineTaskFinalStatus'


@dataclasses.dataclass
class ResourceSpec:
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


@dataclasses.dataclass
class ContainerSpec:
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


@dataclasses.dataclass
class ContainerSpecImplementation:
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

    def __post_init__(self) -> None:
        self._transform_command()
        self._transform_args()
        self._transform_env()

    def _transform_command(self) -> None:
        """Use None instead of empty list for command."""
        self.command = None if self.command == [] else self.command

    def _transform_args(self) -> None:
        """Use None instead of empty list for args."""
        self.args = None if self.args == [] else self.args

    def _transform_env(self) -> None:
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

        return ContainerSpecImplementation(
            image=container_dict['image'],
            command=container_dict.get('command'),
            args=container_dict.get('args'),
            env=None,  # can only be set on tasks
            resources=None)  # can only be set on tasks


@dataclasses.dataclass
class RetryPolicy:
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


@dataclasses.dataclass
class TaskSpec:
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


@dataclasses.dataclass
class ImporterSpec:
    """ImporterSpec definition.

    Attributes:
        artifact_uri: The URI of the artifact.
        schema_title: The schema_title of the artifact.
        schema_version: The schema_version of the artifact.
        reimport: Whether or not import an artifact regardless it has been
         imported before.
        metadata (optional): the properties of the artifact.
    """
    artifact_uri: str
    schema_title: str
    schema_version: str
    reimport: bool
    metadata: Optional[Mapping[str, Any]] = None


@dataclasses.dataclass
class Implementation:
    """Implementation definition.

    Attributes:
        container: container implementation details.
        graph: graph implementation details.
        importer: importer implementation details.
    """
    container: Optional[ContainerSpecImplementation] = None
    importer: Optional[ImporterSpec] = None
    # Use type forward reference to skip the type validation in BaseModel.
    graph: Optional['pipeline_spec_pb2.PipelineSpec'] = None

    @classmethod
    def from_pipeline_spec_dict(cls, pipeline_spec_dict: Dict[str, Any],
                                component_name: str) -> 'Implementation':
        """Creates an Implementation object from a PipelineSpec message in dict
        format.

        Args:
            pipeline_spec_dict (Dict[str, Any]): PipelineSpec message in dict format.
            component_name (str): The name of the component.

        Returns:
            Implementation: An implementation object.
        """
        executor_key = utils.sanitize_executor_label(component_name)
        executor = pipeline_spec_dict['deploymentSpec']['executors'].get(
            executor_key)
        if executor is not None:
            container_spec = ContainerSpecImplementation.from_container_dict(
                executor['container']) if executor else None
            return Implementation(container=container_spec)
        else:
            pipeline_spec = json_format.ParseDict(
                pipeline_spec_dict, pipeline_spec_pb2.PipelineSpec())
            return Implementation(graph=pipeline_spec)


def check_placeholder_references_valid_io_name(
    inputs_dict: Dict[str, InputSpec],
    outputs_dict: Dict[str, OutputSpec],
    arg: placeholders.CommandLineElement,
) -> None:
    """Validates input/output placeholders refer to an existing input/output.

    Args:
        valid_inputs: The existing input names.
        valid_outputs: The existing output names.
        arg: The placeholder argument for checking.

    Raises:
        ValueError: if any placeholder references a nonexistant input or
            output.
        TypeError: if any argument is neither a str nor a placeholder
            instance.
    """
    if isinstance(arg, ContainerComponentArtifactChannel):
        raise ValueError(
            'Cannot access artifact by itself in the container definition. Please use .uri or .path instead to access the artifact.'
        )
    elif isinstance(arg, placeholders.PRIMITIVE_INPUT_PLACEHOLDERS):
        if arg.input_name not in inputs_dict:
            raise ValueError(
                f'Argument "{arg.__class__.__name__}" references nonexistant input: "{arg.input_name}".'
            )
    elif isinstance(arg, placeholders.PRIMITIVE_OUTPUT_PLACEHOLDERS):
        if arg.output_name not in outputs_dict:
            raise ValueError(
                f'Argument "{arg.__class__.__name__}" references nonexistant output: "{arg.output_name}".'
            )
    elif isinstance(arg, placeholders.IfPresentPlaceholder):
        if arg.input_name not in inputs_dict:
            raise ValueError(
                f'Argument "{arg.__class__.__name__}" references nonexistant input: "{arg.input_name}".'
            )

        all_normalized_args: List[placeholders.CommandLineElement] = []
        if arg.then is None:
            pass
        elif isinstance(arg.then, list):
            all_normalized_args.extend(arg.then)
        else:
            all_normalized_args.append(arg.then)

        if arg.else_ is None:
            pass
        elif isinstance(arg.else_, list):
            all_normalized_args.extend(arg.else_)
        else:
            all_normalized_args.append(arg.else_)

        for arg in all_normalized_args:
            check_placeholder_references_valid_io_name(inputs_dict,
                                                       outputs_dict, arg)
    elif isinstance(arg, placeholders.ConcatPlaceholder):
        for arg in arg.items:
            check_placeholder_references_valid_io_name(inputs_dict,
                                                       outputs_dict, arg)
    elif not isinstance(
            arg, placeholders.ExecutorInputPlaceholder) and not isinstance(
                arg, str):
        raise TypeError(f'Unexpected argument "{arg}" of type {type(arg)}.')


@dataclasses.dataclass
class ComponentSpec:
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

    def __post_init__(self) -> None:
        self._transform_name()
        self._transform_inputs()
        self._transform_outputs()
        self._validate_placeholders()

    def _transform_name(self) -> None:
        """Converts the name to a valid name."""
        self.name = utils.maybe_rename_for_k8s(self.name)

    def _transform_inputs(self) -> None:
        """Use None instead of empty list for inputs."""
        self.inputs = None if self.inputs == {} else self.inputs

    def _transform_outputs(self) -> None:
        """Use None instead of empty list for outputs."""
        self.outputs = None if self.outputs == {} else self.outputs

    def _validate_placeholders(self):
        """Validates that input/output placeholders refer to an existing
        input/output."""
        if self.implementation.container is None:
            return

        valid_inputs = {} if self.inputs is None else self.inputs
        valid_outputs = {} if self.outputs is None else self.outputs
        for arg in itertools.chain(
            (self.implementation.container.command or []),
            (self.implementation.container.args or [])):
            check_placeholder_references_valid_io_name(valid_inputs,
                                                       valid_outputs, arg)

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

        if 'implementation' not in component_dict or 'container' not in component_dict[
                'implementation']:
            raise NotImplementedError('Container implementation not found.')

        container = component_dict['implementation']['container']
        command = [
            placeholders.maybe_convert_v1_yaml_placeholder_to_v2_placeholder(
                command, component_dict=component_dict)
            for command in container.get('command', [])
        ]
        args = [
            placeholders.maybe_convert_v1_yaml_placeholder_to_v2_placeholder(
                command, component_dict=component_dict)
            for command in container.get('args', [])
        ]
        env = {
            key:
            placeholders.maybe_convert_v1_yaml_placeholder_to_v2_placeholder(
                command, component_dict=component_dict)
            for key, command in container.get('env', {}).items()
        }
        container_spec = ContainerSpecImplementation.from_container_dict({
            'image': container['image'],
            'command': command,
            'args': args,
            'env': env
        })

        inputs = {}
        for spec in component_dict.get('inputs', []):
            type_ = spec.get('type')
            optional = spec.get('optional', False) or 'default' in spec
            default = spec.get('default')
            default = type_utils.deserialize_v1_component_yaml_default(
                type_=type_, default=default)

            if isinstance(type_, str) and type_ == 'PipelineTaskFinalStatus':
                inputs[utils.sanitize_input_name(spec['name'])] = InputSpec(
                    type=type_, optional=True)
                continue

            elif isinstance(type_, str) and type_.lower(
            ) in type_utils._PARAMETER_TYPES_MAPPING:
                type_enum = type_utils._PARAMETER_TYPES_MAPPING[type_.lower()]
                ir_parameter_type_name = pipeline_spec_pb2.ParameterType.ParameterTypeEnum.Name(
                    type_enum)
                in_memory_parameter_type_name = type_utils.IR_TYPE_TO_IN_MEMORY_SPEC_TYPE[
                    ir_parameter_type_name]
                inputs[utils.sanitize_input_name(spec['name'])] = InputSpec(
                    type=in_memory_parameter_type_name,
                    default=default,
                    optional=optional,
                )
                continue

            elif isinstance(type_, str) and re.match(
                    type_utils._GOOGLE_TYPES_PATTERN, type_):
                schema_title = type_
                schema_version = type_utils._GOOGLE_TYPES_VERSION

            elif isinstance(type_, str) and type_.lower(
            ) in type_utils._ARTIFACT_CLASSES_MAPPING:
                artifact_class = type_utils._ARTIFACT_CLASSES_MAPPING[
                    type_.lower()]
                schema_title = artifact_class.schema_title
                schema_version = artifact_class.schema_version

            elif type_ is None or isinstance(type_, dict) or type_.lower(
            ) not in type_utils._ARTIFACT_CLASSES_MAPPING:
                schema_title = artifact_types.Artifact.schema_title
                schema_version = artifact_types.Artifact.schema_version

            else:
                raise ValueError(f'Unknown input: {type_}')

            if optional:
                # handles optional artifacts with no default value
                inputs[utils.sanitize_input_name(spec['name'])] = InputSpec(
                    type=type_utils.create_bundled_artifact_type(
                        schema_title, schema_version),
                    default=default,
                    optional=optional,
                )
            else:
                inputs[utils.sanitize_input_name(spec['name'])] = InputSpec(
                    type=type_utils.create_bundled_artifact_type(
                        schema_title, schema_version))

        outputs = {}
        for spec in component_dict.get('outputs', []):
            type_ = spec.get('type')

            if isinstance(type_, str) and type_.lower(
            ) in type_utils._PARAMETER_TYPES_MAPPING:
                type_enum = type_utils._PARAMETER_TYPES_MAPPING[type_.lower()]
                ir_parameter_type_name = pipeline_spec_pb2.ParameterType.ParameterTypeEnum.Name(
                    type_enum)
                in_memory_parameter_type_name = type_utils.IR_TYPE_TO_IN_MEMORY_SPEC_TYPE[
                    ir_parameter_type_name]
                outputs[utils.sanitize_input_name(spec['name'])] = OutputSpec(
                    type=in_memory_parameter_type_name)
                continue

            elif isinstance(type_, str) and re.match(
                    type_utils._GOOGLE_TYPES_PATTERN, type_):
                schema_title = type_
                schema_version = type_utils._GOOGLE_TYPES_VERSION

            elif isinstance(type_, str) and type_.lower(
            ) in type_utils._ARTIFACT_CLASSES_MAPPING:
                artifact_class = type_utils._ARTIFACT_CLASSES_MAPPING[
                    type_.lower()]
                schema_title = artifact_class.schema_title
                schema_version = artifact_class.schema_version

            elif type_ is None or isinstance(type_, dict) or type_.lower(
            ) not in type_utils._ARTIFACT_CLASSES_MAPPING:
                schema_title = artifact_types.Artifact.schema_title
                schema_version = artifact_types.Artifact.schema_version

            else:
                raise ValueError(f'Unknown output: {type_}')

            outputs[utils.sanitize_input_name(spec['name'])] = OutputSpec(
                type=type_utils.create_bundled_artifact_type(
                    schema_title, schema_version))

        return ComponentSpec(
            name=component_dict.get('name', 'name'),
            description=component_dict.get('description'),
            implementation=Implementation(container=container_spec),
            inputs=inputs,
            outputs=outputs,
        )

    @classmethod
    def from_pipeline_spec_dict(
            cls, pipeline_spec_dict: Dict[str, Any]) -> 'ComponentSpec':
        raw_name = pipeline_spec_dict['pipelineInfo']['name']

        def inputs_dict_from_component_spec_dict(
                component_spec_dict: Dict[str, Any]) -> Dict[str, InputSpec]:
            parameters = component_spec_dict.get('inputDefinitions',
                                                 {}).get('parameters', {})
            artifacts = component_spec_dict.get('inputDefinitions',
                                                {}).get('artifacts', {})
            all_inputs = {**parameters, **artifacts}
            return {
                name: InputSpec.from_ir_component_inputs_dict(input_dict)
                for name, input_dict in all_inputs.items()
            }

        def outputs_dict_from_component_spec_dict(
                components_spec_dict: Dict[str, Any]) -> Dict[str, OutputSpec]:
            parameters = component_spec_dict.get('outputDefinitions',
                                                 {}).get('parameters', {})
            artifacts = components_spec_dict.get('outputDefinitions',
                                                 {}).get('artifacts', {})
            all_outputs = {**parameters, **artifacts}
            return {
                name: OutputSpec.from_ir_component_outputs_dict(output_dict)
                for name, output_dict in all_outputs.items()
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

        component_key = utils.sanitize_component_name(raw_name)
        component_spec_dict = pipeline_spec_dict['components'].get(
            component_key, pipeline_spec_dict['root'])

        inputs = inputs_dict_from_component_spec_dict(component_spec_dict)
        outputs = outputs_dict_from_component_spec_dict(component_spec_dict)

        implementation = Implementation.from_pipeline_spec_dict(
            pipeline_spec_dict, raw_name)

        description = extract_description_from_command(
            implementation.container.command or
            []) if implementation.container else None

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

        def extract_description(component_yaml: str) -> Union[str, None]:
            heading = '# Description: '
            multi_line_description_prefix = '#             '
            index_of_heading = 2
            if heading in component_yaml:
                description = component_yaml.splitlines()[index_of_heading]

                # Multi line
                comments = component_yaml.splitlines()
                index = index_of_heading + 1
                while comments[index][:len(multi_line_description_prefix
                                          )] == multi_line_description_prefix:
                    description += '\n' + comments[index][
                        len(multi_line_description_prefix) + 1:]
                    index += 1

                return description[len(heading):]
            else:
                return None

        json_component = yaml.safe_load(component_yaml)
        is_v1 = 'implementation' in set(json_component.keys())
        if is_v1:
            v1_component = v1_components._load_component_spec_from_component_text(
                component_yaml)
            return cls.from_v1_component_spec(v1_component)
        else:
            component_spec = ComponentSpec.from_pipeline_spec_dict(
                json_component)
            if not component_spec.description:
                component_spec.description = extract_description(
                    component_yaml=component_yaml)
            return component_spec

    def save_to_component_yaml(self, output_file: str) -> None:
        """Saves ComponentSpec into IR YAML file.

        Args:
            output_file: File path to store the component yaml.
        """
        from kfp.compiler import pipeline_spec_builder as builder

        pipeline_spec = self.to_pipeline_spec()
        builder.write_pipeline_spec_to_file(pipeline_spec, None, output_file)

    def to_pipeline_spec(self) -> pipeline_spec_pb2.PipelineSpec:
        """Creates a pipeline instance and constructs the pipeline spec for a
        single component.

        Args:
            component_spec: The ComponentSpec to convert to PipelineSpec.

        Returns:
            A PipelineSpec proto representing the compiled component.
        """
        # import here to aviod circular module dependency
        from kfp.compiler import compiler_utils
        from kfp.compiler import pipeline_spec_builder as builder
        from kfp.components import pipeline_channel
        from kfp.components import pipeline_task
        from kfp.components import tasks_group

        args_dict = {}
        pipeline_inputs = self.inputs or {}

        for arg_name, input_spec in pipeline_inputs.items():
            args_dict[arg_name] = pipeline_channel.create_pipeline_channel(
                name=arg_name, channel_type=input_spec.type)

        task = pipeline_task.PipelineTask(self, args_dict)

        # instead of constructing a pipeline with pipeline_context.Pipeline,
        # just build the single task group
        group = tasks_group.TasksGroup(
            group_type=tasks_group.TasksGroupType.PIPELINE)
        group.tasks.append(task)

        group.name = uuid.uuid4().hex

        pipeline_name = self.name
        task_group = group

        utils.validate_pipeline_name(pipeline_name)

        pipeline_spec = pipeline_spec_pb2.PipelineSpec()
        pipeline_spec.pipeline_info.name = pipeline_name
        pipeline_spec.sdk_version = f'kfp-{kfp.__version__}'
        # Schema version 2.1.0 is required for kfp-pipeline-spec>0.1.13
        pipeline_spec.schema_version = '2.1.0'

        # if we decide to surface component outputs to pipeline level,
        # can just assign the component_spec_proto directly to .root
        component_spec_proto = builder._build_component_spec_from_component_spec_structure(
            self)
        has_inputs = bool(
            len(component_spec_proto.input_definitions.artifacts) +
            len(component_spec_proto.input_definitions.parameters))
        if has_inputs:
            pipeline_spec.root.input_definitions.CopyFrom(
                component_spec_proto.input_definitions)

        deployment_config = pipeline_spec_pb2.PipelineDeploymentConfig()
        root_group = task_group

        task_name_to_parent_groups, group_name_to_parent_groups = compiler_utils.get_parent_groups(
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
