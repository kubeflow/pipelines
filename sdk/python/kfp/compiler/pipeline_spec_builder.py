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

import copy
import json
import typing
from typing import (Any, DefaultDict, Dict, List, Mapping, Optional, Tuple,
                    Union)
import warnings

from google.protobuf import json_format
from google.protobuf import struct_pb2
import kfp
from kfp import dsl
from kfp.compiler import compiler_utils
from kfp.compiler.compiler_utils import KubernetesManifestOptions
from kfp.dsl import component_factory
from kfp.dsl import for_loop
from kfp.dsl import pipeline_channel
from kfp.dsl import pipeline_config
from kfp.dsl import pipeline_context
from kfp.dsl import pipeline_task
from kfp.dsl import placeholders
from kfp.dsl import structures
from kfp.dsl import tasks_group
from kfp.dsl import utils
from kfp.dsl.types import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2
import yaml

# must be defined here to avoid circular imports
group_type_to_dsl_class = {
    tasks_group.TasksGroupType.PIPELINE: pipeline_context.Pipeline,
    tasks_group.TasksGroupType.CONDITION: tasks_group.Condition,
    tasks_group.TasksGroupType.FOR_LOOP: tasks_group.ParallelFor,
    tasks_group.TasksGroupType.EXIT_HANDLER: tasks_group.ExitHandler,
}


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
    # bool check must be above (int, float) check because bool is a subclass of int so isinstance(True, int) == True
    if isinstance(value, bool):
        return struct_pb2.Value(bool_value=value)
    elif isinstance(value, str):
        return struct_pb2.Value(string_value=value)
    elif isinstance(value, (int, float)):
        return struct_pb2.Value(number_value=value)
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
    if task._task_spec.cache_key:
        pipeline_task_spec.caching_options.cache_key = (
            task._task_spec.cache_key)

    if task._task_spec.retry_policy is not None:
        pipeline_task_spec.retry_policy.CopyFrom(
            task._task_spec.retry_policy.to_proto())

    # Inject resource fields into inputs
    if task.container_spec and task.container_spec.resources:
        for key, val in task.container_spec.resources.__dict__.items():
            if val and pipeline_channel.extract_pipeline_channels_from_any(val):
                task.inputs[key] = val

    if task.container_spec and task.container_spec.image:
        val = task.container_spec.image
        if val and pipeline_channel.extract_pipeline_channels_from_any(val):
            task.inputs['base_image'] = val

    for input_name, input_value in task.inputs.items():
        # Since LoopParameterArgument and LoopArtifactArgument and LoopArgumentVariable are narrower
        # types than PipelineParameterChannel, start with them.

        if isinstance(input_value, for_loop.LoopParameterArgument):

            component_input_parameter = (
                compiler_utils.additional_input_name_for_pipeline_channel(
                    input_value))
            assert component_input_parameter in parent_component_inputs.parameters, \
                f'component_input_parameter: {component_input_parameter} not found. All inputs: {parent_component_inputs}'
            pipeline_task_spec.inputs.parameters[
                input_name].component_input_parameter = (
                    component_input_parameter)

        elif isinstance(input_value, for_loop.LoopArtifactArgument):

            component_input_artifact = (
                compiler_utils.additional_input_name_for_pipeline_channel(
                    input_value))
            assert component_input_artifact in parent_component_inputs.artifacts, \
                f'component_input_artifact: {component_input_artifact} not found. All inputs: {parent_component_inputs}'
            pipeline_task_spec.inputs.artifacts[
                input_name].component_input_artifact = (
                    component_input_artifact)

        elif isinstance(input_value, for_loop.LoopArgumentVariable):

            component_input_parameter = (
                compiler_utils.additional_input_name_for_pipeline_channel(
                    input_value.loop_argument))
            assert component_input_parameter in parent_component_inputs.parameters, \
                f'component_input_parameter: {component_input_parameter} not found. All inputs: {parent_component_inputs}'
            pipeline_task_spec.inputs.parameters[
                input_name].component_input_parameter = (
                    component_input_parameter)
            pipeline_task_spec.inputs.parameters[
                input_name].parameter_expression_selector = (
                    f'parseJson(string_value)["{input_value.subvar_name}"]')
        elif isinstance(input_value,
                        pipeline_channel.PipelineArtifactChannel) or (
                            isinstance(input_value, dsl.Collected) and
                            input_value.is_artifact_channel):

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
                        compiler_utils.
                        additional_input_name_for_pipeline_channel(input_value))
                    assert component_input_artifact in parent_component_inputs.artifacts, \
                        f'component_input_artifact: {component_input_artifact} not found. All inputs: {parent_component_inputs}'
                    pipeline_task_spec.inputs.artifacts[
                        input_name].component_input_artifact = (
                            component_input_artifact)
            else:
                component_input_artifact = input_value.full_name
                if component_input_artifact not in parent_component_inputs.artifacts:
                    component_input_artifact = (
                        compiler_utils.
                        additional_input_name_for_pipeline_channel(input_value))
                pipeline_task_spec.inputs.artifacts[
                    input_name].component_input_artifact = (
                        component_input_artifact)

        elif isinstance(input_value,
                        pipeline_channel.PipelineParameterChannel) or (
                            isinstance(input_value, dsl.Collected) and
                            not input_value.is_artifact_channel):
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
                        compiler_utils.
                        additional_input_name_for_pipeline_channel(input_value))
                    assert component_input_parameter in parent_component_inputs.parameters, \
                        f'component_input_parameter: {component_input_parameter} not found. All inputs: {parent_component_inputs}'
                    pipeline_task_spec.inputs.parameters[
                        input_name].component_input_parameter = (
                            component_input_parameter)
            else:
                # Value is from pipeline input.
                component_input_parameter = input_value.full_name
                if component_input_parameter not in parent_component_inputs.parameters:
                    component_input_parameter = (
                        compiler_utils.
                        additional_input_name_for_pipeline_channel(input_value))
                pipeline_task_spec.inputs.parameters[
                    input_name].component_input_parameter = (
                        component_input_parameter)

        elif isinstance(input_value, (str, int, float, bool, dict, list)):
            pipeline_channels = (
                pipeline_channel.extract_pipeline_channels_from_any(input_value)
            )
            for channel in pipeline_channels:
                # NOTE: case like this   p3 = print_and_return_str(s='Project = {}'.format(project))
                # triggers this code

                # value contains PipelineChannel placeholders which needs to be
                # replaced. And the input needs to be added to the task spec.

                # Form the name for the compiler injected input, and make sure it
                # doesn't collide with any existing input names.
                additional_input_name = (
                    compiler_utils.additional_input_name_for_pipeline_channel(
                        channel))

                # We don't expect collision to happen because we prefix the name
                # of additional input with 'pipelinechannel--'. But just in case
                # collision did happend, throw a RuntimeError so that we don't
                # get surprise at runtime.
                for existing_input_name, _ in task.inputs.items():
                    if existing_input_name == additional_input_name:
                        raise RuntimeError(
                            f'Name collision between existing input name {existing_input_name} and compiler injected input name {additional_input_name}'
                        )

                additional_input_placeholder = placeholders.InputValuePlaceholder(
                    additional_input_name)._to_string()

                input_value = compiler_utils.recursive_replace_placeholders(
                    input_value, channel.pattern, additional_input_placeholder)

                if channel.task_name:
                    # Value is produced by an upstream task.
                    if channel.task_name in tasks_in_current_dag:
                        # Dependent task within the same DAG.
                        pipeline_task_spec.inputs.parameters[
                            additional_input_name].task_output_parameter.producer_task = (
                                utils.sanitize_task_name(channel.task_name))
                        pipeline_task_spec.inputs.parameters[
                            additional_input_name].task_output_parameter.output_parameter_key = (
                                channel.name)
                    else:
                        # Dependent task not from the same DAG.
                        component_input_parameter = (
                            compiler_utils.
                            additional_input_name_for_pipeline_channel(channel))
                        assert component_input_parameter in parent_component_inputs.parameters, \
                            f'component_input_parameter: {component_input_parameter} not found. All inputs: {parent_component_inputs}'
                        pipeline_task_spec.inputs.parameters[
                            additional_input_name].component_input_parameter = (
                                component_input_parameter)
                else:
                    # Value is from pipeline input. (or loop?)
                    component_input_parameter = channel.full_name
                    if component_input_parameter not in parent_component_inputs.parameters:
                        component_input_parameter = (
                            compiler_utils.
                            additional_input_name_for_pipeline_channel(channel))
                    pipeline_task_spec.inputs.parameters[
                        additional_input_name].component_input_parameter = (
                            component_input_parameter)

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
    return build_component_spec_for_task(
        task=task, is_compiled_component=False, is_exit_task=True)


