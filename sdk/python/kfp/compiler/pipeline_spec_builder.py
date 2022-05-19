# Copyright 2021 The Kubeflow Authors
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
"""Functions for creating PipelineSpec proto objects."""

import collections
import json
import re
from typing import Any, Dict, List, Mapping, Optional, Tuple, Union

import kfp
from google.protobuf import json_format
from google.protobuf import struct_pb2
from kfp import dsl
from kfp.compiler import pipeline_spec_builder as builder
from kfp.components import for_loop
from kfp.components import pipeline_channel
from kfp.components import pipeline_task
from kfp.components import structures
from kfp.components import tasks_group
from kfp.components import utils
from kfp.components import utils as component_utils
from kfp.components.types import artifact_types
from kfp.components.types import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2

GroupOrTaskType = Union[tasks_group.TasksGroup, pipeline_task.PipelineTask]


def _additional_input_name_for_pipeline_channel(
        channel_or_name: Union[pipeline_channel.PipelineChannel, str]) -> str:
    """Gets the name for an additional (compiler-injected) input."""

    # Adding a prefix to avoid (reduce chance of) name collision between the
    # original component inputs and the injected input.
    return 'pipelinechannel--' + (
        channel_or_name.full_name if isinstance(
            channel_or_name, pipeline_channel.PipelineChannel) else
        channel_or_name)


def _to_protobuf_value(value: type_utils.PARAMETER_TYPES) -> struct_pb2.Value:
    """Creates a google.protobuf.struct_pb2.Value message out of a provide
    value.

    Args:
        value: The value to be converted to Value message.

    Returns:
         A google.protobuf.struct_pb2.Value message.

    Raises:
        ValueError if the given value is not one of the parameter types.
    """
    if isinstance(value, str):
        return struct_pb2.Value(string_value=value)
    elif isinstance(value, (int, float)):
        return struct_pb2.Value(number_value=value)
    elif isinstance(value, bool):
        return struct_pb2.Value(bool_value=value)
    elif isinstance(value, dict):
        return struct_pb2.Value(
            struct_value=struct_pb2.Struct(
                fields={k: _to_protobuf_value(v) for k, v in value.items()}))
    elif isinstance(value, list):
        return struct_pb2.Value(
            list_value=struct_pb2.ListValue(
                values=[_to_protobuf_value(v) for v in value]))
    else:
        raise ValueError('Value must be one of the following types: '
                         'str, int, float, bool, dict, and list. Got: '
                         f'"{value}" of type "{type(value)}".')


