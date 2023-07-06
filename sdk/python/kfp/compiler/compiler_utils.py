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
from typing import DefaultDict, Dict, List, Mapping, Set, Tuple, Union

from kfp.dsl import for_loop
from kfp.dsl import pipeline_channel
from kfp.dsl import pipeline_context
from kfp.dsl import pipeline_task
from kfp.dsl import tasks_group

GroupOrTaskType = Union[tasks_group.TasksGroup, pipeline_task.PipelineTask]

ILLEGAL_CROSS_DAG_ERROR_PREFIX = 'Illegal task dependency across DSL context managers.'


def additional_input_name_for_pipeline_channel(
        channel_or_name: Union[pipeline_channel.PipelineChannel, str]) -> str:
    """Gets the name for an additional (compiler-injected) input."""

    # Adding a prefix to avoid (reduce chance of) name collision between the
    # original component inputs and the injected input.
    return 'pipelinechannel--' + (
        channel_or_name.full_name if isinstance(
            channel_or_name, pipeline_channel.PipelineChannel) else
        channel_or_name)


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
) -> Tuple[Mapping[str, List[str]], Mapping[str, List[str]]]:
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
    task_name_to_parent_groups: Mapping[str, List[str]],
    group_name_to_parent_groups: Mapping[str, List[str]],
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


class InvalidTopologyException(Exception):
    pass


def validate_parallel_for_fan_in_consumption_legal(
    consumer_task_name: str,
    upstream_groups: List[str],
    group_name_to_group: Dict[str, tasks_group.TasksGroup],
) -> None:
    """Checks that a dsl.Collected object is being used results in an
    unambiguous pipeline topology and is therefore legal.

    Args:
        consumer_task_name: The name of the consumer task.
        upstream_groups: The names of the producer task's upstream groups, ordered from outermost group at beginning to producer task at end. This is produced by produced by _get_uncommon_ancestors.
        group_name_to_group: Map of group name to TasksGroup, for fast lookups.
    """
    # handles cases like this:
    # @dsl.pipeline
    # def my_pipeline():
    #     with dsl.ParallelFor([1, 2, 3]) as x:
    #         t = double(num=x)
    #         x = add(dsl.Collected(t.output))
    #
    # and this:
    # @dsl.pipeline
    # def my_pipeline():
    #     t = double(num=1)
    #     x = add(dsl.Collected(t.output))
    producer_task_idx = -1
    producer_task_name = upstream_groups[producer_task_idx]
    if all(group_name_to_group[group_name].group_type !=
           tasks_group.TasksGroupType.FOR_LOOP
           for group_name in upstream_groups[:producer_task_idx]):
        raise InvalidTopologyException(
            f'dsl.{for_loop.Collected.__name__} can only be used to fan-in outputs produced by a task within a dsl.{tasks_group.ParallelFor.__name__} context to a task outside of the dsl.{tasks_group.ParallelFor.__name__} context. Producer task {producer_task_name} is either not in a dsl.{tasks_group.ParallelFor.__name__} context or is only in a dsl.{tasks_group.ParallelFor.__name__} that also contains consumer task {consumer_task_name}.'
        )

    # illegal if the producer has a parent conditional outside of its outermost for loop, since the for loop may or may not be executed
    # for example, what happens if text == 'b'? the resulting execution behavior is ambiguous.
    #
    # @dsl.pipeline
    # def my_pipeline(text: str = ''):
    #     with dsl.Condition(text == 'a'):
    #         with dsl.ParallelFor([1, 2, 3]) as x:
    #             t = double(num=x)
    #     x = add(nums=dsl.Collected(t.output))
    outermost_uncommon_upstream_group = upstream_groups[0]
    group = group_name_to_group[outermost_uncommon_upstream_group]
    if group.group_type in [
            tasks_group.TasksGroupType.CONDITION,
            tasks_group.TasksGroupType.EXIT_HANDLER,
    ]:
        raise InvalidTopologyException(
            f'{ILLEGAL_CROSS_DAG_ERROR_PREFIX} When using dsl.{for_loop.Collected.__name__} to fan-in outputs from a task within a dsl.{tasks_group.ParallelFor.__name__} context, the dsl.{tasks_group.ParallelFor.__name__} context manager cannot be nested within a dsl.{group.__class__.__name__} context manager unless the consumer task is too. Task {consumer_task_name} consumes from {producer_task_name} within a dsl.{group.__class__.__name__} context.'
        )
    elif group.group_type != tasks_group.TasksGroupType.FOR_LOOP:
        raise ValueError(
            f'Got unexpected group type when validating fanning-in outputs from task in dsl.{tasks_group.ParallelFor.__name__}: {group.group_type}'
        )