def build_component_spec_for_task(
    task: pipeline_task.PipelineTask,
    is_compiled_component: bool,
    is_exit_task: bool = False,
) -> pipeline_spec_pb2.ComponentSpec:
    """Builds ComponentSpec for a pipeline task.

    Args:
        task: The task to build a ComponentSpec for.
        is_exit_task: Whether the task is used as exit task in Exit Handler.

    Returns:
        A ComponentSpec object for the task.
    """
    for input_name, input_spec in (task.component_spec.inputs or {}).items():
        if not is_exit_task and type_utils.is_task_final_status_type(
                input_spec.type
        ) and not is_compiled_component and not task._ignore_upstream_failure_tag:
            raise ValueError(
                f'PipelineTaskFinalStatus can only be used in an exit task. Parameter {input_name} of a non exit task has type PipelineTaskFinalStatus.'
            )

    component_spec = _build_component_spec_from_component_spec_structure(
        task.component_spec)
    component_spec.executor_label = utils.sanitize_executor_label(task.name)
    return component_spec


def _build_component_spec_from_component_spec_structure(
    component_spec_struct: structures.ComponentSpec
) -> pipeline_spec_pb2.ComponentSpec:
    """Builds ComponentSpec proto from ComponentSpec structure."""
    component_spec = pipeline_spec_pb2.ComponentSpec()

    for input_name, input_spec in (component_spec_struct.inputs or {}).items():

        # Special handling for PipelineTaskFinalStatus first.
        if type_utils.is_task_final_status_type(input_spec.type):
            component_spec.input_definitions.parameters[
                input_name].parameter_type = pipeline_spec_pb2.ParameterType.TASK_FINAL_STATUS
            component_spec.input_definitions.parameters[
                input_name].is_optional = True
            if input_spec.description:
                component_spec.input_definitions.parameters[
                    input_name].description = input_spec.description

        elif type_utils.is_parameter_type(input_spec.type):
            component_spec.input_definitions.parameters[
                input_name].parameter_type = type_utils.get_parameter_type(
                    input_spec.type)
            if input_spec.optional:
                component_spec.input_definitions.parameters[
                    input_name].is_optional = True
                _fill_in_component_input_default_value(
                    component_spec=component_spec,
                    input_name=input_name,
                    default_value=input_spec.default,
                )
            if input_spec.description:
                component_spec.input_definitions.parameters[
                    input_name].description = input_spec.description

        else:
            component_spec.input_definitions.artifacts[
                input_name].artifact_type.CopyFrom(
                    type_utils.bundled_artifact_to_artifact_proto(
                        input_spec.type))
            component_spec.input_definitions.artifacts[
                input_name].is_artifact_list = input_spec.is_artifact_list
            if input_spec.optional:
                component_spec.input_definitions.artifacts[
                    input_name].is_optional = True
            if input_spec.description:
                component_spec.input_definitions.artifacts[
                    input_name].description = input_spec.description

    for output_name, output_spec in (component_spec_struct.outputs or
                                     {}).items():
        if type_utils.is_parameter_type(output_spec.type):
            component_spec.output_definitions.parameters[
                output_name].parameter_type = type_utils.get_parameter_type(
                    output_spec.type)
            if output_spec.description:
                component_spec.output_definitions.parameters[
                    output_name].description = output_spec.description
        else:
            component_spec.output_definitions.artifacts[
                output_name].artifact_type.CopyFrom(
                    type_utils.bundled_artifact_to_artifact_proto(
                        output_spec.type))
            component_spec.output_definitions.artifacts[
                output_name].is_artifact_list = output_spec.is_artifact_list
            if output_spec.description:
                component_spec.output_definitions.artifacts[
                    output_name].description = output_spec.description

    return component_spec


def connect_single_dag_output(
    component_spec: pipeline_spec_pb2.ComponentSpec,
    output_name: str,
    output_channel: pipeline_channel.PipelineChannel,
) -> None:
    """Connects a DAG output to a subtask output when the subtask output
    contains only one channel (i.e., not OneOfMixin).

    Args:
        component_spec: The component spec to modify its dag outputs.
        output_name: The name of the dag output.
        output_channel: The pipeline channel selected for the dag output.
    """
    if isinstance(output_channel, pipeline_channel.PipelineArtifactChannel):
        if output_name not in component_spec.output_definitions.artifacts:
            raise ValueError(
                f'Pipeline or component output not defined: {output_name}. You may be missing a type annotation.'
            )
        component_spec.dag.outputs.artifacts[
            output_name].artifact_selectors.append(
                pipeline_spec_pb2.DagOutputsSpec.ArtifactSelectorSpec(
                    producer_subtask=output_channel.task_name,
                    output_artifact_key=output_channel.name,
                ))
    elif isinstance(output_channel, pipeline_channel.PipelineParameterChannel):
        if output_name not in component_spec.output_definitions.parameters:
            raise ValueError(
                f'Pipeline or component output not defined: {output_name}. You may be missing a type annotation.'
            )
        component_spec.dag.outputs.parameters[
            output_name].value_from_parameter.producer_subtask = output_channel.task_name
        component_spec.dag.outputs.parameters[
            output_name].value_from_parameter.output_parameter_key = output_channel.name


def connect_oneof_dag_output(
    component_spec: pipeline_spec_pb2.ComponentSpec,
    output_name: str,
    oneof_output: pipeline_channel.OneOfMixin,
) -> None:
    """Connects a output to the OneOf output returned by the DAG's internal
    condition-branches group.

    Args:
        component_spec: The component spec to modify its DAG outputs.
        output_name: The name of the DAG output.
        oneof_output: The OneOfMixin object returned by the pipeline (OneOf in user code).
    """
    if isinstance(oneof_output, pipeline_channel.OneOfArtifact):
        if output_name not in component_spec.output_definitions.artifacts:
            raise ValueError(
                f'Pipeline or component output not defined: {output_name}. You may be missing a type annotation.'
            )
        for channel in oneof_output.channels:
            component_spec.dag.outputs.artifacts[
                output_name].artifact_selectors.append(
                    pipeline_spec_pb2.DagOutputsSpec.ArtifactSelectorSpec(
                        producer_subtask=channel.task_name,
                        output_artifact_key=channel.name,
                    ))
    if isinstance(oneof_output, pipeline_channel.OneOfParameter):
        if output_name not in component_spec.output_definitions.parameters:
            raise ValueError(
                f'Pipeline or component output not defined: {output_name}. You may be missing a type annotation.'
            )
        for channel in oneof_output.channels:
            component_spec.dag.outputs.parameters[
                output_name].value_from_oneof.parameter_selectors.append(
                    pipeline_spec_pb2.DagOutputsSpec.ParameterSelectorSpec(
                        producer_subtask=channel.task_name,
                        output_parameter_key=channel.name,
                    ))


