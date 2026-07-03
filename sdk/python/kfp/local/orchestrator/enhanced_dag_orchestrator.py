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
import enum
import logging
import re
from typing import Any, Dict, List, Set, Tuple

from kfp.local import config
from kfp.local import graph_utils
from kfp.local import io
from kfp.local import status
from kfp.pipeline_spec import pipeline_spec_pb2

from . import task_executor
from . import task_spec_utils
from .orchestrator_utils import OrchestratorUtils

Outputs = Dict[str, Any]


@enum.unique
class TaskKind(str, enum.Enum):
    """Kinds of local tasks handled by the enhanced DAG orchestrator."""
    EXIT = 'exit'
    CONDITION = 'condition'
    PARALLEL_FOR = 'parallel_for'
    REGULAR = 'regular'


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
            # Try both the stripped name and the full pipelinechannel name
            # since IOStore keys may use either format depending on how
            # the sub-DAG's inputs are defined
            for key in [
                    actual_ref, f'pipelinechannel--{actual_ref}', channel_ref
            ]:
                try:
                    return io_store.get_parent_input(key)
                except (ValueError, KeyError):
                    continue
            logging.warning(f'Could not resolve parent input: {actual_ref}')
            return None

    @staticmethod
    def evaluate_condition(
        condition: str,
        io_store: io.IOStore,
    ) -> bool:
        """Evaluates a condition string using available values from IOStore.

        Handles both simple pipeline channel references and CEL-style
        expressions like inputs.parameter_values['param_name'].

        Args:
            condition: The condition expression to evaluate
            io_store: IOStore containing available values

        Returns:
            True if condition evaluates to True, False otherwise
        """
        if not condition or not condition.strip():
            return True

        try:
            safe_condition = condition

            # Handle CEL-style inputs.parameter_values['param_name'] references
            cel_pattern = r"inputs\.parameter_values\['([^']+)'\]"
            cel_matches = re.findall(cel_pattern, condition)
            for param_name in cel_matches:
                value = ConditionEvaluator._resolve_pipeline_channel(
                    param_name, io_store)
                if value is not None:
                    if isinstance(value, str):
                        replacement = f"'{value}'"
                    else:
                        replacement = str(value)
                    safe_condition = safe_condition.replace(
                        f"inputs.parameter_values['{param_name}']", replacement)
                else:
                    logging.warning(
                        f'Could not resolve CEL parameter: {param_name}')
                    return False

            # Also handle direct pipeline channel references
            pipeline_channels = ConditionEvaluator._extract_pipeline_channels(
                safe_condition)
            for channel_ref in pipeline_channels:
                value = ConditionEvaluator._resolve_pipeline_channel(
                    channel_ref, io_store)
                if value is not None:
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

            # Convert CEL syntax to Python syntax
            safe_condition = safe_condition.replace('&&', ' and ')
            safe_condition = safe_condition.replace('||', ' or ')
            # Convert CEL negation !(...) to Python not (...)
            safe_condition = re.sub(r'!\s*\(', 'not (', safe_condition)

            # Use a restricted evaluation environment for safety.
            # CEL uses lowercase `true`/`false`; expose them as aliases so
            # evaluating a raw CEL condition doesn't NameError.
            allowed_names = {
                '__builtins__': {},
                'True': True,
                'False': False,
                'true': True,
                'false': False,
                'None': None,
                'null': None,
                'not': lambda x: not x,
                'int': int,
                'float': float,
                'str': str,
                'bool': bool,
                'len': len,
                'min': min,
                'max': max,
            }

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


def _has_task_final_status_input(
        task_spec: pipeline_spec_pb2.PipelineTaskSpec) -> bool:
    """True if any parameter input carries a task_final_status reference.

    The compiler emits this exclusively for dsl.ExitHandler exit tasks.
    It is the signal that discriminates a real exit task from an
    `.ignore_upstream_failure()` task — both set
    `trigger_policy.strategy = ALL_UPSTREAM_TASKS_COMPLETED`, but only
    the former consumes PipelineTaskFinalStatus.
    """
    for param_spec in task_spec.inputs.parameters.values():
        if param_spec.HasField('task_final_status'):
            return True
    return False


