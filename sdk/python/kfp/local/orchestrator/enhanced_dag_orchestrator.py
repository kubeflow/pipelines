#!/usr/bin/env python3
# Copyright 2024 The Kubeflow Authors
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
"""Enhanced DAG orchestrator with support for dsl.Condition and
dsl.ParallelFor."""

import concurrent.futures
import logging
import re
from typing import Any, Dict, List, Tuple

from kfp.local import config
from kfp.local import graph_utils
from kfp.local import io
from kfp.local import status
from kfp.pipeline_spec import pipeline_spec_pb2

from .orchestrator_utils import OrchestratorUtils

Outputs = Dict[str, Any]


class ConditionEvaluator:
    """Evaluates condition expressions for dsl.Condition support."""

    @staticmethod
    def _extract_pipeline_channels(condition: str) -> List[str]:
        """Extract pipeline channel references from condition string.

        Args:
            condition: The condition expression

        Returns:
            List of pipeline channel references found
        """
        # Look for patterns like pipelinechannel--task-name-output-name
        pattern = r'pipelinechannel--[a-zA-Z0-9_\-]+'
        return re.findall(pattern, condition)

    @staticmethod
    def _resolve_pipeline_channel(channel_ref: str,
                                  io_store: io.IOStore) -> Any:
        """Resolve a pipeline channel reference to its actual value.

        Args:
            channel_ref: The pipeline channel reference string
            io_store: IOStore containing values

        Returns:
            The resolved value
        """
        # Remove the pipelinechannel-- prefix
        actual_ref = channel_ref.replace('pipelinechannel--', '')

        # Check if this is a task output reference
        if '-Output' in actual_ref or '--' in actual_ref:
            # Parse task output reference: task-name-Output or task-name--output-name
            if '--' in actual_ref:
                parts = actual_ref.split('--')
                output_task_name = parts[0]
                output_key = parts[1] if len(parts) > 1 else 'Output'
            else:
                parts = actual_ref.split('-Output')
                output_task_name = parts[0]
                output_key = 'Output'

            try:
                return io_store.get_task_output(output_task_name, output_key)
            except Exception:
                logging.warning(
                    f'Could not resolve task output: {output_task_name}.{output_key}'
                )
                return None
        else:
            # This is a parent input parameter
            try:
                return io_store.get_parent_input(actual_ref)
            except Exception:
                logging.warning(f'Could not resolve parent input: {actual_ref}')
                return None

    @staticmethod
    def evaluate_condition(
        condition: str,
        io_store: io.IOStore,
    ) -> bool:
        """Evaluates a condition string using available values from IOStore.

        Args:
            condition: The condition expression to evaluate
            io_store: IOStore containing available values

        Returns:
            True if condition evaluates to True, False otherwise
        """
        if not condition or not condition.strip():
            return True

        try:
            # Create a safe evaluation environment
            safe_condition = condition

            # Extract and resolve pipeline channel references
            pipeline_channels = ConditionEvaluator._extract_pipeline_channels(
                condition)

            for channel_ref in pipeline_channels:
                value = ConditionEvaluator._resolve_pipeline_channel(
                    channel_ref, io_store)
                if value is not None:
                    # Replace the reference with the actual value
                    # Convert value to string representation for substitution
                    if isinstance(value, str):
                        safe_condition = safe_condition.replace(
                            channel_ref, f"'{value}'")
                    else:
                        safe_condition = safe_condition.replace(
                            channel_ref, str(value))
                else:
                    logging.warning(
                        f'Could not resolve channel reference: {channel_ref}')
                    return False

            # Use a restricted evaluation environment for safety
            allowed_names = {
                '__builtins__': {},
                'True': True,
                'False': False,
                'None': None,
                'int': int,
                'float': float,
                'str': str,
                'bool': bool,
                'len': len,
                'min': min,
                'max': max,
            }

            # Add basic comparison and logical operators by evaluating in restricted context
            result = eval(safe_condition, allowed_names, {})
            return bool(result)

        except Exception as e:
            logging.warning(
                f'Condition evaluation failed for "{condition}": {e}')
            return False


