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
"""Utility methods for compiler implementation that is IR-agnostic."""

import collections
from copy import deepcopy
from typing import Dict, List, Mapping, Set, Tuple, Union

from kfp.components import for_loop
from kfp.components import pipeline_channel
from kfp.components import pipeline_context
from kfp.components import pipeline_task
from kfp.components import tasks_group

GroupOrTaskType = Union[tasks_group.TasksGroup, pipeline_task.PipelineTask]


def get_all_groups(
    root_group: tasks_group.TasksGroup,) -> List[tasks_group.TasksGroup]:
    """Gets all groups (not including tasks) in a pipeline.

    Args:
        root_group: The root group of a pipeline.

    Returns:
        A list of all groups in topological order (parent first).
    """
    all_groups = []

    def _get_all_groups_helper(
        group: tasks_group.TasksGroup,
        all_groups: List[tasks_group.TasksGroup],
    ):
        all_groups.append(group)
        for group in group.groups:
            _get_all_groups_helper(group, all_groups)

    _get_all_groups_helper(root_group, all_groups)
    return all_groups


def get_parent_groups(
    root_group: tasks_group.TasksGroup,
) -> Tuple[Mapping[str, List[GroupOrTaskType]], Mapping[str,
                                                        List[GroupOrTaskType]]]:
    """Get parent groups that contain the specified tasks.

    Each pipeline has a root group. Each group has a list of tasks (leaf)
    and groups.
    This function traverse the tree and get ancestor groups for all tasks.

    Args:
        root_group: The root group of a pipeline.

    Returns:
        A tuple. The first item is a mapping of task names to parent groups,
        and second item is a mapping of group names to parent groups.
        A list of parent groups is a list of ancestor groups including the
        task/group itself. The list is sorted in a way that the farthest
        parent group is the first and task/group itself is the last.
    """

    def _get_parent_groups_helper(
        current_groups: List[tasks_group.TasksGroup],
        tasks_to_groups: Dict[str, List[GroupOrTaskType]],
        groups_to_groups: Dict[str, List[GroupOrTaskType]],
    ) -> None:
        root_group = current_groups[-1]
        for group in root_group.groups:

            groups_to_groups[group.name] = [x.name for x in current_groups
                                           ] + [group.name]
            current_groups.append(group)

            _get_parent_groups_helper(
                current_groups=current_groups,
                tasks_to_groups=tasks_to_groups,
                groups_to_groups=groups_to_groups,
            )
            del current_groups[-1]

        for task in root_group.tasks:
            tasks_to_groups[task.name] = [x.name for x in current_groups
                                         ] + [task.name]

    tasks_to_groups = {}
    groups_to_groups = {}
    current_groups = [root_group]

    _get_parent_groups_helper(
        current_groups=current_groups,
        tasks_to_groups=tasks_to_groups,
        groups_to_groups=groups_to_groups,
    )
    return (tasks_to_groups, groups_to_groups)


# TODO: do we really need this?
def get_condition_channels_for_tasks(
    root_group: tasks_group.TasksGroup,
) -> Mapping[str, Set[pipeline_channel.PipelineChannel]]:
    """Gets channels referenced in conditions of tasks' parents.

    Args:
        root_group: The root group of a pipeline.

    Returns:
        A mapping of task name to a set of pipeline channels appeared in its
        parent dsl.Condition groups.
    """
    conditions = collections.defaultdict(set)

    def _get_condition_channels_for_tasks_helper(
        group,
        current_conditions_channels,
    ):
        new_current_conditions_channels = current_conditions_channels
        if isinstance(group, tasks_group.Condition):
            new_current_conditions_channels = list(current_conditions_channels)
            if isinstance(group.condition.left_operand,
                          pipeline_channel.PipelineChannel):
                new_current_conditions_channels.append(
                    group.condition.left_operand)
            if isinstance(group.condition.right_operand,
                          pipeline_channel.PipelineChannel):
                new_current_conditions_channels.append(
                    group.condition.right_operand)
        for task in group.tasks:
            for channel in new_current_conditions_channels:
                conditions[task.name].add(channel)
        for group in group.groups:
            _get_condition_channels_for_tasks_helper(
                group, new_current_conditions_channels)

    _get_condition_channels_for_tasks_helper(root_group, [])
    return conditions