def _build_dag_outputs(
    component_spec: pipeline_spec_pb2.ComponentSpec,
    dag_outputs: Dict[str, pipeline_channel.PipelineChannel],
) -> None:
    """Connects the DAG's outputs to a TaskGroup's ComponentSpec and validates
    it is present in the component interface.

    Args:
        component_spec: The ComponentSpec.
        dag_outputs: Dictionary of output key to output channel.
    """
    for output_name, output_channel in dag_outputs.items():
        if not isinstance(output_channel, pipeline_channel.PipelineChannel):
            raise ValueError(
                f"Got unknown pipeline output '{output_name}' of type {output_channel}."
            )
        connect_single_dag_output(component_spec, output_name, output_channel)

    validate_dag_outputs(component_spec)


def validate_dag_outputs(
        component_spec: pipeline_spec_pb2.ComponentSpec) -> None:
    """Validates the DAG's ComponentSpec specifies the source task for all of
    its ComponentSpec inputs (input_definitions) and outputs
    (output_definitions)."""
    for output_name in component_spec.output_definitions.artifacts:
        if output_name not in component_spec.dag.outputs.artifacts:
            raise ValueError(f'Missing pipeline output: {output_name}.')
    for output_name in component_spec.output_definitions.parameters:
        if output_name not in component_spec.dag.outputs.parameters:
            raise ValueError(f'Missing pipeline output: {output_name}.')


