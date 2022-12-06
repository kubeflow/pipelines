# Copyright 2021 The Kubeflow Authors
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
"""Definition for TasksGroup."""

import enum
from typing import Optional, Union

from kfp.components import for_loop
from kfp.components import pipeline_channel
from kfp.components import pipeline_context
from kfp.components import pipeline_task


class TasksGroupType(str, enum.Enum):
    """Types of TasksGroup."""
    PIPELINE = 'pipeline'
    CONDITION = 'condition'
    FOR_LOOP = 'for-loop'
    EXIT_HANDLER = 'exit-handler'


class TasksGroup:
    """Represents a logical group of tasks and groups of TasksGroups.

    This class is the base class for groups of tasks, such as tasks
    sharing an exit handler, a condition branch, or a loop. This class
    is not supposed to be used by pipeline authors. It is useful for
    implementing a compiler.

    Attributes:
        group_type: The type of the TasksGroup.
        tasks: A list of all PipelineTasks in this group.
        groups: A list of TasksGroups in this group.
        display_name: The optional user given name of the group.
        dependencies: A list of tasks or groups this group depends on.
        is_root: If TasksGroup is root group.
    """

    def __init__(
        self,
        group_type: TasksGroupType,
        name: Optional[str] = None,
        is_root: bool = False,
    ):
        """Create a new instance of TasksGroup.

        Args:
          group_type: The type of the group.
          name: The name of the group. Used as display name in UI.
        """
        self.group_type = group_type
        self.tasks = []
        self.groups = []
        self.display_name = name
        self.dependencies = []
        self.is_root = is_root

    def __enter__(self):
        if not pipeline_context.Pipeline.get_default_pipeline():
            raise ValueError('Default pipeline not defined.')

        self._make_name_unique()

        pipeline_context.Pipeline.get_default_pipeline().push_tasks_group(self)
        return self

    def __exit__(self, *unused_args):
        pipeline_context.Pipeline.get_default_pipeline().pop_tasks_group()

    def _make_name_unique(self):
        """Generates a unique TasksGroup name in the pipeline."""
        if not pipeline_context.Pipeline.get_default_pipeline():
            raise ValueError('Default pipeline not defined.')

        group_id = pipeline_context.Pipeline.get_default_pipeline(
        ).get_next_group_id()
        self.name = f'{self.group_type}-{group_id}'
        self.name = self.name.replace('_', '-')

    def remove_task_recursive(self, task: pipeline_task.PipelineTask):
        """Removes a task from the group recursively."""
        if self.tasks and task in self.tasks:
            self.tasks.remove(task)
        for group in self.groups or []:
            group.remove_task_recursive(task)


class ExitHandler(TasksGroup):
    """A class for setting an exit handler task that is invoked upon exiting a
    group of other tasks.

    Args:
        exit_task: The task that is invoked after exiting a group of other tasks.
        name: The name of the exit handler group.

    Example:
      ::

        exit_task = ExitComponent(...)
        with ExitHandler(exit_task):
            task1 = my_component1(...)
            task2 = my_component2(...)
    """

    def __init__(
        self,
        exit_task: pipeline_task.PipelineTask,
        name: Optional[str] = None,
    ):
        """Initializes a Condition task group."""
        super().__init__(
            group_type=TasksGroupType.EXIT_HANDLER,
            name=name,
            is_root=False,
        )

        if exit_task.dependent_tasks:
            raise ValueError('exit_task cannot depend on any other tasks.')

        # Removing exit_task form any group
        pipeline_context.Pipeline.get_default_pipeline(
        ).remove_task_from_groups(exit_task)

        # Set is_exit_handler since the compiler might be using this attribute.
        exit_task.is_exit_handler = True

        self.exit_task = exit_task


class Condition(TasksGroup):
    """A class for creating conditional control flow within a pipeline
    definition.

    Args:
        condition: A comparative expression that evaluates to True or False. At least one of the operands must be an output from an upstream task or a pipeline parameter.
        name: The name of the condition group.

    Example:
      ::

        task1 = my_component1(...)
        with Condition(task1.output=='pizza', 'pizza-condition'):
            task2 = my_component2(...)
    """

    def __init__(
        self,
        condition: pipeline_channel.ConditionOperator,
        name: Optional[str] = None,
    ):
        """Initializes a conditional task group."""
        super().__init__(
            group_type=TasksGroupType.CONDITION,
            name=name,
            is_root=False,
        )
        self.condition = condition


class ParallelFor(TasksGroup):
    """A class for creating parallelized for loop control flow over a static
    set of items within a pipeline definition.

    Args:
        items: The items to loop over. It can be either a constant Python list or a list output from an upstream task.
        name: The name of the for loop group.
        parallelism: The maximum number of concurrent iterations that can be scheduled for execution. A value of 0 represents unconstrained parallelism (default is unconstrained).

    Example:
      ::

        with dsl.ParallelFor(
          items=[{'a': 1, 'b': 10}, {'a': 2, 'b': 20}],
          parallelism=1
        ) as item:
            task1 = my_component(..., number=item.a)
            task2 = my_component(..., number=item.b)

    In the example, the group of tasks containing ``task1`` and ``task2`` would
    be executed twice, once with case ``args=[{'a': 1, 'b': 10}]`` and once with
    case ``args=[{'a': 2, 'b': 20}]``. The ``parallelism=1`` setting causes only
    1 execution to be scheduled at a time.
    """

    def __init__(
        self,
        items: Union[for_loop.ItemList, pipeline_channel.PipelineChannel],
        name: Optional[str] = None,
        parallelism: Optional[int] = None,
    ):
        """Initializes a for loop task group."""
        parallelism = parallelism or 0
        if parallelism < 0:
            raise ValueError(
                f'ParallelFor parallelism must be >= 0. Got: {parallelism}.')

        super().__init__(
            group_type=TasksGroupType.FOR_LOOP,
            name=name,
            is_root=False,
        )

        if isinstance(items, pipeline_channel.PipelineChannel):
            self.loop_argument = for_loop.LoopArgument.from_pipeline_channel(
                items)
            self.items_is_pipeline_channel = True
        else:
            self.loop_argument = for_loop.LoopArgument.from_raw_items(
                raw_items=items,
                name_code=pipeline_context.Pipeline.get_default_pipeline()
                .get_next_group_id(),
            )
            self.items_is_pipeline_channel = False

        self.parallelism_limit = parallelism

    def __enter__(self) -> for_loop.LoopArgument:
        super().__enter__()
        return self.loop_argument
