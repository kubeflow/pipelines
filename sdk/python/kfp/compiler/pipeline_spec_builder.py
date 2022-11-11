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
"""Functions for creating PipelineSpec proto objects."""

import json
from typing import Any, Dict, List, Mapping, Optional, Tuple, Union
import warnings

from google.protobuf import json_format
from google.protobuf import struct_pb2
import kfp
from kfp.compiler import compiler_utils
from kfp.components import for_loop
from kfp.components import pipeline_channel
from kfp.components import pipeline_context
from kfp.components import pipeline_task
from kfp.components import placeholders
from kfp.components import structures
from kfp.components import tasks_group
from kfp.components import utils
from kfp.components.types import artifact_types
from kfp.components.types import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2
import yaml

# must be defined here to avoid circular imports
group_type_to_dsl_class = {
    tasks_group.TasksGroupType.PIPELINE: pipeline_context.Pipeline,
    tasks_group.TasksGroupType.CONDITION: tasks_group.Condition,
    tasks_group.TasksGroupType.FOR_LOOP: tasks_group.ParallelFor,
    tasks_group.TasksGroupType.EXIT_HANDLER: tasks_group.ExitHandler,
}

_SINGLE_OUTPUT_NAME = 'Output'


def _additional_input_name_for_pipeline_channel(
        channel_or_name: Union[pipeline_channel.PipelineChannel, str]) -> str:
    """Gets the name for an additional (compiler-injected) input."""

    # Adding a prefix to avoid (reduce chance of) name collision between the
    # original component inputs and the injected input.
    return 'pipelinechannel--' + (
        channel_or_name.full_name if isinstance(
            channel_or_name, pipeline_channel.PipelineChannel) else
        channel_or_name)


def to_protobuf_value(value: type_utils.PARAMETER_TYPES) -> struct_pb2.Value:
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
                fields={k: to_protobuf_value(v) for k, v in value.items()}))
    elif isinstance(value, list):
        return struct_pb2.Value(
            list_value=struct_pb2.ListValue(
                values=[to_protobuf_value(v) for v in value]))
    else:
        raise ValueError('Value must be one of the following types: '
                         'str, int, float, bool, dict, and list. Got: '
                         f'"{value}" of type "{type(value)}".')


