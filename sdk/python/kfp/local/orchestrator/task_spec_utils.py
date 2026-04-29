# Copyright 2026 The Kubeflow Authors
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
"""Task-spec adaptation helpers for local DAG orchestration."""

import logging
from typing import Any, Dict, List, Optional, Tuple

from kfp.local import io
from kfp.pipeline_spec import pipeline_spec_pb2

from .orchestrator_utils import OrchestratorUtils


def get_artifact_iterator_items(
    task_spec: pipeline_spec_pb2.PipelineTaskSpec,
    io_store: io.IOStore,
    task_name: str,
) -> Tuple[List[Any], str, int]:
    """Resolve items for an artifact iterator."""
    artifact_iter = task_spec.artifact_iterator
    items_spec = artifact_iter.items
    artifact_item_input = artifact_iter.item_input

    if not items_spec.input_artifact:
        raise ValueError(f'Unknown artifact items type for task {task_name}')

    artifact_name = items_spec.input_artifact
    items = None
    if '-' in artifact_name:
        actual_name = artifact_name.replace('pipelinechannel--', '')
        parts = actual_name.rsplit('-', 1)
        if len(parts) == 2:
            producer_task = parts[0]
            output_key = parts[1]
            try:
                items = io_store.get_task_output(producer_task, output_key)
            except ValueError:
                try:
                    items = io_store.get_parent_input(artifact_name)
                except ValueError:
                    items = io_store.get_parent_input(actual_name)
        else:
            items = io_store.get_parent_input(artifact_name)
    else:
        items = io_store.get_parent_input(artifact_name)

    parallelism_limit = 0
    if task_spec.HasField('iterator_policy'):
        parallelism_limit = task_spec.iterator_policy.parallelism_limit

    return items, artifact_item_input, parallelism_limit


def evaluate_parameter_expression(loop_item: Any, expression: str) -> Any:
    """Evaluate a parameter expression selector against a loop item."""
    try:
        if expression.startswith('parseJson('):
            if '["' in expression and '"]' in expression:
                start_idx = expression.find('["') + 2
                end_idx = expression.find('"]')
                field_name = expression[start_idx:end_idx]

                if isinstance(loop_item, dict):
                    if field_name in loop_item:
                        return loop_item[field_name]
                    raise KeyError(
                        f'Field "{field_name}" not found in loop item: {loop_item}'
                    )
                if isinstance(loop_item, str):
                    import json
                    parsed_item = json.loads(loop_item)
                    if field_name in parsed_item:
                        return parsed_item[field_name]
                    raise KeyError(
                        f'Field "{field_name}" not found in parsed loop item: {parsed_item}'
                    )
                logging.warning(
                    f'Cannot parse JSON from loop item type: {type(loop_item)}')
                return loop_item

            if isinstance(loop_item, str):
                import json
                return json.loads(loop_item)
            return loop_item

        if '[' in expression and ']' in expression:
            start_idx = expression.find('["') + 2
            end_idx = expression.find('"]')
            if start_idx > 1 and end_idx > start_idx:
                field_name = expression[start_idx:end_idx]
                if isinstance(loop_item, dict):
                    if field_name in loop_item:
                        return loop_item[field_name]
                    raise KeyError(
                        f'Field "{field_name}" not found in loop item: {loop_item}'
                    )

        return loop_item
    except Exception as exc:
        logging.warning(
            f'Failed to evaluate parameter expression "{expression}": {exc}')
        return loop_item


