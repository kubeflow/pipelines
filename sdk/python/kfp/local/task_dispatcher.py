# Copyright 2023 The Kubeflow Authors
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
"""Code for dispatching a local task execution."""
import logging
from typing import Any, Dict, Tuple

from kfp import local
from kfp.local import config
from kfp.local import docker_task_handler
from kfp.local import executor_input_utils
from kfp.local import executor_output_utils
from kfp.local import logging_utils
from kfp.local import placeholder_utils
from kfp.local import status
from kfp.local import subprocess_task_handler
from kfp.local import task_handler_interface
from kfp.local import utils
from kfp.pipeline_spec import pipeline_spec_pb2


def run_single_task(
    pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    arguments: Dict[str, Any],
) -> Dict[str, Any]:
    """Runs a single component from its compiled PipelineSpec.

    Args:
        pipeline_spec: The PipelineSpec of the component to run.
        arguments: The runtime arguments.

    Returns:
        A LocalTask instance.
    """
    config.LocalExecutionConfig.validate()
    component_name, component_spec = list(pipeline_spec.components.items())[0]
    executor_spec = get_executor_spec(
        pipeline_spec,
        component_spec.executor_label,
    )
    executor_spec = utils.struct_to_executor_spec(executor_spec)
    pipeline_resource_name = executor_input_utils.get_local_pipeline_resource_name(
        pipeline_spec.pipeline_info.name)

    # all global state should be accessed here
    # do not access local config state downstream
    outputs, _ = run_single_task_implementation(
        pipeline_resource_name=pipeline_resource_name,
        component_name=component_name,
        component_spec=component_spec,
        executor_spec=executor_spec,
        arguments=arguments,
        pipeline_root=config.LocalExecutionConfig.instance.pipeline_root,
        runner=config.LocalExecutionConfig.instance.runner,
        raise_on_error=config.LocalExecutionConfig.instance.raise_on_error,
        block_input_artifact=True,
        unique_pipeline_id=placeholder_utils.make_random_id())
    return outputs


def get_executor_spec(
    pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    executor_label: str,
) -> pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec:
    return pipeline_spec.deployment_spec['executors'][executor_label]


Outputs = Dict[str, Any]


def run_single_task_implementation(
    pipeline_resource_name: str,
    component_name: str,
    component_spec: pipeline_spec_pb2.ComponentSpec,
    executor_spec: pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec,
    arguments: Dict[str, Any],
    pipeline_root: str,
    runner: config.LocalRunnerType,
    raise_on_error: bool,
    block_input_artifact: bool,
    unique_pipeline_id: str,
) -> Tuple[Outputs, status.Status]:
    """The implementation of a single component runner.

    Returns a tuple of (outputs, status). If status is FAILURE, outputs
    is an empty dictionary.
    """

    task_resource_name = executor_input_utils.get_local_task_resource_name(
        component_name)
    task_root = executor_input_utils.construct_local_task_root(
        pipeline_root=pipeline_root,
        pipeline_resource_name=pipeline_resource_name,
        task_resource_name=task_resource_name,
    )
    executor_input = executor_input_utils.construct_executor_input(
        component_spec=component_spec,
        arguments=arguments,
        task_root=task_root,
        block_input_artifact=block_input_artifact,
    )

    container = executor_spec.container
    image = container.image
    full_command = list(container.command) + list(container.args)

    executor_input_dict = executor_input_utils.executor_input_to_dict(
        executor_input=executor_input,
        component_spec=component_spec,
    )
    full_command = placeholder_utils.replace_placeholders(
        full_command=full_command,
        executor_input_dict=executor_input_dict,
        pipeline_resource_name=pipeline_resource_name,
        task_resource_name=task_resource_name,
        pipeline_root=pipeline_root,
        unique_pipeline_id=unique_pipeline_id,
    )

    runner_type = type(runner)
    task_handler_map: Dict[
        local.LocalRunnerType, task_handler_interface.ITaskHandler] = {
            local.SubprocessRunner:
                subprocess_task_handler.SubprocessTaskHandler,
            local.DockerRunner:
                docker_task_handler.DockerTaskHandler,
        }
    TaskHandler = task_handler_map[runner_type]

    with logging_utils.local_logger_context():
        task_name_for_logs = logging_utils.format_task_name(task_resource_name)

        logging.info(f'Executing task {task_name_for_logs}')
        task_handler = TaskHandler(
            image=image,
            full_command=full_command,
            pipeline_root=pipeline_root,
            runner=runner,
        )

        # trailing newline helps visually separate subprocess logs
        logging.info(f'Streamed logs:\n')

        with logging_utils.indented_print():
            # subprocess logs printed here
            task_status = task_handler.run()

        if task_status == status.Status.SUCCESS:
            logging.info(
                f'Task {task_name_for_logs} finished with status {logging_utils.format_status(task_status)}'
            )

            outputs = executor_output_utils.get_outputs_for_task(
                executor_input=executor_input,
                component_spec=component_spec,
            )
            if outputs:
                output_string = [
                    f'Task {task_name_for_logs} outputs:',
                    *logging_utils.make_log_lines_for_outputs(outputs),
                ]
                logging.info('\n'.join(output_string))
            else:
                logging.info(f'Task {task_name_for_logs} has no outputs')

        elif task_status == status.Status.FAILURE:
            msg = f'Task {task_name_for_logs} finished with status {logging_utils.format_status(task_status)}'
            if raise_on_error:
                raise RuntimeError(msg)
            else:
                logging.error(msg)
            outputs = {}

        else:
            # for developers; user should never hit this
            raise ValueError(f'Got unknown status: {task_status}')
        logging_utils.print_horizontal_line()

        return outputs, task_status
