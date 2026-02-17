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
import uuid

from kfp import dsl
from kfp.compiler import pipeline_spec_builder as builder
from kfp.dsl import base_component
from kfp.dsl import pipeline_channel
from kfp.dsl import pipeline_config
from kfp.dsl import pipeline_context
from kfp.dsl import pipeline_task
from kfp.dsl import structures
from kfp.dsl import tasks_group
from kfp.pipeline_spec import pipeline_spec_pb2


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
                ))

        with pipeline_context.Pipeline(
                self.component_spec.name) as dsl_pipeline:
            pipeline_outputs = pipeline_func(*args_list)

        if not dsl_pipeline.tasks:
            raise ValueError('Task is missing from pipeline.')

        # Validate workspace configuration if workspace features are used
        self._validate_workspace_requirements(dsl_pipeline, pipeline_config)

        # Making the pipeline group name unique to prevent name clashes with
        # templates
        pipeline_group = dsl_pipeline.groups[0]
        pipeline_group.name = uuid.uuid4().hex

        pipeline_spec, platform_spec = builder.create_pipeline_spec(
            pipeline=dsl_pipeline,
            component_spec=self.component_spec,
            pipeline_outputs=pipeline_outputs,
            pipeline_config=pipeline_config,
        )

        pipeline_root = getattr(pipeline_func, 'pipeline_root', None)
        if pipeline_root is not None:
            pipeline_spec.default_pipeline_root = pipeline_root
        if display_name is not None:
            pipeline_spec.pipeline_info.display_name = display_name
        if component_spec.description is not None:
            pipeline_spec.pipeline_info.description = component_spec.description

        self.component_spec.implementation.graph = pipeline_spec
        self.component_spec.platform_spec = platform_spec

    def _detect_workspace_usage_in_tasks(
            self, tasks: List[pipeline_task.PipelineTask]) -> bool:
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
                        if (dsl.WORKSPACE_PATH_PLACEHOLDER in input_value or
                                input_value.startswith(
                                    dsl.constants.WORKSPACE_MOUNT_PATH)):
                            return True

        return False

    def _detect_workspace_usage_in_groups(
            self, groups: List[tasks_group.TasksGroup]) -> bool:
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
            self, pipeline: pipeline_context.Pipeline,
            pipeline_config: Optional[pipeline_config.PipelineConfig]) -> None:
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
                pipeline.groups)

        # If workspace features are used, ensure workspace is configured
        if workspace_used:
            if (pipeline_config is None or pipeline_config.workspace is None or
                    not getattr(pipeline_config.workspace, 'size', None)):
                raise ValueError(
                    'Workspace features are used (e.g., dsl.WORKSPACE_PATH_PLACEHOLDER) but PipelineConfig.workspace.size is not set.'
                )

    @property
    def pipeline_spec(self) -> pipeline_spec_pb2.PipelineSpec:
        """Returns the pipeline spec of the component."""
        return self.component_spec.implementation.graph

    def execute(self, **kwargs):
        raise RuntimeError('Graph component has no local execution mode.')