def get_inputs_for_all_groups(
    pipeline: pipeline_context.Pipeline,
    task_name_to_parent_groups: Mapping[str, List[GroupOrTaskType]],
    group_name_to_parent_groups: Mapping[str, List[tasks_group.TasksGroup]],
    condition_channels: Mapping[str,
                                Set[pipeline_channel.PipelineParameterChannel]],
    name_to_for_loop_group: Mapping[str, tasks_group.ParallelFor],
) -> Mapping[str, List[Tuple[pipeline_channel.PipelineChannel, str]]]:
    """Get inputs and outputs of each group and op.

    Args:
        pipeline: The instantiated pipeline object.
        task_name_to_parent_groups: The dict of task name to list of parent
            groups.
        group_name_to_parent_groups: The dict of group name to list of
            parent groups.
        condition_channels: The dict of task name to a set of pipeline
            channels referenced by its parent condition groups.
        name_to_for_loop_group: The dict of for loop group name to loop
            group.

    Returns:
        A mapping  with key being the group/task names and values being list
        of tuples (channel, producing_task_name).
        producing_task_name is the name of the task that produces the
        channel. If the channel is a pipeline argument (no producer task),
        then producing_task_name is None.
    """
    inputs = collections.defaultdict(set)

    for task in pipeline.tasks.values():
        # task's inputs and all channels used in conditions for that task are
        # considered.
        task_condition_inputs = list(condition_channels[task.name])

        for channel in task.channel_inputs + task_condition_inputs:

            # If the value is already provided (immediate value), then no
            # need to expose it as input for its parent groups.
            if getattr(channel, 'value', None):
                continue

            # channels_to_add could be a list of PipelineChannels when loop
            # args are involved. Given a nested loops example as follows:
            #
            #  def my_pipeline(loop_parameter: list):
            #       with dsl.ParallelFor(loop_parameter) as item:
            #           with dsl.ParallelFor(item.p_a) as item_p_a:
            #               print_op(item_p_a.q_a)
            #
            # The print_op takes an input of
            # {{channel:task=;name=loop_parameter-loop-item-subvar-p_a-loop-item-subvar-q_a;}}.
            # Given this, we calculate the list of PipelineChannels potentially
            # needed by across DAG levels as follows:
            #
            # [{{channel:task=;name=loop_parameter-loop-item-subvar-p_a-loop-item-subvar-q_a}},
            #  {{channel:task=;name=loop_parameter-loop-item-subvar-p_a-loop-item}},
            #  {{channel:task=;name=loop_parameter-loop-item-subvar-p_a}},
            #  {{channel:task=;name=loop_parameter-loop-item}},
            #  {{chaenel:task=;name=loop_parameter}}]
            #
            # For the above example, the first loop needs the input of
            # {{channel:task=;name=loop_parameter}},
            # the second loop needs the input of
            # {{channel:task=;name=loop_parameter-loop-item}}
            # and the print_op needs the input of
            # {{channel:task=;name=loop_parameter-loop-item-subvar-p_a-loop-item}}
            #
            # When we traverse a DAG in a top-down direction, we add channels
            # from the end, and pop it out when it's no longer needed by the
            # sub-DAG.
            # When we traverse a DAG in a bottom-up direction, we add
            # channels from the front, and pop it out when it's no longer
            #  needed by the parent DAG.
            channels_to_add = collections.deque()
            channel_to_add = channel

            while isinstance(channel_to_add, (
                    for_loop.LoopArgument,
                    for_loop.LoopArgumentVariable,
            )):
                channels_to_add.append(channel_to_add)
                if isinstance(channel_to_add, for_loop.LoopArgumentVariable):
                    channel_to_add = channel_to_add.loop_argument
                else:
                    channel_to_add = channel_to_add.items_or_pipeline_channel

            if isinstance(channel_to_add, pipeline_channel.PipelineChannel):
                channels_to_add.append(channel_to_add)

            if channel.task_name:
                # The PipelineChannel is produced by a task.

                upstream_task = pipeline.tasks[channel.task_name]
                upstream_groups, downstream_groups = (
                    _get_uncommon_ancestors(
                        task_name_to_parent_groups=task_name_to_parent_groups,
                        group_name_to_parent_groups=group_name_to_parent_groups,
                        task1=upstream_task,
                        task2=task,
                    ))

                for i, group_name in enumerate(downstream_groups):
                    if i == 0:
                        # If it is the first uncommon downstream group, then
                        # the input comes from the first uncommon upstream
                        # group.
                        producer_task = upstream_groups[0]
                    else:
                        # If not the first downstream group, then the input
                        # is passed down from its ancestor groups so the
                        # upstream group is None.
                        producer_task = None

                    inputs[group_name].add((channels_to_add[-1], producer_task))

                    if group_name in name_to_for_loop_group:
                        loop_group = name_to_for_loop_group[group_name]

                        # Pop out the last elements from channels_to_add if it
                        # is found in the current (loop) DAG. Downstreams
                        # would only need the more specific versions for it.
                        if channels_to_add[
                                -1].full_name in loop_group.loop_argument.full_name:
                            channels_to_add.pop()
                            if not channels_to_add:
                                break

            else:
                # The PipelineChannel is not produced by a task. It's either
                # a top-level pipeline input, or a constant value to loop
                # items.

                # TODO: revisit if this is correct.
                if getattr(task, 'is_exit_handler', False):
                    continue

                # For PipelineChannel as a result of constant value used as
                # loop items, we have to go from bottom-up because the
                # PipelineChannel can be originated from the middle a DAG,
                # which is not needed and visible to its parent DAG.
                if isinstance(
                        channel,
                    (for_loop.LoopArgument, for_loop.LoopArgumentVariable
                    )) and channel.is_with_items_loop_argument:
                    for group_name in task_name_to_parent_groups[
                            task.name][::-1]:

                        inputs[group_name].add((channels_to_add[0], None))
                        if group_name in name_to_for_loop_group:
                            # for example:
                            #   loop_group.loop_argument.name = 'loop-item-param-1'
                            #   channel.name = 'loop-item-param-1-subvar-a'
                            loop_group = name_to_for_loop_group[group_name]

                            if channels_to_add[
                                    0].full_name in loop_group.loop_argument.full_name:
                                channels_to_add.popleft()
                                if not channels_to_add:
                                    break
                else:
                    # For PipelineChannel from pipeline input, go top-down
                    # just like we do for PipelineChannel produced by a task.
                    for group_name in task_name_to_parent_groups[task.name]:

                        inputs[group_name].add((channels_to_add[-1], None))
                        if group_name in name_to_for_loop_group:
                            loop_group = name_to_for_loop_group[group_name]

                            if channels_to_add[
                                    -1].full_name in loop_group.loop_argument.full_name:
                                channels_to_add.pop()
                                if not channels_to_add:
                                    break

    return inputs