def make_new_channel_for_collected_outputs(
    channel_name: str,
    starting_channel: pipeline_channel.PipelineChannel,
    task_name: str,
) -> pipeline_channel.PipelineChannel:
    """Creates a new PipelineParameterChannel/PipelineArtifactChannel list for
    a dsl.Collected channel from the original task output."""
    if isinstance(starting_channel, pipeline_channel.PipelineParameterChannel):
        return pipeline_channel.PipelineParameterChannel(
            channel_name, channel_type='LIST', task_name=task_name)
    elif isinstance(starting_channel, pipeline_channel.PipelineArtifactChannel):
        return pipeline_channel.PipelineArtifactChannel(
            channel_name,
            channel_type=starting_channel.channel_type,
            task_name=task_name,
            is_artifact_list=True)
    else:
        ValueError(
            f'Got unknown PipelineChannel: {starting_channel!r}. Expected an instance of {pipeline_channel.PipelineArtifactChannel.__name__!r} or {pipeline_channel.PipelineParameterChannel.__name__!r}.'
        )


def get_outputs_for_all_groups(
    pipeline: pipeline_context.Pipeline,
    task_name_to_parent_groups: Mapping[str, List[str]],
    group_name_to_parent_groups: Mapping[str, List[str]],
    all_groups: List[tasks_group.TasksGroup],
    pipeline_outputs_dict: Dict[str, pipeline_channel.PipelineChannel]
) -> Tuple[DefaultDict[str, Dict[str, pipeline_channel.PipelineChannel]], Dict[
        str, pipeline_channel.PipelineChannel]]:
    """Gets a dictionary of all TasksGroup names to an inner dictionary. The
    inner dictionary is TasksGroup output keys to channels corresponding to
    those keys.

    It constructs this dictionary from both data passing within the pipeline body, as well as the outputs returned from the pipeline (e.g., return dsl.Collected(...)).

    Also returns as the second item of tuple the updated pipeline_outputs_dict. This dict is modified so that the values (PipelineChannel) references the group that surfaces the task output, instead of the original task that produced it.
    """

    # unlike inputs, which will be surfaced as component input parameters,
    # consumers of surfaced outputs need to have a reference to what the parent
    # component calls them when they surface them, which will be different than
    # the producer task name and channel name (the information contained in the
    # pipeline channel)
    # for this reason, we use additional_input_name_for_pipeline_channel here
    # to set the name of the surfaced output once

    group_name_to_group = {group.name: group for group in all_groups}
    group_name_to_children = {
        group.name: [group.name for group in group.groups] +
        [task.name for task in group.tasks] for group in all_groups
    }

    outputs = collections.defaultdict(dict)

    # handle dsl.Collected consumed by tasks
    for task in pipeline.tasks.values():
        for channel in task.channel_inputs:
            if not isinstance(channel, for_loop.Collected):
                continue
            producer_task = pipeline.tasks[channel.task_name]
            consumer_task = task

            upstream_groups, downstream_groups = (
                _get_uncommon_ancestors(
                    task_name_to_parent_groups=task_name_to_parent_groups,
                    group_name_to_parent_groups=group_name_to_parent_groups,
                    task1=producer_task,
                    task2=consumer_task,
                ))
            validate_parallel_for_fan_in_consumption_legal(
                consumer_task_name=consumer_task.name,
                upstream_groups=upstream_groups,
                group_name_to_group=group_name_to_group,
            )

            # producer_task's immediate parent group and the name by which
            # to surface the channel
            surfaced_output_name = additional_input_name_for_pipeline_channel(
                channel)

            # the highest-level task group that "consumes" the
            # collected output
            parent_consumer = downstream_groups[0]
            producer_task_name = upstream_groups.pop()

            # process from the upstream groups from the inside out
            for upstream_name in reversed(upstream_groups):
                outputs[upstream_name][
                    surfaced_output_name] = make_new_channel_for_collected_outputs(
                        channel_name=channel.name,
                        starting_channel=channel.output,
                        task_name=producer_task_name,
                    )

                # on each iteration, mutate the channel being consumed so
                # that it references the last parent group surfacer
                channel.name = surfaced_output_name
                channel.task_name = upstream_name

                # for the next iteration, set the consumer to the current
                # surfacer (parent group)
                producer_task_name = upstream_name

                parent_of_current_surfacer = group_name_to_parent_groups[
                    upstream_name][-2]
                if parent_consumer in group_name_to_children[
                        parent_of_current_surfacer]:
                    break

        # handle dsl.Collected returned from pipeline
        for output_key, channel in pipeline_outputs_dict.items():
            if isinstance(channel, for_loop.Collected):
                surfaced_output_name = additional_input_name_for_pipeline_channel(
                    channel)
                upstream_groups = task_name_to_parent_groups[
                    channel.task_name][1:]
                producer_task_name = upstream_groups.pop()
                # process upstream groups from the inside out, until getting to the pipeline level
                for upstream_name in reversed(upstream_groups):
                    new_channel = make_new_channel_for_collected_outputs(
                        channel_name=channel.name,
                        starting_channel=channel.output,
                        task_name=producer_task_name,
                    )

                    # on each iteration, mutate the channel being consumed so
                    # that it references the last parent group surfacer
                    channel.name = surfaced_output_name
                    channel.task_name = upstream_name

                    # for the next iteration, set the consumer to the current
                    # surfacer (parent group)
                    producer_task_name = upstream_name
                    outputs[upstream_name][surfaced_output_name] = new_channel

                # after surfacing from all inner TasksGroup, change the PipelineChannel output to also return from the correct TasksGroup
                pipeline_outputs_dict[
                    output_key] = make_new_channel_for_collected_outputs(
                        channel_name=surfaced_output_name,
                        starting_channel=channel.output,
                        task_name=upstream_name,
                    )
    return outputs, pipeline_outputs_dict


