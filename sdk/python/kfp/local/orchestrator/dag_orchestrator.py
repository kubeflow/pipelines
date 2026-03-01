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
"""Code for locally executing a DAG within a pipeline."""
import logging
from typing import Any, Dict, List, Tuple

from kfp.local import config
from kfp.local import graph_utils
from kfp.local import io
from kfp.local import status
from kfp.pipeline_spec import pipeline_spec_pb2

Outputs = Dict[str, Any]

# Trigger strategy enum value for ALL_UPSTREAM_TASKS_COMPLETED
ALL_UPSTREAM_TASKS_COMPLETED = 2


def _is_exit_handler_task(task_spec: pipeline_spec_pb2.PipelineTaskSpec) -> bool:
    """Check if a task is an exit handler task.

    Exit handler tasks have trigger_policy.strategy == ALL_UPSTREAM_TASKS_COMPLETED,
    which means they should run regardless of upstream task success/failure.

    Args:
        task_spec: The task specification.

    Returns:
        True if the task is an exit handler task.
    """
    return task_spec.trigger_policy.strategy == ALL_UPSTREAM_TASKS_COMPLETED


def run_dag(
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
    """Runs a DAGSpec.

    Args:
        pipeline_resource_name: The root pipeline resource name.
        dag_component_spec: The ComponentSpec which defines the DAG to execute.
        executors: The ExecutorSpecs corresponding to the DAG.
        components: The ComponentSpecs corresponding to the DAG.
        dag_arguments: The arguments to the DAG's outer ComponentSpec.
        io_store: The IOStore instance corresponding to this DAG.
        pipeline_root: The local pipeline root.
        runner: The user-specified local runner.
        unique_pipeline_id: A unique identifier for the pipeline for placeholder resolution.
        fail_stack: Mutable stack of failures. If a primitive task in the DAG fails, the task name is appended. If a multitask DAG fails, the DAG name is appended. If the pipeline executes successfully, fail_stack will be empty throughout the full local execution call stack.

    Returns:
        A two-tuple of (outputs, status). If status is FAILURE, outputs is an empty dictionary.
    """

    # Original DAG execution logic for simple pipelines
    from .orchestrator_utils import OrchestratorUtils

    dag_arguments_with_defaults = OrchestratorUtils.join_user_inputs_and_defaults(
        dag_arguments=dag_arguments,
        dag_inputs_spec=dag_component_spec.input_definitions,
    )

    # prepare IOStore for DAG
    io_store = io.IOStore()
    for k, v in dag_arguments_with_defaults.items():
        io_store.put_parent_input(k, v)

    # Separate exit handler tasks from regular tasks
    dag_spec = dag_component_spec.dag
    regular_tasks = {}
    exit_handler_tasks = {}

    for task_name, task_spec in dag_spec.tasks.items():
        if _is_exit_handler_task(task_spec):
            exit_handler_tasks[task_name] = task_spec
        else:
            regular_tasks[task_name] = task_spec

    # Execute regular tasks in order
    dag_failure = False
    failed_task_name = None
    error_message = None

    if regular_tasks:
        sorted_tasks = graph_utils.topological_sort_tasks(regular_tasks)
        while sorted_tasks:
            task_name = sorted_tasks.pop()
            task_spec = regular_tasks[task_name]
            component_name = task_spec.component_ref.name
            component_spec = components[component_name]
            implementation = component_spec.WhichOneof('implementation')
            if implementation == 'dag':
                # unlikely to exceed default max recursion depth of 1000
                outputs, task_status = run_dag(
                    pipeline_resource_name=pipeline_resource_name,
                    dag_component_spec=component_spec,
                    components=components,
                    executors=executors,
                    dag_arguments=OrchestratorUtils.make_task_arguments(
                        task_spec.inputs,
                        io_store,
                    ),
                    pipeline_root=pipeline_root,
                    runner=runner,
                    unique_pipeline_id=unique_pipeline_id,
                    fail_stack=fail_stack,
                )
            else:
                # Use consolidated task execution logic from OrchestratorUtils
                outputs, task_status = OrchestratorUtils.execute_single_task(
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

            # Store task status for exit handler tasks
            if task_status == status.Status.FAILURE:
                dag_failure = True
                failed_task_name = task_name
                error_message = f"Task '{task_name}' failed during execution"
                io_store.put_task_status(task_name, task_status, error_message)
            else:
                io_store.put_task_status(task_name, task_status)
                # Don't return immediately if there are exit handler tasks
                if not exit_handler_tasks:
                    fail_stack.append(task_name)
                    return {}, status.Status.FAILURE
                # Stop executing remaining regular tasks
                break

            # update IO store on success
            elif task_status == status.Status.SUCCESS:
                for key, output in outputs.items():
                    io_store.put_task_output(
                        task_name,
                        key,
                        output,
                    )
            else:
                raise ValueError(f'Got unknown task status: {task_status.name}')

    # Execute exit handler tasks (they run regardless of success/failure)
    for task_name, task_spec in exit_handler_tasks.items():
        logging.info(f'Running exit handler task: {task_name}')
        component_name = task_spec.component_ref.name
        component_spec = components[component_name]
        implementation = component_spec.WhichOneof('implementation')

        try:
            if implementation == 'dag':
                outputs, task_status = run_dag(
                    pipeline_resource_name=pipeline_resource_name,
                    dag_component_spec=component_spec,
                    components=components,
                    executors=executors,
                    dag_arguments=OrchestratorUtils.make_task_arguments(
                        task_spec.inputs,
                        io_store,
                    ),
                    pipeline_root=pipeline_root,
                    runner=runner,
                    unique_pipeline_id=unique_pipeline_id,
                    fail_stack=fail_stack,
                )
            else:
                outputs, task_status = OrchestratorUtils.execute_single_task(
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

            # Store exit handler task outputs
            if task_status == status.Status.SUCCESS:
                for key, output in outputs.items():
                    io_store.put_task_output(task_name, key, output)

        except Exception as e:
            logging.error(f'Exit handler task {task_name} failed with exception: {e}')
            # Exit handler failures don't affect the overall DAG status
            # (the DAG already failed if regular tasks failed)

    # If a regular task failed, return failure status
    if dag_failure:
        fail_stack.append(failed_task_name)
        return {}, status.Status.FAILURE

    dag_outputs = OrchestratorUtils.get_dag_outputs(
        dag_outputs_spec=dag_component_spec.dag.outputs,
        io_store=io_store,
    )
    return dag_outputs, status.Status.SUCCESS