class ParallelExecutor:
    """Handles parallel execution of tasks for dsl.ParallelFor support."""

    def __init__(self, max_workers: int = None):
        """Initialize parallel executor.

        Args:
            max_workers: Maximum number of parallel workers. If None, uses conservative default.
        """
        # Use conservative default to avoid thread explosion with nested ParallelFor loops
        # Conservative max of 2 prevents exponential thread growth:
        # - 2 levels deep: 2 × 2 = 4 threads
        # - 3 levels deep: 2 × 2 × 2 = 8 threads
        self.conservative_max = 2
        # If max_workers not specified (None), use conservative default
        # If specified, still cap it to prevent thread explosion in nested loops
        if max_workers is None:
            self.max_workers = self.conservative_max
        else:
            # Cap user-specified max_workers at conservative maximum
            self.max_workers = min(max_workers, self.conservative_max)

    def execute_parallel_tasks(
        self,
        tasks: List[Tuple[str, pipeline_spec_pb2.PipelineTaskSpec]],
        pipeline_resource_name: str,
        components: Dict[str, pipeline_spec_pb2.ComponentSpec],
        executors: Dict[
            str, pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec],
        io_store: io.IOStore,
        pipeline_root: str,
        runner: config.LocalRunnerType,
        unique_pipeline_id: str,
        fail_stack: List[str],
        parallelism_limit: int = 0,
    ) -> Tuple[Dict[str, Outputs], status.Status]:
        """Execute tasks in parallel with optional parallelism limit.

        Args:
            tasks: List of (task_name, task_spec) tuples to execute
            pipeline_resource_name: The root pipeline resource name
            components: Component specifications
            executors: Executor specifications
            io_store: IOStore for this execution context
            pipeline_root: Local pipeline root directory
            runner: Local runner configuration
            unique_pipeline_id: Unique pipeline identifier
            fail_stack: Mutable failure stack
            parallelism_limit: Maximum parallel executions (0 = unlimited)

        Returns:
            Tuple of (all_outputs, overall_status)
        """
        # Determine max_workers: use parallelism_limit if specified, otherwise use default
        requested_workers = parallelism_limit if parallelism_limit > 0 else self.max_workers
        # Always cap at conservative maximum to prevent thread explosion in nested loops
        max_workers = min(requested_workers, self.conservative_max)
        all_outputs = {}

        executor = concurrent.futures.ThreadPoolExecutor(
            max_workers=max_workers)
        try:
            # Submit all tasks
            future_to_task = {}
            for task_name, task_spec in tasks:
                future = executor.submit(
                    execute_task,
                    task_name=task_name,
                    task_spec=task_spec,
                    pipeline_resource_name=pipeline_resource_name,
                    components=components,
                    executors=executors,
                    io_store=io_store,
                    pipeline_root=pipeline_root,
                    runner=runner,
                    unique_pipeline_id=unique_pipeline_id,
                    fail_stack=fail_stack,
                )
                future_to_task[future] = task_name

            # Collect results
            for future in concurrent.futures.as_completed(future_to_task):
                task_name = future_to_task[future]
                try:
                    outputs, task_status = future.result()

                    if task_status == status.Status.FAILURE:
                        fail_stack.append(task_name)
                        return {}, status.Status.FAILURE

                    all_outputs[task_name] = outputs

                    # Update IO store with outputs
                    for key, output in outputs.items():
                        io_store.put_task_output(task_name, key, output)

                except Exception as e:
                    logging.error(
                        f'Task {task_name} failed with exception: {e}')
                    fail_stack.append(task_name)
                    return {}, status.Status.FAILURE

            return all_outputs, status.Status.SUCCESS
        finally:
            # Explicitly shutdown the executor and wait for threads to terminate
            executor.shutdown(wait=True)