def _get_uncommon_ancestors(
    task_name_to_parent_groups: Mapping[str, List[str]],
    group_name_to_parent_groups: Mapping[str, List[str]],
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
    task_name_to_parent_groups: Mapping[str, List[str]],
    group_name_to_parent_groups: Mapping[str, List[str]],
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

            # TODO: this logic can be simplified: the logic to check for consumption from a task in an upstream
            # Condition and ExitHandler should be the same and can be unified with the ParallelFor checks
            if uncommon_upstream_groups:
                dependent_group = group_name_to_group.get(
                    uncommon_upstream_groups[0], None)

                if isinstance(dependent_group,
                              (tasks_group.Condition, tasks_group.ExitHandler)):
                    raise InvalidTopologyException(
                        f'{ILLEGAL_CROSS_DAG_ERROR_PREFIX} A downstream task cannot depend on an upstream task within a dsl.{dependent_group.__class__.__name__} context unless the downstream is within that context too. Found task {task.name} which depends on upstream task {upstream_task.name} within an uncommon dsl.{dependent_group.__class__.__name__} context.'
                    )
                elif isinstance(dependent_group, tasks_group.ParallelFor):
                    raise InvalidTopologyException(
                        f'{ILLEGAL_CROSS_DAG_ERROR_PREFIX} A downstream task cannot depend on an upstream task within a dsl.{dependent_group.__class__.__name__} context unless the downstream is within that context too or the outputs are begin fanned-in to a list using dsl.{for_loop.Collected.__name__}. Found task {task.name} which depends on upstream task {upstream_task.name} within an uncommon dsl.{dependent_group.__class__.__name__} context.'
                    )

            # tasks in a deeper part of a nested ParallelFor cannot depend directly on a task in a shallower part of the nested ParallelFor
            # to check for this we need to know that:
            # - the upstream task has at least one ParallelFor parent group
            # - the downstream task has at least one ParallelFor parent group uncommon to the upstream task
            # - the downstream consumes from the upstream or has an explicit dependency (.after) on it (a dependency transitively via the parent task group doesn't satisfy the error condition, since the ParallelFor should be permitted to iterate over the upstreams list output)
            if isinstance(upstream_task, pipeline_task.PipelineTask):
                upstream_parent_tasks = task_name_to_parent_groups[
                    upstream_task.name]

                downstream_parallelfor_parents = [
                    group_name_to_group.get(group, None)
                    for group in downstream_groups
                    if isinstance(
                        group_name_to_group.get(group, None),
                        tasks_group.ParallelFor)
                ]
                downstream_in_parallelfor = bool(downstream_parallelfor_parents)

                upstream_parallelfor_parents = [
                    parent_group for parent_group in upstream_parent_tasks
                    if isinstance(
                        group_name_to_group.get(parent_group, None),
                        tasks_group.ParallelFor)
                ]
                upstream_in_parallelfor = bool(upstream_parallelfor_parents)

                downstream_consumes_from_upstream = bool([
                    inp.name
                    for inp in task._task_spec.inputs.values()
                    if isinstance(inp, pipeline_channel.PipelineChannel) and
                    inp.name == upstream_task.name
                ])

                after_called_on_upstream = upstream_task.name in task._run_after

                if downstream_in_parallelfor and upstream_in_parallelfor and (
                        downstream_consumes_from_upstream or
                        after_called_on_upstream):

                    raise InvalidTopologyException(
                        f'{ILLEGAL_CROSS_DAG_ERROR_PREFIX} Downstream tasks in a nested {tasks_group.ParallelFor.__name__} group cannot depend on an upstream task in a shallower {tasks_group.ParallelFor.__name__} group. Task {task.name} depends on upstream task {upstream_task.name}, while {downstream_parallelfor_parents[0]} is nested in {upstream_parallelfor_parents[0]}.'
                    )

            dependencies[downstream_groups[0]].add(upstream_groups[0])

    return dependencies