def build_oneof_dag_outputs(
    component_spec: pipeline_spec_pb2.ComponentSpec,
    oneof_outputs: Dict[str, pipeline_channel.OneOfMixin],
) -> None:
    """Connects the DAG's OneOf outputs to a TaskGroup's ComponentSpec and
    validates it is present in the component interface.

    Args:
        component_spec: The ComponentSpec.
        oneof_outputs: Dictionary of output key to OneOf output channel.
    """
    for output_name, oneof_output in oneof_outputs.items():
        for channel in oneof_output.channels:
            if not isinstance(channel, pipeline_channel.PipelineChannel):
                raise ValueError(
                    f"Got unknown pipeline output '{output_name}' of type {type(channel)}."
                )
        connect_oneof_dag_output(
            component_spec,
            output_name,
            oneof_output,
        )
    validate_dag_outputs(component_spec)


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
                  pipeline_channel.PipelineChannel):
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

    def convert_to_placeholder(input_value: str) -> str:
        """Checks if input is a pipeline channel and if so, converts to
        compiler injected input name."""
        pipeline_channels = (
            pipeline_channel.extract_pipeline_channels_from_any(input_value))
        if pipeline_channels:
            assert len(pipeline_channels) == 1
            channel = pipeline_channels[0]
            additional_input_name = (
                compiler_utils.additional_input_name_for_pipeline_channel(
                    channel))
            additional_input_placeholder = placeholders.InputValuePlaceholder(
                additional_input_name)._to_string()
            input_value = input_value.replace(channel.pattern,
                                              additional_input_placeholder)
        return input_value

    container_spec = (
        pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec(
            image=convert_to_placeholder(task.container_spec.image),
            command=task.container_spec.command,
            args=task.container_spec.args,
            env=[
                pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec
                .EnvVar(name=name, value=value)
                for name, value in (task.container_spec.env or {}).items()
            ]))

    # All the fields with the resource_ prefix are newer fields to support pipeline input parameters. The below code
    # will check if the value is a placeholder and if not, it will also set the value on the old deprecated fields
    # without the resource_ prefix to work on older KFP installations.
    if task.container_spec.resources is not None:
        if task.container_spec.resources.cpu_request is not None:
            placeholder = convert_to_placeholder(
                task.container_spec.resources.cpu_request)
            container_spec.resources.resource_cpu_request = placeholder

            if task.container_spec.resources.cpu_request == placeholder:
                container_spec.resources.cpu_request = compiler_utils._cpu_to_float(
                    task.container_spec.resources.cpu_request)
        if task.container_spec.resources.cpu_limit is not None:
            placeholder = convert_to_placeholder(
                task.container_spec.resources.cpu_limit)
            container_spec.resources.resource_cpu_limit = placeholder

            if task.container_spec.resources.cpu_limit == placeholder:
                container_spec.resources.cpu_limit = compiler_utils._cpu_to_float(
                    task.container_spec.resources.cpu_limit)
        if task.container_spec.resources.memory_request is not None:
            placeholder = convert_to_placeholder(
                task.container_spec.resources.memory_request)
            container_spec.resources.resource_memory_request = placeholder

            if task.container_spec.resources.memory_request == placeholder:
                container_spec.resources.memory_request = compiler_utils._memory_to_float(
                    task.container_spec.resources.memory_request)
        if task.container_spec.resources.memory_limit is not None:
            placeholder = convert_to_placeholder(
                task.container_spec.resources.memory_limit)
            container_spec.resources.resource_memory_limit = placeholder

            if task.container_spec.resources.memory_limit == placeholder:
                container_spec.resources.memory_limit = compiler_utils._memory_to_float(
                    task.container_spec.resources.memory_limit)
        if task.container_spec.resources.accelerator_count is not None:
            ac_type = None
            ac_type_placholder = convert_to_placeholder(
                task.container_spec.resources.accelerator_type)
            if task.container_spec.resources.accelerator_type == ac_type_placholder:
                ac_type = task.container_spec.resources.accelerator_type

            ac_count = None
            ac_count_placeholder = convert_to_placeholder(
                task.container_spec.resources.accelerator_count)
            if task.container_spec.resources.accelerator_count == ac_count_placeholder:
                ac_count = int(task.container_spec.resources.accelerator_count)

            container_spec.resources.accelerator.CopyFrom(
                pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec
                .ResourceSpec.AcceleratorConfig(
                    resource_type=ac_type_placholder,
                    resource_count=ac_count_placeholder,
                    type=ac_type,
                    count=ac_count,
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
                to_protobuf_value(default_value))
    elif pipeline_spec_pb2.ParameterType.LIST == parameter_type:
        component_spec.input_definitions.parameters[
            input_name].default_value.CopyFrom(
                to_protobuf_value(default_value))


def build_component_spec_for_group(
    input_pipeline_channels: List[pipeline_channel.PipelineChannel],
    output_pipeline_channels: Dict[str, pipeline_channel.PipelineChannel],
) -> pipeline_spec_pb2.ComponentSpec:
    """Builds ComponentSpec for a TasksGroup.

    Args:
        group: The group to build a ComponentSpec for.
        pipeline_channels: The list of pipeline channels referenced by the group.

    Returns:
        A PipelineTaskSpec object representing the loop group.
    """
    component_spec = pipeline_spec_pb2.ComponentSpec()

    for channel in input_pipeline_channels:
        input_name = compiler_utils.additional_input_name_for_pipeline_channel(
            channel)

        if isinstance(channel, (pipeline_channel.PipelineArtifactChannel,
                                for_loop.LoopArtifactArgument)):
            component_spec.input_definitions.artifacts[
                input_name].artifact_type.CopyFrom(
                    type_utils.bundled_artifact_to_artifact_proto(
                        channel.channel_type))
            component_spec.input_definitions.artifacts[
                input_name].is_artifact_list = channel.is_artifact_list
        elif isinstance(channel,
                        (pipeline_channel.PipelineParameterChannel,
                         for_loop.LoopParameterArgument,
                         for_loop.LoopArgumentVariable, dsl.Collected)):
            component_spec.input_definitions.parameters[
                input_name].parameter_type = type_utils.get_parameter_type(
                    channel.channel_type)
        else:
            raise TypeError(
                f'Expected PipelineParameterChannel, PipelineArtifactChannel, LoopParameterArgument, LoopArtifactArgument, LoopArgumentVariable, or Collected, got {type(channel)}.'
            )

    for output_name, output in output_pipeline_channels.items():
        if isinstance(output, pipeline_channel.PipelineArtifactChannel):
            component_spec.output_definitions.artifacts[
                output_name].artifact_type.CopyFrom(
                    type_utils.bundled_artifact_to_artifact_proto(
                        output.channel_type))
            component_spec.output_definitions.artifacts[
                output_name].is_artifact_list = output.is_artifact_list
        else:
            component_spec.output_definitions.parameters[
                output_name].parameter_type = type_utils.get_parameter_type(
                    output.channel_type)

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
        input_parameter_name = compiler_utils.additional_input_name_for_pipeline_channel(
            loop_items_channel)
        loop_argument_item_name = compiler_utils.additional_input_name_for_pipeline_channel(
            group.loop_argument.full_name)

        loop_arguments_item = f'{input_parameter_name}-{for_loop.LOOP_ITEM_NAME_BASE}'
        assert loop_arguments_item == loop_argument_item_name

        if isinstance(group.loop_argument, for_loop.LoopParameterArgument):
            pipeline_task_spec.parameter_iterator.items.input_parameter = (
                input_parameter_name)
            pipeline_task_spec.parameter_iterator.item_input = (
                loop_argument_item_name)

            _pop_input_from_task_spec(
                task_spec=pipeline_task_spec,
                input_name=pipeline_task_spec.parameter_iterator.item_input)

        elif isinstance(group.loop_argument, for_loop.LoopArtifactArgument):
            input_artifact_name = compiler_utils.additional_input_name_for_pipeline_channel(
                loop_items_channel)

            pipeline_task_spec.artifact_iterator.items.input_artifact = input_artifact_name
            pipeline_task_spec.artifact_iterator.item_input = (
                loop_argument_item_name)

            _pop_input_from_task_spec(
                task_spec=pipeline_task_spec,
                input_name=pipeline_task_spec.artifact_iterator.item_input)
        else:
            raise TypeError(
                f'Expected LoopParameterArgument or LoopArtifactArgument, got {type(group.loop_argument)}.'
            )

        # If the loop items itself is a loop arguments variable, handle the
        # subvar name.
        if isinstance(loop_items_channel, for_loop.LoopArgumentVariable):
            pipeline_task_spec.inputs.parameters[
                input_parameter_name].parameter_expression_selector = (
                    f'parseJson(string_value)["{loop_items_channel.subvar_name}"]'
                )
            pipeline_task_spec.inputs.parameters[
                input_parameter_name].component_input_parameter = (
                    compiler_utils.additional_input_name_for_pipeline_channel(
                        loop_items_channel.loop_argument))

    else:
        input_parameter_name = compiler_utils.additional_input_name_for_pipeline_channel(
            group.loop_argument)
        raw_values = group.loop_argument.items_or_pipeline_channel

        pipeline_task_spec.parameter_iterator.items.raw = json.dumps(
            raw_values, sort_keys=True)
        pipeline_task_spec.parameter_iterator.item_input = (
            input_parameter_name)

        _pop_input_from_task_spec(
            task_spec=pipeline_task_spec,
            input_name=pipeline_task_spec.parameter_iterator.item_input)

    if (group.parallelism_limit > 0):
        pipeline_task_spec.iterator_policy.parallelism_limit = (
            group.parallelism_limit)


def _binary_operations_to_cel_conjunctive(
        operations: List[pipeline_channel.ConditionOperation]) -> str:
    """Converts a list of ConditionOperation to a CEL string with placeholders.
    Each ConditionOperation will be joined the others via the conjunctive (&&).

    Args:
        operations: The binary operations to convert to convert and join.

    Returns:
        The binary operations as a CEL string.
    """
    operands = [
        _single_binary_operation_to_cel_condition(operation=bin_op)
        for bin_op in operations
    ]
    return ' && '.join(operands)


def _single_binary_operation_to_cel_condition(
        operation: pipeline_channel.ConditionOperation) -> str:
    """Converts a ConditionOperation to a CEL string with placeholders.

    Args:
        operation: The binary operation to convert to a string.

    Returns:
        The binary operation as a CEL string.
    """
    left_operand = operation.left_operand
    right_operand = operation.right_operand

    # cannot make comparisons involving particular types
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
                input_name = compiler_utils.additional_input_name_for_pipeline_channel(
                    value_or_reference)
                raise ValueError(
                    f'Conditional requires primitive parameter values for comparison. Found input "{input_name}" of type {value_or_reference.channel_type} in pipeline definition instead.'
                )

    # ensure the types compared are the same or compatible
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
                f'Types: {parameter_types} [{left_operand} {right_operand}]'
            assert pipeline_spec_pb2.ParameterType.NUMBER_INTEGER in parameter_types, \
                f'Types: {parameter_types} [{left_operand} {right_operand}]'
            canonical_parameter_type = pipeline_spec_pb2.ParameterType.NUMBER_DOUBLE
    elif len(parameter_types) == 1:  # Both operands are the same type.
        canonical_parameter_type = parameter_types.pop()
    else:
        # Probably shouldn't happen.
        raise ValueError(
            f'Unable to determine operand types for "{left_operand}" and "{right_operand}"'
        )

    operand_values = []
    for value_or_reference in [left_operand, right_operand]:
        if isinstance(value_or_reference, pipeline_channel.PipelineChannel):
            input_name = compiler_utils.additional_input_name_for_pipeline_channel(
                value_or_reference)
            operand_value = f"inputs.parameter_values['{input_name}']"
            parameter_type = type_utils.get_parameter_type(
                value_or_reference.channel_type)
            if parameter_type == pipeline_spec_pb2.ParameterType.NUMBER_INTEGER:
                operand_value = f'int({operand_value})'
        elif isinstance(value_or_reference, str):
            operand_value = f"'{value_or_reference}'"
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
                operand_value = f"'{operand_value}'"
            elif canonical_parameter_type == pipeline_spec_pb2.ParameterType.BOOLEAN:
                assert parameter_type in [
                    pipeline_spec_pb2.ParameterType.NUMBER_INTEGER,
                    pipeline_spec_pb2.ParameterType.NUMBER_DOUBLE,
                ]
                operand_value = 'true' if int(operand_value) == 0 else 'false'
            else:
                assert canonical_parameter_type == pipeline_spec_pb2.ParameterType.NUMBER_DOUBLE
                assert parameter_type == pipeline_spec_pb2.ParameterType.NUMBER_INTEGER
                operand_value = f'double({operand_value})'

        operand_values.append(operand_value)

    left_operand_value, right_operand_value = tuple(operand_values)

    condition_string = (
        f'{left_operand_value} {operation.operator} {right_operand_value}')

    return f'!({condition_string})' if operation.negate else condition_string


def _update_task_spec_for_condition_group(
    group: tasks_group._ConditionBase,
    pipeline_task_spec: pipeline_spec_pb2.PipelineTaskSpec,
) -> None:
    """Updates PipelineTaskSpec for condition group.

    Args:
        group: The condition group to update task spec for.
        pipeline_task_spec: The pipeline task spec to update in place.
    """
    condition = _binary_operations_to_cel_conjunctive(group.conditions)
    pipeline_task_spec.trigger_policy.CopyFrom(
        pipeline_spec_pb2.PipelineTaskSpec.TriggerPolicy(condition=condition))


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

        input_name = compiler_utils.additional_input_name_for_pipeline_channel(
            channel)

        channel_name = channel.name
        if subvar_name:
            pipeline_task_spec.inputs.parameters[
                input_name].parameter_expression_selector = (
                    f'parseJson(string_value)["{subvar_name}"]')
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
        elif channel.task_name and channel.task_name in tasks_in_current_dag:
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
                    compiler_utils.additional_input_name_for_pipeline_channel(
                        channel_full_name))

    if isinstance(group, tasks_group.ParallelFor):
        _update_task_spec_for_loop_group(
            group=group,
            pipeline_task_spec=pipeline_task_spec,
        )
    elif isinstance(group, tasks_group._ConditionBase):
        _update_task_spec_for_condition_group(
            group=group,
            pipeline_task_spec=pipeline_task_spec,
        )

    return pipeline_task_spec


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
            raise ValueError(
                f'Pipeline parameter {input_name} does not match any known pipeline input.'
            )

    return pipeline_spec