def _has_valid_iterator(task_spec: pipeline_spec_pb2.PipelineTaskSpec) -> bool:
    """Check if a task spec has a valid iterator configuration for ParallelFor.

    Args:
        task_spec: The pipeline task specification

    Returns:
        True if this is a valid ParallelFor task, False otherwise
    """
    iterator_type = task_spec.WhichOneof('iterator')
    if not iterator_type:
        return False

    if iterator_type == 'parameter_iterator':
        param_iter = task_spec.parameter_iterator
        items_spec = param_iter.items
        # Check if the iterator has valid items configuration
        return (items_spec.HasField('input_parameter') or
                items_spec.HasField('raw'))
    elif iterator_type == 'artifact_iterator':
        artifact_iter = task_spec.artifact_iterator
        items_spec = artifact_iter.items
        # Check if the iterator has valid items configuration
        return bool(items_spec.input_artifact)

    return False


def _get_artifact_iterator_items(
    task_spec: pipeline_spec_pb2.PipelineTaskSpec,
    io_store: io.IOStore,
    task_name: str,
) -> Tuple[List[Any], str, int]:
    """Get items from an artifact iterator.

    Args:
        task_spec: The pipeline task specification with artifact_iterator
        io_store: IOStore containing values
        task_name: The task name for logging

    Returns:
        Tuple of (items list, artifact_item_input name, parallelism_limit)

    Raises:
        ValueError: If items cannot be resolved
    """
    artifact_iter = task_spec.artifact_iterator
    items_spec = artifact_iter.items
    artifact_item_input = artifact_iter.item_input

    if not items_spec.input_artifact:
        raise ValueError(f'Unknown artifact items type for task {task_name}')

    # Get artifacts from input artifact (from IOStore)
    artifact_name = items_spec.input_artifact

    items = None
    # Check if this is a task output artifact or a parent input artifact
    if '-' in artifact_name:
        # Try to parse as task-output format
        # Format could be: task-name-output-key or pipelinechannel--task-name-output-key
        actual_name = artifact_name.replace('pipelinechannel--', '')
        # Try to find the producer task and output key
        # The format is typically: producer-task-output-key
        parts = actual_name.rsplit('-', 1)
        if len(parts) == 2:
            producer_task = parts[0]
            output_key = parts[1]
            try:
                items = io_store.get_task_output(producer_task, output_key)
            except ValueError:
                # Try with full name as parent input
                try:
                    items = io_store.get_parent_input(artifact_name)
                except ValueError:
                    items = io_store.get_parent_input(actual_name)
        else:
            items = io_store.get_parent_input(artifact_name)
    else:
        items = io_store.get_parent_input(artifact_name)

    # Get parallelism limit from iterator_policy if available
    parallelism_limit = 0
    if task_spec.HasField('iterator_policy'):
        parallelism_limit = task_spec.iterator_policy.parallelism_limit

    return items, artifact_item_input, parallelism_limit


def _evaluate_parameter_expression(loop_item: Any, expression: str) -> Any:
    """Evaluate a parameter expression selector against a loop item.

    Args:
        loop_item: The current loop item value
        expression: The parameter expression selector (e.g., 'parseJson(string_value)["A_a"]')

    Returns:
        The extracted value from the loop item
    """
    try:
        # Handle parseJson expressions
        if expression.startswith('parseJson('):
            # Extract the field path from the expression
            # Expression format: parseJson(string_value)["field_name"]
            if '["' in expression and '"]' in expression:
                start_idx = expression.find('["') + 2
                end_idx = expression.find('"]')
                field_name = expression[start_idx:end_idx]

                # If loop_item is already a dict, extract the field directly
                if isinstance(loop_item, dict):
                    if field_name in loop_item:
                        return loop_item[field_name]
                    else:
                        raise KeyError(
                            f'Field "{field_name}" not found in loop item: {loop_item}'
                        )
                # If loop_item is a JSON string, parse it first
                elif isinstance(loop_item, str):
                    import json
                    parsed_item = json.loads(loop_item)
                    if field_name in parsed_item:
                        return parsed_item[field_name]
                    else:
                        raise KeyError(
                            f'Field "{field_name}" not found in parsed loop item: {parsed_item}'
                        )
                else:
                    logging.warning(
                        f'Cannot parse JSON from loop item type: {type(loop_item)}'
                    )
                    return loop_item
            else:
                # No field extraction, just parse JSON
                if isinstance(loop_item, str):
                    import json
                    return json.loads(loop_item)
                else:
                    return loop_item

        # Handle other expressions (can be extended as needed)
        elif '[' in expression and ']' in expression:
            # Handle direct field access like item["field"]
            start_idx = expression.find('["') + 2
            end_idx = expression.find('"]')
            if start_idx > 1 and end_idx > start_idx:
                field_name = expression[start_idx:end_idx]
                if isinstance(loop_item, dict):
                    if field_name in loop_item:
                        return loop_item[field_name]
                    else:
                        raise KeyError(
                            f'Field "{field_name}" not found in loop item: {loop_item}'
                        )

        # Default: return the loop item as-is
        return loop_item

    except Exception as e:
        logging.warning(
            f'Failed to evaluate parameter expression "{expression}": {e}')
        return loop_item