def build_task_spec_for_task(
    task: pipeline_task.PipelineTask,
    parent_component_inputs: pipeline_spec_pb2.ComponentInputsSpec,
    tasks_in_current_dag: List[str],
    input_parameters_in_current_dag: List[str],
    input_artifacts_in_current_dag: List[str],
) -> pipeline_spec_pb2.PipelineTaskSpec:
    """Builds PipelineTaskSpec for a pipeline task.

    A task input may reference an output outside its immediate DAG.
    For instance::

        random_num = random_num_op(...)
        with dsl.Condition(random_num.output > 5):
            print_op('%s > 5' % random_num.output)

    In this example, `dsl.Condition` forms a subDAG with one task from `print_op`
    inside the subDAG. The task of `print_op` references output from `random_num`
    task, which is outside the sub-DAG. When compiling to IR, such cross DAG
    reference is disallowed. So we need to "punch a hole" in the sub-DAG to make
    the input available in the subDAG component inputs if it's not already there,
    Next, we can call this method to fix the tasks inside the subDAG to make them
    reference the component inputs instead of directly referencing the original
    producer task.

    Args:
        task: The task to build a PipelineTaskSpec for.
        parent_component_inputs: The task's parent component's input specs.
        tasks_in_current_dag: The list of tasks names for tasks in the same dag.
        input_parameters_in_current_dag: The list of input parameters in the DAG
            component.
        input_artifacts_in_current_dag: The list of input artifacts in the DAG
            component.

    Returns:
        A PipelineTaskSpec object representing the task.
    """
    pipeline_task_spec = pipeline_spec_pb2.PipelineTaskSpec()
    pipeline_task_spec.task_info.name = (
        task.task_spec.display_name or task.name)
    # Use task.name for component_ref.name because we may customize component
    # spec for individual tasks to work around the lack of optional inputs
    # support in IR.
    pipeline_task_spec.component_ref.name = (
        component_utils.sanitize_component_name(task.name))
    pipeline_task_spec.caching_options.enable_cache = (
        task.task_spec.enable_caching)

    for input_name, input_value in task.inputs.items():
        if isinstance(input_value, pipeline_channel.PipelineArtifactChannel):

            if input_value.task_name:
                # Value is produced by an upstream task.
                if input_value.task_name in tasks_in_current_dag:
                    # Dependent task within the same DAG.
                    pipeline_task_spec.inputs.artifacts[
                        input_name].task_output_artifact.producer_task = (
                            component_utils.sanitize_task_name(
                                input_value.task_name))
                    pipeline_task_spec.inputs.artifacts[
                        input_name].task_output_artifact.output_artifact_key = (
                            input_value.name)
                else:
                    # Dependent task not from the same DAG.
                    component_input_artifact = (
                        _additional_input_name_for_pipeline_channel(input_value)
                    )
                    assert component_input_artifact in parent_component_inputs.artifacts, \
                        'component_input_artifact: {} not found. All inputs: {}'.format(
                            component_input_artifact, parent_component_inputs)
                    pipeline_task_spec.inputs.artifacts[
                        input_name].component_input_artifact = (
                            component_input_artifact)
            else:
                raise RuntimeError(
                    f'Artifacts must be produced by a task. Got {input_value}.')

        elif isinstance(input_value, pipeline_channel.PipelineParameterChannel):

            if input_value.task_name:
                # Value is produced by an upstream task.
                if input_value.task_name in tasks_in_current_dag:
                    # Dependent task within the same DAG.
                    pipeline_task_spec.inputs.parameters[
                        input_name].task_output_parameter.producer_task = (
                            component_utils.sanitize_task_name(
                                input_value.task_name))
                    pipeline_task_spec.inputs.parameters[
                        input_name].task_output_parameter.output_parameter_key = (
                            input_value.name)
                else:
                    # Dependent task not from the same DAG.
                    component_input_parameter = (
                        _additional_input_name_for_pipeline_channel(input_value)
                    )
                    assert component_input_parameter in parent_component_inputs.parameters, \
                        'component_input_parameter: {} not found. All inputs: {}'.format(
                            component_input_parameter, parent_component_inputs)
                    pipeline_task_spec.inputs.parameters[
                        input_name].component_input_parameter = (
                            component_input_parameter)
            else:
                # Value is from pipeline input.
                component_input_parameter = input_value.full_name
                if component_input_parameter not in parent_component_inputs.parameters:
                    component_input_parameter = (
                        _additional_input_name_for_pipeline_channel(input_value)
                    )
                pipeline_task_spec.inputs.parameters[
                    input_name].component_input_parameter = (
                        component_input_parameter)

        elif isinstance(input_value, for_loop.LoopArgument):

            component_input_parameter = (
                _additional_input_name_for_pipeline_channel(input_value))
            assert component_input_parameter in parent_component_inputs.parameters, \
                'component_input_parameter: {} not found. All inputs: {}'.format(
                    component_input_parameter, parent_component_inputs)
            pipeline_task_spec.inputs.parameters[
                input_name].component_input_parameter = (
                    component_input_parameter)

        elif isinstance(input_value, for_loop.LoopArgumentVariable):

            component_input_parameter = (
                _additional_input_name_for_pipeline_channel(
                    input_value.loop_argument))
            assert component_input_parameter in parent_component_inputs.parameters, \
                'component_input_parameter: {} not found. All inputs: {}'.format(
                    component_input_parameter, parent_component_inputs)
            pipeline_task_spec.inputs.parameters[
                input_name].component_input_parameter = (
                    component_input_parameter)
            pipeline_task_spec.inputs.parameters[
                input_name].parameter_expression_selector = (
                    'parseJson(string_value)["{}"]'.format(
                        input_value.subvar_name))

        elif isinstance(input_value, str):

            # Handle extra input due to string concat
            pipeline_channels = (
                pipeline_channel.extract_pipeline_channels_from_any(input_value)
            )
            for channel in pipeline_channels:
                # value contains PipelineChannel placeholders which needs to be
                # replaced. And the input needs to be added to the task spec.

                # Form the name for the compiler injected input, and make sure it
                # doesn't collide with any existing input names.
                additional_input_name = (
                    _additional_input_name_for_pipeline_channel(channel))

                # We don't expect collision to happen because we prefix the name
                # of additional input with 'pipelinechannel--'. But just in case
                # collision did happend, throw a RuntimeError so that we don't
                # get surprise at runtime.
                for existing_input_name, _ in task.inputs.items():
                    if existing_input_name == additional_input_name:
                        raise RuntimeError(
                            'Name collision between existing input name '
                            '{} and compiler injected input name {}'.format(
                                existing_input_name, additional_input_name))

                additional_input_placeholder = structures.InputValuePlaceholder(
                    additional_input_name).to_placeholder()
                input_value = input_value.replace(channel.pattern,
                                                  additional_input_placeholder)

                if channel.task_name:
                    # Value is produced by an upstream task.
                    if channel.task_name in tasks_in_current_dag:
                        # Dependent task within the same DAG.
                        pipeline_task_spec.inputs.parameters[
                            additional_input_name].task_output_parameter.producer_task = (
                                component_utils.sanitize_task_name(
                                    channel.task_name))
                        pipeline_task_spec.inputs.parameters[
                            input_name].task_output_parameter.output_parameter_key = (
                                channel.name)
                    else:
                        # Dependent task not from the same DAG.
                        component_input_parameter = (
                            _additional_input_name_for_pipeline_channel(channel)
                        )
                        assert component_input_parameter in parent_component_inputs.parameters, \
                            'component_input_parameter: {} not found. All inputs: {}'.format(
                                component_input_parameter, parent_component_inputs)
                        pipeline_task_spec.inputs.parameters[
                            additional_input_name].component_input_parameter = (
                                component_input_parameter)
                else:
                    # Value is from pipeline input. (or loop?)
                    component_input_parameter = channel.full_name
                    if component_input_parameter not in parent_component_inputs.parameters:
                        component_input_parameter = (
                            _additional_input_name_for_pipeline_channel(channel)
                        )
                    pipeline_task_spec.inputs.parameters[
                        additional_input_name].component_input_parameter = (
                            component_input_parameter)

            pipeline_task_spec.inputs.parameters[
                input_name].runtime_value.constant.string_value = input_value

        elif isinstance(input_value, (str, int, float, bool, dict, list)):

            pipeline_task_spec.inputs.parameters[
                input_name].runtime_value.constant.CopyFrom(
                    _to_protobuf_value(input_value))

        else:
            raise ValueError(
                'Input argument supports only the following types: '
                'str, int, float, bool, dict, and list.'
                f'Got {input_value} of type {type(input_value)}.')

    return pipeline_task_spec


def build_component_spec_for_exit_task(
    task: pipeline_task.PipelineTask,) -> pipeline_spec_pb2.ComponentSpec:
    """Builds ComponentSpec for an exit task.

    Args:
        task: The task to build a ComponentSpec for.

    Returns:
        A ComponentSpec object for the exit task.
    """
    return build_component_spec_for_task(task=task, is_exit_task=True)


