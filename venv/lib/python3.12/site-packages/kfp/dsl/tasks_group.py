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

import copy
import enum
from typing import List, Optional, Union
import warnings

from kfp.dsl import for_loop
from kfp.dsl import pipeline_channel
from kfp.dsl import pipeline_context
from kfp.dsl import pipeline_task
from kfp.dsl.pipeline_channel import PipelineParameterChannel


class TasksGroupType(str, enum.Enum):
    """Types of TasksGroup."""
    PIPELINE = 'pipeline'
    CONDITION = 'condition'
    CONDITION_BRANCHES = 'condition-branches'
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
    ) -> None:
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
        # backref to parent, set when the pipeline is called in pipeline_context
        self.parent_task_group: Optional[TasksGroup] = None

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
        self.name = f'{self.group_type.value}-{group_id}'
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
    ) -> None:
        """Initializes a Condition task group."""
        super().__init__(
            group_type=TasksGroupType.EXIT_HANDLER,
            name=name,
            is_root=False,
        )

        self.exit_task = exit_task

        if self.__has_dependent_tasks():
            raise ValueError('exit_task cannot depend on any other tasks.')

        # Removing exit_task form any group
        pipeline_context.Pipeline.get_default_pipeline(
        ).remove_task_from_groups(exit_task)

        # Set is_exit_handler since the compiler might be using this attribute.
        exit_task.is_exit_handler = True

    def __has_dependent_tasks(self) -> bool:
        if self.exit_task.dependent_tasks:
            return True

        if not self.exit_task.inputs:
            return False

        for task_input in self.exit_task.inputs.values():
            if isinstance(
                    task_input,
                    PipelineParameterChannel) and task_input.task is not None:
                return True
        return False


class ConditionBranches(TasksGroup):
    _oneof_id = 0

    def __init__(self) -> None:
        super().__init__(
            group_type=TasksGroupType.CONDITION_BRANCHES,
            name=None,
            is_root=False,
        )

    def get_oneof_id(self) -> int:
        """Incrementor for uniquely identifying a OneOf for the parent
        ConditionBranches group.

        This is analogous to incrementing a unique identifier for tasks
        groups belonging to a pipeline.
        """
        self._oneof_id += 1
        return self._oneof_id


class _ConditionBase(TasksGroup):
    """Parent class for condition control flow context managers (Condition, If,
    Elif, Else).

    Args:
        condition: A list of binary operations to be combined via conjunction.
        name: The name of the condition group.
    """

    def __init__(
        self,
        conditions: List[pipeline_channel.ConditionOperation],
        name: Optional[str] = None,
    ) -> None:
        super().__init__(
            group_type=TasksGroupType.CONDITION,
            name=name,
            is_root=False,
        )
        self.conditions: List[pipeline_channel.ConditionOperation] = conditions


class If(_ConditionBase):
    """A class for creating a conditional control flow "if" block within a
    pipeline.

    Args:
        condition: A comparative expression that evaluates to True or False. At least one of the operands must be an output from an upstream task or a pipeline parameter.
        name: The name of the condition group.

    Example:
      ::

        task1 = my_component1(...)
        with dsl.If(task1.output=='pizza', 'pizza-condition'):
            task2 = my_component2(...)
    """

    def __init__(
        self,
        condition,
        name: Optional[str] = None,
    ) -> None:
        super().__init__(
            conditions=[condition],
            name=name,
        )
        if isinstance(condition, bool):
            raise ValueError(
                f'Got constant boolean {condition} as a condition. This is likely because the provided condition evaluated immediately. At least one of the operands must be an output from an upstream task or a pipeline parameter.'
            )
        copied_condition = copy.copy(condition)
        copied_condition.negate = True
        self._negated_upstream_conditions = [copied_condition]


class Condition(If):
    """Deprecated.

    Use dsl.If instead.
    """

    def __enter__(self):
        super().__enter__()
        warnings.warn(
            'dsl.Condition is deprecated. Please use dsl.If instead.',
            category=DeprecationWarning,
            stacklevel=2)
        return self


class Elif(_ConditionBase):
    """A class for creating a conditional control flow "else if" block within a
    pipeline. Can be used following an upstream dsl.If or dsl.Elif.

    Args:
        condition: A comparative expression that evaluates to True or False. At least one of the operands must be an output from an upstream task or a pipeline parameter.
        name: The name of the condition group.

    Example:
      ::

        task1 = my_component1(...)
        task2 = my_component2(...)
        with dsl.If(task1.output=='pizza', 'pizza-condition'):
            task3 = my_component3(...)

        with dsl.Elif(task2.output=='pasta', 'pasta-condition'):
            task4 = my_component4(...)
    """

    def __init__(
        self,
        condition,
        name: Optional[str] = None,
    ) -> None:
        prev_cond = pipeline_context.Pipeline.get_default_pipeline(
        ).get_last_tasks_group()
        if not isinstance(prev_cond, (Condition, If, Elif)):
            # prefer pushing toward dsl.If rather than dsl.Condition for syntactic consistency with the if-elif-else keywords in Python
            raise InvalidControlFlowException(
                'dsl.Elif can only be used following an upstream dsl.If or dsl.Elif.'
            )

        if isinstance(condition, bool):
            raise ValueError(
                f'Got constant boolean {condition} as a condition. This is likely because the provided condition evaluated immediately. At least one of the operands must be an output from an upstream task or a pipeline parameter.'
            )

        copied_condition = copy.copy(condition)
        copied_condition.negate = True
        self._negated_upstream_conditions = _shallow_copy_list_of_binary_operations(
            prev_cond._negated_upstream_conditions) + [copied_condition]

        conditions = _shallow_copy_list_of_binary_operations(
            prev_cond._negated_upstream_conditions)
        conditions.append(condition)

        super().__init__(
            conditions=conditions,
            name=name,
        )

    def __enter__(self):
        if not pipeline_context.Pipeline.get_default_pipeline():
            raise ValueError('Default pipeline not defined.')

        pipeline = pipeline_context.Pipeline.get_default_pipeline()

        maybe_make_and_insert_conditional_branches_group(pipeline)

        self._make_name_unique()
        pipeline.push_tasks_group(self)
        return self