def create_parameter_loop_iteration_task_spec(
    original_task_spec: pipeline_spec_pb2.PipelineTaskSpec,
    loop_item: Any,
    loop_task_name: str,
) -> pipeline_spec_pb2.PipelineTaskSpec:
    """Create a modified task spec for one parameter-iterator iteration."""
    from kfp.compiler import pipeline_spec_builder
    from kfp.local import executor_output_utils

    iteration_task_spec = pipeline_spec_pb2.PipelineTaskSpec()
    iteration_task_spec.CopyFrom(original_task_spec)

    loop_item_param_name: Optional[str] = None
    if original_task_spec.WhichOneof('iterator') == 'parameter_iterator':
        loop_item_param_name = original_task_spec.parameter_iterator.item_input

    if loop_item_param_name:
        new_input = iteration_task_spec.inputs.parameters[loop_item_param_name]
        new_constant = pipeline_spec_builder.to_protobuf_value(loop_item)
        new_input.runtime_value.constant.CopyFrom(new_constant)

    iteration_task_spec.ClearField('parameter_iterator')
    iteration_task_spec.ClearField('artifact_iterator')

    for input_spec in iteration_task_spec.inputs.parameters.values():
        if input_spec.HasField('runtime_value'):
            runtime_value = input_spec.runtime_value
            if runtime_value.WhichOneof('value') == 'constant':
                constant_value = executor_output_utils.pb2_value_to_python(
                    runtime_value.constant)
                if (isinstance(constant_value, str) and
                        'pipelinechannel--' in constant_value and
                        f'{loop_task_name}-loop-item' in constant_value):
                    new_constant = pipeline_spec_builder.to_protobuf_value(
                        loop_item)
                    runtime_value.constant.CopyFrom(new_constant)

        elif input_spec.HasField('task_output_parameter'):
            input_spec.ClearField('task_output_parameter')
            new_constant = pipeline_spec_builder.to_protobuf_value(loop_item)
            input_spec.runtime_value.constant.CopyFrom(new_constant)

        elif input_spec.HasField('component_input_parameter'):
            param_name = input_spec.component_input_parameter
            has_expression_selector = bool(
                input_spec.parameter_expression_selector)

            if (param_name.endswith('-loop-item') or
                    f'{loop_task_name}-loop-item' in param_name):
                if has_expression_selector:
                    expression = input_spec.parameter_expression_selector
                    extracted_value = evaluate_parameter_expression(
                        loop_item, expression)
                    input_spec.ClearField('component_input_parameter')
                    input_spec.ClearField('parameter_expression_selector')
                    if extracted_value is None:
                        raise ValueError('Extracted value is None')
                    new_constant = pipeline_spec_builder.to_protobuf_value(
                        extracted_value)
                    input_spec.runtime_value.constant.CopyFrom(new_constant)
                else:
                    input_spec.ClearField('component_input_parameter')
                    if loop_item is None:
                        raise ValueError('Loop item is None')
                    new_constant = pipeline_spec_builder.to_protobuf_value(
                        loop_item)
                    input_spec.runtime_value.constant.CopyFrom(new_constant)
            elif loop_item_param_name and param_name == loop_item_param_name:
                input_spec.ClearField('component_input_parameter')
                new_constant = pipeline_spec_builder.to_protobuf_value(
                    loop_item)
                input_spec.runtime_value.constant.CopyFrom(new_constant)

    return iteration_task_spec


def create_artifact_loop_iteration_task_spec(
    original_task_spec: pipeline_spec_pb2.PipelineTaskSpec,
    iteration_index: int,
    loop_task_name: str,
    artifact_item_input: str,
) -> pipeline_spec_pb2.PipelineTaskSpec:
    """Create a modified task spec for one artifact-iterator iteration."""
    iteration_task_spec = pipeline_spec_pb2.PipelineTaskSpec()
    iteration_task_spec.CopyFrom(original_task_spec)

    iteration_task_spec.ClearField('parameter_iterator')
    iteration_task_spec.ClearField('artifact_iterator')

    for input_spec in iteration_task_spec.inputs.artifacts.values():
        if input_spec.HasField('component_input_artifact'):
            artifact_name = input_spec.component_input_artifact
            if (artifact_name == artifact_item_input or
                    artifact_name.endswith('-loop-item') or
                    f'{loop_task_name}-loop-item' in artifact_name):
                iteration_key = f'{artifact_name}-iteration-{iteration_index}'
                input_spec.component_input_artifact = iteration_key

    return iteration_task_spec


def build_nested_dag_arguments(
    task_name: str,
    task_spec: pipeline_spec_pb2.PipelineTaskSpec,
    component_spec: pipeline_spec_pb2.ComponentSpec,
    io_store: io.IOStore,
) -> Dict[str, Any]:
    """Build arguments for a nested DAG task from the current IO store."""
    if not task_spec.inputs.parameters:
        task_arguments = OrchestratorUtils.join_user_inputs_and_defaults(
            dag_arguments={},
            dag_inputs_spec=component_spec.input_definitions,
        )
    else:
        task_arguments = OrchestratorUtils.make_task_arguments(
            task_spec.inputs, io_store)

    for input_name in component_spec.input_definitions.parameters:
        if input_name not in task_arguments:
            try:
                parent_value = io_store.get_parent_input(input_name)
                task_arguments[input_name] = parent_value
            except Exception:
                pass

    iteration_index = _get_iteration_index(task_name)
    for input_name in component_spec.input_definitions.artifacts:
        if input_name not in task_arguments:
            try:
                if iteration_index is not None:
                    iteration_key = f'{input_name}-iteration-{iteration_index}'
                    try:
                        parent_value = io_store.get_parent_input(iteration_key)
                        task_arguments[input_name] = parent_value
                        continue
                    except Exception:
                        pass
                parent_value = io_store.get_parent_input(input_name)
                task_arguments[input_name] = parent_value
            except Exception:
                pass

    return task_arguments


def get_nested_pipeline_resource_name(
    task_name: str,
    pipeline_resource_name: str,
) -> str:
    """Build the pipeline resource name for a nested DAG task."""
    if '-iteration-' in task_name:
        return f'{pipeline_resource_name}/{task_name}'
    return pipeline_resource_name


def _get_iteration_index(task_name: str) -> Optional[int]:
    if '-iteration-' not in task_name:
        return None
    try:
        return int(task_name.split('-iteration-')[-1])
    except ValueError:
        return None