def _create_loop_iteration_task_spec(
    original_task_spec: pipeline_spec_pb2.PipelineTaskSpec,
    loop_item: Any,
    iteration_index: int,
    loop_task_name: str,
    components: Dict[str, pipeline_spec_pb2.ComponentSpec] = None,
) -> pipeline_spec_pb2.PipelineTaskSpec:
    """Create a modified task spec for a loop iteration.

    Args:
        original_task_spec: The original task specification
        loop_item: The current loop item value
        iteration_index: The iteration index
        loop_task_name: The original loop task name
        components: Component specifications (needed to get loop item parameter name)

    Returns:
        Modified task spec for this iteration
    """
    from kfp.compiler import pipeline_spec_builder
    from kfp.local import executor_output_utils

    # Create a proper copy of the original task spec using protobuf methods
    # (deepcopy doesn't work correctly with protobuf objects)
    iteration_task_spec = pipeline_spec_pb2.PipelineTaskSpec()
    iteration_task_spec.CopyFrom(original_task_spec)

    # Get the loop item parameter name from the iterator
    iterator_type = original_task_spec.WhichOneof('iterator')
    loop_item_param_name = None
    if iterator_type == 'parameter_iterator':
        param_iter = original_task_spec.parameter_iterator
        loop_item_param_name = param_iter.item_input

    # The nested DAG component's input definition expects the loop-item parameter name,
    # so we ALWAYS need to add it with the correct name. This is required because the
    # component's input definitions use the -loop-item suffix while the task spec
    # might use a different parameter name.
    if loop_item_param_name:
        # Create the input parameter for the loop item with the correct name
        new_input = iteration_task_spec.inputs.parameters[loop_item_param_name]
        new_constant = pipeline_spec_builder.to_protobuf_value(loop_item)
        new_input.runtime_value.constant.CopyFrom(new_constant)

    # Remove the iterator since this is now a regular task
    iteration_task_spec.ClearField('parameter_iterator')
    iteration_task_spec.ClearField('artifact_iterator')

    # Find and replace loop item references in task inputs
    for input_name, input_spec in iteration_task_spec.inputs.parameters.items():

        if input_spec.HasField('runtime_value'):
            # Check if this references the loop item
            runtime_value = input_spec.runtime_value
            if runtime_value.WhichOneof('value') == 'constant':
                # This is a constant - might need to substitute loop item
                constant_value = executor_output_utils.pb2_value_to_python(
                    runtime_value.constant)

                # Check if the constant value contains loop item placeholders
                if isinstance(constant_value,
                              str) and 'pipelinechannel--' in constant_value:
                    # This might be a loop item reference
                    if f'{loop_task_name}-loop-item' in constant_value:
                        # Replace with actual loop item value
                        new_constant = pipeline_spec_builder.to_protobuf_value(
                            loop_item)
                        runtime_value.constant.CopyFrom(new_constant)

        elif input_spec.HasField('task_output_parameter'):
            # This is a reference to an upstream task output
            # In ParallelFor, we need to replace this with the loop item value
            input_spec.ClearField('task_output_parameter')
            new_constant = pipeline_spec_builder.to_protobuf_value(loop_item)
            input_spec.runtime_value.constant.CopyFrom(new_constant)

        elif input_spec.HasField('component_input_parameter'):
            # Check if this references a loop item parameter
            param_name = input_spec.component_input_parameter

            # Check if parameter expression selector exists (scalar field - check value not presence)
            has_expression_selector = bool(
                input_spec.parameter_expression_selector)

            if param_name.endswith(
                    '-loop-item'
            ) or f'{loop_task_name}-loop-item' in param_name:
                # Handle parameter expression selector if present
                if has_expression_selector:
                    # Extract the specific value using the expression selector
                    expression = input_spec.parameter_expression_selector
                    extracted_value = _evaluate_parameter_expression(
                        loop_item, expression)

                    # Replace with the extracted value
                    input_spec.ClearField('component_input_parameter')
                    input_spec.ClearField('parameter_expression_selector')
                    if extracted_value is None:
                        raise ValueError("Extracted value is None")
                    new_constant = pipeline_spec_builder.to_protobuf_value(
                        extracted_value)
                    input_spec.runtime_value.constant.CopyFrom(new_constant)
                else:
                    # Replace with the entire loop item value
                    input_spec.ClearField('component_input_parameter')
                    if loop_item is None:
                        raise ValueError("Loop item is None")
                    new_constant = pipeline_spec_builder.to_protobuf_value(
                        loop_item)
                    input_spec.runtime_value.constant.CopyFrom(new_constant)
            else:
                # Only replace if this parameter matches the specific loop item parameter name
                # from this ParallelFor's iterator. This avoids incorrectly replacing other
                # parent pipeline inputs that nested loops may need to access.
                if loop_item_param_name and param_name == loop_item_param_name:
                    input_spec.ClearField('component_input_parameter')
                    new_constant = pipeline_spec_builder.to_protobuf_value(
                        loop_item)
                    input_spec.runtime_value.constant.CopyFrom(new_constant)

    return iteration_task_spec