def build_component_spec_for_task(
    task: pipeline_task.PipelineTask,
    is_exit_task: bool = False,
) -> pipeline_spec_pb2.ComponentSpec:
    """Builds ComponentSpec for a pipeline task.

    Args:
        task: The task to build a ComponentSpec for.
        is_exit_task: Whether the task is used as exit task in Exit Handler.

    Returns:
        A ComponentSpec object for the task.
    """
    component_spec = pipeline_spec_pb2.ComponentSpec()
    component_spec.executor_label = component_utils.sanitize_executor_label(
        task.name)

    for input_name, input_spec in (task.component_spec.inputs or {}).items():

        # Special handling for PipelineTaskFinalStatus first.
        if type_utils.is_task_final_status_type(input_spec.type):
            if not is_exit_task:
                raise ValueError(
                    'PipelineTaskFinalStatus can only be used in an exit task.')
            component_spec.input_definitions.parameters[
                input_name].parameter_type = pipeline_spec_pb2.ParameterType.STRUCT
            continue

        # skip inputs not present, as a workaround to support optional inputs.
        if input_name not in task.inputs and input_spec.default is None:
            continue

        if type_utils.is_parameter_type(input_spec.type):
            component_spec.input_definitions.parameters[
                input_name].parameter_type = type_utils.get_parameter_type(
                    input_spec.type)
            if input_spec.default is not None:
                component_spec.input_definitions.parameters[
                    input_name].default_value.CopyFrom(
                        _to_protobuf_value(input_spec.default))

        else:
            component_spec.input_definitions.artifacts[
                input_name].artifact_type.CopyFrom(
                    type_utils.get_artifact_type_schema(input_spec.type))

    for output_name, output_spec in (task.component_spec.outputs or {}).items():
        if type_utils.is_parameter_type(output_spec.type):
            component_spec.output_definitions.parameters[
                output_name].parameter_type = type_utils.get_parameter_type(
                    output_spec.type)
        else:
            component_spec.output_definitions.artifacts[
                output_name].artifact_type.CopyFrom(
                    type_utils.get_artifact_type_schema(output_spec.type))

    return component_spec


def build_importer_spec_for_task(
    task: pipeline_task.PipelineTask
) -> pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec:
    """Builds ImporterSpec for a pipeline task.

    Args:
        task: The task to build a ComponentSpec for.

    Returns:
        A ImporterSpec object for the task.
    """
    type_schema = type_utils.get_artifact_type_schema(
        task.importer_spec.type_schema)
    importer_spec = pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec(
        type_schema=type_schema, reimport=task.importer_spec.reimport)

    if task.importer_spec.metadata:
        metadata_protobuf_struct = struct_pb2.Struct()
        metadata_protobuf_struct.update(task.importer_spec.metadata)
        importer_spec.metadata.CopyFrom(metadata_protobuf_struct)

    if isinstance(task.importer_spec.artifact_uri,
                  pipeline_channel.PipelineParameterChannel):
        importer_spec.artifact_uri.runtime_parameter = 'uri'
    elif isinstance(task.importer_spec.artifact_uri, str):
        importer_spec.artifact_uri.constant.string_value = task.importer_spec.artifact_uri

    return importer_spec


def build_container_spec_for_task(
    task: pipeline_task.PipelineTask
) -> pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec:
    """Builds PipelineContainerSpec for a pipeline task.

    Args:
        task: The task to build a ComponentSpec for.

    Returns:
        A PipelineContainerSpec object for the task.
    """
    container_spec = (
        pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec(
            image=task.container_spec.image,
            command=task.container_spec.command,
            args=task.container_spec.args,
            env=[
                pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec
                .EnvVar(name=name, value=value)
                for name, value in (task.container_spec.env or {}).items()
            ]))

    if task.container_spec.resources is not None:
        container_spec.resources.cpu_limit = (
            task.container_spec.resources.cpu_limit)
        container_spec.resources.memory_limit = (
            task.container_spec.resources.memory_limit)
        if task.container_spec.resources.accelerator_count is not None:
            container_spec.resources.accelerator.CopyFrom(
                pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec
                .ResourceSpec.AcceleratorConfig(
                    type=task.container_spec.resources.accelerator_type,
                    count=task.container_spec.resources.accelerator_count,
                ))

    return container_spec


def _fill_in_component_input_default_value(
    component_spec: pipeline_spec_pb2.ComponentSpec,
    input_name: str,
    default_value: Optional[type_utils.PARAMETER_TYPES],
) -> None:
    """Fills in the default of component input parameter.

    Args:
        component_spec: The ComponentSpec to update in place.
        input_name: The name of the input parameter.
        default_value: The default value of the input parameter.
    """
    if default_value is None:
        return

    parameter_type = component_spec.input_definitions.parameters[
        input_name].parameter_type
    if pipeline_spec_pb2.ParameterType.NUMBER_INTEGER == parameter_type:
        component_spec.input_definitions.parameters[
            input_name].default_value.number_value = default_value
    elif pipeline_spec_pb2.ParameterType.NUMBER_DOUBLE == parameter_type:
        component_spec.input_definitions.parameters[
            input_name].default_value.number_value = default_value
    elif pipeline_spec_pb2.ParameterType.STRING == parameter_type:
        component_spec.input_definitions.parameters[
            input_name].default_value.string_value = default_value
    elif pipeline_spec_pb2.ParameterType.BOOLEAN == parameter_type:
        component_spec.input_definitions.parameters[
            input_name].default_value.bool_value = default_value
    elif pipeline_spec_pb2.ParameterType.STRUCT == parameter_type:
        component_spec.input_definitions.parameters[
            input_name].default_value.CopyFrom(
                _to_protobuf_value(default_value))
    elif pipeline_spec_pb2.ParameterType.LIST == parameter_type:
        component_spec.input_definitions.parameters[
            input_name].default_value.CopyFrom(
                _to_protobuf_value(default_value))


def build_component_spec_for_group(
    pipeline_channels: List[pipeline_channel.PipelineChannel],
    is_root_group: bool,
) -> pipeline_spec_pb2.ComponentSpec:
    """Builds ComponentSpec for a TasksGroup.

    Args:
        group: The group to build a ComponentSpec for.
        pipeline_channels: The list of pipeline channels referenced by the group.

    Returns:
        A PipelineTaskSpec object representing the loop group.
    """
    component_spec = pipeline_spec_pb2.ComponentSpec()

    for channel in pipeline_channels:

        input_name = (
            channel.name if is_root_group else
            _additional_input_name_for_pipeline_channel(channel))

        if isinstance(channel, pipeline_channel.PipelineArtifactChannel):
            component_spec.input_definitions.artifacts[
                input_name].artifact_type.CopyFrom(
                    type_utils.get_artifact_type_schema(channel.channel_type))
        else:
            # channel is one of PipelineParameterChannel, LoopArgument, or
            # LoopArgumentVariable.
            component_spec.input_definitions.parameters[
                input_name].parameter_type = type_utils.get_parameter_type(
                    channel.channel_type)

            if is_root_group:
                _fill_in_component_input_default_value(
                    component_spec=component_spec,
                    input_name=input_name,
                    default_value=channel.value,
                )

    return component_spec