def build_spec_by_group(
    pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    deployment_config: pipeline_spec_pb2.PipelineDeploymentConfig,
    group: tasks_group.TasksGroup,
    inputs: Mapping[str, List[Tuple[pipeline_channel.PipelineChannel, str]]],
    outputs: DefaultDict[str, Dict[str, pipeline_channel.PipelineChannel]],
    dependencies: Dict[str, List[compiler_utils.GroupOrTaskType]],
    rootgroup_name: str,
    task_name_to_parent_groups: Mapping[str,
                                        List[compiler_utils.GroupOrTaskType]],
    group_name_to_parent_groups: Mapping[str, List[tasks_group.TasksGroup]],
    name_to_for_loop_group: Mapping[str, tasks_group.ParallelFor],
    platform_spec: pipeline_spec_pb2.PlatformSpec,
    is_compiled_component: bool,
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

        subgroup_input_channels = [
            channel for channel, _ in inputs.get(subgroup.name, [])
        ]
        subgroup_output_channels = outputs.get(subgroup.name, {})

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
                task=subgroup,
                is_compiled_component=is_compiled_component,
            )
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
                single_task_platform_spec = platform_config_to_platform_spec(
                    subgroup.platform_config,
                    executor_label,
                )
                merge_platform_specs(
                    platform_spec,
                    single_task_platform_spec,
                )

            elif subgroup.importer_spec is not None:
                subgroup_importer_spec = build_importer_spec_for_task(
                    task=subgroup)
                deployment_config.executors[executor_label].importer.CopyFrom(
                    subgroup_importer_spec)
            elif subgroup.pipeline_spec is not None:
                if subgroup.platform_config:
                    raise ValueError(
                        'Platform-specific features can only be set on primitive components. Found platform-specific feature set on a pipeline.'
                    )
                sub_pipeline_spec, platform_spec = merge_deployment_spec_and_component_spec(
                    main_pipeline_spec=pipeline_spec,
                    main_deployment_config=deployment_config,
                    sub_pipeline_spec=subgroup.pipeline_spec,
                    main_platform_spec=platform_spec,
                    sub_platform_spec=subgroup.platform_spec,
                )
                subgroup_component_spec = sub_pipeline_spec.root
            else:
                raise RuntimeError
        elif isinstance(subgroup, tasks_group.ParallelFor):

            # "Punch the hole", adding additional inputs (other than loop
            # arguments which will be handled separately) needed by its
            # subgroups or tasks.
            loop_subgroup_channels = []

            for channel in subgroup_input_channels:
                # Skip 'withItems' loop arguments if it's from an inner loop.
                if isinstance(channel, (
                        for_loop.LoopParameterArgument,
                        for_loop.LoopArtifactArgument,
                        for_loop.LoopArgumentVariable,
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
                input_pipeline_channels=loop_subgroup_channels,
                output_pipeline_channels=subgroup_output_channels,
            )

            subgroup_task_spec = build_task_spec_for_group(
                group=subgroup,
                pipeline_channels=loop_subgroup_channels,
                tasks_in_current_dag=tasks_in_current_dag,
                is_parent_component_root=is_parent_component_root,
            )

            _build_dag_outputs(subgroup_component_spec,
                               subgroup_output_channels)

        elif isinstance(subgroup, tasks_group._ConditionBase):

            # "Punch the hole", adding inputs needed by its subgroups or
            # tasks.
            condition_subgroup_channels = list(subgroup_input_channels)

            compiler_utils.get_channels_from_condition(
                subgroup.conditions, condition_subgroup_channels)

            subgroup_component_spec = build_component_spec_for_group(
                input_pipeline_channels=condition_subgroup_channels,
                output_pipeline_channels=subgroup_output_channels,
            )

            subgroup_task_spec = build_task_spec_for_group(
                group=subgroup,
                pipeline_channels=condition_subgroup_channels,
                tasks_in_current_dag=tasks_in_current_dag,
                is_parent_component_root=is_parent_component_root,
            )

            _build_dag_outputs(subgroup_component_spec,
                               subgroup_output_channels)

        elif isinstance(subgroup, tasks_group.ExitHandler):

            subgroup_component_spec = build_component_spec_for_group(
                input_pipeline_channels=subgroup_input_channels,
                output_pipeline_channels={},
            )

            subgroup_task_spec = build_task_spec_for_group(
                group=subgroup,
                pipeline_channels=subgroup_input_channels,
                tasks_in_current_dag=tasks_in_current_dag,
                is_parent_component_root=is_parent_component_root,
            )

        # handles the conditional group wrapping only
        elif isinstance(subgroup, tasks_group.ConditionBranches):
            subgroup_component_spec = build_component_spec_for_group(
                input_pipeline_channels=subgroup_input_channels,
                output_pipeline_channels=subgroup_output_channels,
            )

            subgroup_task_spec = build_task_spec_for_group(
                group=subgroup,
                pipeline_channels=subgroup_input_channels,
                tasks_in_current_dag=tasks_in_current_dag,
                is_parent_component_root=is_parent_component_root,
            )
            # oneof is the only type of output a ConditionBranches group can have
            build_oneof_dag_outputs(subgroup_component_spec,
                                    subgroup_output_channels)

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

        # Modify the task inputs for PipelineTaskFinalStatus if ignore_upstream_failure is used
        # Must be done after dependencies are added
        if isinstance(subgroup, pipeline_task.PipelineTask):
            modify_task_for_ignore_upstream_failure(
                task=subgroup, pipeline_task_spec=subgroup_task_spec)
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


def modify_task_for_ignore_upstream_failure(
    task: pipeline_task.PipelineTask,
    pipeline_task_spec: pipeline_spec_pb2.PipelineTaskSpec,
):
    if task._ignore_upstream_failure_tag:
        pipeline_task_spec.trigger_policy.strategy = (
            pipeline_spec_pb2.PipelineTaskSpec.TriggerPolicy.TriggerStrategy
            .ALL_UPSTREAM_TASKS_COMPLETED)

        for input_name, input_spec in (task.component_spec.inputs or
                                       {}).items():
            if not type_utils.is_task_final_status_type(input_spec.type):
                continue

            if len(pipeline_task_spec.dependent_tasks) == 0:
                if task.parent_task_group.group_type == tasks_group.TasksGroupType.PIPELINE:
                    raise compiler_utils.InvalidTopologyException(
                        f"Tasks that use '.ignore_upstream_failure()' and 'PipelineTaskFinalStatus' must have exactly one dependent upstream task. Got task '{pipeline_task_spec.task_info.name} with no upstream dependencies."
                    )
                else:
                    # TODO: permit additional PipelineTaskFinalStatus flexibility by "punching the hole" through Condition and ParallelFor groups
                    raise compiler_utils.InvalidTopologyException(
                        f"Tasks that use '.ignore_upstream_failure()' and 'PipelineTaskFinalStatus' must have exactly one dependent upstream task within the same control flow scope. Got task '{pipeline_task_spec.task_info.name}' beneath a 'dsl.{group_type_to_dsl_class[task.parent_task_group.group_type].__name__}' that does not also contain the upstream dependent task."
                    )

            # if >1 dependent task, ambiguous to which upstream task the PipelineTaskFinalStatus should correspond, since there is no ExitHandler that bundles these together
            if len(pipeline_task_spec.dependent_tasks) > 1:
                raise compiler_utils.InvalidTopologyException(
                    f"Tasks that use '.ignore_upstream_failure()' and 'PipelineTaskFinalStatus' must have exactly one dependent upstream task. Got {len(pipeline_task_spec.dependent_tasks)} dependent tasks: {pipeline_task_spec.dependent_tasks}."
                )

            pipeline_task_spec.inputs.parameters[
                input_name].task_final_status.producer_task = pipeline_task_spec.dependent_tasks[
                    0]


def platform_config_to_platform_spec(
    platform_config: dict,
    executor_label: str,
) -> pipeline_spec_pb2.PlatformSpec:
    """Converts a single task's pipeline_task.platform_config dictionary to a
    PlatformSpec message using the executor_label for the task."""
    platform_spec_msg = pipeline_spec_pb2.PlatformSpec()
    json_format.ParseDict(
        {
            'platforms': {
                platform_key: {
                    'deployment_spec': {
                        'executors': {
                            executor_label: config
                        }
                    }
                } for platform_key, config in platform_config.items()
            }
        }, platform_spec_msg)
    return platform_spec_msg


def merge_platform_specs(
    main_msg: pipeline_spec_pb2.PlatformSpec,
    sub_msg: pipeline_spec_pb2.PlatformSpec,
) -> None:
    """Merges a sub_msg PlatformSpec into the main_msg PlatformSpec, leaving
    the sub_msg unchanged."""
    for platform_key, single_platform_spec in sub_msg.platforms.items():
        merge_platform_deployment_config(
            main_msg.platforms[platform_key].deployment_spec,
            single_platform_spec.deployment_spec,
        )


def merge_platform_deployment_config(
    main_msg: pipeline_spec_pb2.PlatformDeploymentConfig,
    sub_msg: pipeline_spec_pb2.PlatformDeploymentConfig,
) -> None:
    """Merges a sub_msg PlatformDeploymentConfig into the main_msg
    PlatformDeploymentConfig, leaving the sub_msg unchanged."""
    for executor_label, addtl_body in sub_msg.executors.items():
        body = main_msg.executors.get(executor_label, struct_pb2.Struct())
        body.update(addtl_body)
        main_msg.executors[executor_label].CopyFrom(body)


def build_exit_handler_groups_recursively(
    parent_group: tasks_group.TasksGroup,
    pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    deployment_config: pipeline_spec_pb2.PipelineDeploymentConfig,
    platform_spec: pipeline_spec_pb2.PlatformSpec,
) -> None:
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
                single_task_platform_spec = platform_config_to_platform_spec(
                    exit_task.platform_config,
                    executor_label,
                )
                merge_platform_specs(
                    platform_spec,
                    single_task_platform_spec,
                )
            elif exit_task.pipeline_spec is not None:
                exit_task_pipeline_spec, updated_platform_spec = merge_deployment_spec_and_component_spec(
                    main_pipeline_spec=pipeline_spec,
                    main_deployment_config=deployment_config,
                    sub_pipeline_spec=exit_task.pipeline_spec,
                    main_platform_spec=platform_spec,
                    sub_platform_spec=exit_task.platform_spec)
                exit_task_component_spec = exit_task_pipeline_spec.root
                # assign the new PlatformSpec data to the existing main PlatformSpec object
                platform_spec.CopyFrom(updated_platform_spec)
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
            deployment_config=deployment_config,
            platform_spec=platform_spec,
        )