def _create_artifact_loop_iteration_task_spec(
    original_task_spec: pipeline_spec_pb2.PipelineTaskSpec,
    loop_item: Any,
    iteration_index: int,
    loop_task_name: str,
    artifact_item_input: str,
) -> pipeline_spec_pb2.PipelineTaskSpec:
    """Create a modified task spec for an artifact iterator loop iteration.

    Args:
        original_task_spec: The original task specification
        loop_item: The current loop item (an artifact)
        iteration_index: The iteration index
        loop_task_name: The original loop task name
        artifact_item_input: The name of the artifact input for the loop item

    Returns:
        Modified task spec for this iteration
    """
    # Create a proper copy of the original task spec using protobuf methods
    iteration_task_spec = pipeline_spec_pb2.PipelineTaskSpec()
    iteration_task_spec.CopyFrom(original_task_spec)

    # Remove the iterator since this is now a regular task
    iteration_task_spec.ClearField('parameter_iterator')
    iteration_task_spec.ClearField('artifact_iterator')

    # For artifact iterators, we need to handle artifact inputs
    # Update artifact input references to use iteration-specific keys
    # This allows parallel execution without race conditions

    # Find and update artifact inputs that reference the loop item
    for input_name, input_spec in iteration_task_spec.inputs.artifacts.items():
        if input_spec.HasField('component_input_artifact'):
            artifact_name = input_spec.component_input_artifact
            # Check if this references the loop item artifact
            if (artifact_name == artifact_item_input or
                    artifact_name.endswith('-loop-item') or
                    f'{loop_task_name}-loop-item' in artifact_name):
                # Update the reference to use an iteration-specific key
                # This allows each iteration to have its own artifact in the shared io_store
                iteration_key = f'{artifact_name}-iteration-{iteration_index}'
                input_spec.component_input_artifact = iteration_key

    return iteration_task_spec


