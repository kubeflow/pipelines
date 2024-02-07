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
import copy
from typing import Any, Dict, List, Tuple

from kfp.local import config
from kfp.local import graph_utils
from kfp.local import importer_handler
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
        fail_stack: Mutable stack of failures. If a primitive task in the DAG fails, the task name is appended. If a multi-task DAG fails, the DAG name is appended. If the pipeline executes successfully, fail_stack will be empty throughout the full local execution call stack.

    Returns:
        A two-tuple of (outputs, status). If status is FAILURE, outputs is an empty dictionary.
    """
    from kfp.local import task_dispatcher

    dag_arguments_with_defaults = join_user_inputs_and_defaults(
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
        # TODO: support control flow features
        validate_task_spec_not_loop_or_condition(task_spec=task_spec)
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
                dag_arguments=make_task_arguments(
                    task_spec.inputs,
                    io_store,
                ),
                pipeline_root=pipeline_root,
                runner=runner,
                unique_pipeline_id=unique_pipeline_id,
                fail_stack=fail_stack,
            )

        elif implementation == 'executor_label':
            executor_spec = executors[component_spec.executor_label]
            task_arguments = make_task_arguments(
                task_inputs_spec=dag_spec.tasks[task_name].inputs,
                io_store=io_store,
            )

            if executor_spec.WhichOneof('spec') == 'importer':
                outputs, task_status = importer_handler.run_importer(
                    pipeline_resource_name=pipeline_resource_name,
                    component_name=component_name,
                    component_spec=component_spec,
                    executor_spec=executor_spec,
                    arguments=task_arguments,
                    pipeline_root=pipeline_root,
                    unique_pipeline_id=unique_pipeline_id,
                )
            elif executor_spec.WhichOneof('spec') == 'container':
                outputs, task_status = task_dispatcher.run_single_task_implementation(
                    pipeline_resource_name=pipeline_resource_name,
                    component_name=component_name,
                    component_spec=component_spec,
                    executor_spec=executor_spec,
                    arguments=task_arguments,
                    pipeline_root=pipeline_root,
                    runner=runner,
                    # let the outer pipeline raise the error
                    raise_on_error=False,
                    # components may consume input artifacts when passed from upstream
                    # outputs or parent component inputs
                    block_input_artifact=False,
                    # provide the same unique job id for each task for
                    # consistent placeholder resolution
                    unique_pipeline_id=unique_pipeline_id,
                )
            else:
                raise ValueError(
                    "Got unknown spec in ExecutorSpec. Only 'dsl.component', 'dsl.container_component', and 'dsl.importer' are supported in local pipeline execution."
                )
        else:
            raise ValueError(
                f'Got unknown component implementation: {implementation}')

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

    dag_outputs = get_dag_outputs(
        dag_outputs_spec=dag_component_spec.dag.outputs,
        io_store=io_store,
    )
    return dag_outputs, status.Status.SUCCESS


def join_user_inputs_and_defaults(
    dag_arguments: Dict[str, Any],
    dag_inputs_spec: pipeline_spec_pb2.ComponentInputsSpec,
) -> Dict[str, Any]:
    """Collects user-provided arguments and default arguments (when no user-
    provided argument) into a dictionary. Returns the dictionary.

    Args:
        dag_arguments: The user-provided arguments to the DAG.
        dag_inputs_spec: The ComponentInputSpec for the DAG.

    Returns:
        The complete DAG inputs, with defaults included where the user-provided argument is missing.
    """
    from kfp.local import executor_output_utils

    copied_dag_arguments = copy.deepcopy(dag_arguments)

    for input_name, input_spec in dag_inputs_spec.parameters.items():
        if input_name not in copied_dag_arguments:
            copied_dag_arguments[
                input_name] = executor_output_utils.pb2_value_to_python(
                    input_spec.default_value)
    return copied_dag_arguments


def make_task_arguments(
    task_inputs_spec: pipeline_spec_pb2.TaskInputsSpec,
    io_store: io.IOStore,
) -> Dict[str, Any]:
    """Obtains a dictionary of arguments required to execute the task
    corresponding to TaskInputsSpec.

    Args:
        task_inputs_spec: The TaskInputsSpec for the task.
        io_store: The IOStore of the current DAG. Used to obtain task arguments which come from upstream task outputs and parent component inputs.

    Returns:
        The arguments for the task.
    """
    from kfp.local import executor_output_utils

    task_arguments = {}
    # handle parameters
    for input_name, input_spec in task_inputs_spec.parameters.items():

        # handle constants
        if input_spec.HasField('runtime_value'):
            # runtime_value's value should always be constant for the v2 compiler
            if input_spec.runtime_value.WhichOneof('value') != 'constant':
                raise ValueError('Expected constant.')
            task_arguments[
                input_name] = executor_output_utils.pb2_value_to_python(
                    input_spec.runtime_value.constant)

        # handle upstream outputs
        elif input_spec.HasField('task_output_parameter'):
            task_arguments[input_name] = io_store.get_task_output(
                input_spec.task_output_parameter.producer_task,
                input_spec.task_output_parameter.output_parameter_key,
            )

        # handle parent pipeline input parameters
        elif input_spec.HasField('component_input_parameter'):
            task_arguments[input_name] = io_store.get_parent_input(
                input_spec.component_input_parameter)

        # TODO: support dsl.ExitHandler
        elif input_spec.HasField('task_final_status'):
            raise NotImplementedError(
                "'dsl.ExitHandler' is not yet support for local execution.")

        else:
            raise ValueError(f'Missing input for parameter {input_name}.')

    # handle artifacts
    for input_name, input_spec in task_inputs_spec.artifacts.items():
        if input_spec.HasField('task_output_artifact'):
            task_arguments[input_name] = io_store.get_task_output(
                input_spec.task_output_artifact.producer_task,
                input_spec.task_output_artifact.output_artifact_key,
            )
        elif input_spec.HasField('component_input_artifact'):
            task_arguments[input_name] = io_store.get_parent_input(
                input_spec.component_input_artifact)
        else:
            raise ValueError(f'Missing input for artifact {input_name}.')

    return task_arguments


def get_dag_output_parameters(
    dag_outputs_spec: pipeline_spec_pb2.DagOutputsSpec,
    io_store: io.IOStore,
) -> Dict[str, Any]:
    """Gets the DAG output parameter values from a DagOutputsSpec and the DAG's
    IOStore.

    Args:
        dag_outputs_spec: DagOutputsSpec corresponding to the DAG.
        io_store: IOStore corresponding to the DAG.

    Returns:
        The DAG output parameters.
    """
    outputs = {}
    for root_output_key, parameter_selector_spec in dag_outputs_spec.parameters.items(
    ):
        kind = parameter_selector_spec.WhichOneof('kind')
        if kind == 'value_from_parameter':
            value_from_parameter = parameter_selector_spec.value_from_parameter
            outputs[root_output_key] = io_store.get_task_output(
                value_from_parameter.producer_subtask,
                value_from_parameter.output_parameter_key,
            )
        elif kind == 'value_from_oneof':
            raise NotImplementedError(
                "'dsl.OneOf' is not yet supported in local execution.")
        else:
            raise ValueError(
                f"Got unknown 'parameter_selector_spec' kind: {kind}")
    return outputs


def get_dag_output_artifacts(
    dag_outputs_spec: pipeline_spec_pb2.DagOutputsSpec,
    io_store: io.IOStore,
) -> Dict[str, Any]:
    """Gets the DAG output artifact values from a DagOutputsSpec and the DAG's
    IOStore.

    Args:
        dag_outputs_spec: DagOutputsSpec corresponding to the DAG.
        io_store: IOStore corresponding to the DAG.

    Returns:
        The DAG output artifacts.
    """
    outputs = {}
    for root_output_key, artifact_selector_spec in dag_outputs_spec.artifacts.items(
    ):
        len_artifact_selectors = len(artifact_selector_spec.artifact_selectors)
        if len_artifact_selectors != 1:
            raise ValueError(
                f'Expected 1 artifact in ArtifactSelectorSpec. Got: {len_artifact_selectors}'
            )
        artifact_selector = artifact_selector_spec.artifact_selectors[0]
        outputs[root_output_key] = io_store.get_task_output(
            artifact_selector.producer_subtask,
            artifact_selector.output_artifact_key,
        )
    return outputs


def get_dag_outputs(
    dag_outputs_spec: pipeline_spec_pb2.DagOutputsSpec,
    io_store: io.IOStore,
) -> Dict[str, Any]:
    """Gets the DAG output values from a DagOutputsSpec and the DAG's IOStore.

    Args:
        dag_outputs_spec: DagOutputsSpec corresponding to the DAG.
        io_store: IOStore corresponding to the DAG.

    Returns:
        The DAG outputs.
    """
    output_params = get_dag_output_parameters(
        dag_outputs_spec=dag_outputs_spec,
        io_store=io_store,
    )
    output_artifacts = get_dag_output_artifacts(
        dag_outputs_spec=dag_outputs_spec,
        io_store=io_store,
    )
    return {**output_params, **output_artifacts}


def validate_task_spec_not_loop_or_condition(
        task_spec: pipeline_spec_pb2.PipelineTaskSpec) -> None:
    if task_spec.trigger_policy.condition:
        raise NotImplementedError(
            "'dsl.Condition' is not supported by local pipeline execution.")
    elif task_spec.WhichOneof('iterator'):
        raise NotImplementedError(
            "'dsl.ParallelFor' is not supported by local pipeline execution.")
