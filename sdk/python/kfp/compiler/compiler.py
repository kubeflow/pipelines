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
"""KFP DSL compiler.

Implementation of KFP compiler that compiles KFP pipeline into Pipeline IR:
https://docs.google.com/document/d/1PUDuSQ8vmeKSBloli53mp7GIvzekaY7sggg6ywy35Dk/
"""
import collections
import inspect
import json
from typing import (Any, Callable, Dict, List, Mapping, Optional, Set, Tuple,
                    Union)
import uuid
import warnings

from google.protobuf import json_format
import kfp
from kfp import dsl
from kfp.compiler import pipeline_spec_builder as builder
from kfp.compiler.pipeline_spec_builder import GroupOrTaskType
from kfp.components import base_component
from kfp.components import component_factory
from kfp.components import for_loop
from kfp.components import pipeline_context
from kfp.components import pipeline_task
from kfp.components import structures
from kfp.components import tasks_group
from kfp.components import utils as component_utils
from kfp.components.types import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2
import yaml


class Compiler:
    """Experimental DSL compiler that targets the PipelineSpec IR.

    It compiles pipeline function into PipelineSpec json string.
    PipelineSpec is the IR protobuf message that defines a pipeline:
    https://github.com/kubeflow/pipelines/blob/237795539f7b85bac77435e2464367226ee19391/api/v2alpha1/pipeline_spec.proto#L8
    In this initial implementation, we only support components authored through
    Component yaml spec. And we don't support advanced features like conditions,
    static and dynamic loops, etc.

    Example::

        @dsl.pipeline(
          name='name',
          description='description',
        )
        def my_pipeline(a: int = 1, b: str = "default value"):
            ...

        kfp.compiler.Compiler().compile(
            pipeline_func=my_pipeline,
            package_path='path/to/pipeline.json',
        )
    """

    def compile(
        self,
        pipeline_func: Union[Callable[..., Any], base_component.BaseComponent],
        package_path: str,
        pipeline_name: Optional[str] = None,
        pipeline_parameters: Optional[Mapping[str, Any]] = None,
        type_check: bool = True,
    ) -> None:
        """Compile the given pipeline function or component into pipeline job
        json.

        Args:
            pipeline_func: Pipeline function with @dsl.pipeline or component with @dsl.component decorator.
            package_path: The output pipeline spec .yaml file path. For example, "~/pipeline.yaml" or "~/component.yaml".
            pipeline_name: Optional; the name of the pipeline.
            pipeline_parameters: Optional; the mapping from parameter names to
                values.
            type_check: Optional; whether to enable the type check or not.
                Default is True.
        """

        with type_utils.TypeCheckManager(enable=type_check):
            if isinstance(pipeline_func, base_component.BaseComponent):
                component_spec = builder.modify_component_spec_for_compile(
                    component_spec=pipeline_func.component_spec,
                    pipeline_name=pipeline_name,
                    pipeline_parameters_override=pipeline_parameters,
                )
                pipeline_spec = component_spec.to_pipeline_spec()
            elif pipeline_context.Pipeline.is_pipeline_func(pipeline_func):
                pipeline_spec = self._create_pipeline(
                    pipeline_func=pipeline_func,
                    pipeline_name=pipeline_name,
                    pipeline_parameters_override=pipeline_parameters,
                )
            else:
                raise ValueError(
                    'Unsupported pipeline_func type. Expected '
                    'subclass of `base_component.BaseComponent` or '
                    '`Callable` constructed with @dsl.pipeline '
                    f'decorator. Got: {type(pipeline_func)}')
            write_pipeline_spec_to_file(
                pipeline_spec=pipeline_spec, package_path=package_path)

    def _create_pipeline(
        self,
        pipeline_func: Callable[..., Any],
        pipeline_name: Optional[str] = None,
        pipeline_parameters_override: Optional[Mapping[str, Any]] = None,
    ) -> pipeline_spec_pb2.PipelineSpec:
        """Creates a pipeline instance and constructs the pipeline spec from
        it.

        Args:
            pipeline_func: The pipeline function with @dsl.pipeline decorator.
            pipeline_name: Optional; the name of the pipeline.
            pipeline_parameters_override: Optional; the mapping from parameter
                names to values.

        Returns:
            A PipelineSpec proto representing the compiled pipeline.
        """

        # Create the arg list with no default values and call pipeline function.
        # Assign type information to the PipelineChannel
        pipeline_meta = component_factory.extract_component_interface(
            pipeline_func)
        pipeline_name = pipeline_name or pipeline_meta.name

        pipeline_root = getattr(pipeline_func, 'pipeline_root', None)

        args_list = []
        signature = inspect.signature(pipeline_func)

        for arg_name in signature.parameters:
            arg_type = pipeline_meta.inputs[arg_name].type
            if not type_utils.is_parameter_type(arg_type):
                raise TypeError(
                    builder.make_invalid_input_type_error_msg(
                        arg_name, arg_type))
            args_list.append(
                dsl.PipelineParameterChannel(
                    name=arg_name, channel_type=arg_type))

        with pipeline_context.Pipeline(pipeline_name) as dsl_pipeline:
            pipeline_func(*args_list)

        if not dsl_pipeline.tasks:
            raise ValueError('Task is missing from pipeline.')

        self._validate_exit_handler(dsl_pipeline)

        pipeline_inputs = pipeline_meta.inputs or {}

        # Verify that pipeline_parameters_override contains only input names
        # that match the pipeline inputs definition.
        pipeline_parameters_override = pipeline_parameters_override or {}
        for input_name in pipeline_parameters_override:
            if input_name not in pipeline_inputs:
                raise ValueError(
                    'Pipeline parameter {} does not match any known '
                    'pipeline argument.'.format(input_name))

        # Fill in the default values.
        args_list_with_defaults = [
            dsl.PipelineParameterChannel(
                name=input_name,
                channel_type=input_spec.type,
                value=pipeline_parameters_override.get(input_name) or
                input_spec.default,
            ) for input_name, input_spec in pipeline_inputs.items()
        ]

        # Making the pipeline group name unique to prevent name clashes with
        # templates
        pipeline_group = dsl_pipeline.groups[0]
        pipeline_group.name = uuid.uuid4().hex

        pipeline_spec = self._create_pipeline_spec(
            pipeline_args=args_list_with_defaults,
            pipeline=dsl_pipeline,
        )

        if pipeline_root:
            pipeline_spec.default_pipeline_root = pipeline_root

        return pipeline_spec

    def _create_pipeline_from_component_spec(
        self,
        component_spec: structures.ComponentSpec,
    ) -> pipeline_spec_pb2.PipelineSpec:
        """Creates a pipeline instance and constructs the pipeline spec for a
        primitive component.

        Args:
            component_spec: The ComponentSpec to convert to PipelineSpec.

        Returns:
            A PipelineSpec proto representing the compiled component.
        """
        args_dict = {}

        for arg_name, input_spec in component_spec.inputs.items():
            arg_type = input_spec.type
            if not type_utils.is_parameter_type(
                    arg_type) or type_utils.is_task_final_status_type(arg_type):
                raise TypeError(
                    builder.make_invalid_input_type_error_msg(
                        arg_name, arg_type))
            args_dict[arg_name] = dsl.PipelineParameterChannel(
                name=arg_name, channel_type=arg_type)

        task = pipeline_task.PipelineTask(component_spec, args_dict)

        # instead of constructing a pipeline with pipeline_context.Pipeline,
        # just build the single task group
        group = tasks_group.TasksGroup(
            group_type=tasks_group.TasksGroupType.PIPELINE)
        group.tasks.append(task)

        pipeline_inputs = component_spec.inputs or {}

        # Fill in the default values.
        args_list_with_defaults = [
            dsl.PipelineParameterChannel(
                name=input_name,
                channel_type=input_spec.type,
                value=input_spec.default,
            ) for input_name, input_spec in pipeline_inputs.items()
        ]
        group.name = uuid.uuid4().hex

        return builder.create_pipeline_spec_for_component(
            pipeline_name=component_spec.name,
            pipeline_args=args_list_with_defaults,
            task_group=group,
        )

    def _validate_exit_handler(self,
                               pipeline: pipeline_context.Pipeline) -> None:
        """Makes sure there is only one global exit handler.

        This is temporary to be compatible with KFP v1.

        Raises:
            ValueError if there are more than one exit handler.
        """

        def _validate_exit_handler_helper(
            group: tasks_group.TasksGroup,
            exiting_task_names: List[str],
            handler_exists: bool,
        ) -> None:

            if isinstance(group, dsl.ExitHandler):
                if handler_exists or len(exiting_task_names) > 1:
                    raise ValueError(
                        'Only one global exit_handler is allowed and all ops need to be included.'
                    )
                handler_exists = True

            if group.tasks:
                exiting_task_names.extend([x.name for x in group.tasks])

            for group in group.groups:
                _validate_exit_handler_helper(
                    group=group,
                    exiting_task_names=exiting_task_names,
                    handler_exists=handler_exists,
                )

        _validate_exit_handler_helper(
            group=pipeline.groups[0],
            exiting_task_names=[],
            handler_exists=False,
        )

    def _create_pipeline_spec(
        self,
        pipeline_args: List[dsl.PipelineChannel],
        pipeline: pipeline_context.Pipeline,
    ) -> pipeline_spec_pb2.PipelineSpec:
        """Creates a pipeline spec object.

        Args:
            pipeline_args: The list of pipeline input parameters.
            pipeline: The instantiated pipeline object.

        Returns:
            A PipelineSpec proto representing the compiled pipeline.

        Raises:
            ValueError if the argument is of unsupported types.
        """
        builder.validate_pipeline_name(pipeline.name)

        deployment_config = pipeline_spec_pb2.PipelineDeploymentConfig()
        pipeline_spec = pipeline_spec_pb2.PipelineSpec()

        pipeline_spec.pipeline_info.name = pipeline.name
        pipeline_spec.sdk_version = f'kfp-{kfp.__version__}'
        # Schema version 2.1.0 is required for kfp-pipeline-spec>0.1.13
        pipeline_spec.schema_version = '2.1.0'

        pipeline_spec.root.CopyFrom(
            builder.build_component_spec_for_group(
                pipeline_channels=pipeline_args,
                is_root_group=True,
            ))

        root_group = pipeline.groups[0]

        all_groups = self._get_all_groups(root_group)
        group_name_to_group = {group.name: group for group in all_groups}
        task_name_to_parent_groups, group_name_to_parent_groups = (
            builder.get_parent_groups(root_group))
        condition_channels = self._get_condition_channels_for_tasks(root_group)
        name_to_for_loop_group = {
            group_name: group
            for group_name, group in group_name_to_group.items()
            if isinstance(group, dsl.ParallelFor)
        }
        inputs = self._get_inputs_for_all_groups(
            pipeline=pipeline,
            pipeline_args=pipeline_args,
            root_group=root_group,
            task_name_to_parent_groups=task_name_to_parent_groups,
            group_name_to_parent_groups=group_name_to_parent_groups,
            condition_channels=condition_channels,
            name_to_for_loop_group=name_to_for_loop_group,
        )
        dependencies = self._get_dependencies(
            pipeline=pipeline,
            root_group=root_group,
            task_name_to_parent_groups=task_name_to_parent_groups,
            group_name_to_parent_groups=group_name_to_parent_groups,
            group_name_to_group=group_name_to_group,
            condition_channels=condition_channels,
        )

        for group in all_groups:
            builder.build_spec_by_group(
                pipeline_spec=pipeline_spec,
                deployment_config=deployment_config,
                group=group,
                inputs=inputs,
                dependencies=dependencies,
                rootgroup_name=root_group.name,
                task_name_to_parent_groups=task_name_to_parent_groups,
                group_name_to_parent_groups=group_name_to_parent_groups,
                name_to_for_loop_group=name_to_for_loop_group,
            )

        # TODO: refactor to support multiple exit handler per pipeline.
        if pipeline.groups[0].groups:
            first_group = pipeline.groups[0].groups[0]
            if isinstance(first_group, dsl.ExitHandler):
                exit_task = first_group.exit_task
                exit_task_name = component_utils.sanitize_task_name(
                    exit_task.name)
                exit_handler_group_task_name = component_utils.sanitize_task_name(
                    first_group.name)
                input_parameters_in_current_dag = [
                    input_name for input_name in
                    pipeline_spec.root.input_definitions.parameters
                ]
                exit_task_task_spec = builder.build_task_spec_for_exit_task(
                    task=exit_task,
                    dependent_task=exit_handler_group_task_name,
                    pipeline_inputs=pipeline_spec.root.input_definitions,
                )

                exit_task_component_spec = builder.build_component_spec_for_exit_task(
                    task=exit_task)

                exit_task_container_spec = builder.build_container_spec_for_task(
                    task=exit_task)

                # Add exit task task spec
                pipeline_spec.root.dag.tasks[exit_task_name].CopyFrom(
                    exit_task_task_spec)

                # Add exit task component spec if it does not exist.
                component_name = exit_task_task_spec.component_ref.name
                if component_name not in pipeline_spec.components:
                    pipeline_spec.components[component_name].CopyFrom(
                        exit_task_component_spec)

                # Add exit task container spec if it does not exist.
                executor_label = exit_task_component_spec.executor_label
                if executor_label not in deployment_config.executors:
                    deployment_config.executors[
                        executor_label].container.CopyFrom(
                            exit_task_container_spec)
                    pipeline_spec.deployment_spec.update(
                        json_format.MessageToDict(deployment_config))

        return pipeline_spec

    def _get_all_groups(
        self,
        root_group: tasks_group.TasksGroup,
    ) -> List[tasks_group.TasksGroup]:
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

    # TODO: do we really need this?
    def _get_condition_channels_for_tasks(
        self,
        root_group: tasks_group.TasksGroup,
    ) -> Mapping[str, Set[dsl.PipelineChannel]]:
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
            if isinstance(group, dsl.Condition):
                new_current_conditions_channels = list(
                    current_conditions_channels)
                if isinstance(group.condition.left_operand,
                              dsl.PipelineChannel):
                    new_current_conditions_channels.append(
                        group.condition.left_operand)
                if isinstance(group.condition.right_operand,
                              dsl.PipelineChannel):
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

    def _get_inputs_for_all_groups(
        self,
        pipeline: pipeline_context.Pipeline,
        pipeline_args: List[dsl.PipelineChannel],
        root_group: tasks_group.TasksGroup,
        task_name_to_parent_groups: Mapping[str, List[GroupOrTaskType]],
        group_name_to_parent_groups: Mapping[str, List[tasks_group.TasksGroup]],
        condition_channels: Mapping[str, Set[dsl.PipelineParameterChannel]],
        name_to_for_loop_group: Mapping[str, dsl.ParallelFor],
    ) -> Mapping[str, List[Tuple[dsl.PipelineChannel, str]]]:
        """Get inputs and outputs of each group and op.

        Args:
            pipeline: The instantiated pipeline object.
            pipeline_args: The list of pipeline function arguments as
                PipelineChannel.
            root_group: The root group of the pipeline.
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
                    if isinstance(channel_to_add,
                                  for_loop.LoopArgumentVariable):
                        channel_to_add = channel_to_add.loop_argument
                    else:
                        channel_to_add = channel_to_add.items_or_pipeline_channel

                if isinstance(channel_to_add, dsl.PipelineChannel):
                    channels_to_add.append(channel_to_add)

                if channel.task_name:
                    # The PipelineChannel is produced by a task.

                    upstream_task = pipeline.tasks[channel.task_name]
                    upstream_groups, downstream_groups = (
                        self._get_uncommon_ancestors(
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

                        inputs[group_name].add(
                            (channels_to_add[-1], producer_task))

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
        self,
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

    def _get_dependencies(
        self,
        pipeline: pipeline_context.Pipeline,
        root_group: tasks_group.TasksGroup,
        task_name_to_parent_groups: Mapping[str, List[GroupOrTaskType]],
        group_name_to_parent_groups: Mapping[str, List[tasks_group.TasksGroup]],
        group_name_to_group: Mapping[str, tasks_group.TasksGroup],
        condition_channels: Dict[str, dsl.PipelineChannel],
    ) -> Mapping[str, List[GroupOrTaskType]]:
        """Gets dependent groups and tasks for all tasks and groups.

        Args:
            pipeline: The instantiated pipeline object.
            root_group: The root group of the pipeline.
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

                upstream_groups, downstream_groups = self._get_uncommon_ancestors(
                    task_name_to_parent_groups=task_name_to_parent_groups,
                    group_name_to_parent_groups=group_name_to_parent_groups,
                    task1=upstream_task,
                    task2=task,
                )

                # If a task depends on a condition group or a loop group, it
                # must explicitly dependent on a task inside the group. This
                # should not be allowed, because it leads to ambiguous
                # expectations for runtime behaviors.
                dependent_group = group_name_to_group.get(
                    upstream_groups[0], None)
                if isinstance(dependent_group,
                              (tasks_group.Condition, tasks_group.ParallelFor)):
                    raise RuntimeError(
                        f'Task {task.name} cannot dependent on any task inside'
                        f' the group: {upstream_groups[0]}.')

                dependencies[downstream_groups[0]].add(upstream_groups[0])

        return dependencies


def write_pipeline_spec_to_file(pipeline_spec: pipeline_spec_pb2.PipelineSpec,
                                package_path: str) -> None:
    """Writes PipelienSpec into a YAML or JSON (deprecated) file.

    Args:
        pipeline_spec (pipeline_spec_pb2.PipelineSpec): The PipelineSpec.
        package_path (str): The path to which to write the PipelineSpec.
    """
    json_dict = json_format.MessageToDict(pipeline_spec)

    if package_path.endswith('.json'):
        warnings.warn(
            ('Compiling to JSON is deprecated and will be '
             'removed in a future version. Please compile to a YAML file by '
             'providing a file path with a .yaml extension instead.'),
            category=DeprecationWarning,
            stacklevel=2,
        )
        with open(package_path, 'w') as json_file:
            json.dump(json_dict, json_file, indent=2, sort_keys=True)

    elif package_path.endswith(('.yaml', '.yml')):
        with open(package_path, 'w') as yaml_file:
            yaml.dump(json_dict, yaml_file, sort_keys=True)

    else:
        raise ValueError(
            f'The output path {package_path} should end with ".yaml".')