def run_enhanced_dag(
    pipeline_resource_name: str,
    dag_component_spec: pipeline_spec_pb2.ComponentSpec,
    executors: Dict[str,
                    pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec],
    components: Dict[str, pipeline_spec_pb2.ComponentSpec],
    dag_arguments: Dict[str, Any],
    pipeline_root: str,
    runner: config.LocalRunnerType,
    unique_pipeline_id: str,
    fail_stack: List[str],
) -> Tuple[Outputs, status.Status]:
    """Enhanced DAG runner with support for dsl.Condition and dsl.ParallelFor.

    This is an enhanced version of dag_orchestrator.run_dag that
    supports control flow features like conditions and parallel loops.
    """
    dag_arguments_with_defaults = OrchestratorUtils.join_user_inputs_and_defaults(
        dag_arguments=dag_arguments,
        dag_inputs_spec=dag_component_spec.input_definitions,
    )

    # prepare IOStore for DAG
    io_store = io.IOStore()
    for k, v in dag_arguments_with_defaults.items():
        io_store.put_parent_input(k, v)

    dag_spec = dag_component_spec.dag

    # Group tasks by control flow type
    regular_tasks = []
    condition_tasks = []
    parallel_for_tasks = []

    for task_name, task_spec in dag_spec.tasks.items():
        if task_spec.trigger_policy.condition:
            condition_tasks.append((task_name, task_spec))
        elif task_spec.WhichOneof('iterator') and _has_valid_iterator(
                task_spec):
            parallel_for_tasks.append((task_name, task_spec))
        else:
            regular_tasks.append((task_name, task_spec))

    # Execute regular tasks first (topologically sorted)
    regular_task_dict = {name: spec for name, spec in regular_tasks}
    if regular_task_dict:
        sorted_task_names = graph_utils.topological_sort_tasks(
            regular_task_dict)

        while sorted_task_names:
            task_name = sorted_task_names.pop()
            task_spec = regular_task_dict[task_name]

            # Use execute_task which handles both executor and nested DAG implementations
            outputs, task_status = execute_task(
                task_name=task_name,
                task_spec=task_spec,
                pipeline_resource_name=pipeline_resource_name,
                components=components,
                executors=executors,
                io_store=io_store,
                pipeline_root=pipeline_root,
                runner=runner,
                unique_pipeline_id=unique_pipeline_id,
                fail_stack=fail_stack,
            )

            if task_status == status.Status.FAILURE:
                return {}, status.Status.FAILURE

            # Update IO store on success
            for key, output in outputs.items():
                io_store.put_task_output(task_name, key, output)

    # Execute conditional tasks
    condition_evaluator = ConditionEvaluator()
    for task_name, task_spec in condition_tasks:
        # Evaluate condition
        condition_expr = task_spec.trigger_policy.condition
        should_execute = condition_evaluator.evaluate_condition(
            condition_expr, io_store)

        if should_execute:
            outputs, task_status = execute_task(
                task_name=task_name,
                task_spec=task_spec,
                pipeline_resource_name=pipeline_resource_name,
                components=components,
                executors=executors,
                io_store=io_store,
                pipeline_root=pipeline_root,
                runner=runner,
                unique_pipeline_id=unique_pipeline_id,
                fail_stack=fail_stack,
            )

            if task_status == status.Status.FAILURE:
                return {}, status.Status.FAILURE

            # Update IO store on success
            for key, output in outputs.items():
                io_store.put_task_output(task_name, key, output)
        else:
            logging.info(
                f'Skipping conditional task {task_name} (condition evaluated to False)'
            )

    # Execute parallel for tasks
    parallel_executor = ParallelExecutor()
    for task_name, task_spec in parallel_for_tasks:
        # Get iterator configuration
        iterator = task_spec.WhichOneof('iterator')
        if iterator == 'parameter_iterator':
            param_iter = task_spec.parameter_iterator
            # Get items from the ItemsSpec
            items_spec = param_iter.items
            if items_spec.HasField('input_parameter'):
                # Get items from input parameter (from IOStore)
                param_name = items_spec.input_parameter
                # Check if this is a task output parameter or a parent input parameter
                if param_name.startswith('pipelinechannel--'):
                    # Remove the prefix but check if it contains a task output reference
                    actual_param = param_name.replace('pipelinechannel--',
                                                      '').replace(
                                                          '-loop-item', '')
                    # Check if this looks like a task output (task-name-output-name)
                    if '-Output' in actual_param:
                        # This is a task output, split it
                        parts = actual_param.split('-Output')
                        output_task_name = parts[
                            0]  # Don't overwrite the loop variable!
                        output_key = 'Output'
                        items = io_store.get_task_output(
                            output_task_name, output_key)
                    else:
                        # This is a parent input - try both the stripped name and
                        # the full pipelinechannel name since IOStore keys may use either
                        try:
                            items = io_store.get_parent_input(actual_param)
                        except Exception:
                            # Try with the pipelinechannel prefix
                            items = io_store.get_parent_input(
                                f'pipelinechannel--{actual_param}')
                else:
                    # Direct parameter name
                    items = io_store.get_parent_input(param_name)
            elif items_spec.HasField('raw'):
                # Parse raw JSON string
                import json
                items = json.loads(items_spec.raw)
            else:
                logging.warning(f'Unknown items type for task {task_name}')
                continue
            parallelism_limit = getattr(param_iter, 'parallelism_limit', 0)
        elif iterator == 'artifact_iterator':
            # Handle artifact iterator using helper function
            try:
                items, artifact_item_input, parallelism_limit = _get_artifact_iterator_items(
                    task_spec, io_store, task_name)
            except ValueError as e:
                logging.warning(f'Failed to get artifact iterator items: {e}')
                continue
        else:
            logging.warning(
                f'Unknown iterator type for task {task_name}: {iterator}')
            continue

        # Create parallel tasks for each loop item
        loop_tasks = []
        for i, item in enumerate(items):
            # Create a unique task name for this iteration
            iteration_task_name = f"{task_name}-iteration-{i}"

            # Create a modified task spec for this iteration
            if iterator == 'artifact_iterator':
                iteration_task_spec = _create_artifact_loop_iteration_task_spec(
                    task_spec, item, i, task_name, artifact_item_input)
                # For artifact iterators, store the artifact in io_store with iteration-specific keys
                # Use artifact_item_input as base since that's what the nested component expects
                # (e.g., pipelinechannel--make-artifacts-Output-loop-item-iteration-0)
                iteration_key = f'{artifact_item_input}-iteration-{i}'
                io_store.put_parent_input(iteration_key, item)
            else:
                iteration_task_spec = _create_loop_iteration_task_spec(
                    task_spec, item, i, task_name)

            loop_tasks.append((iteration_task_name, iteration_task_spec))

        # Execute all loop iterations in parallel
        if loop_tasks:

            parallel_outputs, parallel_status = parallel_executor.execute_parallel_tasks(
                tasks=loop_tasks,
                pipeline_resource_name=pipeline_resource_name,
                components=components,
                executors=executors,
                io_store=io_store,
                pipeline_root=pipeline_root,
                runner=runner,
                unique_pipeline_id=unique_pipeline_id,
                fail_stack=fail_stack,
                parallelism_limit=parallelism_limit,
            )

            if parallel_status == status.Status.FAILURE:
                return {}, status.Status.FAILURE

            # Aggregate loop outputs - collect all iteration outputs under the main task name
            aggregated_outputs = {}
            for iteration_name, outputs in parallel_outputs.items():
                # Store outputs with iteration index for later aggregation
                iteration_index = iteration_name.split('-iteration-')[-1]
                for output_key, output_value in outputs.items():
                    if output_key not in aggregated_outputs:
                        aggregated_outputs[output_key] = {}
                    aggregated_outputs[output_key][
                        iteration_index] = output_value

            # Store the aggregated outputs in the main task's namespace
            for output_key, iteration_outputs in aggregated_outputs.items():
                # Create a list of outputs in iteration order
                ordered_outputs = [
                    iteration_outputs.get(str(i)) for i in range(len(items))
                ]
                io_store.put_task_output(task_name, output_key, ordered_outputs)

    # Get DAG outputs
    dag_outputs = OrchestratorUtils.get_dag_outputs(
        dag_outputs_spec=dag_component_spec.dag.outputs,
        io_store=io_store,
    )

    return dag_outputs, status.Status.SUCCESS