def merge_deployment_spec_and_component_spec(
    main_pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    main_deployment_config: pipeline_spec_pb2.PipelineDeploymentConfig,
    sub_pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    main_platform_spec: pipeline_spec_pb2.PlatformSpec,
    sub_platform_spec: pipeline_spec_pb2.PlatformSpec,
) -> Tuple[pipeline_spec_pb2.PipelineSpec, pipeline_spec_pb2.PlatformSpec]:
    """Merges deployment spec and component spec from a sub pipeline spec into
    the main spec.

    Also merges sub_platform_spec into main_platform_spec, with updated executor labels.

    We need to make sure that we keep the original sub PipelineSpec and PlatformSpec
    unchanged--in case the pipeline is reused (instantiated) multiple times,
    the "template" should not carry any "signs of usage".


    Args:
        main_pipeline_spec: The main PipelineSpec to merge into.
        main_deployment_config: The main PipelineDeploymentConfig to merge into.
        sub_pipeline_spec: The PipelineSpec of an inner pipeline whose
            deployment specs and component specs need to be copied into the main
            specs.
        main_platform_spec: The PlatformSpec corresponding to main_pipeline_spec.
        sub_platform_spec: The PlatformSpec corresponding to sub_pipeline_spec.

    Returns:
        The possibly modified version of sub_pipeline_spec and the possibly modified version of the the main_platform_spec. The sub_pipeline_spec is "folded" into the outer pipeline, whereas the main_platform_spec is updated to contain the sub_pipeline_spec's configuration.
    """
    # Make a copy of the messages of the inner pipeline so that the "template" remains
    # unchanged and works even the pipeline is reused multiple times.
    sub_pipeline_spec_copy = pipeline_spec_pb2.PipelineSpec()
    sub_pipeline_spec_copy.CopyFrom(sub_pipeline_spec)
    sub_platform_spec_copy = pipeline_spec_pb2.PlatformSpec()
    sub_platform_spec_copy.CopyFrom(sub_platform_spec)

    _merge_deployment_spec(
        main_deployment_config=main_deployment_config,
        sub_pipeline_spec=sub_pipeline_spec_copy,
        main_platform_spec=main_platform_spec,
        sub_platform_spec=sub_platform_spec_copy,
    )
    _merge_component_spec(
        main_pipeline_spec=main_pipeline_spec,
        sub_pipeline_spec=sub_pipeline_spec_copy,
    )
    return sub_pipeline_spec_copy, main_platform_spec