def build_task_spec_for_task(
    task: pipeline_task.PipelineTask,
    parent_component_inputs: pipeline_spec_pb2.ComponentInputsSpec,
    tasks_in_current_dag: List[str],
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

    Returns:
        A PipelineTaskSpec object representing the task.
    """
    pipeline_task_spec = pipeline_spec_pb2.PipelineTaskSpec()
    pipeline_task_spec.task_info.name = (
        task._task_spec.display_name or task.name)
    # Use task.name for component_ref.name because we may customize component
    # spec for individual tasks to work around the lack of optional inputs
    # support in IR.
    pipeline_task_spec.component_ref.name = (
        utils.sanitize_component_name(task.name))
    pipeline_task_spec.caching_options.enable_cache = (
        task._task_spec.enable_caching)

    if task._task_spec.retry_policy is not None:
        pipeline_task_spec.retry_policy.CopyFrom(
            task._task_spec.retry_policy.to_proto())

    for input_name, input_value in task.inputs.items():
        if isinstance(input_value, pipeline_channel.PipelineArtifactChannel):

            if input_value.task_name:
                # Value is produced by an upstream task.
                if input_value.task_name in tasks_in_current_dag:
                    # Dependent task within the same DAG.
                    pipeline_task_spec.inputs.artifacts[
                        input_name].task_output_artifact.producer_task = (
                            utils.sanitize_task_name(input_value.task_name))
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
                component_input_artifact = input_value.full_name
                if component_input_artifact not in parent_component_inputs.artifacts:
                    component_input_artifact = (
                        _additional_input_name_for_pipeline_channel(input_value)
                    )
                pipeline_task_spec.inputs.artifacts[
                    input_name].component_input_artifact = (
                        component_input_artifact)

        elif isinstance(input_value, pipeline_channel.PipelineParameterChannel):

            if input_value.task_name:
                # Value is produced by an upstream task.
                if input_value.task_name in tasks_in_current_dag:
                    # Dependent task within the same DAG.
                    pipeline_task_spec.inputs.parameters[
                        input_name].task_output_parameter.producer_task = (
                            utils.sanitize_task_name(input_value.task_name))
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

                additional_input_placeholder = placeholders.InputValuePlaceholder(
                    additional_input_name)._to_string()
                input_value = input_value.replace(channel.pattern,
                                                  additional_input_placeholder)

                if channel.task_name:
                    # Value is produced by an upstream task.
                    if channel.task_name in tasks_in_current_dag:
                        # Dependent task within the same DAG.
                        pipeline_task_spec.inputs.parameters[
                            additional_input_name].task_output_parameter.producer_task = (
                                utils.sanitize_task_name(channel.task_name))
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
                    to_protobuf_value(input_value))

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
    component_spec.executor_label = utils.sanitize_executor_label(task.name)

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
                _fill_in_component_input_default_value(
                    component_spec=component_spec,
                    input_name=input_name,
                    default_value=input_spec.default,
                )

        else:
            component_spec.input_definitions.artifacts[
                input_name].artifact_type.CopyFrom(
                    type_utils.bundled_artifact_to_artifact_proto(
                        input_spec.type))

    for output_name, output_spec in (task.component_spec.outputs or {}).items():
        if type_utils.is_parameter_type(output_spec.type):
            component_spec.output_definitions.parameters[
                output_name].parameter_type = type_utils.get_parameter_type(
                    output_spec.type)
        else:
            component_spec.output_definitions.artifacts[
                output_name].artifact_type.CopyFrom(
                    type_utils.bundled_artifact_to_artifact_proto(
                        output_spec.type))

    return component_spec


# TODO(chensun): merge with build_component_spec_for_task
def _build_component_spec_from_component_spec_structure(
    component_spec_struct: structures.ComponentSpec,
) -> pipeline_spec_pb2.ComponentSpec:
    """Builds ComponentSpec proto from ComponentSpec structure."""
    component_spec = pipeline_spec_pb2.ComponentSpec()

    for input_name, input_spec in (component_spec_struct.inputs or {}).items():

        # Special handling for PipelineTaskFinalStatus first.
        if type_utils.is_task_final_status_type(input_spec.type):
            component_spec.input_definitions.parameters[
                input_name].parameter_type = pipeline_spec_pb2.ParameterType.STRUCT
            continue

        if type_utils.is_parameter_type(input_spec.type):
            component_spec.input_definitions.parameters[
                input_name].parameter_type = type_utils.get_parameter_type(
                    input_spec.type)
            if input_spec.default is not None:
                _fill_in_component_input_default_value(
                    component_spec=component_spec,
                    input_name=input_name,
                    default_value=input_spec.default,
                )

        else:
            component_spec.input_definitions.artifacts[
                input_name].artifact_type.CopyFrom(
                    type_utils.bundled_artifact_to_artifact_proto(
                        input_spec.type))

    for output_name, output_spec in (component_spec_struct.outputs or
                                     {}).items():
        if type_utils.is_parameter_type(output_spec.type):
            component_spec.output_definitions.parameters[
                output_name].parameter_type = type_utils.get_parameter_type(
                    output_spec.type)
        else:
            component_spec.output_definitions.artifacts[
                output_name].artifact_type.CopyFrom(
                    type_utils.bundled_artifact_to_artifact_proto(
                        output_spec.type))

    return component_spec


def _connect_dag_outputs(
    component_spec: pipeline_spec_pb2.ComponentSpec,
    output_name: str,
    output_channel: pipeline_channel.PipelineChannel,
) -> None:
    """Connects dag ouptut to a subtask output.

    Args:
        component_spec: The component spec to modify its dag outputs.
        output_name: The name of the dag output.
        output_channel: The pipeline channel selected for the dag output.
    """
    if isinstance(output_channel, pipeline_channel.PipelineArtifactChannel):
        if output_name not in component_spec.output_definitions.artifacts:
            raise ValueError(f'Pipeline output not defined: {output_name}.')
        component_spec.dag.outputs.artifacts[
            output_name].artifact_selectors.append(
                pipeline_spec_pb2.DagOutputsSpec.ArtifactSelectorSpec(
                    producer_subtask=output_channel.task_name,
                    output_artifact_key=output_channel.name,
                ))
    elif isinstance(output_channel, pipeline_channel.PipelineParameterChannel):
        if output_name not in component_spec.output_definitions.parameters:
            raise ValueError(f'Pipeline output not defined: {output_name}.')
        component_spec.dag.outputs.parameters[
            output_name].value_from_parameter.producer_subtask = output_channel.task_name
        component_spec.dag.outputs.parameters[
            output_name].value_from_parameter.output_parameter_key = output_channel.name


def _build_dag_outputs(
    component_spec: pipeline_spec_pb2.ComponentSpec,
    dag_outputs: Optional[Any],
) -> None:
    """Builds DAG output spec."""
    if dag_outputs is not None:
        if isinstance(dag_outputs, pipeline_channel.PipelineChannel):
            _connect_dag_outputs(
                component_spec=component_spec,
                output_name=_SINGLE_OUTPUT_NAME,
                output_channel=dag_outputs,
            )
        elif isinstance(dag_outputs, tuple) and hasattr(dag_outputs, '_asdict'):
            for output_name, output_channel in dag_outputs._asdict().items():
                _connect_dag_outputs(
                    component_spec=component_spec,
                    output_name=output_name,
                    output_channel=output_channel,
                )
    # Valid dag outputs covers all outptus in component definition.
    for output_name in component_spec.output_definitions.artifacts:
        if output_name not in component_spec.dag.outputs.artifacts:
            raise ValueError(f'Missing pipeline output: {output_name}.')
    for output_name in component_spec.output_definitions.parameters:
        if output_name not in component_spec.dag.outputs.parameters:
            raise ValueError(f'Missing pipeline output: {output_name}.')


def build_importer_spec_for_task(
    task: pipeline_task.PipelineTask
) -> pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec:
    """Builds ImporterSpec for a pipeline task.

    Args:
        task: The task to build a ComponentSpec for.

    Returns:
        A ImporterSpec object for the task.
    """
    type_schema = type_utils.bundled_artifact_to_artifact_proto(
        task.importer_spec.schema_title)
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
        # cast to int to support v1 component YAML where NUMBER_INTEGER defaults are included as strings
        # for example, input Limit: https://raw.githubusercontent.com/kubeflow/pipelines/60a2612541ec08c6a85c237d2ec7525b12543a43/components/datasets/Chicago_Taxi_Trips/component.yaml
        component_spec.input_definitions.parameters[
            input_name].default_value.number_value = int(default_value)
        # cast to int to support v1 component YAML where NUMBER_DOUBLE defaults are included as strings
        # for example, input learning_rate: https://raw.githubusercontent.com/kubeflow/pipelines/567c04c51ff00a1ee525b3458425b17adbe3df61/components/XGBoost/Train/component.yaml
    elif pipeline_spec_pb2.ParameterType.NUMBER_DOUBLE == parameter_type:
        component_spec.input_definitions.parameters[
            input_name].default_value.number_value = float(default_value)
    elif pipeline_spec_pb2.ParameterType.STRING == parameter_type:
        component_spec.input_definitions.parameters[
            input_name].default_value.string_value = default_value
    elif pipeline_spec_pb2.ParameterType.BOOLEAN == parameter_type:
        component_spec.input_definitions.parameters[
            input_name].default_value.bool_value = default_value
    elif pipeline_spec_pb2.ParameterType.STRUCT == parameter_type:
        component_spec.input_definitions.parameters[
            input_name].default_value.CopyFrom(
                to_protobuf_value(default_value))
    elif pipeline_spec_pb2.ParameterType.LIST == parameter_type:
        component_spec.input_definitions.parameters[
            input_name].default_value.CopyFrom(
                to_protobuf_value(default_value))


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
                    type_utils.bundled_artifact_to_artifact_proto(
                        channel.channel_type))
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

    if (group.parallelism_limit > 0):
        pipeline_task_spec.iterator_policy.parallelism_limit = (
            group.parallelism_limit)

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
    )
    pipeline_task_spec.dependent_tasks.extend([dependent_task])
    pipeline_task_spec.trigger_policy.strategy = (
        pipeline_spec_pb2.PipelineTaskSpec.TriggerPolicy.TriggerStrategy
        .ALL_UPSTREAM_TASKS_COMPLETED)

    for input_name, input_spec in (task.component_spec.inputs or {}).items():
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
        utils.sanitize_component_name(group.name))

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
                        utils.sanitize_task_name(channel.task_name))
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
                        utils.sanitize_task_name(channel.task_name))
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
    task_name_to_parent_groups: Mapping[str,
                                        List[compiler_utils.GroupOrTaskType]],
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
                (utils.sanitize_component_name(group_name),
                 utils.sanitize_task_name(group_name)))
        # Reverse the order to make the farthest group in the end.
        parent_components_and_tasks.reverse()

        for output_name, artifact_spec in \
            component_spec.output_definitions.artifacts.items():

            if artifact_spec.artifact_type.WhichOneof(
                    'kind'
            ) == 'schema_title' and artifact_spec.artifact_type.schema_title in [
                    artifact_types.Metrics.schema_title,
                    artifact_types.ClassificationMetrics.schema_title,
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


def modify_pipeline_spec_with_override(
    pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    pipeline_name: Optional[str],
    pipeline_parameters: Optional[Mapping[str, Any]],
) -> pipeline_spec_pb2.PipelineSpec:
    """Modifies the PipelineSpec using arguments passed to the Compiler.compile
    method.

    Args:
        pipeline_spec (pipeline_spec_pb2.PipelineSpec): PipelineSpec to modify.
        pipeline_name (Optional[str]): Name of the pipeline. Overrides component name.
        pipeline_parameters (Optional[Mapping[str, Any]]): Pipeline parameters. Overrides component input default values.

    Returns:
        The modified PipelineSpec copy.
    Raises:
        ValueError: If a parameter is passed to the compiler that is not a component input.
    """
    pipeline_spec_new = pipeline_spec_pb2.PipelineSpec()
    pipeline_spec_new.CopyFrom(pipeline_spec)
    pipeline_spec = pipeline_spec_new

    if pipeline_name is not None:
        pipeline_spec.pipeline_info.name = pipeline_name

    # Verify that pipeline_parameters contains only input names
    # that match the pipeline inputs definition.
    for input_name, input_value in (pipeline_parameters or {}).items():
        if input_name in pipeline_spec.root.input_definitions.parameters:
            _fill_in_component_input_default_value(pipeline_spec.root,
                                                   input_name, input_value)
        elif input_name in pipeline_spec.root.input_definitions.artifacts:
            raise NotImplementedError(
                'Default value for artifact input is not supported.')
        else:
            raise ValueError('Pipeline parameter {} does not match any known '
                             'pipeline input.'.format(input_name))

    return pipeline_spec


def build_spec_by_group(
    pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    deployment_config: pipeline_spec_pb2.PipelineDeploymentConfig,
    group: tasks_group.TasksGroup,
    inputs: Mapping[str, List[Tuple[pipeline_channel.PipelineChannel, str]]],
    dependencies: Dict[str, List[compiler_utils.GroupOrTaskType]],
    rootgroup_name: str,
    task_name_to_parent_groups: Mapping[str,
                                        List[compiler_utils.GroupOrTaskType]],
    group_name_to_parent_groups: Mapping[str, List[tasks_group.TasksGroup]],
    name_to_for_loop_group: Mapping[str, tasks_group.ParallelFor],
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
    group_component_name = utils.sanitize_component_name(group.name)

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

        subgroup_component_name = (utils.sanitize_component_name(subgroup.name))

        tasks_in_current_dag = [
            utils.sanitize_task_name(subgroup.name) for subgroup in subgroups
        ]
        is_parent_component_root = (group_component_spec == pipeline_spec.root)

        if isinstance(subgroup, pipeline_task.PipelineTask):

            subgroup_task_spec = build_task_spec_for_task(
                task=subgroup,
                parent_component_inputs=group_component_spec.input_definitions,
                tasks_in_current_dag=tasks_in_current_dag,
            )
            task_name_to_task_spec[subgroup.name] = subgroup_task_spec

            subgroup_component_spec = build_component_spec_for_task(
                task=subgroup)
            task_name_to_component_spec[subgroup.name] = subgroup_component_spec

            if subgroup_component_spec.executor_label:
                executor_label = utils.make_name_unique_by_adding_index(
                    name=subgroup_component_spec.executor_label,
                    collection=list(deployment_config.executors.keys()),
                    delimiter='-')
                subgroup_component_spec.executor_label = executor_label

            if subgroup.container_spec is not None:
                subgroup_container_spec = build_container_spec_for_task(
                    task=subgroup)
                deployment_config.executors[executor_label].container.CopyFrom(
                    subgroup_container_spec)
            elif subgroup.importer_spec is not None:
                subgroup_importer_spec = build_importer_spec_for_task(
                    task=subgroup)
                deployment_config.executors[executor_label].importer.CopyFrom(
                    subgroup_importer_spec)
            elif subgroup.pipeline_spec is not None:
                sub_pipeline_spec = merge_deployment_spec_and_component_spec(
                    main_pipeline_spec=pipeline_spec,
                    main_deployment_config=deployment_config,
                    sub_pipeline_spec=subgroup.pipeline_spec,
                )
                subgroup_component_spec = sub_pipeline_spec.root
            else:
                raise RuntimeError
        elif isinstance(subgroup, tasks_group.ParallelFor):

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

            subgroup_component_spec = build_component_spec_for_group(
                pipeline_channels=loop_subgroup_channels,
                is_root_group=False,
            )

            subgroup_task_spec = build_task_spec_for_group(
                group=subgroup,
                pipeline_channels=loop_subgroup_channels,
                tasks_in_current_dag=tasks_in_current_dag,
                is_parent_component_root=is_parent_component_root,
            )

        elif isinstance(subgroup, tasks_group.Condition):

            # "Punch the hole", adding inputs needed by its subgroups or
            # tasks.
            condition_subgroup_channels = list(subgroup_channels)
            for operand in [
                    subgroup.condition.left_operand,
                    subgroup.condition.right_operand,
            ]:
                if isinstance(operand, pipeline_channel.PipelineChannel):
                    condition_subgroup_channels.append(operand)

            subgroup_component_spec = build_component_spec_for_group(
                pipeline_channels=condition_subgroup_channels,
                is_root_group=False,
            )

            subgroup_task_spec = build_task_spec_for_group(
                group=subgroup,
                pipeline_channels=condition_subgroup_channels,
                tasks_in_current_dag=tasks_in_current_dag,
                is_parent_component_root=is_parent_component_root,
            )

        elif isinstance(subgroup, tasks_group.ExitHandler):

            subgroup_component_spec = build_component_spec_for_group(
                pipeline_channels=subgroup_channels,
                is_root_group=False,
            )

            subgroup_task_spec = build_task_spec_for_group(
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
            subgroup_task_spec.dependent_tasks.extend(
                [utils.sanitize_task_name(dep) for dep in group_dependencies])

        # Add component spec
        subgroup_component_name = utils.make_name_unique_by_adding_index(
            name=subgroup_component_name,
            collection=list(pipeline_spec.components.keys()),
            delimiter='-')

        subgroup_task_spec.component_ref.name = subgroup_component_name
        pipeline_spec.components[subgroup_component_name].CopyFrom(
            subgroup_component_spec)

        # Add task spec
        group_component_spec.dag.tasks[subgroup.name].CopyFrom(
            subgroup_task_spec)

    pipeline_spec.deployment_spec.update(
        json_format.MessageToDict(deployment_config))

    # Surface metrics outputs to the top.
    populate_metrics_in_dag_outputs(
        tasks=group.tasks,
        task_name_to_parent_groups=task_name_to_parent_groups,
        task_name_to_component_spec=task_name_to_component_spec,
        pipeline_spec=pipeline_spec,
    )


def build_exit_handler_groups_recursively(
    parent_group: tasks_group.TasksGroup,
    pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    deployment_config: pipeline_spec_pb2.PipelineDeploymentConfig,
):
    if not parent_group.groups:
        return
    for group in parent_group.groups:
        if isinstance(group, tasks_group.ExitHandler):

            # remove this if block to support nested exit handlers
            if not parent_group.is_root:
                raise ValueError(
                    f'{tasks_group.ExitHandler.__name__} can only be used within the outermost scope of a pipeline function definition. Using an {tasks_group.ExitHandler.__name__} within {group_type_to_dsl_class[parent_group.group_type].__name__} {parent_group.name} is not allowed.'
                )

            exit_task = group.exit_task
            exit_task_name = utils.sanitize_task_name(exit_task.name)
            exit_handler_group_task_name = utils.sanitize_task_name(group.name)

            exit_task_task_spec = build_task_spec_for_exit_task(
                task=exit_task,
                dependent_task=exit_handler_group_task_name,
                pipeline_inputs=pipeline_spec.root.input_definitions,
            )

            exit_task_component_spec = build_component_spec_for_exit_task(
                task=exit_task)

            # Add exit task container spec if applicable.
            if exit_task.container_spec is not None:
                exit_task_container_spec = build_container_spec_for_task(
                    task=exit_task)
                executor_label = utils.make_name_unique_by_adding_index(
                    name=exit_task_component_spec.executor_label,
                    collection=list(deployment_config.executors.keys()),
                    delimiter='-')
                exit_task_component_spec.executor_label = executor_label
                deployment_config.executors[executor_label].container.CopyFrom(
                    exit_task_container_spec)
            elif exit_task.pipeline_spec is not None:
                exit_task_pipeline_spec = merge_deployment_spec_and_component_spec(
                    main_pipeline_spec=pipeline_spec,
                    main_deployment_config=deployment_config,
                    sub_pipeline_spec=exit_task.pipeline_spec,
                )
                exit_task_component_spec = exit_task_pipeline_spec.root
            else:
                raise RuntimeError(
                    f'Exit task {exit_task_name} is missing both container spec and pipeline spec.'
                )

            # Add exit task component spec.
            component_name = utils.make_name_unique_by_adding_index(
                name=exit_task_task_spec.component_ref.name,
                collection=list(pipeline_spec.components.keys()),
                delimiter='-')
            exit_task_task_spec.component_ref.name = component_name
            pipeline_spec.components[component_name].CopyFrom(
                exit_task_component_spec)

            # Add exit task task spec.
            parent_dag = pipeline_spec.root.dag
            parent_dag.tasks[exit_task_name].CopyFrom(exit_task_task_spec)

            pipeline_spec.deployment_spec.update(
                json_format.MessageToDict(deployment_config))

        build_exit_handler_groups_recursively(
            parent_group=group,
            pipeline_spec=pipeline_spec,
            deployment_config=deployment_config)


def merge_deployment_spec_and_component_spec(
    main_pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    main_deployment_config: pipeline_spec_pb2.PipelineDeploymentConfig,
    sub_pipeline_spec: pipeline_spec_pb2.PipelineSpec,
) -> pipeline_spec_pb2.PipelineSpec:
    """Merges deployment spec and component spec from a sub pipeline spec into
    the main spec.

    We need to make sure that we keep the original sub pipeline spec
    unchanged--in case the pipeline is reused (instantiated) multiple times,
    the "template" should not carry any "signs of usage".

    Args:
        main_pipeline_spec: The main pipeline spec to merge into.
        main_deployment_config: The main deployment config to merge into.
        sub_pipeline_spec: The pipeline spec of an inner pipeline whose
            deployment specs and component specs need to be copied into the main
            specs.

    Returns:
        The possibly modified copy of pipeline spec.
    """
    # Make a copy of the sub_pipeline_spec so that the "template" remains
    # unchanged and works even the pipeline is reused multiple times.
    sub_pipeline_spec_copy = pipeline_spec_pb2.PipelineSpec()
    sub_pipeline_spec_copy.CopyFrom(sub_pipeline_spec)

    _merge_deployment_spec(
        main_deployment_config=main_deployment_config,
        sub_pipeline_spec=sub_pipeline_spec_copy)
    _merge_component_spec(
        main_pipeline_spec=main_pipeline_spec,
        sub_pipeline_spec=sub_pipeline_spec_copy,
    )
    return sub_pipeline_spec_copy


def _merge_deployment_spec(
    main_deployment_config: pipeline_spec_pb2.PipelineDeploymentConfig,
    sub_pipeline_spec: pipeline_spec_pb2.PipelineSpec,
) -> None:
    """Merges deployment config from a sub pipeline spec into the main config.

    During the merge we need to ensure all executor specs have unique executor
    labels, that means we might need to update the `executor_label` referenced
    from component specs in sub_pipeline_spec.

    Args:
        main_deployment_config: The main deployment config to merge into.
        sub_pipeline_spec: The pipeline spec of an inner pipeline whose
            deployment configs need to be merged into the main config.
    """

    def _rename_executor_labels(
        pipeline_spec: pipeline_spec_pb2.PipelineSpec,
        old_executor_label: str,
        new_executor_label: str,
    ) -> None:
        """Renames the old executor_label to the new one in component spec."""
        for _, component_spec in pipeline_spec.components.items():
            if component_spec.executor_label == old_executor_label:
                component_spec.executor_label = new_executor_label

    sub_deployment_config = pipeline_spec_pb2.PipelineDeploymentConfig()
    json_format.ParseDict(
        json_format.MessageToDict(sub_pipeline_spec.deployment_spec),
        sub_deployment_config)

    for executor_label, executor_spec in sub_deployment_config.executors.items(
    ):
        old_executor_label = executor_label
        executor_label = utils.make_name_unique_by_adding_index(
            name=executor_label,
            collection=list(main_deployment_config.executors.keys()),
            delimiter='-')
        if executor_label != old_executor_label:
            _rename_executor_labels(
                pipeline_spec=sub_pipeline_spec,
                old_executor_label=old_executor_label,
                new_executor_label=executor_label)

        main_deployment_config.executors[executor_label].CopyFrom(executor_spec)


def _merge_component_spec(
    main_pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    sub_pipeline_spec: pipeline_spec_pb2.PipelineSpec,
) -> None:
    """Merges component spec from a sub pipeline spec into the main config.

    During the merge we need to ensure all component specs have unique component
    name, that means we might need to update the `component_ref` referenced from
    task specs in sub_pipeline_spec.

    Args:
        main_pipeline_spec: The main pipeline spec to merge into.
        sub_pipeline_spec: The pipeline spec of an inner pipeline whose
            component specs need to be merged into the global config.
    """

    def _rename_component_refs(
        pipeline_spec: pipeline_spec_pb2.PipelineSpec,
        old_component_ref: str,
        new_component_ref: str,
    ) -> None:
        """Renames the old component_ref to the new one in task spec."""
        for _, component_spec in pipeline_spec.components.items():
            if not component_spec.dag:
                continue
            for _, task_spec in component_spec.dag.tasks.items():
                if task_spec.component_ref.name == old_component_ref:
                    task_spec.component_ref.name = new_component_ref

        for _, task_spec in pipeline_spec.root.dag.tasks.items():
            if task_spec.component_ref.name == old_component_ref:
                task_spec.component_ref.name = new_component_ref

    # Do all the renaming in place, then do the acutal merge of component specs
    # in a second pass. This would ensure all component specs are in the final
    # state at the time of merging.
    old_name_to_new_name = {}
    for component_name, component_spec in sub_pipeline_spec.components.items():
        old_component_name = component_name
        new_component_name = utils.make_name_unique_by_adding_index(
            name=component_name,
            collection=list(main_pipeline_spec.components.keys()),
            delimiter='-')
        old_name_to_new_name[old_component_name] = new_component_name

        if new_component_name != old_component_name:
            _rename_component_refs(
                pipeline_spec=sub_pipeline_spec,
                old_component_ref=old_component_name,
                new_component_ref=new_component_name)

    for old_component_name, component_spec in sub_pipeline_spec.components.items(
    ):
        main_pipeline_spec.components[
            old_name_to_new_name[old_component_name]].CopyFrom(component_spec)


def create_pipeline_spec(
    pipeline: pipeline_context.Pipeline,
    component_spec: structures.ComponentSpec,
    pipeline_outputs: Optional[Any] = None,
) -> pipeline_spec_pb2.PipelineSpec:
    """Creates a pipeline spec object.

    Args:
        pipeline: The instantiated pipeline object.
        component_spec: The component spec structures.
        pipeline_outputs: The pipeline outputs via return.

    Returns:
        A PipelineSpec proto representing the compiled pipeline.

    Raises:
        ValueError if the argument is of unsupported types.
    """
    utils.validate_pipeline_name(pipeline.name)

    deployment_config = pipeline_spec_pb2.PipelineDeploymentConfig()
    pipeline_spec = pipeline_spec_pb2.PipelineSpec()

    pipeline_spec.pipeline_info.name = pipeline.name
    pipeline_spec.sdk_version = f'kfp-{kfp.__version__}'
    # Schema version 2.1.0 is required for kfp-pipeline-spec>0.1.13
    pipeline_spec.schema_version = '2.1.0'

    pipeline_spec.root.CopyFrom(
        _build_component_spec_from_component_spec_structure(component_spec))

    _build_dag_outputs(
        component_spec=pipeline_spec.root, dag_outputs=pipeline_outputs)

    root_group = pipeline.groups[0]

    all_groups = compiler_utils.get_all_groups(root_group)
    group_name_to_group = {group.name: group for group in all_groups}
    task_name_to_parent_groups, group_name_to_parent_groups = (
        compiler_utils.get_parent_groups(root_group))
    condition_channels = compiler_utils.get_condition_channels_for_tasks(
        root_group)
    name_to_for_loop_group = {
        group_name: group
        for group_name, group in group_name_to_group.items()
        if isinstance(group, tasks_group.ParallelFor)
    }
    inputs = compiler_utils.get_inputs_for_all_groups(
        pipeline=pipeline,
        task_name_to_parent_groups=task_name_to_parent_groups,
        group_name_to_parent_groups=group_name_to_parent_groups,
        condition_channels=condition_channels,
        name_to_for_loop_group=name_to_for_loop_group,
    )
    dependencies = compiler_utils.get_dependencies(
        pipeline=pipeline,
        task_name_to_parent_groups=task_name_to_parent_groups,
        group_name_to_parent_groups=group_name_to_parent_groups,
        group_name_to_group=group_name_to_group,
        condition_channels=condition_channels,
    )

    for group in all_groups:
        build_spec_by_group(
            pipeline_spec=pipeline_spec,
            deployment_config=deployment_config,
            group=group,
            inputs=inputs,
            dependencies=dependencies,
            rootgroup_name=root_group.name,
            task_name_to_parent_groups=task_name_to_parent_groups,
            group_name_to_parent_groups=group_name_to_parent_groups,
            name_to_for_loop_group=name_to_for_loop_group,
        )

    build_exit_handler_groups_recursively(
        parent_group=root_group,
        pipeline_spec=pipeline_spec,
        deployment_config=deployment_config,
    )

    return pipeline_spec


def write_pipeline_spec_to_file(pipeline_spec: pipeline_spec_pb2.PipelineSpec,
                                package_path: str) -> None:
    """Writes PipelineSpec into a YAML or JSON (deprecated) file.

    Args:
        pipeline_spec (pipeline_spec_pb2.PipelineSpec): The PipelineSpec.
        package_path (str): The path to which to write the PipelineSpec.
    """
    json_dict = json_format.MessageToDict(pipeline_spec)

    if package_path.endswith('.json'):
        warnings.warn(
            ('Compiling to JSON is deprecated and will be '
             'removed in a future version. Please compile to a YAML file by '
             'providing a file path with a .yaml extension instead.'),
            category=DeprecationWarning,
            stacklevel=2,
        )
        with open(package_path, 'w') as json_file:
            json.dump(json_dict, json_file, indent=2, sort_keys=True)

    elif package_path.endswith(('.yaml', '.yml')):
        with open(package_path, 'w') as yaml_file:
            yaml.dump(json_dict, yaml_file, sort_keys=True)

    else:
        raise ValueError(
            f'The output path {package_path} should end with ".yaml".')