def execute_task(
    task_name: str,
    task_spec: pipeline_spec_pb2.PipelineTaskSpec,
    pipeline_resource_name: str,
    components: Dict[str, pipeline_spec_pb2.ComponentSpec],
    executors: Dict[str,
                    pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec],
    io_store: io.IOStore,
    pipeline_root: str,
    runner: config.LocalRunnerType,
    unique_pipeline_id: str,
    fail_stack: List[str],
) -> Tuple[Outputs, status.Status]:
    """Execute a single task."""
    component_name = task_spec.component_ref.name
    component_spec = components[component_name]
    implementation = component_spec.WhichOneof('implementation')

    if implementation == 'dag':
        # For ParallelFor iteration tasks, we need to handle the case where
        # the iteration task spec has no inputs but we need to pass arguments
        # to the nested DAG based on the component's input definitions
        if not task_spec.inputs.parameters:
            # This is likely a ParallelFor iteration with no inputs
            # Use the component's input definitions with default values
            task_arguments = OrchestratorUtils.join_user_inputs_and_defaults(
                dag_arguments={},
                dag_inputs_spec=component_spec.input_definitions,
            )
        else:
            task_arguments = OrchestratorUtils.make_task_arguments(
                task_spec.inputs, io_store)

        # For nested DAGs (especially ParallelFor iterations), we need to ensure that
        # any parent inputs required by the component's input definitions are passed through.
        # This handles cases like nested ParallelFor loops that reference parent pipeline inputs.
        for input_name in component_spec.input_definitions.parameters:
            if input_name not in task_arguments:
                # This input is needed by the component but not in task_arguments
                # Try to get it from the parent IOStore
                try:
                    parent_value = io_store.get_parent_input(input_name)
                    task_arguments[input_name] = parent_value
                except Exception:
                    # Input not available in parent IOStore, will use default if available
                    pass

        # Also handle artifact inputs for artifact iterators
        # For iteration tasks (e.g., for-loop-1-iteration-0), extract the iteration index
        # to look up artifacts stored with iteration-specific keys
        iteration_index = None
        if '-iteration-' in task_name:
            try:
                iteration_index = int(task_name.split('-iteration-')[-1])
            except ValueError:
                pass

        for input_name in component_spec.input_definitions.artifacts:
            if input_name not in task_arguments:
                # This artifact is needed by the component but not in task_arguments
                # Try to get it from the parent IOStore
                try:
                    # For iteration tasks, try the iteration-specific key first
                    if iteration_index is not None:
                        iteration_key = f'{input_name}-iteration-{iteration_index}'
                        try:
                            parent_value = io_store.get_parent_input(
                                iteration_key)
                            task_arguments[input_name] = parent_value
                            continue
                        except Exception:
                            pass
                    # Fall back to the regular key
                    parent_value = io_store.get_parent_input(input_name)
                    task_arguments[input_name] = parent_value
                except Exception:
                    # Artifact not available in parent IOStore
                    pass

        # For nested DAGs (especially ParallelFor iterations), include the task name
        # in the pipeline_resource_name to ensure unique output paths for each iteration
        nested_resource_name = f"{pipeline_resource_name}/{task_name}"

        return run_enhanced_dag(
            pipeline_resource_name=nested_resource_name,
            dag_component_spec=component_spec,
            components=components,
            executors=executors,
            dag_arguments=task_arguments,
            pipeline_root=pipeline_root,
            runner=runner,
            unique_pipeline_id=unique_pipeline_id,
            fail_stack=fail_stack,
        )

    else:
        return OrchestratorUtils.execute_single_task(
            task_name=task_name,
            task_spec=task_spec,
            pipeline_resource_name=pipeline_resource_name,
            components=components,
            executors=executors,
            io_store=io_store,
            pipeline_root=pipeline_root,
            runner=runner,
            unique_pipeline_id=unique_pipeline_id,
            fail_stack=fail_stack,
        )