def _pop_input_from_task_spec(
    task_spec: pipeline_spec_pb2.PipelineTaskSpec,
    input_name: str,
) -> None:
    """Removes an input from task spec inputs.

    Args:
      task_spec: The pipeline task spec to update in place.
      input_name: The name of the input, which could be an artifact or paremeter.
    """
    task_spec.inputs.artifacts.pop(input_name)
    task_spec.inputs.parameters.pop(input_name)

    if task_spec.inputs == pipeline_spec_pb2.TaskInputsSpec():
        task_spec.ClearField('inputs')


def _update_task_spec_for_loop_group(
    group: tasks_group.ParallelFor,
    pipeline_task_spec: pipeline_spec_pb2.PipelineTaskSpec,
) -> None:
    """Updates PipelineTaskSpec for loop group.

    Args:
        group: The loop group to update task spec for.
        pipeline_task_spec: The pipeline task spec to update in place.
    """
    if group.items_is_pipeline_channel:
        loop_items_channel = group.loop_argument.items_or_pipeline_channel
        input_parameter_name = _additional_input_name_for_pipeline_channel(
            loop_items_channel)
        loop_argument_item_name = _additional_input_name_for_pipeline_channel(
            group.loop_argument.full_name)

        loop_arguments_item = '{}-{}'.format(
            input_parameter_name, for_loop.LoopArgument.LOOP_ITEM_NAME_BASE)
        assert loop_arguments_item == loop_argument_item_name

        pipeline_task_spec.parameter_iterator.items.input_parameter = (
            input_parameter_name)
        pipeline_task_spec.parameter_iterator.item_input = (
            loop_argument_item_name)

        # If the loop items itself is a loop arguments variable, handle the
        # subvar name.
        if isinstance(loop_items_channel, for_loop.LoopArgumentVariable):
            pipeline_task_spec.inputs.parameters[
                input_parameter_name].parameter_expression_selector = (
                    'parseJson(string_value)["{}"]'.format(
                        loop_items_channel.subvar_name))
            pipeline_task_spec.inputs.parameters[
                input_parameter_name].component_input_parameter = (
                    _additional_input_name_for_pipeline_channel(
                        loop_items_channel.loop_argument))

    else:
        input_parameter_name = _additional_input_name_for_pipeline_channel(
            group.loop_argument)
        raw_values = group.loop_argument.items_or_pipeline_channel

        pipeline_task_spec.parameter_iterator.items.raw = json.dumps(
            raw_values, sort_keys=True)
        pipeline_task_spec.parameter_iterator.item_input = (
            input_parameter_name)

    _pop_input_from_task_spec(
        task_spec=pipeline_task_spec,
        input_name=pipeline_task_spec.parameter_iterator.item_input)


def _resolve_condition_operands(
    left_operand: Union[str, pipeline_channel.PipelineChannel],
    right_operand: Union[str, pipeline_channel.PipelineChannel],
) -> Tuple[str, str]:
    """Resolves values and PipelineChannels for condition operands.

    Args:
        left_operand: The left operand of a condition expression.
        right_operand: The right operand of a condition expression.

    Returns:
        A tuple of the resolved operands values:
        (left_operand_value, right_operand_value).
    """

    # Pre-scan the operand to get the type of constant value if there's any.
    for value_or_reference in [left_operand, right_operand]:
        if isinstance(value_or_reference, pipeline_channel.PipelineChannel):
            parameter_type = type_utils.get_parameter_type(
                value_or_reference.channel_type)
            if parameter_type in [
                    pipeline_spec_pb2.ParameterType.STRUCT,
                    pipeline_spec_pb2.ParameterType.LIST,
                    pipeline_spec_pb2.ParameterType
                    .PARAMETER_TYPE_ENUM_UNSPECIFIED,
            ]:
                input_name = _additional_input_name_for_pipeline_channel(
                    value_or_reference)
                raise ValueError('Conditional requires scalar parameter values'
                                 ' for comparison. Found input "{}" of type {}'
                                 ' in pipeline definition instead.'.format(
                                     input_name,
                                     value_or_reference.channel_type))
    parameter_types = set()
    for value_or_reference in [left_operand, right_operand]:
        if isinstance(value_or_reference, pipeline_channel.PipelineChannel):
            parameter_type = type_utils.get_parameter_type(
                value_or_reference.channel_type)
        else:
            parameter_type = type_utils.get_parameter_type(
                type(value_or_reference).__name__)

        parameter_types.add(parameter_type)

    if len(parameter_types) == 2:
        # Two different types being compared. The only possible types are
        # String, Boolean, Double and Integer. We'll promote the other type
        # using the following precedence:
        # String > Boolean > Double > Integer
        if pipeline_spec_pb2.ParameterType.STRING in parameter_types:
            canonical_parameter_type = pipeline_spec_pb2.ParameterType.STRING
        elif pipeline_spec_pb2.ParameterType.BOOLEAN in parameter_types:
            canonical_parameter_type = pipeline_spec_pb2.ParameterType.BOOLEAN
        else:
            # Must be a double and int, promote to double.
            assert pipeline_spec_pb2.ParameterType.NUMBER_DOUBLE in parameter_types, \
                'Types: {} [{} {}]'.format(
                parameter_types, left_operand, right_operand)
            assert pipeline_spec_pb2.ParameterType.NUMBER_INTEGER in parameter_types, \
                'Types: {} [{} {}]'.format(
                parameter_types, left_operand, right_operand)
            canonical_parameter_type = pipeline_spec_pb2.ParameterType.NUMBER_DOUBLE
    elif len(parameter_types) == 1:  # Both operands are the same type.
        canonical_parameter_type = parameter_types.pop()
    else:
        # Probably shouldn't happen.
        raise ValueError('Unable to determine operand types for'
                         ' "{}" and "{}"'.format(left_operand, right_operand))

    operand_values = []
    for value_or_reference in [left_operand, right_operand]:
        if isinstance(value_or_reference, pipeline_channel.PipelineChannel):
            input_name = _additional_input_name_for_pipeline_channel(
                value_or_reference)
            operand_value = "inputs.parameter_values['{input_name}']".format(
                input_name=input_name)
            parameter_type = type_utils.get_parameter_type(
                value_or_reference.channel_type)
            if parameter_type == pipeline_spec_pb2.ParameterType.NUMBER_INTEGER:
                operand_value = 'int({})'.format(operand_value)
        elif isinstance(value_or_reference, str):
            operand_value = "'{}'".format(value_or_reference)
            parameter_type = pipeline_spec_pb2.ParameterType.STRING
        elif isinstance(value_or_reference, bool):
            # Booleans need to be compared as 'true' or 'false' in CEL.
            operand_value = str(value_or_reference).lower()
            parameter_type = pipeline_spec_pb2.ParameterType.BOOLEAN
        elif isinstance(value_or_reference, int):
            operand_value = str(value_or_reference)
            parameter_type = pipeline_spec_pb2.ParameterType.NUMBER_INTEGER
        else:
            assert isinstance(value_or_reference, float), value_or_reference
            operand_value = str(value_or_reference)
            parameter_type = pipeline_spec_pb2.ParameterType.NUMBER_DOUBLE

        if parameter_type != canonical_parameter_type:
            # Type-cast to so CEL does not complain.
            if canonical_parameter_type == pipeline_spec_pb2.ParameterType.STRING:
                assert parameter_type in [
                    pipeline_spec_pb2.ParameterType.BOOLEAN,
                    pipeline_spec_pb2.ParameterType.NUMBER_INTEGER,
                    pipeline_spec_pb2.ParameterType.NUMBER_DOUBLE,
                ]
                operand_value = "'{}'".format(operand_value)
            elif canonical_parameter_type == pipeline_spec_pb2.ParameterType.BOOLEAN:
                assert parameter_type in [
                    pipeline_spec_pb2.ParameterType.NUMBER_INTEGER,
                    pipeline_spec_pb2.ParameterType.NUMBER_DOUBLE,
                ]
                operand_value = 'true' if int(operand_value) == 0 else 'false'
            else:
                assert canonical_parameter_type == pipeline_spec_pb2.ParameterType.NUMBER_DOUBLE
                assert parameter_type == pipeline_spec_pb2.ParameterType.NUMBER_INTEGER
                operand_value = 'double({})'.format(operand_value)

        operand_values.append(operand_value)

    return tuple(operand_values)


