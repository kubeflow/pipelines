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
"""Helper functions for DSL compiler."""
import re
from typing import Any, Dict, List, Mapping, Optional, Tuple, Union

from google.protobuf import json_format
from kfp import dsl
from kfp.compiler import pipeline_spec_builder as builder
from kfp.components import for_loop
from kfp.components import pipeline_task
from kfp.components import structures
from kfp.components import tasks_group
from kfp.components import utils
from kfp.components import utils as component_utils
from kfp.pipeline_spec import pipeline_spec_pb2

_GroupOrTask = Union[tasks_group.TasksGroup, pipeline_task.PipelineTask]


def _make_invalid_input_type_error_msg(arg_name: str, arg_type: Any) -> str:
    valid_types = (str.__name__, int.__name__, float.__name__, bool.__name__,
                   dict.__name__, list.__name__)
    return f"The pipeline parameter '{arg_name}' of type {arg_type} is not a valid input for this component. Passing artifacts as pipeline inputs is not supported. Consider annotating the parameter with a primitive type such as {valid_types}."


def _modify_component_spec_for_compile(
    component_spec: structures.ComponentSpec,
    pipeline_name: Optional[str],
    pipeline_parameters_override: Optional[Mapping[str, Any]],
) -> structures.ComponentSpec:
    """Modifies the ComponentSpec using arguments passed to the
    Compiler.compile method.

    Args:
        component_spec (structures.ComponentSpec): ComponentSpec to modify.
        pipeline_name (Optional[str]): Name of the pipeline. Overrides component name.
        pipeline_parameters_override (Optional[Mapping[str, Any]]): Pipeline parameters. Overrides component input default values.

    Raises:
        ValueError: If a parameter is passed to the compiler that is not a component input.

    Returns:
        structures.ComponentSpec: The modified ComponentSpec.
    """
    pipeline_name = pipeline_name or component_utils.sanitize_component_name(
        component_spec.name).replace(utils._COMPONENT_NAME_PREFIX, '')

    component_spec.name = pipeline_name
    if component_spec.inputs is not None:
        pipeline_parameters_override = pipeline_parameters_override or {}
        for input_name in pipeline_parameters_override:
            if input_name not in component_spec.inputs:
                raise ValueError(
                    f'Parameter {input_name} does not match any known component parameters.'
                )
            component_spec.inputs[
                input_name].default = pipeline_parameters_override[input_name]

    return component_spec


