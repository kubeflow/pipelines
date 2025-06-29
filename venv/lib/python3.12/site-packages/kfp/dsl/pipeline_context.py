# Copyright 2021-2022 The Kubeflow Authors
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
"""Definition for Pipeline."""

import functools
import os
from typing import Callable, Optional

from kfp.dsl import component_factory
from kfp.dsl import pipeline_config
from kfp.dsl import pipeline_task
from kfp.dsl import tasks_group
from kfp.dsl import utils


def pipeline(
        func: Optional[Callable] = None,
        *,
        name: Optional[str] = None,
        description: Optional[str] = None,
        pipeline_root: Optional[str] = None,
        display_name: Optional[str] = None,
        pipeline_config: pipeline_config.PipelineConfig = None) -> Callable:
    """Decorator used to construct a pipeline.

    Example
      ::

        @pipeline(
          name='my-pipeline',
          description='My ML Pipeline.'
          pipeline_root='gs://my-bucket/my-output-path'
        )
        def my_pipeline(a: str, b: int):
          ...

    Args:
        func: The Python function that defines a pipeline.
        name: The pipeline name. Defaults to a sanitized version of the
            decorated function name.
        description: A human-readable description of the pipeline.
        pipeline_root: The root directory from which to read input and output
            parameters and artifacts.
        display_name: A human-readable name for the pipeline.
        pipeline_config: Pipeline-level config options.
    """
    if func is None:
        return functools.partial(
            pipeline,
            name=name,
            description=description,
            pipeline_root=pipeline_root,
            display_name=display_name,
            pipeline_config=pipeline_config,
        )

    if pipeline_root:
        func.pipeline_root = pipeline_root

    return component_factory.create_graph_component_from_func(
        func,
        name=name,
        description=description,
        display_name=display_name,
        pipeline_config=pipeline_config,
    )


class Pipeline:
    """A pipeline contains a list of tasks.

    This class is not supposed to be used by pipeline authors since pipeline
    authors can use pipeline functions (decorated with @pipeline) to reference
    their pipelines.
    This class is useful for implementing a compiler. For example, the compiler
    can use the following to get the pipeline object and its tasks:

    Example:
      ::

        with Pipeline() as p:
            pipeline_func(*args_list)

        traverse(p.tasks)

    Attributes:
        name:
        tasks:
        groups:
    """

    # _default_pipeline is set when the compiler runs "with Pipeline()"
    _default_pipeline = None

    @staticmethod
    def get_default_pipeline():
        """Gets the default pipeline."""
        return Pipeline._default_pipeline

    # _execution_caching_default can be disabled via the click option --disable-execution-caching-by-default
    # or the env var KFP_DISABLE_EXECUTION_CACHING_BY_DEFAULT.
    # align with click's treatment of env vars for boolean flags.
    # per click doc, "1", "true", "t", "yes", "y", and "on" are all converted to True
    _execution_caching_default = not str(
        os.getenv('KFP_DISABLE_EXECUTION_CACHING_BY_DEFAULT')).strip().lower(
        ) in {'1', 'true', 't', 'yes', 'y', 'on'}

    @staticmethod
    def get_execution_caching_default():
        """Gets the default execution caching."""
        return Pipeline._execution_caching_default

    def __init__(self, name: str):
        """Creates a new instance of Pipeline.

        Args:
            name: The name of the pipeline.
        """
        self.name = name
        self.tasks = {}
        # Add the root group.
        self.groups = [
            tasks_group.TasksGroup(
                group_type=tasks_group.TasksGroupType.PIPELINE,
                name=name,
                is_root=True)
        ]
        self._group_id = 0

    def __enter__(self):

        if Pipeline._default_pipeline:
            raise Exception('Nested pipelines are not allowed.')

        Pipeline._default_pipeline = self

        def register_task_and_generate_id(task: pipeline_task.PipelineTask):
            return self.add_task(
                task=task,
                add_to_group=not getattr(task, 'is_exit_handler', False))

        self._old_register_task_handler = (
            pipeline_task.PipelineTask._register_task_handler)
        pipeline_task.PipelineTask._register_task_handler = (
            register_task_and_generate_id)
        return self

    def __exit__(self, *unused_args):

        Pipeline._default_pipeline = None
        pipeline_task.PipelineTask._register_task_handler = (
            self._old_register_task_handler)

    def add_task(
        self,
        task: pipeline_task.PipelineTask,
        add_to_group: bool,
    ) -> str:
        """Adds a new task.

        Args:
            task: A PipelineTask instance.
            add_to_group: Whether add the task into the current group. Expect
                True for all tasks expect for exit handler.

        Returns:
            A unique task name.
        """
        # Sanitizing the task name.
        # Technically this could be delayed to the compilation stage, but string
        # serialization of PipelineChannels make unsanitized names problematic.
        task_name = utils.maybe_rename_for_k8s(task.component_spec.name)
        #If there is an existing task with this name then generate a new name.
        task_name = utils.make_name_unique_by_adding_index(
            task_name, list(self.tasks.keys()), '-')
        if task_name == '':
            task_name = utils.make_name_unique_by_adding_index(
                'task', list(self.tasks.keys()), '-')

        self.tasks[task_name] = task
        if add_to_group:
            task.parent_task_group = self.groups[-1]
            self.groups[-1].tasks.append(task)

        return task_name

    def push_tasks_group(self, group: 'tasks_group.TasksGroup'):
        """Pushes a TasksGroup into the stack.

        Args:
            group: A TasksGroup. Typically it is one of ExitHandler, Condition,
                and ParallelFor.
        """
        group.parent_task_group = self.get_parent_group()
        self.groups[-1].groups.append(group)
        self.groups.append(group)

    def pop_tasks_group(self):
        """Removes the current TasksGroup from the stack."""
        del self.groups[-1]

    def get_last_tasks_group(self) -> Optional['tasks_group.TasksGroup']:
        """Gets the last TasksGroup added to the pipeline at the current level
        of the pipeline definition."""
        groups = self.groups[-1].groups
        return groups[-1] if groups else None

    def get_parent_group(self) -> 'tasks_group.TasksGroup':
        return self.groups[-1]

    def remove_task_from_groups(self, task: pipeline_task.PipelineTask):
        """Removes a task from the pipeline.

        This is useful for excluding exit handler from the pipeline.
        """
        for group in self.groups:
            group.remove_task_recursive(task)

    def get_next_group_id(self) -> str:
        """Gets the next id for a new group."""
        self._group_id += 1
        return str(self._group_id)