def _update_task_spec_for_condition_group(
    group: tasks_group.Condition,
    pipeline_task_spec: pipeline_spec_pb2.PipelineTaskSpec,
) -> None:
    """Updates PipelineTaskSpec for condition group.

    Args:
        group: The condition group to update task spec for.
        pipeline_task_spec: The pipeline task spec to update in place.
    """
    left_operand_value, right_operand_value = _resolve_condition_operands(
        group.condition.left_operand, group.condition.right_operand)

    condition_string = (
        f'{left_operand_value} {group.condition.operator} {right_operand_value}'
    )
    pipeline_task_spec.trigger_policy.CopyFrom(
        pipeline_spec_pb2.PipelineTaskSpec.TriggerPolicy(
            condition=condition_string))


def build_task_spec_for_exit_task(
    task: pipeline_task.PipelineTask,
    dependent_task: str,
    pipeline_inputs: pipeline_spec_pb2.ComponentInputsSpec,
) -> pipeline_spec_pb2.PipelineTaskSpec:
    """Builds PipelineTaskSpec for an exit handler's exit task.

    Args:
        tasks: The exit handler's exit task to build task spec for.
        dependent_task: The dependent task name for the exit task, i.e. the name
            of the exit handler group.
        pipeline_inputs: The pipeline level input definitions.

    Returns:
        A PipelineTaskSpec object representing the exit task.
    """
    pipeline_task_spec = build_task_spec_for_task(
        task=task,
        parent_component_inputs=pipeline_inputs,
        tasks_in_current_dag=[],  # Does not matter for exit task
        input_parameters_in_current_dag=pipeline_inputs.parameters.keys(),
        input_artifacts_in_current_dag=[],
    )
    pipeline_task_spec.dependent_tasks.extend([dependent_task])
    pipeline_task_spec.trigger_policy.strategy = (
        pipeline_spec_pb2.PipelineTaskSpec.TriggerPolicy.TriggerStrategy
        .ALL_UPSTREAM_TASKS_COMPLETED)

    for input_name, input_spec in task.component_spec.inputs.items():
        if type_utils.is_task_final_status_type(input_spec.type):
            pipeline_task_spec.inputs.parameters[
                input_name].task_final_status.producer_task = dependent_task

    return pipeline_task_spec


