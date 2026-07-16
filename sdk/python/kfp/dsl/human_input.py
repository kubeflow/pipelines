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
"""Human input (suspend) task for KFP pipelines.

A human input task pauses pipeline execution at a specific step and waits for
a human operator to supply parameter values via the KFP UI (or API) before the
run continues.  The supplied values become the task's outputs and can be
referenced by downstream tasks exactly like any other component output.

Example – static approval gate::

    import kfp
    from kfp import dsl

    @dsl.pipeline
    def approval_pipeline():
        gate = dsl.human_input(
            name='approval-gate',
            parameters={
                'decision': dsl.HumanInputParameter(
                    description='Approve or reject the deployment?',
                    enum=['YES', 'NO'],
                    default='NO',
                ),
            },
        )
        deploy_op(approved=gate.outputs['decision'])
"""

from typing import Dict, Optional

from kfp.dsl import base_component
from kfp.dsl import pipeline_task
from kfp.dsl import structures

# Sentinel image recognised by the backend Argo compiler to generate a
# suspend template instead of a regular container task.  This value MUST be
# kept in sync with ``humanInputSentinelImage`` in
# ``backend/src/v2/compiler/argocompiler/suspend.go``.
HUMAN_INPUT_SENTINEL_IMAGE = 'kfp://human-input'

# Type alias for backwards-compatible import
HumanInputParameter = structures.HumanInputParameterSpec


class _HumanInputComponent(base_component.BaseComponent):
    """Internal component class backing a human input task.

    Users should create human input tasks through :func:`human_input`
    rather than instantiating this class directly.
    """

    def __init__(self, component_spec: structures.ComponentSpec) -> None:
        super().__init__(component_spec=component_spec)

    def execute(self, **kwargs):
        raise NotImplementedError(
            'Human input tasks cannot be executed locally; they require the '
            'Argo-backed KFP runtime to suspend the workflow and wait for '
            'operator input.')


def human_input(
    name: str,
    parameters: Dict[str, structures.HumanInputParameterSpec],
    display_name: Optional[str] = None,
) -> pipeline_task.PipelineTask:
    """Creates a pipeline task that pauses for human-supplied parameter values.

    When the pipeline reaches this task the Argo Workflows runtime suspends
    the node.  A human operator can then open the task's node-details panel in
    the KFP UI, fill in the declared parameters, and click *Submit* to resume
    the run.  The submitted values become this task's output parameters and can
    be consumed by downstream tasks.

    Args:
        name: Name for the task, used as its identity within the pipeline DAG.
        parameters: Mapping from output parameter name to a
            :class:`HumanInputParameter` (alias for
            :class:`~kfp.dsl.structures.HumanInputParameterSpec`) that
            describes the parameter's constraints (description, default value,
            and/or enumerated choices).
        display_name: Optional human-readable label shown in the KFP UI.  When
            omitted the *name* is used as the display label.

    Returns:
        A :class:`~kfp.dsl.pipeline_task.PipelineTask` whose
        :attr:`~kfp.dsl.pipeline_task.PipelineTask.outputs` dictionary
        contains a :class:`~kfp.dsl.pipeline_channel.PipelineChannel` for
        each declared parameter.  Reference these channels as inputs to
        downstream tasks to wire up the data dependency.

    Raises:
        ValueError: If *parameters* is empty.

    Examples::

        @dsl.pipeline
        def my_pipeline():
            gate = dsl.human_input(
                name='approval-gate',
                parameters={
                    'decision': dsl.HumanInputParameter(
                        description='Approve or reject?',
                        enum=['YES', 'NO'],
                        default='NO',
                    ),
                },
            )
            next_step(approved=gate.outputs['decision'])
    """
    if not parameters:
        raise ValueError(
            'human_input() requires at least one parameter to be declared.')

    # Build the component's output spec - all human-input outputs are strings.
    outputs = {
        param_name: structures.OutputSpec(type='String')
        for param_name in parameters
    }

    component_spec = structures.ComponentSpec(
        name=name,
        implementation=structures.Implementation(
            human_input=structures.HumanInputSpec(parameters=parameters)),
        inputs=None,
        outputs=outputs,
    )
    component = _HumanInputComponent(component_spec=component_spec)
    task = component()
    if display_name:
        task.set_display_name(display_name)
    return task
