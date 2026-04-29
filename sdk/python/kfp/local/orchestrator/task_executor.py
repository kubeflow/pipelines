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
"""Shared local task execution policy for orchestrators."""

import logging
import time
from typing import Any, Dict, List, Tuple
import warnings

from kfp.local import cache as cache_module
from kfp.local import config
from kfp.local import importer_handler
from kfp.local import io
from kfp.local import status
from kfp.local import task_dispatcher
from kfp.pipeline_spec import pipeline_spec_pb2

from .orchestrator_utils import OrchestratorUtils

Outputs = Dict[str, Any]


def _warn_local_only_noops(
    task_name: str,
    task_spec: pipeline_spec_pb2.PipelineTaskSpec,
    container: pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec,
) -> None:
    """Emit warnings for task settings that have no local effect."""
    no_op_settings: List[str] = []

    if (task_spec.HasField('task_info') and task_spec.task_info.name and
            task_spec.task_info.name != task_name):
        no_op_settings.append('display_name')

    if container.HasField('resources'):
        resources = container.resources
        if resources.cpu_limit or resources.resource_cpu_limit:
            no_op_settings.append('cpu_limit')
        if resources.memory_limit or resources.resource_memory_limit:
            no_op_settings.append('memory_limit')
        if resources.cpu_request or resources.resource_cpu_request:
            no_op_settings.append('cpu_request')
        if resources.memory_request or resources.resource_memory_request:
            no_op_settings.append('memory_request')
        if resources.HasField('accelerator') and (
                resources.accelerator.type or
                resources.accelerator.resource_type or
                resources.accelerator.count or
                resources.accelerator.resource_count):
            no_op_settings.append('accelerator')

    if no_op_settings:
        warnings.warn(
            f'Task {task_name!r}: settings {no_op_settings} have '
            'no effect in the current local runner and will be ignored.',
            stacklevel=2,
        )


def execute_single_task(
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
    """Execute one local task with shared cache/retry/importer policy."""
    component_name = task_spec.component_ref.name
    component_spec = components[component_name]
    implementation = component_spec.WhichOneof('implementation')

    if implementation != 'executor_label':
        raise ValueError(
            f'Got unknown component implementation: {implementation}')

    executor_spec = executors[component_spec.executor_label]
    task_arguments = OrchestratorUtils.make_task_arguments(
        task_inputs_spec=task_spec.inputs,
        io_store=io_store,
    )

    if executor_spec.WhichOneof('spec') == 'container':
        _warn_local_only_noops(task_name, task_spec, executor_spec.container)

    local_cache = cache_module.get_local_cache()
    cache_enabled_for_task = task_spec.caching_options.enable_cache
    cache_key = None
    if (local_cache is not None and cache_enabled_for_task and
            executor_spec.WhichOneof('spec') == 'container'):
        container = executor_spec.container
        full_command = list(container.command) + list(container.args)
        env_vars = {env_var.name: env_var.value for env_var in container.env}
        cache_key = local_cache.compute_key(
            component_name=component_name,
            image=container.image,
            full_command=full_command,
            arguments=task_arguments,
            env_vars=env_vars,
            custom_cache_key=(task_spec.caching_options.cache_key or None),
        )
        cached_outputs = local_cache.get(cache_key)
        if cached_outputs is not None:
            logging.info(f'Task {task_name} cache hit (key={cache_key[:12]}); '
                         'skipping execution.')
            return cached_outputs, status.Status.SUCCESS

    retry_policy = task_spec.retry_policy
    max_retries = retry_policy.max_retry_count if retry_policy.max_retry_count > 0 else 0
    backoff_duration = retry_policy.backoff_duration.seconds if retry_policy.HasField(
        'backoff_duration') else 0
    backoff_factor = retry_policy.backoff_factor if retry_policy.backoff_factor > 0 else 2.0
    backoff_max_duration = retry_policy.backoff_max_duration.seconds if retry_policy.HasField(
        'backoff_max_duration') else 3600

    for attempt in range(max_retries + 1):
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
                raise_on_error=False,
                block_input_artifact=False,
                unique_pipeline_id=unique_pipeline_id,
            )
        else:
            raise ValueError(
                "Got unknown spec in ExecutorSpec. Only 'dsl.component', "
                "'dsl.container_component', and 'dsl.importer' are supported.")

        if task_status == status.Status.SUCCESS or attempt >= max_retries:
            if (task_status == status.Status.SUCCESS and
                    local_cache is not None and cache_key is not None):
                try:
                    local_cache.put(cache_key, outputs)
                except Exception as exc:  # pragma: no cover - defensive
                    logging.warning(f'Failed to write cache entry for task '
                                    f'{task_name}: {exc}')
            return outputs, task_status

        delay = backoff_duration * (backoff_factor**attempt)
        delay = min(delay, backoff_max_duration)
        logging.info(f'Task {task_name} failed (attempt {attempt + 1}/'
                     f'{max_retries + 1}). Retrying in {delay:.1f}s...')
        time.sleep(delay)

    return outputs, task_status