def build_task_spec_for_group(
    group: tasks_group.TasksGroup,
    pipeline_channels: List[pipeline_channel.PipelineChannel],
    tasks_in_current_dag: List[str],
    is_parent_component_root: bool,
) -> pipeline_spec_pb2.PipelineTaskSpec:
    """Builds PipelineTaskSpec for a group.

    Args:
        group: The group to build PipelineTaskSpec for.
        pipeline_channels: The list of pipeline channels referenced by the group.
        tasks_in_current_dag: The list of tasks names for tasks in the same dag.
        is_parent_component_root: Whether the parent component is the pipeline's
            root dag.

    Returns:
        A PipelineTaskSpec object representing the group.
    """
    pipeline_task_spec = pipeline_spec_pb2.PipelineTaskSpec()
    pipeline_task_spec.task_info.name = group.display_name or group.name
    pipeline_task_spec.component_ref.name = (
        component_utils.sanitize_component_name(group.name))

    for channel in pipeline_channels:

        channel_full_name = channel.full_name
        subvar_name = None
        if isinstance(channel, for_loop.LoopArgumentVariable):
            channel_full_name = channel.loop_argument.full_name
            subvar_name = channel.subvar_name

        input_name = _additional_input_name_for_pipeline_channel(channel)

        channel_name = channel.name
        if subvar_name:
            pipeline_task_spec.inputs.parameters[
                input_name].parameter_expression_selector = (
                    'parseJson(string_value)["{}"]'.format(subvar_name))
            if not channel.is_with_items_loop_argument:
                channel_name = channel.items_or_pipeline_channel.name

        if isinstance(channel, pipeline_channel.PipelineArtifactChannel):
            if channel.task_name and channel.task_name in tasks_in_current_dag:
                pipeline_task_spec.inputs.artifacts[
                    input_name].task_output_artifact.producer_task = (
                        component_utils.sanitize_task_name(channel.task_name))
                pipeline_task_spec.inputs.artifacts[
                    input_name].task_output_artifact.output_artifact_key = (
                        channel_name)
            else:
                pipeline_task_spec.inputs.artifacts[
                    input_name].component_input_artifact = (
                        channel_full_name
                        if is_parent_component_root else input_name)
        else:
            # channel is one of PipelineParameterChannel, LoopArgument, or
            # LoopArgumentVariable
            if channel.task_name and channel.task_name in tasks_in_current_dag:
                pipeline_task_spec.inputs.parameters[
                    input_name].task_output_parameter.producer_task = (
                        component_utils.sanitize_task_name(channel.task_name))
                pipeline_task_spec.inputs.parameters[
                    input_name].task_output_parameter.output_parameter_key = (
                        channel_name)
            else:
                pipeline_task_spec.inputs.parameters[
                    input_name].component_input_parameter = (
                        channel_full_name if is_parent_component_root else
                        _additional_input_name_for_pipeline_channel(
                            channel_full_name))

    if isinstance(group, tasks_group.ParallelFor):
        _update_task_spec_for_loop_group(
            group=group,
            pipeline_task_spec=pipeline_task_spec,
        )
    elif isinstance(group, tasks_group.Condition):
        _update_task_spec_for_condition_group(
            group=group,
            pipeline_task_spec=pipeline_task_spec,
        )

    return pipeline_task_spec


def populate_metrics_in_dag_outputs(
    tasks: List[pipeline_task.PipelineTask],
    task_name_to_parent_groups: Mapping[str, List[GroupOrTaskType]],
    task_name_to_task_spec: Mapping[str, pipeline_spec_pb2.PipelineTaskSpec],
    task_name_to_component_spec: Mapping[str, pipeline_spec_pb2.ComponentSpec],
    pipeline_spec: pipeline_spec_pb2.PipelineSpec,
) -> None:
    """Populates metrics artifacts in DAG outputs.

    Args:
        tasks: The list of tasks that may produce metrics outputs.
        task_name_to_parent_groups: The dict of task name to parent groups.
            Key is the task's name. Value is a list of ancestor groups including
            the task itself. The list of a given op is sorted in a way that the
            farthest group is the first and the task itself is the last.
        task_name_to_task_spec: The dict of task name to PipelineTaskSpec.
        task_name_to_component_spec: The dict of task name to ComponentSpec.
        pipeline_spec: The pipeline_spec to update in-place.
    """
    for task in tasks:
        component_spec = task_name_to_component_spec[task.name]

        # Get the tuple of (component_name, task_name) of all its parent groups.
        parent_components_and_tasks = [('_root', '')]
        # skip the op itself and the root group which cannot be retrived via name.
        for group_name in task_name_to_parent_groups[task.name][1:-1]:
            parent_components_and_tasks.append(
                (component_utils.sanitize_component_name(group_name),
                 component_utils.sanitize_task_name(group_name)))
        # Reverse the order to make the farthest group in the end.
        parent_components_and_tasks.reverse()

        for output_name, artifact_spec in \
            component_spec.output_definitions.artifacts.items():

            if artifact_spec.artifact_type.WhichOneof(
                    'kind'
            ) == 'schema_title' and artifact_spec.artifact_type.schema_title in [
                    artifact_types.Metrics.TYPE_NAME,
                    artifact_types.ClassificationMetrics.TYPE_NAME,
            ]:
                unique_output_name = '{}-{}'.format(task.name, output_name)

                sub_task_name = task.name
                sub_task_output = output_name
                for component_name, task_name in parent_components_and_tasks:
                    group_component_spec = (
                        pipeline_spec.root if component_name == '_root' else
                        pipeline_spec.components[component_name])
                    group_component_spec.output_definitions.artifacts[
                        unique_output_name].CopyFrom(artifact_spec)
                    group_component_spec.dag.outputs.artifacts[
                        unique_output_name].artifact_selectors.append(
                            pipeline_spec_pb2.DagOutputsSpec
                            .ArtifactSelectorSpec(
                                producer_subtask=sub_task_name,
                                output_artifact_key=sub_task_output,
                            ))
                    sub_task_name = task_name
                    sub_task_output = unique_output_name


def make_invalid_input_type_error_msg(arg_name: str, arg_type: Any) -> str:
    valid_types = (
        str.__name__,
        int.__name__,
        float.__name__,
        bool.__name__,
        dict.__name__,
        list.__name__,
    )
    return f"The pipeline parameter '{arg_name}' of type {arg_type} is not a valid input for this component. Passing artifacts as pipeline inputs is not supported. Consider annotating the parameter with a primitive type such as {valid_types}."


def modify_component_spec_for_compile(
    component_spec: structures.ComponentSpec,
    pipeline_name: Optional[str],
    pipeline_parameters_override: Optional[Mapping[str, Any]],
) -> structures.ComponentSpec:
    """Modifies the ComponentSpec using arguments passed to the
    Compiler.compile method.

    Args:
        component_spec (structures.ComponentSpec): ComponentSpec to modify.
        pipeline_name (Optional[str]): Name of the pipeline. Overrides component name.
        pipeline_parameters_override (Optional[Mapping[str, Any]]): Pipeline parameters. Overrides component input default values.

    Raises:
        ValueError: If a parameter is passed to the compiler that is not a component input.

    Returns:
        structures.ComponentSpec: The modified ComponentSpec.
    """
    pipeline_name = pipeline_name or component_utils.sanitize_component_name(
        component_spec.name).replace(utils._COMPONENT_NAME_PREFIX, '')

    component_spec.name = pipeline_name
    if component_spec.inputs is not None:
        pipeline_parameters_override = pipeline_parameters_override or {}
        for input_name in pipeline_parameters_override:
            if input_name not in component_spec.inputs:
                raise ValueError(
                    f'Parameter {input_name} does not match any known component parameters.'
                )
            component_spec.inputs[
                input_name].default = pipeline_parameters_override[input_name]

    return component_spec