class Else(_ConditionBase):
    """A class for creating a conditional control flow "else" block within a
    pipeline. Can be used following an upstream dsl.If or dsl.Elif.

    Args:
        name: The name of the condition group.

    Example:
      ::

        task1 = my_component1(...)
        task2 = my_component2(...)
        with dsl.If(task1.output=='pizza', 'pizza-condition'):
            task3 = my_component3(...)

        with dsl.Elif(task2.output=='pasta', 'pasta-condition'):
            task4 = my_component4(...)

        with dsl.Else():
            my_component5(...)
    """

    def __init__(
        self,
        name: Optional[str] = None,
    ) -> None:
        prev_cond = pipeline_context.Pipeline.get_default_pipeline(
        ).get_last_tasks_group()

        # if it immediately follows as TasksGroup, this is because it immediately
        # follows Else in the user code and we wrap Else in a TasksGroup
        if isinstance(prev_cond, ConditionBranches):
            # prefer pushing toward dsl.If rather than dsl.Condition for syntactic consistency with the if-elif-else keywords in Python
            raise InvalidControlFlowException(
                'Cannot use dsl.Else following another dsl.Else. dsl.Else can only be used following an upstream dsl.If or dsl.Elif.'
            )
        if not isinstance(prev_cond, (Condition, If, Elif)):
            # prefer pushing toward dsl.If rather than dsl.Condition for syntactic consistency with the if-elif-else keywords in Python
            raise InvalidControlFlowException(
                'dsl.Else can only be used following an upstream dsl.If or dsl.Elif.'
            )

        super().__init__(
            conditions=prev_cond._negated_upstream_conditions,
            name=name,
        )

    def __enter__(self):
        if not pipeline_context.Pipeline.get_default_pipeline():
            raise ValueError('Default pipeline not defined.')

        pipeline = pipeline_context.Pipeline.get_default_pipeline()

        maybe_make_and_insert_conditional_branches_group(pipeline)

        self._make_name_unique()
        pipeline.push_tasks_group(self)
        return self

    def __exit__(self, *unused_args):
        pipeline = pipeline_context.Pipeline.get_default_pipeline()
        pipeline.pop_tasks_group()

        # since this is an else, also pop off the parent dag for conditional branches
        # this parent TasksGroup is not a context manager, so we simulate its
        # __exit__ call with this
        pipeline.pop_tasks_group()


def maybe_make_and_insert_conditional_branches_group(
        pipeline: 'pipeline_context.Pipeline') -> None:

    already_has_pipeline_wrapper = isinstance(
        pipeline.get_last_tasks_group(),
        Elif,
    )
    if already_has_pipeline_wrapper:
        return

    condition_wrapper_group = ConditionBranches()
    condition_wrapper_group._make_name_unique()

    # swap outer and inner group ids so that numbering stays sequentially consistent with how such hypothetical code would be authored
    def swap_group_ids(parent: TasksGroup, cond: TasksGroup):
        parent_name, parent_id = parent.name.rsplit('-', 1)
        cond_name, cond_id = cond.name.split('-')
        cond.name = f'{cond_name}-{parent_id}'
        parent.name = f'{parent_name}-{cond_id}'

    # replace last pushed group (If or Elif) with condition group
    last_pushed_group = pipeline.groups[-1].groups.pop()
    swap_group_ids(condition_wrapper_group, last_pushed_group)
    pipeline.push_tasks_group(condition_wrapper_group)

    # then repush (__enter__) and pop (__exit__) the last pushed group
    # before the wrapper to emulate re-entering and exiting its context
    pipeline.push_tasks_group(last_pushed_group)
    pipeline.pop_tasks_group()


class InvalidControlFlowException(Exception):
    pass


def _shallow_copy_list_of_binary_operations(
    operations: List[pipeline_channel.ConditionOperation]
) -> List[pipeline_channel.ConditionOperation]:
    # shallow copy is sufficient to allow us to invert the negate flag of a ConditionOperation without affecting copies. deep copy not needed and would result in many copies of the full pipeline since PipelineChannels hold references to the pipeline.
    return [copy.copy(operation) for operation in operations]


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
    ) -> None:
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

        if isinstance(items, pipeline_channel.PipelineParameterChannel):
            self.loop_argument = for_loop.LoopParameterArgument.from_pipeline_channel(
                items)
            self.items_is_pipeline_channel = True
        elif isinstance(items, pipeline_channel.PipelineArtifactChannel):
            self.loop_argument = for_loop.LoopArtifactArgument.from_pipeline_channel(
                items)
            self.items_is_pipeline_channel = True
        else:
            self.loop_argument = for_loop.LoopParameterArgument.from_raw_items(
                raw_items=items,
                name_code=pipeline_context.Pipeline.get_default_pipeline()
                .get_next_group_id(),
            )
            self.items_is_pipeline_channel = False
        # TODO: support artifact constants here.

        self.parallelism_limit = parallelism

    def __enter__(
        self
    ) -> Union[for_loop.LoopParameterArgument, for_loop.LoopArtifactArgument]:
        super().__enter__()
        return self.loop_argument