def _merge_deployment_spec(
    main_deployment_config: pipeline_spec_pb2.PipelineDeploymentConfig,
    sub_pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    main_platform_spec: pipeline_spec_pb2.PlatformSpec,
    sub_platform_spec: pipeline_spec_pb2.PlatformSpec,
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

    def _rename_platform_config_executor_labels(
        sub_platform_spec: pipeline_spec_pb2.PlatformSpec,
        old_executor_label: str,
        new_executor_label: str,
    ) -> None:
        # make a copy so that map size doesn't change during iteration
        sub_platform_spec_copy = copy.deepcopy(sub_platform_spec)
        for platform_key, platform_config in sub_platform_spec_copy.platforms.items(
        ):
            for cur_exec_label, task_config in platform_config.deployment_spec.executors.items(
            ):
                if cur_exec_label == old_executor_label:
                    del sub_platform_spec.platforms[
                        platform_key].deployment_spec.executors[
                            old_executor_label]
                    sub_platform_spec.platforms[
                        platform_key].deployment_spec.executors[
                            new_executor_label].CopyFrom(task_config)

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
            _rename_platform_config_executor_labels(
                sub_platform_spec,
                old_executor_label=old_executor_label,
                new_executor_label=executor_label)

        main_deployment_config.executors[executor_label].CopyFrom(executor_spec)
        merge_platform_specs(
            main_msg=main_platform_spec, sub_msg=sub_platform_spec)


def _merge_component_spec(
    main_pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    sub_pipeline_spec: pipeline_spec_pb2.PipelineSpec,
) -> None:
    """Merges component spec from a sub pipeline spec into the main config.

    During the merge we need to ensure all component specs have a unique component
    name, that means we might need to update the `component_ref` referenced from
    task specs in sub_pipeline_spec.

    Uniqueness is determined by existing component names in main_pipeline_spec
    and sub_pipeline_spec.

    Renaming is first done in place, specifically in a LIFO order. This is to avoid
    a rename causing a name collision with a later rename. Then, the actual merge
    of component specs is done in a second pass. This ensures all component specs
    are in the final state at the time of merging.

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

    old_name_to_new_name = {}
    existing_main_comp_names = list(main_pipeline_spec.components.keys())
    for component_name, _ in sub_pipeline_spec.components.items():
        old_component_name = component_name
        current_comp_name_collection = [
            key for pair in old_name_to_new_name.items() for key in pair
        ]
        new_component_name = utils.make_name_unique_by_adding_index(
            name=component_name,
            collection=existing_main_comp_names + current_comp_name_collection,
            delimiter='-')
        old_name_to_new_name[old_component_name] = new_component_name

    ordered_names = enumerate(old_name_to_new_name.items())
    lifo_ordered_names = sorted(ordered_names, key=lambda x: x[0], reverse=True)
    for _, (old_component_name, new_component_name) in lifo_ordered_names:
        if new_component_name != old_component_name:
            _rename_component_refs(
                pipeline_spec=sub_pipeline_spec,
                old_component_ref=old_component_name,
                new_component_ref=new_component_name)

    for old_component_name, component_spec in sub_pipeline_spec.components.items(
    ):
        main_pipeline_spec.components[
            old_name_to_new_name[old_component_name]].CopyFrom(component_spec)


def validate_pipeline_outputs_dict(
        pipeline_outputs_dict: Dict[str, pipeline_channel.PipelineChannel]):
    for channel in pipeline_outputs_dict.values():
        if isinstance(channel, dsl.Collected):
            # this validation doesn't apply to Collected
            continue

        elif isinstance(channel, pipeline_channel.OneOfMixin):
            if channel.condition_branches_group.parent_task_group.group_type != tasks_group.TasksGroupType.PIPELINE:
                raise compiler_utils.InvalidTopologyException(
                    f'Pipeline outputs may only be returned from the top level of the pipeline function scope. Got pipeline output dsl.{pipeline_channel.OneOf.__name__} from within the control flow group dsl.{channel.condition_branches_group.parent_task_group.__class__.__name__}.'
                )

        elif isinstance(channel, pipeline_channel.PipelineChannel):
            if channel.task.parent_task_group.group_type != tasks_group.TasksGroupType.PIPELINE:
                raise compiler_utils.InvalidTopologyException(
                    f'Pipeline outputs may only be returned from the top level of the pipeline function scope. Got pipeline output from within the control flow group dsl.{channel.task.parent_task_group.__class__.__name__}.'
                )
        else:
            raise ValueError(
                f'Got unknown pipeline output, {channel}, of type {type(channel)}.'
            )


def create_pipeline_spec(
    pipeline: pipeline_context.Pipeline,
    component_spec: structures.ComponentSpec,
    pipeline_outputs: Optional[Any] = None,
    pipeline_config: pipeline_config.PipelineConfig = None,
) -> Tuple[pipeline_spec_pb2.PipelineSpec, pipeline_spec_pb2.PlatformSpec]:
    """Creates a pipeline spec object.

    Args:
        pipeline: The instantiated pipeline object.
        component_spec: The component spec structures.
        pipeline_outputs: The pipeline outputs via return.
        pipeline_config: The pipeline config object.

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

    # TODO: add validation of returned outputs -- it's possible to return
    # an output from a task in a condition group, for example, which isn't
    # caught until submission time using Vertex SDK client
    pipeline_outputs_dict = convert_pipeline_outputs_to_dict(pipeline_outputs)
    validate_pipeline_outputs_dict(pipeline_outputs_dict)

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
    outputs, modified_pipeline_outputs_dict = compiler_utils.get_outputs_for_all_groups(
        pipeline=pipeline,
        task_name_to_parent_groups=task_name_to_parent_groups,
        group_name_to_parent_groups=group_name_to_parent_groups,
        all_groups=all_groups,
        pipeline_outputs_dict=pipeline_outputs_dict)
    dependencies = compiler_utils.get_dependencies(
        pipeline=pipeline,
        task_name_to_parent_groups=task_name_to_parent_groups,
        group_name_to_parent_groups=group_name_to_parent_groups,
        group_name_to_group=group_name_to_group,
        condition_channels=condition_channels,
    )

    platform_spec = pipeline_spec_pb2.PlatformSpec()
    if pipeline_config is not None:
        _merge_pipeline_config(
            pipelineConfig=pipeline_config, platformSpec=platform_spec)

    for group in all_groups:
        build_spec_by_group(
            pipeline_spec=pipeline_spec,
            deployment_config=deployment_config,
            group=group,
            inputs=inputs,
            outputs=outputs,
            dependencies=dependencies,
            rootgroup_name=root_group.name,
            task_name_to_parent_groups=task_name_to_parent_groups,
            group_name_to_parent_groups=group_name_to_parent_groups,
            name_to_for_loop_group=name_to_for_loop_group,
            platform_spec=platform_spec,
            is_compiled_component=False,
        )

    build_exit_handler_groups_recursively(
        parent_group=root_group,
        pipeline_spec=pipeline_spec,
        deployment_config=deployment_config,
        platform_spec=platform_spec,
    )

    _build_dag_outputs(
        component_spec=pipeline_spec.root,
        dag_outputs=modified_pipeline_outputs_dict,
    )
    # call _build_dag_outputs first to verify the presence of an output annotation
    # at all, then validate that the annotation is correct with _validate_dag_output_types
    _validate_dag_output_types(
        dag_outputs=modified_pipeline_outputs_dict,
        structures_component_spec=component_spec)

    return pipeline_spec, platform_spec


def _validate_dag_output_types(
        dag_outputs: Dict[str, pipeline_channel.PipelineChannel],
        structures_component_spec: structures.ComponentSpec) -> None:
    for output_name, output_channel in dag_outputs.items():
        output_spec = structures_component_spec.outputs[output_name]
        output_name = '' if len(dag_outputs) == 1 else f'{output_name!r} '
        error_message_prefix = f'Incompatible return type provided for output {output_name}of pipeline {structures_component_spec.name!r}. '
        type_utils.verify_type_compatibility(
            output_channel,
            output_spec,
            error_message_prefix,
            checks_input=False,
        )


def convert_pipeline_outputs_to_dict(
    pipeline_outputs: Union[pipeline_channel.PipelineChannel, typing.NamedTuple,
                            None]
) -> Dict[str, pipeline_channel.PipelineChannel]:
    """Converts the outputs from a pipeline function into a dictionary of
    output name to PipelineChannel."""
    if pipeline_outputs is None:
        return {}
    elif isinstance(pipeline_outputs, dict):
        # This condition is required to support the case where a nested pipeline
        # returns a namedtuple but its output is converted into a dict by
        # earlier invocations of this function (a few lines down).
        return pipeline_outputs
    elif isinstance(pipeline_outputs, pipeline_channel.PipelineChannel):
        return {component_factory.SINGLE_OUTPUT_NAME: pipeline_outputs}
    elif isinstance(pipeline_outputs, tuple) and hasattr(
            pipeline_outputs, '_asdict'):
        return dict(pipeline_outputs._asdict())
    else:
        raise ValueError(
            f'Got unknown pipeline output, {pipeline_outputs}, of type {type(pipeline_outputs)}.'
        )


def write_pipeline_spec_to_file(
    pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    pipeline_description: Union[str, None],
    platform_spec: pipeline_spec_pb2.PlatformSpec,
    package_path: str,
    kubernetes_manifest_options: Optional[KubernetesManifestOptions] = None,
    kubernetes_manifest_format: bool = False,
) -> None:
    """Writes PipelineSpec into a YAML or JSON (deprecated) file.

    Args:
        pipeline_spec: The PipelineSpec.
        pipeline_description: Description from pipeline docstring.
        platform_spec: The PlatformSpec.
        package_path: The path to which to write the PipelineSpec.
        kubernetes_manifest_options: KubernetesManifestOptions object with manifest options.
        kubernetes_manifest_format: Output the compiled pipeline as a Kubernetes manifest.
    """
    if kubernetes_manifest_format:
        opts = kubernetes_manifest_options or KubernetesManifestOptions()
        opts.set_pipeline_spec(pipeline_spec)
        _write_kubernetes_manifest_to_file(
            package_path=package_path,
            opts=opts,
            pipeline_spec=pipeline_spec,
            platform_spec=platform_spec,
            pipeline_description=pipeline_description,
        )
        return

    pipeline_spec_dict = json_format.MessageToDict(pipeline_spec)

    yaml_comments = extract_comments_from_pipeline_spec(pipeline_spec_dict,
                                                        pipeline_description)
    has_platform_specific_features = len(platform_spec.platforms) > 0

    if package_path.endswith('.json'):
        warnings.warn(
            ('Compiling to JSON is deprecated and will be '
             'removed in a future version. Please compile to a YAML file by '
             'providing a file path with a .yaml extension instead.'),
            category=DeprecationWarning,
            stacklevel=2,
        )
        with open(package_path, 'w') as json_file:
            if has_platform_specific_features:
                raise ValueError(
                    f'Platform-specific features are only supported when serializing to YAML. Argument for {"package_path"!r} has file extension {".json"!r}.'
                )
            json.dump(pipeline_spec_dict, json_file, indent=2, sort_keys=True)

    elif package_path.endswith(('.yaml', '.yml')):
        with open(package_path, 'w') as yaml_file:
            yaml_file.write(yaml_comments)
            documents = [pipeline_spec_dict]
            if has_platform_specific_features:
                documents.append(json_format.MessageToDict(platform_spec))
            yaml.dump_all(documents, yaml_file, sort_keys=True)

    else:
        raise ValueError(
            f'The output path {package_path} should end with ".yaml".')


def _write_kubernetes_manifest_to_file(
    package_path: str,
    opts: KubernetesManifestOptions,
    pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    platform_spec: pipeline_spec_pb2.PlatformSpec,
    pipeline_description: Union[str, None] = None,
) -> None:
    pipeline_name = opts.pipeline_name
    pipeline_display_name = opts.pipeline_display_name
    pipeline_version_display_name = opts.pipeline_version_display_name
    pipeline_version_name = opts.pipeline_version_name
    namespace = opts.namespace
    include_pipeline_manifest = opts.include_pipeline_manifest

    pipeline_spec_dict = json_format.MessageToDict(pipeline_spec)
    platform_spec_dict = json_format.MessageToDict(platform_spec)

    documents = []

    # Pipeline manifest
    if include_pipeline_manifest:
        pipeline_metadata = {'name': pipeline_name}
        if namespace:
            pipeline_metadata['namespace'] = namespace
        pipeline_manifest = {
            'apiVersion': 'pipelines.kubeflow.org/v2beta1',
            'kind': 'Pipeline',
            'metadata': pipeline_metadata,
            'spec': {
                'displayName': pipeline_display_name,
            },
        }
        if pipeline_description:
            pipeline_manifest['spec']['description'] = pipeline_description
        documents.append(pipeline_manifest)

    # PipelineVersion manifest
    pipeline_version_metadata = {'name': pipeline_version_name}
    if namespace:
        pipeline_version_metadata['namespace'] = namespace
    pipeline_version_manifest = {
        'apiVersion': 'pipelines.kubeflow.org/v2beta1',
        'kind': 'PipelineVersion',
        'metadata': pipeline_version_metadata,
        'spec': {
            'displayName': pipeline_version_display_name,
            'pipelineName': pipeline_name,
            'pipelineSpec': pipeline_spec_dict,
        },
    }

    if platform_spec_dict:
        pipeline_version_manifest['spec']['platformSpec'] = platform_spec_dict

    if pipeline_description:
        pipeline_version_manifest['spec']['description'] = pipeline_description
    documents.append(pipeline_version_manifest)

    with open(package_path, 'w') as yaml_file:
        yaml.dump_all(
            documents=documents,
            stream=yaml_file,
            sort_keys=True,
        )


def _merge_pipeline_config(pipelineConfig: pipeline_config.PipelineConfig,
                           platformSpec: pipeline_spec_pb2.PlatformSpec):
    workspace = pipelineConfig.workspace
    if workspace is None:
        return platformSpec

    json_format.ParseDict(
        {'pipelineConfig': {
            'workspace': workspace.get_workspace(),
        }}, platformSpec.platforms['kubernetes'])

    return platformSpec


def extract_comments_from_pipeline_spec(pipeline_spec: dict,
                                        pipeline_description: str) -> str:

    map_headings = {
        'inputDefinitions': '# Inputs:',
        'outputDefinitions': '# Outputs:'
    }

    def collect_pipeline_signatures(root_dict: dict,
                                    signature_type: str) -> List[str]:
        comment_strings = []
        if signature_type in root_dict:
            signature = root_dict[signature_type]
            comment_strings.append(map_headings[signature_type])

            # Collect data
            array_of_signatures = []
            for parameter_name, parameter_body in signature.get(
                    'parameters', {}).items():
                data = {}
                data['name'] = parameter_name
                data[
                    'parameterType'] = type_utils.IR_TYPE_TO_COMMENT_TYPE_STRING[
                        parameter_body['parameterType']]
                if 'defaultValue' in signature['parameters'][parameter_name]:
                    data['defaultValue'] = signature['parameters'][
                        parameter_name]['defaultValue']
                    if isinstance(data['defaultValue'], str):
                        data['defaultValue'] = "'" + data['defaultValue'] + "'"
                array_of_signatures.append(data)

            for artifact_name, artifact_body in signature.get('artifacts',
                                                              {}).items():
                data = {
                    'name':
                        artifact_name,
                    'parameterType':
                        artifact_body['artifactType']['schemaTitle']
                }
                array_of_signatures.append(data)

            array_of_signatures = sorted(
                array_of_signatures, key=lambda d: d.get('name'))

            # Present data
            for signature in array_of_signatures:
                string = '#    ' + signature['name'] + ': ' + signature[
                    'parameterType']
                if 'defaultValue' in signature:
                    string += ' [Default: ' + str(
                        signature['defaultValue']) + ']'
                comment_strings.append(string)

        return comment_strings

    multi_line_description_prefix = '#              '
    comment_sections = []
    comment_sections.append('# PIPELINE DEFINITION')
    comment_sections.append('# Name: ' + pipeline_spec['pipelineInfo']['name'])
    if pipeline_description:
        pipeline_description = f'\n{multi_line_description_prefix}'.join(
            pipeline_description.splitlines())
        comment_sections.append('# Description: ' + pipeline_description)
    comment_sections.extend(
        collect_pipeline_signatures(pipeline_spec['root'], 'inputDefinitions'))
    comment_sections.extend(
        collect_pipeline_signatures(pipeline_spec['root'], 'outputDefinitions'))

    comment = '\n'.join(comment_sections) + '\n'

    return comment