def build_spec_by_group(
    pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    deployment_config: pipeline_spec_pb2.PipelineDeploymentConfig,
    group: tasks_group.TasksGroup,
    inputs: Mapping[str, List[Tuple[dsl.PipelineChannel, str]]],
    dependencies: Dict[str, List[GroupOrTaskType]],
    rootgroup_name: str,
    task_name_to_parent_groups: Mapping[str, List[GroupOrTaskType]],
    group_name_to_parent_groups: Mapping[str, List[tasks_group.TasksGroup]],
    name_to_for_loop_group: Mapping[str, dsl.ParallelFor],
) -> None:
    """Generates IR spec given a TasksGroup.

    Args:
        pipeline_spec: The pipeline_spec to update in place.
        deployment_config: The deployment_config to hold all executors. The
            spec is updated in place.
        group: The TasksGroup to generate spec for.
        inputs: The inputs dictionary. The keys are group/task names and the
            values are lists of tuples (channel, producing_task_name).
        dependencies: The group dependencies dictionary. The keys are group
            or task names, and the values are lists of dependent groups or
            tasks.
        rootgroup_name: The name of the group root. Used to determine whether
            the component spec for the current group should be the root dag.
        task_name_to_parent_groups: The dict of task name to parent groups.
            Key is task name. Value is a list of ancestor groups including
            the task itself. The list of a given task is sorted in a way that
            the farthest group is the first and the task itself is the last.
        group_name_to_parent_groups: The dict of group name to parent groups.
            Key is the group name. Value is a list of ancestor groups
            including the group itself. The list of a given group is sorted
            in a way that the farthest group is the first and the group
            itself is the last.
        name_to_for_loop_group: The dict of for loop group name to loop
            group.
    """
    group_component_name = component_utils.sanitize_component_name(group.name)

    if group.name == rootgroup_name:
        group_component_spec = pipeline_spec.root
    else:
        group_component_spec = pipeline_spec.components[group_component_name]

    task_name_to_task_spec = {}
    task_name_to_component_spec = {}

    # Generate task specs and component specs for the dag.
    subgroups = group.groups + group.tasks
    for subgroup in subgroups:

        subgroup_inputs = inputs.get(subgroup.name, [])
        subgroup_channels = [channel for channel, _ in subgroup_inputs]

        subgroup_component_name = (
            component_utils.sanitize_component_name(subgroup.name))

        tasks_in_current_dag = [
            component_utils.sanitize_task_name(subgroup.name)
            for subgroup in subgroups
        ]
        input_parameters_in_current_dag = [
            input_name
            for input_name in group_component_spec.input_definitions.parameters
        ]
        input_artifacts_in_current_dag = [
            input_name
            for input_name in group_component_spec.input_definitions.artifacts
        ]
        is_parent_component_root = (group_component_spec == pipeline_spec.root)

        if isinstance(subgroup, pipeline_task.PipelineTask):

            subgroup_task_spec = builder.build_task_spec_for_task(
                task=subgroup,
                parent_component_inputs=group_component_spec.input_definitions,
                tasks_in_current_dag=tasks_in_current_dag,
                input_parameters_in_current_dag=input_parameters_in_current_dag,
                input_artifacts_in_current_dag=input_artifacts_in_current_dag,
            )
            task_name_to_task_spec[subgroup.name] = subgroup_task_spec

            subgroup_component_spec = builder.build_component_spec_for_task(
                task=subgroup)
            task_name_to_component_spec[subgroup.name] = subgroup_component_spec

            executor_label = subgroup_component_spec.executor_label

            if executor_label not in deployment_config.executors:
                if subgroup.container_spec is not None:
                    subgroup_container_spec = builder.build_container_spec_for_task(
                        task=subgroup)
                    deployment_config.executors[
                        executor_label].container.CopyFrom(
                            subgroup_container_spec)
                elif subgroup.importer_spec is not None:
                    subgroup_importer_spec = builder.build_importer_spec_for_task(
                        task=subgroup)
                    deployment_config.executors[
                        executor_label].importer.CopyFrom(
                            subgroup_importer_spec)
        elif isinstance(subgroup, dsl.ParallelFor):

            # "Punch the hole", adding additional inputs (other than loop
            # arguments which will be handled separately) needed by its
            # subgroups or tasks.
            loop_subgroup_channels = []

            for channel in subgroup_channels:
                # Skip 'withItems' loop arguments if it's from an inner loop.
                if isinstance(
                        channel,
                    (for_loop.LoopArgument, for_loop.LoopArgumentVariable
                    )) and channel.is_with_items_loop_argument:
                    withitems_loop_arg_found_in_self_or_upstream = False
                    for group_name in group_name_to_parent_groups[
                            subgroup.name][::-1]:
                        if group_name in name_to_for_loop_group:
                            loop_group = name_to_for_loop_group[group_name]
                            if channel.name in loop_group.loop_argument.name:
                                withitems_loop_arg_found_in_self_or_upstream = True
                                break
                    if not withitems_loop_arg_found_in_self_or_upstream:
                        continue
                loop_subgroup_channels.append(channel)

            if subgroup.items_is_pipeline_channel:
                # This loop_argument is based on a pipeline channel, i.e.,
                # rather than a static list, it is either the output of
                # another task or an input as global pipeline parameters.
                loop_subgroup_channels.append(
                    subgroup.loop_argument.items_or_pipeline_channel)

            loop_subgroup_channels.append(subgroup.loop_argument)

            subgroup_component_spec = builder.build_component_spec_for_group(
                pipeline_channels=loop_subgroup_channels,
                is_root_group=False,
            )

            subgroup_task_spec = builder.build_task_spec_for_group(
                group=subgroup,
                pipeline_channels=loop_subgroup_channels,
                tasks_in_current_dag=tasks_in_current_dag,
                is_parent_component_root=is_parent_component_root,
            )

        elif isinstance(subgroup, dsl.Condition):

            # "Punch the hole", adding inputs needed by its subgroups or
            # tasks.
            condition_subgroup_channels = list(subgroup_channels)
            for operand in [
                    subgroup.condition.left_operand,
                    subgroup.condition.right_operand,
            ]:
                if isinstance(operand, dsl.PipelineChannel):
                    condition_subgroup_channels.append(operand)

            subgroup_component_spec = builder.build_component_spec_for_group(
                pipeline_channels=condition_subgroup_channels,
                is_root_group=False,
            )

            subgroup_task_spec = builder.build_task_spec_for_group(
                group=subgroup,
                pipeline_channels=condition_subgroup_channels,
                tasks_in_current_dag=tasks_in_current_dag,
                is_parent_component_root=is_parent_component_root,
            )

        elif isinstance(subgroup, dsl.ExitHandler):

            subgroup_component_spec = builder.build_component_spec_for_group(
                pipeline_channels=subgroup_channels,
                is_root_group=False,
            )

            subgroup_task_spec = builder.build_task_spec_for_group(
                group=subgroup,
                pipeline_channels=subgroup_channels,
                tasks_in_current_dag=tasks_in_current_dag,
                is_parent_component_root=is_parent_component_root,
            )

        else:
            raise RuntimeError(
                f'Unexpected task/group type: Got {subgroup} of type '
                f'{type(subgroup)}.')

        # Generate dependencies section for this task.
        if dependencies.get(subgroup.name, None):
            group_dependencies = list(dependencies[subgroup.name])
            group_dependencies.sort()
            subgroup_task_spec.dependent_tasks.extend([
                component_utils.sanitize_task_name(dep)
                for dep in group_dependencies
            ])

        # Add component spec if not exists
        if subgroup_component_name not in pipeline_spec.components:
            pipeline_spec.components[subgroup_component_name].CopyFrom(
                subgroup_component_spec)

        # Add task spec
        group_component_spec.dag.tasks[subgroup.name].CopyFrom(
            subgroup_task_spec)

    pipeline_spec.deployment_spec.update(
        json_format.MessageToDict(deployment_config))

    # Surface metrics outputs to the top.
    builder.populate_metrics_in_dag_outputs(
        tasks=group.tasks,
        task_name_to_parent_groups=task_name_to_parent_groups,
        task_name_to_task_spec=task_name_to_task_spec,
        task_name_to_component_spec=task_name_to_component_spec,
        pipeline_spec=pipeline_spec,
    )