def _run_parallel_for_task(
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
    parallel_executor: 'ParallelExecutor',
) -> bool:
    """Run all iterations of a single ParallelFor task.

    Resolves the iterator's items from IOStore, materializes an
    iteration task spec per item, executes them in parallel, and writes
    aggregated outputs back to IOStore under the loop task name so
    downstream Collected consumers can read them.

    Returns True if the parallel_for failed and the caller should
    propagate failure; False otherwise.
    """
    iterator = task_spec.WhichOneof('iterator')
    artifact_item_input = None

    if iterator == 'parameter_iterator':
        param_iter = task_spec.parameter_iterator
        items_spec = param_iter.items
        if items_spec.HasField('input_parameter'):
            param_name = items_spec.input_parameter
            if param_name.startswith('pipelinechannel--'):
                actual_param = param_name.replace('pipelinechannel--',
                                                  '').replace('-loop-item', '')
                if '-Output' in actual_param:
                    output_task_name = actual_param.split('-Output')[0]
                    items = io_store.get_task_output(output_task_name, 'Output')
                else:
                    try:
                        items = io_store.get_parent_input(actual_param)
                    except Exception:
                        items = io_store.get_parent_input(
                            f'pipelinechannel--{actual_param}')
            else:
                items = io_store.get_parent_input(param_name)
        elif items_spec.HasField('raw'):
            import json
            items = json.loads(items_spec.raw)
        else:
            logging.warning(f'Unknown items type for task {task_name}')
            return False
        parallelism_limit = 0
        if task_spec.HasField('iterator_policy'):
            parallelism_limit = task_spec.iterator_policy.parallelism_limit
    elif iterator == 'artifact_iterator':
        try:
            items, artifact_item_input, parallelism_limit = (
                task_spec_utils.get_artifact_iterator_items(
                    task_spec, io_store, task_name))
        except ValueError as e:
            logging.warning(f'Failed to get artifact iterator items: {e}')
            return False
    else:
        logging.warning(
            f'Unknown iterator type for task {task_name}: {iterator}')
        return False

    loop_tasks = []
    for i, item in enumerate(items):
        iteration_task_name = f"{task_name}-iteration-{i}"
        if iterator == 'artifact_iterator':
            iteration_task_spec = (
                task_spec_utils.create_artifact_loop_iteration_task_spec(
                    task_spec, i, task_name, artifact_item_input))
            iteration_key = f'{artifact_item_input}-iteration-{i}'
            io_store.put_parent_input(iteration_key, item)
        else:
            iteration_task_spec = (
                task_spec_utils.create_parameter_loop_iteration_task_spec(
                    task_spec, item, task_name))
        loop_tasks.append((iteration_task_name, iteration_task_spec))

    if not loop_tasks:
        return False

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
        return True

    # Aggregate iteration outputs into lists keyed by output name, in
    # iteration order, and publish under the loop task's name so that
    # downstream dsl.Collected consumers resolve via IOStore.
    aggregated_outputs: Dict[str, Dict[str, Any]] = {}
    for iteration_name, outputs in parallel_outputs.items():
        iteration_index = iteration_name.split('-iteration-')[-1]
        for output_key, output_value in outputs.items():
            aggregated_outputs.setdefault(output_key,
                                          {})[iteration_index] = output_value

    for output_key, iteration_outputs in aggregated_outputs.items():
        ordered_outputs = [
            iteration_outputs.get(str(i)) for i in range(len(items))
        ]
        io_store.put_task_output(task_name, output_key, ordered_outputs)

    io_store.put_task_status(task_name, 'SUCCEEDED')
    return False


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
    parent_io_store: 'io.IOStore' = None,
) -> Tuple[Outputs, status.Status]:
    """Enhanced DAG runner with support for dsl.Condition and dsl.ParallelFor.

    This is an enhanced version of dag_orchestrator.run_dag that
    supports control flow features like conditions and parallel loops.
    """
    dag_arguments_with_defaults = OrchestratorUtils.join_user_inputs_and_defaults(
        dag_arguments=dag_arguments,
        dag_inputs_spec=dag_component_spec.input_definitions,
    )

    # prepare IOStore for DAG, chained to the enclosing scope (if any) so
    # nested references can walk up on miss
    io_store = io.IOStore(parent=parent_io_store)
    for k, v in dag_arguments_with_defaults.items():
        io_store.put_parent_input(k, v)

    dag_spec = dag_component_spec.dag

    # Classify every task by control-flow kind. Exit tasks are separated out
    # because they must run AFTER every other task regardless of success /
    # failure. All other tasks (regular, condition, parallel_for) share a
    # single topological order over the full body, so that cross-kind
    # dependencies (e.g. a regular task consuming dsl.Collected output from a
    # parallel_for) are honored.
    #
    # ExitHandler tasks and ignore_upstream_failure() tasks BOTH compile to
    # trigger_policy.strategy == ALL_UPSTREAM_TASKS_COMPLETED. The
    # discriminator is that a real exit task takes a PipelineTaskFinalStatus
    # input; ignore_upstream_failure() tasks do not. Only true exit tasks go
    # into the exit phase — ignore_upstream_failure() tasks stay in the body
    # and are honored by the dependency-failure check below.
    TRIGGER_STRATEGY = (
        pipeline_spec_pb2.PipelineTaskSpec.TriggerPolicy.TriggerStrategy)

    task_kinds: Dict[str, TaskKind] = {}
    exit_tasks: List[Tuple[str, pipeline_spec_pb2.PipelineTaskSpec]] = []
    body_tasks: Dict[str, pipeline_spec_pb2.PipelineTaskSpec] = {}

    for task_name, task_spec in dag_spec.tasks.items():
        if (task_spec.trigger_policy.strategy
                == TRIGGER_STRATEGY.ALL_UPSTREAM_TASKS_COMPLETED and
                _has_task_final_status_input(task_spec)):
            task_kinds[task_name] = TaskKind.EXIT
            exit_tasks.append((task_name, task_spec))
        elif task_spec.trigger_policy.condition:
            task_kinds[task_name] = TaskKind.CONDITION
            body_tasks[task_name] = task_spec
        elif (task_spec.WhichOneof('iterator') and
              _has_valid_iterator(task_spec)):
            task_kinds[task_name] = TaskKind.PARALLEL_FOR
            body_tasks[task_name] = task_spec
        else:
            task_kinds[task_name] = TaskKind.REGULAR
            body_tasks[task_name] = task_spec

    # Track overall body status for exit handler support, plus the set of
    # task names that have failed (or were skipped because their upstream
    # failed). failed_tasks feeds the dependency-failure gating below.
    body_failed = False
    failed_tasks: Set[str] = set()

    condition_evaluator = ConditionEvaluator()
    parallel_executor = ParallelExecutor()

    if body_tasks:
        sorted_body_tasks = graph_utils.topological_sort_tasks(body_tasks)

        while sorted_body_tasks:
            task_name = sorted_body_tasks.pop()
            task_spec = body_tasks[task_name]
            kind = task_kinds[task_name]

            # Dependency-failure gating. A task whose upstream failed is
            # skipped unless it explicitly opts in to running on completion
            # (ignore_upstream_failure → ALL_UPSTREAM_TASKS_COMPLETED).
            failed_deps = [
                dep for dep in task_spec.dependent_tasks if dep in failed_tasks
            ]
            if failed_deps and (
                    task_spec.trigger_policy.strategy
                    != TRIGGER_STRATEGY.ALL_UPSTREAM_TASKS_COMPLETED):
                logging.info(f'Skipping task {task_name}: upstream task(s) '
                             f'{sorted(failed_deps)} failed.')
                failed_tasks.add(task_name)
                io_store.put_task_status(task_name, 'FAILED')
                body_failed = True
                continue

            if kind == TaskKind.REGULAR:
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

                status_str = ('SUCCEEDED' if task_status
                              == status.Status.SUCCESS else 'FAILED')
                io_store.put_task_status(task_name, status_str)

                if task_status == status.Status.FAILURE:
                    fail_stack.append(task_name)
                    failed_tasks.add(task_name)
                    body_failed = True
                    continue

                for key, output in outputs.items():
                    io_store.put_task_output(task_name, key, output)

            elif kind == TaskKind.CONDITION:
                condition_expr = task_spec.trigger_policy.condition
                should_execute = condition_evaluator.evaluate_condition(
                    condition_expr, io_store)

                if not should_execute:
                    logging.info(f'Skipping conditional task {task_name} '
                                 '(condition evaluated to False)')
                    continue

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

                status_str = ('SUCCEEDED' if task_status
                              == status.Status.SUCCESS else 'FAILED')
                io_store.put_task_status(task_name, status_str)

                if task_status == status.Status.FAILURE:
                    fail_stack.append(task_name)
                    failed_tasks.add(task_name)
                    body_failed = True
                    continue

                for key, output in outputs.items():
                    io_store.put_task_output(task_name, key, output)

            elif kind == TaskKind.PARALLEL_FOR:
                parallel_for_failed = _run_parallel_for_task(
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
                    parallel_executor=parallel_executor,
                )

                if parallel_for_failed:
                    failed_tasks.add(task_name)
                    body_failed = True

    # If the body failed and there are no exit tasks to run afterwards, bail
    # out now. Exit tasks always run below regardless of body outcome.
    if body_failed and not exit_tasks:
        return {}, status.Status.FAILURE

    # Execute exit tasks after all body tasks, regardless of body success or
    # failure. An exit task failure propagates to the overall DAG status so
    # callers see pipeline-level FAILURE even when the body succeeded.
    exit_failed = False
    if exit_tasks:
        for task_name, task_spec in exit_tasks:
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

            status_str = ('SUCCEEDED'
                          if task_status == status.Status.SUCCESS else 'FAILED')
            io_store.put_task_status(task_name, status_str)

            if task_status == status.Status.SUCCESS:
                for key, output in outputs.items():
                    io_store.put_task_output(task_name, key, output)
            else:
                fail_stack.append(task_name)
                exit_failed = True

        if body_failed or exit_failed:
            return {}, status.Status.FAILURE

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
        task_arguments = task_spec_utils.build_nested_dag_arguments(
            task_name=task_name,
            task_spec=task_spec,
            component_spec=component_spec,
            io_store=io_store,
        )
        nested_resource_name = task_spec_utils.get_nested_pipeline_resource_name(
            task_name=task_name,
            pipeline_resource_name=pipeline_resource_name,
        )

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
            parent_io_store=io_store,
        )

    else:
        return task_executor.execute_single_task(
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