def _build_spec_by_group(
    pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    deployment_config: pipeline_spec_pb2.PipelineDeploymentConfig,
    group: tasks_group.TasksGroup,
    inputs: Mapping[str, List[Tuple[dsl.PipelineChannel, str]]],
    dependencies: Dict[str, List[_GroupOrTask]],
    rootgroup_name: str,
    task_name_to_parent_groups: Mapping[str, List[_GroupOrTask]],
    group_name_to_parent_groups: Mapping[str, List[tasks_group.TasksGroup]],
    name_to_for_loop_group: Mapping[str, dsl.ParallelFor],
) -> None:
    """Generates IR spec given a TasksGroup.

    Args:
        pipeline_spec: The pipeline_spec to update in place.
        deployment_config: The deployment_config to hold all executors. The
            spec is updated in place.
        group: The TasksGroup to generate spec for.
        inputs: The inputs dictionary. The keys are group/task names and the
            values are lists of tuples (channel, producing_task_name).
        dependencies: The group dependencies dictionary. The keys are group
            or task names, and the values are lists of dependent groups or
            tasks.
        rootgroup_name: The name of the group root. Used to determine whether
            the component spec for the current group should be the root dag.
        task_name_to_parent_groups: The dict of task name to parent groups.
            Key is task name. Value is a list of ancestor groups including
            the task itself. The list of a given task is sorted in a way that
            the farthest group is the first and the task itself is the last.
        group_name_to_parent_groups: The dict of group name to parent groups.
            Key is the group name. Value is a list of ancestor groups
            including the group itself. The list of a given group is sorted
            in a way that the farthest group is the first and the group
            itself is the last.
        name_to_for_loop_group: The dict of for loop group name to loop
            group.
    """
    group_component_name = component_utils.sanitize_component_name(group.name)

    if group.name == rootgroup_name:
        group_component_spec = pipeline_spec.root
    else:
        group_component_spec = pipeline_spec.components[group_component_name]

    task_name_to_task_spec = {}
    task_name_to_component_spec = {}

    # Generate task specs and component specs for the dag.
    subgroups = group.groups + group.tasks
    for subgroup in subgroups:

        subgroup_inputs = inputs.get(subgroup.name, [])
        subgroup_channels = [channel for channel, _ in subgroup_inputs]

        subgroup_component_name = (
            component_utils.sanitize_component_name(subgroup.name))

        tasks_in_current_dag = [
            component_utils.sanitize_task_name(subgroup.name)
            for subgroup in subgroups
        ]
        input_parameters_in_current_dag = [
            input_name
            for input_name in group_component_spec.input_definitions.parameters
        ]
        input_artifacts_in_current_dag = [
            input_name
            for input_name in group_component_spec.input_definitions.artifacts
        ]
        is_parent_component_root = (group_component_spec == pipeline_spec.root)

        if isinstance(subgroup, pipeline_task.PipelineTask):

            subgroup_task_spec = builder.build_task_spec_for_task(
                task=subgroup,
                parent_component_inputs=group_component_spec.input_definitions,
                tasks_in_current_dag=tasks_in_current_dag,
                input_parameters_in_current_dag=input_parameters_in_current_dag,
                input_artifacts_in_current_dag=input_artifacts_in_current_dag,
            )
            task_name_to_task_spec[subgroup.name] = subgroup_task_spec

            subgroup_component_spec = builder.build_component_spec_for_task(
                task=subgroup)
            task_name_to_component_spec[subgroup.name] = subgroup_component_spec

            executor_label = subgroup_component_spec.executor_label

            if executor_label not in deployment_config.executors:
                if subgroup.container_spec is not None:
                    subgroup_container_spec = builder.build_container_spec_for_task(
                        task=subgroup)
                    deployment_config.executors[
                        executor_label].container.CopyFrom(
                            subgroup_container_spec)
                elif subgroup.importer_spec is not None:
                    subgroup_importer_spec = builder.build_importer_spec_for_task(
                        task=subgroup)
                    deployment_config.executors[
                        executor_label].importer.CopyFrom(
                            subgroup_importer_spec)
        elif isinstance(subgroup, dsl.ParallelFor):

            # "Punch the hole", adding additional inputs (other than loop
            # arguments which will be handled separately) needed by its
            # subgroups or tasks.
            loop_subgroup_channels = []

            for channel in subgroup_channels:
                # Skip 'withItems' loop arguments if it's from an inner loop.
                if isinstance(
                        channel,
                    (for_loop.LoopArgument, for_loop.LoopArgumentVariable
                    )) and channel.is_with_items_loop_argument:
                    withitems_loop_arg_found_in_self_or_upstream = False
                    for group_name in group_name_to_parent_groups[
                            subgroup.name][::-1]:
                        if group_name in name_to_for_loop_group:
                            loop_group = name_to_for_loop_group[group_name]
                            if channel.name in loop_group.loop_argument.name:
                                withitems_loop_arg_found_in_self_or_upstream = True
                                break
                    if not withitems_loop_arg_found_in_self_or_upstream:
                        continue
                loop_subgroup_channels.append(channel)

            if subgroup.items_is_pipeline_channel:
                # This loop_argument is based on a pipeline channel, i.e.,
                # rather than a static list, it is either the output of
                # another task or an input as global pipeline parameters.
                loop_subgroup_channels.append(
                    subgroup.loop_argument.items_or_pipeline_channel)

            loop_subgroup_channels.append(subgroup.loop_argument)

            subgroup_component_spec = builder.build_component_spec_for_group(
                pipeline_channels=loop_subgroup_channels,
                is_root_group=False,
            )

            subgroup_task_spec = builder.build_task_spec_for_group(
                group=subgroup,
                pipeline_channels=loop_subgroup_channels,
                tasks_in_current_dag=tasks_in_current_dag,
                is_parent_component_root=is_parent_component_root,
            )

        elif isinstance(subgroup, dsl.Condition):

            # "Punch the hole", adding inputs needed by its subgroups or
            # tasks.
            condition_subgroup_channels = list(subgroup_channels)
            for operand in [
                    subgroup.condition.left_operand,
                    subgroup.condition.right_operand,
            ]:
                if isinstance(operand, dsl.PipelineChannel):
                    condition_subgroup_channels.append(operand)

            subgroup_component_spec = builder.build_component_spec_for_group(
                pipeline_channels=condition_subgroup_channels,
                is_root_group=False,
            )

            subgroup_task_spec = builder.build_task_spec_for_group(
                group=subgroup,
                pipeline_channels=condition_subgroup_channels,
                tasks_in_current_dag=tasks_in_current_dag,
                is_parent_component_root=is_parent_component_root,
            )

        elif isinstance(subgroup, dsl.ExitHandler):

            subgroup_component_spec = builder.build_component_spec_for_group(
                pipeline_channels=subgroup_channels,
                is_root_group=False,
            )

            subgroup_task_spec = builder.build_task_spec_for_group(
                group=subgroup,
                pipeline_channels=subgroup_channels,
                tasks_in_current_dag=tasks_in_current_dag,
                is_parent_component_root=is_parent_component_root,
            )

        else:
            raise RuntimeError(
                f'Unexpected task/group type: Got {subgroup} of type '
                f'{type(subgroup)}.')

        # Generate dependencies section for this task.
        if dependencies.get(subgroup.name, None):
            group_dependencies = list(dependencies[subgroup.name])
            group_dependencies.sort()
            subgroup_task_spec.dependent_tasks.extend([
                component_utils.sanitize_task_name(dep)
                for dep in group_dependencies
            ])

        # Add component spec if not exists
        if subgroup_component_name not in pipeline_spec.components:
            pipeline_spec.components[subgroup_component_name].CopyFrom(
                subgroup_component_spec)

        # Add task spec
        group_component_spec.dag.tasks[subgroup.name].CopyFrom(
            subgroup_task_spec)

    pipeline_spec.deployment_spec.update(
        json_format.MessageToDict(deployment_config))

    # Surface metrics outputs to the top.
    builder.populate_metrics_in_dag_outputs(
        tasks=group.tasks,
        task_name_to_parent_groups=task_name_to_parent_groups,
        task_name_to_task_spec=task_name_to_task_spec,
        task_name_to_component_spec=task_name_to_component_spec,
        pipeline_spec=pipeline_spec,
    )


def _get_parent_groups(
    root_group: tasks_group.TasksGroup,
) -> Tuple[Mapping[str, List[_GroupOrTask]], Mapping[str, List[_GroupOrTask]]]:
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
        tasks_to_groups: Dict[str, List[_GroupOrTask]],
        groups_to_groups: Dict[str, List[_GroupOrTask]],
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


def _validate_pipeline_name(name: str) -> None:
    """Validate pipeline name.

    A valid pipeline name should match ^[a-z0-9][a-z0-9-]{0,127}$.

    Args:
        name: The pipeline name.

    Raises:
        ValueError if the pipeline name doesn't conform to the regular expression.
    """
    pattern = re.compile(r'^[a-z0-9][a-z0-9-]{0,127}$')
    if not pattern.match(name):
        raise ValueError(
            'Invalid pipeline name: %s.\n'
            'Please specify a pipeline name that matches the regular '
            'expression "^[a-z0-9][a-z0-9-]{0,127}$" using '
            '`dsl.pipeline(name=...)` decorator.' % name)