def get_parent_groups(
    root_group: tasks_group.TasksGroup,
) -> Tuple[Mapping[str, List[GroupOrTaskType]], Mapping[str,
                                                        List[GroupOrTaskType]]]:
    """Get parent groups that contain the specified tasks.

    Each pipeline has a root group. Each group has a list of tasks (leaf)
    and groups.
    This function traverse the tree and get ancestor groups for all tasks.

    Args:
        root_group: The root group of a pipeline.

    Returns:
        A tuple. The first item is a mapping of task names to parent groups,
        and second item is a mapping of group names to parent groups.
        A list of parent groups is a list of ancestor groups including the
        task/group itself. The list is sorted in a way that the farthest
        parent group is the first and task/group itself is the last.
    """

    def _get_parent_groups_helper(
        current_groups: List[tasks_group.TasksGroup],
        tasks_to_groups: Dict[str, List[GroupOrTaskType]],
        groups_to_groups: Dict[str, List[GroupOrTaskType]],
    ) -> None:
        root_group = current_groups[-1]
        for group in root_group.groups:

            groups_to_groups[group.name] = [x.name for x in current_groups
                                           ] + [group.name]
            current_groups.append(group)

            _get_parent_groups_helper(
                current_groups=current_groups,
                tasks_to_groups=tasks_to_groups,
                groups_to_groups=groups_to_groups,
            )
            del current_groups[-1]

        for task in root_group.tasks:
            tasks_to_groups[task.name] = [x.name for x in current_groups
                                         ] + [task.name]

    tasks_to_groups = {}
    groups_to_groups = {}
    current_groups = [root_group]

    _get_parent_groups_helper(
        current_groups=current_groups,
        tasks_to_groups=tasks_to_groups,
        groups_to_groups=groups_to_groups,
    )
    return (tasks_to_groups, groups_to_groups)


def validate_pipeline_name(name: str) -> None:
    """Validate pipeline name.

    A valid pipeline name should match ^[a-z0-9][a-z0-9-]{0,127}$.

    Args:
        name: The pipeline name.

    Raises:
        ValueError if the pipeline name doesn't conform to the regular expression.
    """
    pattern = re.compile(r'^[a-z0-9][a-z0-9-]{0,127}$')
    if not pattern.match(name):
        raise ValueError(
            'Invalid pipeline name: %s.\n'
            'Please specify a pipeline name that matches the regular '
            'expression "^[a-z0-9][a-z0-9-]{0,127}$" using '
            '`dsl.pipeline(name=...)` decorator.' % name)


def create_pipeline_spec_for_component(
        pipeline_name: str, pipeline_args: List[dsl.PipelineChannel],
        task_group: tasks_group.TasksGroup) -> pipeline_spec_pb2.PipelineSpec:
    """Creates a pipeline spec object for a component (single-component
    pipeline).

    Args:
        pipeline_name: The pipeline name.
        pipeline_args: The PipelineChannel arguments to the pipeline.
        task_group: The pipeline's single task group (containing a single
            task).

    Returns:
        A PipelineSpec proto representing the compiled pipeline.

    Raises:
        ValueError: If the argument is of unsupported types.
    """

    # this is method is essentially a simplified version
    # of _create_pipeline_spec

    # one-by-one building up the arguments for self._build_spec_by_group
    validate_pipeline_name(pipeline_name)

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

    task_name_to_parent_groups, group_name_to_parent_groups = get_parent_groups(
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

    build_spec_by_group(
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