def _get_uncommon_ancestors(
    task_name_to_parent_groups: Mapping[str, List[GroupOrTaskType]],
    group_name_to_parent_groups: Mapping[str, List[tasks_group.TasksGroup]],
    task1: GroupOrTaskType,
    task2: GroupOrTaskType,
) -> Tuple[List[GroupOrTaskType], List[GroupOrTaskType]]:
    """Gets the unique ancestors between two tasks.

    For example, task1's ancestor groups are [root, G1, G2, G3, task1],
    task2's ancestor groups are [root, G1, G4, task2], then it returns a
    tuple ([G2, G3, task1], [G4, task2]).

    Args:
        task_name_to_parent_groups: The dict of task name to list of parent
            groups.
        group_name_tor_parent_groups: The dict of group name to list of
            parent groups.
        task1: One of the two tasks.
        task2: The other task.

    Returns:
        A tuple which are lists of uncommon ancestors for each task.
    """
    if task1.name in task_name_to_parent_groups:
        task1_groups = task_name_to_parent_groups[task1.name]
    elif task1.name in group_name_to_parent_groups:
        task1_groups = group_name_to_parent_groups[task1.name]
    else:
        raise ValueError(task1.name + ' does not exist.')

    if task2.name in task_name_to_parent_groups:
        task2_groups = task_name_to_parent_groups[task2.name]
    elif task2.name in group_name_to_parent_groups:
        task2_groups = group_name_to_parent_groups[task2.name]
    else:
        raise ValueError(task2.name + ' does not exist.')

    both_groups = [task1_groups, task2_groups]
    common_groups_len = sum(
        1 for x in zip(*both_groups) if x == (x[0],) * len(x))
    group1 = task1_groups[common_groups_len:]
    group2 = task2_groups[common_groups_len:]
    return (group1, group2)


