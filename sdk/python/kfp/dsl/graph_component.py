# Copyright 2022 The Kubeflow Authors
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
"""Pipeline as a component (aka graph component)."""

import inspect
from typing import Callable, List, Optional

from kfp import dsl
from kfp.dsl import (
    base_component,
    pipeline_channel,
    pipeline_config,
    pipeline_context,
    pipeline_task,
    structures,
    tasks_group,
)
from kfp.dsl.types import type_utils


class GraphComponent(base_component.BaseComponent):
    """A component defined via @dsl.pipeline decorator.

    Attribute:
        pipeline_func: The function that becomes the implementation of this component.
    """

    def __init__(
        self,
        component_spec: structures.ComponentSpec,
        pipeline_func: Callable,
        display_name: Optional[str] = None,
        pipeline_config: pipeline_config.PipelineConfig = None,
    ):
        super().__init__(component_spec=component_spec)
        self.pipeline_func = pipeline_func
        self.pipeline_config = pipeline_config

        args_list = []
        signature = inspect.signature(pipeline_func)

        for arg_name in signature.parameters:
            input_spec = component_spec.inputs[arg_name]
            args_list.append(
                pipeline_channel.create_pipeline_channel(
                    name=arg_name,
                    channel_type=input_spec.type,
                    is_artifact_list=input_spec.is_artifact_list,
                )
            )

        with pipeline_context.Pipeline(
            self.component_spec.name
        ) as dsl_pipeline:
            pipeline_outputs = pipeline_func(*args_list)

        if not dsl_pipeline.tasks:
            raise ValueError('Task is missing from pipeline.')

        # Return validation belongs to graph capture, not a removed IR backend.
        returned = (
            pipeline_outputs._asdict()
            if hasattr(pipeline_outputs, '_asdict')
            else pipeline_outputs
            if isinstance(pipeline_outputs, dict)
            else {'Output': pipeline_outputs}
            if pipeline_outputs is not None
            else {}
        )
        for name, output in (component_spec.outputs or {}).items():
            if name not in returned:
                raise ValueError(
                    f'Pipeline output {name!r} is missing; return the declared output channel.'
                )
            type_utils.verify_type_compatibility(
                given_value=returned[name],
                expected_spec=output,
                error_message_prefix=f'Incompatible return type provided for output of pipeline {self.name!r}: ',
            )

        # Validate workspace configuration if workspace features are used
        self._validate_workspace_requirements(dsl_pipeline, pipeline_config)

        self.component_spec.implementation.graph = structures.CapturedGraph(
            pipeline=dsl_pipeline,
            outputs=pipeline_outputs,
            config=pipeline_config,
            pipeline_root=getattr(pipeline_func, 'pipeline_root', None),
            display_name=display_name,
        )

    def _detect_workspace_usage_in_tasks(
        self, tasks: List[pipeline_task.PipelineTask]
    ) -> bool:
        """Detects if any task in the list uses workspace features.

        Args:
            tasks: List of pipeline tasks to check for workspace usage.

        Returns:
            True if any task uses workspace features, False otherwise.
        """

        for task in tasks:
            if hasattr(task, 'inputs') and task.inputs:
                for input_value in task.inputs.values():
                    if isinstance(input_value, str):
                        if (
                            dsl.WORKSPACE_PATH_PLACEHOLDER in input_value
                            or input_value.startswith(
                                dsl.constants.WORKSPACE_MOUNT_PATH
                            )
                        ):
                            return True

        return False

    def _detect_workspace_usage_in_groups(
        self, groups: List[tasks_group.TasksGroup]
    ) -> bool:
        """Detects if any task group uses workspace features.

        Args:
            groups: List of task groups to check for workspace usage.

        Returns:
            True if any task uses workspace features, False otherwise.
        """
        for group in groups:
            # Check tasks in the current group
            if hasattr(group, 'tasks') and group.tasks:
                if self._detect_workspace_usage_in_tasks(group.tasks):
                    return True

            # Recursively check nested groups
            if hasattr(group, 'groups') and group.groups:
                if self._detect_workspace_usage_in_groups(group.groups):
                    return True

        return False

    def _validate_workspace_requirements(
        self,
        pipeline: pipeline_context.Pipeline,
        pipeline_config: Optional[pipeline_config.PipelineConfig],
    ) -> None:
        """Validates that workspace is configured if workspace features are
        used in the pipeline.

        Args:
            pipeline: The pipeline instance to validate.
            pipeline_config: The pipeline configuration.

        Raises:
            ValueError: If workspace features are used but workspace is not configured in PipelineConfig.
        """

        workspace_used = False

        if pipeline.groups:
            workspace_used = self._detect_workspace_usage_in_groups(
                pipeline.groups
            )

        # If workspace features are used, ensure workspace is configured
        if workspace_used:
            if (
                pipeline_config is None
                or pipeline_config.workspace is None
                or not getattr(pipeline_config.workspace, 'size', None)
            ):
                raise ValueError(
                    'Workspace features are used (e.g., dsl.WORKSPACE_PATH_PLACEHOLDER) but PipelineConfig.workspace.size is not set.'
                )

    @property
    def pipeline_spec(self) -> structures.CapturedGraph:
        """Returns the captured DSL graph, not a serialized PipelineSpec IR."""
        return self.component_spec.implementation.graph
