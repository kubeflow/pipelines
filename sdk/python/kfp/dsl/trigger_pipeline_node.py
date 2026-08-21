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
"""Utility function for building TriggerPipeline Node spec."""

from typing import Any, Dict, Mapping, Optional

from kfp.dsl import pipeline_channel
from kfp.dsl import pipeline_task
from kfp.dsl import structures
from kfp.dsl import trigger_pipeline_component

RUN_ID_KEY = 'run_id'
STATE_KEY = 'state'
PIPELINE_VERSION_ID_KEY = 'pipeline_version_id'


def _infer_parameter_type(value: Any) -> tuple[str, Any]:
    """Returns (KFP type name, possibly coerced value) for a constant argument."""
    if isinstance(value, bool):
        return 'Boolean', value
    if isinstance(value, int):
        return 'Integer', value
    if isinstance(value, float):
        return 'Float', value
    if isinstance(value, dict):
        return 'JsonObject', value
    if isinstance(value, list):
        return 'JsonArray', value
    if isinstance(value, str):
        return 'String', value
    return 'String', str(value)


def trigger_pipeline(
    pipeline_name: str,
    arguments: Optional[Mapping[str, Any]] = None,
    *,
    pipeline_version_id: str = '',
    wait_for_completion: bool = True,
    poke_interval_seconds: int = 30,
) -> pipeline_task.PipelineTask:
    """Triggers an independent run of a registered pipeline.

    Unlike pipelines-as-components (nested DAG in the same Workflow), this
    creates a separate KFP Run. Use when you need Airflow-like TriggerDagRun
    semantics and a lightweight parent Workflow.

    Args:
        pipeline_name: Name of the registered pipeline to run.
        arguments: Parameter map passed to the child run. Values may be
            constants or ``PipelineParameterChannel`` from upstream tasks.
        pipeline_version_id: Optional PipelineVersion ID. When empty, the
            launcher prefers a child version whose ``display_name``/``name``
            matches the parent run's PipelineVersion; if none matches, it
            falls back to the latest child version (``created_at`` desc).
        wait_for_completion: When True, wait until the child reaches a
            terminal state; fail the parent task if not SUCCEEDED.
        poke_interval_seconds: Polling interval when waiting.

    Returns:
        A task with outputs ``run_id``, ``state``, and ``pipeline_version_id``.

    Examples::

        @dsl.pipeline
        def parent():
            task = dsl.trigger_pipeline(
                pipeline_name='get-sasrec-recommendations',
                arguments={'model_name': 'SASRecV1'},
                wait_for_completion=True,
            )
            print_op(msg=task.outputs['run_id'])
    """
    if not pipeline_name or not str(pipeline_name).strip():
        raise ValueError('pipeline_name must be a non-empty string')
    if poke_interval_seconds <= 0:
        raise ValueError('poke_interval_seconds must be positive')

    arguments = dict(arguments or {})
    component_inputs: Dict[str, structures.InputSpec] = {}
    call_inputs: Dict[str, Any] = {}

    for name, value in arguments.items():
        if not isinstance(name, str) or not name:
            raise ValueError(
                f'argument names must be non-empty strings, got {name!r}')
        if isinstance(value, pipeline_channel.PipelineParameterChannel):
            component_inputs[name] = structures.InputSpec(type=value.channel_type)
            call_inputs[name] = value
        elif isinstance(value, pipeline_channel.PipelineChannel):
            raise TypeError(
                f'trigger_pipeline arguments[{name!r}] must be a parameter '
                f'(str/int/float/bool/dict/list or PipelineParameterChannel), '
                f'not artifact channel {type(value)}')
        else:
            type_name, coerced = _infer_parameter_type(value)
            component_inputs[name] = structures.InputSpec(type=type_name)
            call_inputs[name] = coerced

    component_spec = structures.ComponentSpec(
        name='trigger-pipeline',
        implementation=structures.Implementation(
            trigger_pipeline=structures.TriggerPipelineSpec(
                pipeline_name=pipeline_name,
                pipeline_version_id=pipeline_version_id or '',
                wait_for_completion=wait_for_completion,
                poke_interval_seconds=poke_interval_seconds,
            )),
        inputs=component_inputs or None,
        outputs={
            RUN_ID_KEY: structures.OutputSpec(type='String'),
            STATE_KEY: structures.OutputSpec(type='String'),
            PIPELINE_VERSION_ID_KEY: structures.OutputSpec(type='String'),
        },
    )
    component = trigger_pipeline_component.TriggerPipelineComponent(
        component_spec=component_spec)
    return component(**call_inputs)