def get_dependencies(
    pipeline: pipeline_context.Pipeline,
    task_name_to_parent_groups: Mapping[str, List[GroupOrTaskType]],
    group_name_to_parent_groups: Mapping[str, List[tasks_group.TasksGroup]],
    group_name_to_group: Mapping[str, tasks_group.TasksGroup],
    condition_channels: Dict[str, pipeline_channel.PipelineChannel],
) -> Mapping[str, List[GroupOrTaskType]]:
    """Gets dependent groups and tasks for all tasks and groups.

    Args:
        pipeline: The instantiated pipeline object.
        task_name_to_parent_groups: The dict of task name to list of parent
            groups.
        group_name_to_parent_groups: The dict of group name to list of
            parent groups.
        group_name_to_group: The dict of group name to group.
        condition_channels: The dict of task name to a set of pipeline
            channels referenced by its parent condition groups.

    Returns:
        A Mapping where key is group/task name, value is a list of dependent
        groups/tasks. The dependencies are calculated in the following way:
        if task2 depends on task1, and their ancestors are
        [root, G1, G2, task1] and [root, G1, G3, G4, task2], then G3 is
        dependent on G2. Basically dependency only exists in the first
        uncommon ancesters in their ancesters chain. Only sibling
        groups/tasks can have dependencies.

    Raises:
        RuntimeError: if a task depends on a task inside a condition or loop
            group.
    """
    dependencies = collections.defaultdict(set)
    for task in pipeline.tasks.values():
        upstream_task_names = set()
        task_condition_inputs = list(condition_channels[task.name])
        for channel in task.channel_inputs + task_condition_inputs:
            if channel.task_name:
                upstream_task_names.add(channel.task_name)
        upstream_task_names |= set(task.dependent_tasks)

        for upstream_task_name in upstream_task_names:
            # the dependent op could be either a BaseOp or an opsgroup
            if upstream_task_name in pipeline.tasks:
                upstream_task = pipeline.tasks[upstream_task_name]
            elif upstream_task_name in group_name_to_group:
                upstream_task = group_name_to_group[upstream_task_name]
            else:
                raise ValueError(
                    f'Compiler cannot find task: {upstream_task_name}.')

            upstream_groups, downstream_groups = _get_uncommon_ancestors(
                task_name_to_parent_groups=task_name_to_parent_groups,
                group_name_to_parent_groups=group_name_to_parent_groups,
                task1=upstream_task,
                task2=task,
            )

            # uncommon upstream ancestor check
            uncommon_upstream_groups = deepcopy(upstream_groups)
            uncommon_upstream_groups.remove(
                upstream_task.name
            )  # because a task's `upstream_groups` contains the task's name
            if uncommon_upstream_groups:
                dependent_group = group_name_to_group.get(
                    uncommon_upstream_groups[0], None)
                if isinstance(dependent_group, tasks_group.ExitHandler):
                    task_group_type = 'an ' + tasks_group.ExitHandler.__name__

                elif isinstance(dependent_group, tasks_group.Condition):
                    task_group_type = 'a ' + tasks_group.Condition.__name__

                else:
                    task_group_type = 'a ' + tasks_group.ParallelFor.__name__

                raise RuntimeError(
                    f'Tasks cannot depend on an upstream task inside {task_group_type} that is not a common ancestor of both tasks. Task {task.name} depends on upstream task {upstream_task.name}.'
                )

            # ParralelFor Nested Check
            # if there is a parrallelFor group type in the upstream parents tasks and there also exists a parallelFor in the uncommon_ancestors of downstream: this means a nested for loop exists in the DAG
            upstream_parent_tasks = task_name_to_parent_groups[
                upstream_task.name]
            for group in downstream_groups:
                if isinstance(
                        group_name_to_group.get(group, None),
                        tasks_group.ParallelFor):
                    for parent_task in upstream_parent_tasks:
                        if isinstance(
                                group_name_to_group.get(parent_task, None),
                                tasks_group.ParallelFor):
                            raise RuntimeError(
                                f'Downstream tasks in a nested {tasks_group.ParallelFor.__name__} group cannot depend on an upstream task in a shallower {tasks_group.ParallelFor.__name__} group. Task {task.name} depends on upstream task {upstream_task.name}, while {group} is nested in {parent_task}.'
                            )

            dependencies[downstream_groups[0]].add(upstream_groups[0])

    return dependencies
