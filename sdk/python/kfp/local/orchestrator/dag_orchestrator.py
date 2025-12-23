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
from typing import Any, Dict, List, Tuple

from kfp.local import config
from kfp.local import graph_utils
from kfp.local import io
from kfp.local import status
from kfp.pipeline_spec import pipeline_spec_pb2

Outputs = Dict[str, Any]


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

    # execute tasks in order
    dag_spec = dag_component_spec.dag
    sorted_tasks = graph_utils.topological_sort_tasks(dag_spec.tasks)
    while sorted_tasks:
        task_name = sorted_tasks.pop()
        task_spec = dag_spec.tasks[task_name]
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

        if task_status == status.Status.FAILURE:
            fail_stack.append(task_name)
            return {}, status.Status.FAILURE

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

    dag_outputs = OrchestratorUtils.get_dag_outputs(
        dag_outputs_spec=dag_component_spec.dag.outputs,
        io_store=io_store,
    )
    return dag_outputs, status.Status.SUCCESS