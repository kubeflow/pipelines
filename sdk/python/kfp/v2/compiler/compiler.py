# Copyright 2020 The Kubeflow Authors
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
"""KFP DSL v2 compiler.

This is an experimental implementation of KFP compiler that compiles KFP
pipeline into Pipeline IR:
https://docs.google.com/document/d/1PUDuSQ8vmeKSBloli53mp7GIvzekaY7sggg6ywy35Dk/
"""

import collections
import inspect
import json
import uuid
import warnings
from typing import Any, Callable, Dict, List, Mapping, Optional, Set, Tuple, Union

import kfp
from kfp.compiler._k8s_helper import sanitize_k8s_name
from kfp.components import _python_op
from kfp import dsl
from kfp.dsl import _for_loop
from kfp.dsl import _pipeline_param
from kfp.v2.compiler import compiler_utils
from kfp.dsl import component_spec as dsl_component_spec
from kfp.dsl import dsl_utils
from kfp.dsl import io_types
from kfp.dsl import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2

from google.protobuf import json_format

_GroupOrOp = Union[dsl.OpsGroup, dsl.BaseOp]


class Compiler(object):
  """Experimental DSL compiler that targets the PipelineSpec IR.

  It compiles pipeline function into PipelineSpec json string.
  PipelineSpec is the IR protobuf message that defines a pipeline:
  https://github.com/kubeflow/pipelines/blob/237795539f7b85bac77435e2464367226ee19391/api/v2alpha1/pipeline_spec.proto#L8
  In this initial implementation, we only support components authored through
  Component yaml spec. And we don't support advanced features like conditions,
  static and dynamic loops, etc.

  Example:
    How to use the compiler to construct pipeline_spec json:

      @dsl.pipeline(
        name='name',
        description='description'
      )
      def my_pipeline(a: int = 1, b: str = "default value"):
        ...

      kfp.v2.compiler.Compiler().compile(my_pipeline, 'path/to/pipeline.json')
  """

  def _get_groups_for_ops(self,
                          root_group: dsl.OpsGroup) -> Dict[str, List[str]]:
    """Helper function to get groups that contain the specified ops.

    Each pipeline has a root group. Each group has a list of operators (leaf)
    and groups.
    This function traverse the tree and get all ancestor groups for all
    operators.

    Args:
     root_group: The root node of a ops tree or subtree.

    Returns:
      A dict. Key is the operator's name. Value is a list of ancestor groups
      including the op itself. The list of a given operator is sorted in a way
      that the farthest group is the first and operator itself is the last.
    """

    def _get_op_groups_helper(
        current_groups: List[dsl.OpsGroup],
        ops_to_groups: Dict[str, List[dsl.OpsGroup]]) -> None:
      root_group = current_groups[-1]
      for g in root_group.groups:
        # Add recursive opsgroup in the ops_to_groups
        # such that the i/o dependency can be propagated to the ancester opsgroups
        if g.recursive_ref:
          ops_to_groups[g.name] = [x.name for x in current_groups] + [g.name]
          continue
        current_groups.append(g)
        _get_op_groups_helper(current_groups, ops_to_groups)
        del current_groups[-1]
      for op in root_group.ops:
        ops_to_groups[op.name] = [x.name for x in current_groups] + [op.name]

    ops_to_groups = {}
    current_groups = [root_group]
    _get_op_groups_helper(current_groups, ops_to_groups)
    return ops_to_groups

  #TODO: combine with the _get_groups_for_ops
  def _get_groups_for_opsgroups(
      self, root_group: dsl.OpsGroup) -> Dict[str, List[str]]:
    """Helper function to get groups that contain the specified opsgroup.

    Each pipeline has a root group. Each group has a list of operators (leaf)
    and groups.
    This function traverse the tree and get all ancestor groups for all
    opsgroups.

    Args:
     root_group: The root node of a groups tree or subtree.

    Returns:
      A dict. Key is the opsgroup's name. Value is a list of ancestor groups
      including the opsgroup itself. The list of a given opsgroup is sorted in a
      way that the farthest group is the first and opsgroup itself is the last.
    """

    def _get_opsgroup_groups_helper(
        current_groups: dsl.OpsGroup,
        opsgroups_to_groups: Dict[str, List[dsl.OpsGroup]]) -> None:
      root_group = current_groups[-1]
      for g in root_group.groups:
        # Add recursive opsgroup in the ops_to_groups
        # such that the i/o dependency can be propagated to the ancester opsgroups
        if g.recursive_ref:
          continue
        opsgroups_to_groups[g.name] = [x.name for x in current_groups
                                      ] + [g.name]
        current_groups.append(g)
        _get_opsgroup_groups_helper(current_groups, opsgroups_to_groups)
        del current_groups[-1]

    opsgroups_to_groups = {}
    current_groups = [root_group]
    _get_opsgroup_groups_helper(current_groups, opsgroups_to_groups)
    return opsgroups_to_groups

  def _get_groups(self, root_group: dsl.OpsGroup) -> Dict[str, dsl.OpsGroup]:
    """Helper function to get all groups (not including ops) in a pipeline."""

    def _get_groups_helper(group):
      groups = {group.name: group}
      for g in group.groups:
        # Skip the recursive opsgroup because no templates
        # need to be generated for the recursive opsgroups.
        if not g.recursive_ref:
          groups.update(_get_groups_helper(g))
      return groups

    return _get_groups_helper(root_group)

  def _get_uncommon_ancestors(
      self,
      op_groups: Dict[str, List[str]],
      opsgroup_groups: Dict[str, List[str]],
      op1: dsl.BaseOp,
      op2: dsl.BaseOp,
  ) -> Tuple[List[_GroupOrOp], List[_GroupOrOp]]:
    """Helper function to get unique ancestors between two ops.

    For example, op1's ancestor groups are [root, G1, G2, G3, op1], op2's
    ancestor groups are
    [root, G1, G4, op2], then it returns a tuple ([G2, G3, op1], [G4, op2]).
    """
    #TODO: extract a function for the following two code module
    if op1.name in op_groups:
      op1_groups = op_groups[op1.name]
    elif op1.name in opsgroup_groups:
      op1_groups = opsgroup_groups[op1.name]
    else:
      raise ValueError(op1.name + ' does not exist.')

    if op2.name in op_groups:
      op2_groups = op_groups[op2.name]
    elif op2.name in opsgroup_groups:
      op2_groups = opsgroup_groups[op2.name]
    else:
      raise ValueError(op2.name + ' does not exist.')

    both_groups = [op1_groups, op2_groups]
    common_groups_len = sum(
        1 for x in zip(*both_groups) if x == (x[0],) * len(x))
    group1 = op1_groups[common_groups_len:]
    group2 = op2_groups[common_groups_len:]
    return (group1, group2)

  def _get_condition_params_for_ops(
      self, root_group: dsl.OpsGroup) -> Dict[str, dsl.PipelineParam]:
    """Get parameters referenced in conditions of ops."""
    conditions = collections.defaultdict(set)

    def _get_condition_params_for_ops_helper(group, current_conditions_params):
      new_current_conditions_params = current_conditions_params
      if group.type == 'condition':
        new_current_conditions_params = list(current_conditions_params)
        if isinstance(group.condition.operand1, dsl.PipelineParam):
          new_current_conditions_params.append(group.condition.operand1)
        if isinstance(group.condition.operand2, dsl.PipelineParam):
          new_current_conditions_params.append(group.condition.operand2)
      for op in group.ops:
        for param in new_current_conditions_params:
          conditions[op.name].add(param)
      for g in group.groups:
        # If the subgroup is a recursive opsgroup, propagate the pipelineparams
        # in the condition expression, similar to the ops.
        if g.recursive_ref:
          for param in new_current_conditions_params:
            conditions[g.name].add(param)
        else:
          _get_condition_params_for_ops_helper(g, new_current_conditions_params)

    _get_condition_params_for_ops_helper(root_group, [])
    return conditions

  def _get_next_group_or_op(self, to_visit: List, already_visited: Set):
    """Get next group or op to visit."""
    if len(to_visit) == 0:
      return None
    next = to_visit.pop(0)
    while next in already_visited:
      next = to_visit.pop(0)
    already_visited.add(next)
    return next

  def _get_for_loop_ops(self, new_root) -> Dict[str, dsl.ParallelFor]:
    to_visit = self._get_all_subgroups_and_ops(new_root)
    op_name_to_op = {}
    already_visited = set()

    while len(to_visit):
      next_op = self._get_next_group_or_op(to_visit, already_visited)
      if next_op is None:
        break
      to_visit.extend(self._get_all_subgroups_and_ops(next_op))
      if isinstance(next_op, dsl.ParallelFor):
        op_name_to_op[next_op.name] = next_op

    return op_name_to_op

  def _get_all_subgroups_and_ops(self, group: dsl.OpsGroup):
    """Get all ops and groups contained within this group."""
    subgroups = []
    if hasattr(group, 'ops'):
      subgroups.extend(group.ops)
    if hasattr(group, 'groups'):
      subgroups.extend(group.groups)
    return subgroups

  def _get_inputs_outputs(
      self,
      pipeline: dsl.Pipeline,
      args: List[dsl.PipelineParam],
      root_group: dsl.OpsGroup,
      op_groups: Dict[str, List[str]],
      opsgroup_groups: Dict[str, List[str]],
      condition_params: Dict[str, dsl.PipelineParam],
      op_name_to_for_loop_op: Dict[str, dsl.ParallelFor],
  ) -> Tuple[Dict[str, List[Tuple[dsl.PipelineParam, str]]], Dict[
      str, List[Tuple[dsl.PipelineParam, str]]]]:
    """Get inputs and outputs of each group and op.

    Args:
      pipeline: The instantiated pipeline object.
      args: The list of pipeline function arguments as PipelineParam.
      root_group: The root OpsGroup.
      op_groups: The dict of op name to parent groups.
      opsgroup_groups: The dict of opsgroup name to parent groups.
      condition_params: The dict of group name to pipeline params referenced in
        the conditions in that group.
      op_name_to_for_loop_op: The dict of op name to loop ops.

    Returns:
      A tuple (inputs, outputs).
      inputs and outputs are dicts with key being the group/op names and values
      being list of tuples (param, producing_op_name). producing_op_name is the
      name of the op that produces the param. If the param is a pipeline param
      (no producer op), then producing_op_name is None.
    """
    inputs = collections.defaultdict(set)
    outputs = collections.defaultdict(set)

    # Fix possible missing type -- PipelineParam parsed from command line args
    # doesn't contain the type information, as the `.param_type` is not included
    # during PipelineParam serialization.
    all_params = {param.pattern: param for param in args}
    for op in pipeline.ops.values():
      for param in op.inputs + list(op.outputs.values()) + list(
          condition_params[op.name]):
        if param.pattern not in all_params:
          all_params[param.pattern] = param
        else:
          param.param_type = param.param_type or all_params[
              param.pattern].param_type
          all_params[param.pattern].param_type = param.param_type

    for op in pipeline.ops.values():
      # op's inputs and all params used in conditions for that op are both
      # considered.
      for param in op.inputs + list(condition_params[op.name]):

        # if the value is already provided (immediate value), then no need to
        # expose it as input for its parent groups.
        if param.value:
          continue
        if param.op_name:
          upstream_op = pipeline.ops[param.op_name]
          upstream_groups, downstream_groups = (
              self._get_uncommon_ancestors(op_groups, opsgroup_groups,
                                           upstream_op, op))
          for i, group_name in enumerate(downstream_groups):
            if i == 0:
              # If it is the first uncommon downstream group, then the input
              # comes from the first uncommon upstream group.
              inputs[group_name].add((param, upstream_groups[0]))
            else:
              # If not the first downstream group, then the input is passed down
              # from its ancestor groups so the upstream group is None.
              inputs[group_name].add((param, None))
          for i, group_name in enumerate(upstream_groups):
            if i == len(upstream_groups) - 1:
              # If last upstream group, it is an operator and output comes from container.
              outputs[group_name].add((param, None))
            else:
              # If not last upstream group, output value comes from one of its child.
              outputs[group_name].add((param, upstream_groups[i + 1]))
        else:
          if not op.is_exit_handler:
            for group_name in op_groups[op.name][::-1]:
              # if group is for loop group and param is that loop's param, then the param
              # is created by that for loop ops_group and it shouldn't be an input to
              # any of its parent groups.
              inputs[group_name].add((param, None))
              if group_name in op_name_to_for_loop_op:
                # for example:
                #   loop_group.loop_args.name = 'loop-item-param-99ca152e'
                #   param.name =                'loop-item-param-99ca152e--a'
                loop_group = op_name_to_for_loop_op[group_name]
                if loop_group.loop_args.name in param.name:
                  break

    # Generate the input/output for recursive opsgroups
    # It propagates the recursive opsgroups IO to their ancester opsgroups
    def _get_inputs_outputs_recursive_opsgroup(group: dsl.OpsGroup):
      #TODO: refactor the following codes with the above
      if group.recursive_ref:
        params = [(param, False) for param in group.inputs]
        params.extend([
            (param, True) for param in list(condition_params[group.name])
        ])
        for param, is_condition_param in params:
          if param.value:
            continue

          if param.op_name:
            upstream_op = pipeline.ops[param.op_name]
            upstream_groups, downstream_groups = \
              self._get_uncommon_ancestors(op_groups, opsgroup_groups, upstream_op, group)
            for i, g in enumerate(downstream_groups):
              if i == 0:
                inputs[g].add((param, upstream_groups[0]))
              # There is no need to pass the condition param as argument to the downstream ops.
              #TODO: this might also apply to ops. add a TODO here and think about it.
              elif i == len(downstream_groups) - 1 and is_condition_param:
                continue
              else:
                inputs[g].add((param, None))
            for i, g in enumerate(upstream_groups):
              if i == len(upstream_groups) - 1:
                outputs[g].add((param, None))
              else:
                outputs[g].add((param, upstream_groups[i + 1]))
          elif not is_condition_param:
            for g in op_groups[group.name]:
              inputs[g].add((param, None))
      for subgroup in group.groups:
        _get_inputs_outputs_recursive_opsgroup(subgroup)

    _get_inputs_outputs_recursive_opsgroup(root_group)

    # Generate the input for SubGraph along with parallelfor
    for subgraph in opsgroup_groups:
      if subgraph in op_name_to_for_loop_op:
        # The opsgroup list is sorted with the farthest group as the first and the opsgroup
        # itself as the last. To get the latest opsgroup which is not the opsgroup itself -2 is used.
        parent = opsgroup_groups[subgraph][-2]
        if parent and parent.startswith('subgraph'):
          # propagate only op's pipeline param from subgraph to parallelfor
          loop_op = op_name_to_for_loop_op[subgraph]
          pipeline_param = loop_op.loop_args.items_or_pipeline_param
          if loop_op.items_is_pipeline_param and pipeline_param.op_name:
            inputs[parent].add((pipeline_param, pipeline_param.op_name))

    return inputs, outputs

  def _get_dependencies(
      self,
      pipeline: dsl.Pipeline,
      root_group: dsl.OpsGroup,
      op_groups: Dict[str, List[str]],
      opsgroups_groups: Dict[str, List[str]],
      opsgroups: Dict[str, dsl.OpsGroup],
      condition_params: Dict[str, dsl.PipelineParam],
  ) -> Dict[str, List[_GroupOrOp]]:
    """Get dependent groups and ops for all ops and groups.

    Args:
      pipeline: The instantiated pipeline object.
      root_group: The root OpsGroup.
      op_groups: The dict of op name to parent groups.
      opsgroup_groups: The dict of opsgroup name to parent groups.
      opsgroups: The dict of opsgroup name to opsgroup.
      condition_params: The dict of group name to pipeline params referenced in
        the conditions in that group.

    Returns:
      A dict. Key is group/op name, value is a list of dependent groups/ops.
      The dependencies are calculated in the following way: if op2 depends on
      op1, and their ancestors are [root, G1, G2, op1] and
      [root, G1, G3, G4, op2], then G3 is dependent on G2. Basically dependency
      only exists in the first uncommon ancesters in their ancesters chain. Only
      sibling groups/ops can have dependencies.
    """
    dependencies = collections.defaultdict(set)
    for op in pipeline.ops.values():
      upstream_op_names = set()
      for param in op.inputs + list(condition_params[op.name]):
        if param.op_name:
          upstream_op_names.add(param.op_name)
      upstream_op_names |= set(op.dependent_names)

      for upstream_op_name in upstream_op_names:
        # the dependent op could be either a BaseOp or an opsgroup
        if upstream_op_name in pipeline.ops:
          upstream_op = pipeline.ops[upstream_op_name]
        elif upstream_op_name in opsgroups:
          upstream_op = opsgroups[upstream_op_name]
        else:
          raise ValueError('compiler cannot find the ' + upstream_op_name)

        upstream_groups, downstream_groups = self._get_uncommon_ancestors(
            op_groups, opsgroups_groups, upstream_op, op)
        dependencies[downstream_groups[0]].add(upstream_groups[0])

    # Generate dependencies based on the recursive opsgroups
    #TODO: refactor the following codes with the above
    def _get_dependency_opsgroup(
        group: dsl.OpsGroup, dependencies: Dict[str, List[_GroupOrOp]]) -> None:
      upstream_op_names = set(
          [dependency.name for dependency in group.dependencies])
      if group.recursive_ref:
        for param in group.inputs + list(condition_params[group.name]):
          if param.op_name:
            upstream_op_names.add(param.op_name)

      for op_name in upstream_op_names:
        if op_name in pipeline.ops:
          upstream_op = pipeline.ops[op_name]
        elif op_name in opsgroups:
          upstream_op = opsgroups[op_name]
        else:
          raise ValueError('compiler cannot find the ' + op_name)
        upstream_groups, downstream_groups = (
            self._get_uncommon_ancestors(op_groups, opsgroups_groups,
                                         upstream_op, group))
        dependencies[downstream_groups[0]].add(upstream_groups[0])

      for subgroup in group.groups:
        _get_dependency_opsgroup(subgroup, dependencies)

    _get_dependency_opsgroup(root_group, dependencies)

    return dependencies

  def _resolve_condition_operands(
      self, operand1: Union[str, dsl.PipelineParam],
      operand2: Union[str, dsl.PipelineParam]) -> Tuple[str, str]:
    """Resolves values and PipelineParams for condition operands."""

    # Pre-scan the operand to get the type of constant value if there's any.
    # The value_type can be used to backfill missing PipelineParam.param_type.
    value_type = None
    for value_or_reference in [operand1, operand2]:
      if isinstance(value_or_reference, dsl.PipelineParam):
        continue
      if isinstance(value_or_reference, float):
        value_type = 'Float'
      elif isinstance(value_or_reference, int):
        value_type = 'Integer'
      else:
        value_type = 'String'

    operand_values = []
    for value_or_reference in [operand1, operand2]:
      if isinstance(value_or_reference, dsl.PipelineParam):
        input_name = dsl_component_spec.additional_input_name_for_pipelineparam(
            value_or_reference)
        # Condition operand is always parameters for now.
        value_or_reference.param_type = (
            value_or_reference.param_type or value_type)
        operand_values.append(
            "inputs.parameters['{input_name}'].{value_field}".format(
                input_name=input_name,
                value_field=type_utils.get_parameter_type_field_name(
                    value_or_reference.param_type)))
      else:
        if isinstance(value_or_reference, str):
          operand_values.append("'{}'".format(value_or_reference))
        else:
          operand_values.append(str(value_or_reference))

    return tuple(operand_values)

  def _populate_metrics_in_dag_outputs(
      self,
      ops: List[dsl.ContainerOp],
      op_to_parent_groups: Dict[str, List[str]],
      pipeline_spec: pipeline_spec_pb2.PipelineSpec,
  ) -> None:
    """Populates metrics artifacts in dag outputs.

    Args:
      ops: The list of ops that may produce metrics outputs.
      op_to_parent_groups: The dict of op name to parent groups. Key is the op's
        name. Value is a list of ancestor groups including the op itself. The
        list of a given op is sorted in a way that the farthest group is the
        first and the op itself is the last.
      pipeline_spec: The pipeline_spec to update in-place.
    """
    for op in ops:
      op_task_spec = getattr(op, 'task_spec',
                             pipeline_spec_pb2.PipelineTaskSpec())
      op_component_spec = getattr(op, 'component_spec',
                                  pipeline_spec_pb2.ComponentSpec())

      # Get the tuple of (component_name, task_name) of all its parent groups.
      parent_components_and_tasks = [('_root', '')]
      # skip the op itself and the root group which cannot be retrived via name.
      for group_name in op_to_parent_groups[op.name][1:-1]:
        parent_components_and_tasks.append(
            (dsl_utils.sanitize_component_name(group_name),
             dsl_utils.sanitize_task_name(group_name)))
      # Reverse the order to make the farthest group in the end.
      parent_components_and_tasks.reverse()

      for output_name, artifact_spec in \
          op_component_spec.output_definitions.artifacts.items():

        if artifact_spec.artifact_type.WhichOneof(
            'kind'
        ) == 'schema_title' and artifact_spec.artifact_type.schema_title in [
            io_types.Metrics.TYPE_NAME,
            io_types.ClassificationMetrics.TYPE_NAME,
        ]:
          unique_output_name = '{}-{}'.format(op_task_spec.task_info.name,
                                              output_name)

          sub_task_name = op_task_spec.task_info.name
          sub_task_output = output_name
          for component_name, task_name in parent_components_and_tasks:
            group_component_spec = (
                pipeline_spec.root if component_name == '_root' else
                pipeline_spec.components[component_name])
            group_component_spec.output_definitions.artifacts[
                unique_output_name].CopyFrom(artifact_spec)
            group_component_spec.dag.outputs.artifacts[
                unique_output_name].artifact_selectors.append(
                    pipeline_spec_pb2.DagOutputsSpec.ArtifactSelectorSpec(
                        producer_subtask=sub_task_name,
                        output_artifact_key=sub_task_output,
                    ))
            sub_task_name = task_name
            sub_task_output = unique_output_name

  def _group_to_dag_spec(
      self,
      group: dsl.OpsGroup,
      inputs: Dict[str, List[Tuple[dsl.PipelineParam, str]]],
      outputs: Dict[str, List[Tuple[dsl.PipelineParam, str]]],
      dependencies: Dict[str, List[_GroupOrOp]],
      pipeline_spec: pipeline_spec_pb2.PipelineSpec,
      deployment_config: pipeline_spec_pb2.PipelineDeploymentConfig,
      rootgroup_name: str,
      op_to_parent_groups: Dict[str, List[str]],
  ) -> None:
    """Generate IR spec given an OpsGroup.

    Args:
      group: The OpsGroup to generate spec for.
      inputs: The inputs dictionary. The keys are group/op names and values are
        lists of tuples (param, producing_op_name).
      outputs: The outputs dictionary. The keys are group/op names and values
        are lists of tuples (param, producing_op_name).
      dependencies: The group dependencies dictionary. The keys are group/op
        names, and the values are lists of dependent groups/ops.
      pipeline_spec: The pipeline_spec to update in-place.
      deployment_config: The deployment_config to hold all executors.
      rootgroup_name: The name of the group root. Used to determine whether the
        component spec for the current group should be the root dag.
      op_to_parent_groups: The dict of op name to parent groups. Key is the op's
        name. Value is a list of ancestor groups including the op itself. The
        list of a given op is sorted in a way that the farthest group is the
        first and the op itself is the last.
    """
    group_component_name = dsl_utils.sanitize_component_name(group.name)

    if group.name == rootgroup_name:
      group_component_spec = pipeline_spec.root
    else:
      group_component_spec = pipeline_spec.components[group_component_name]

    # Generate task specs and component specs for the dag.
    subgroups = group.groups + group.ops
    for subgroup in subgroups:
      subgroup_task_spec = getattr(subgroup, 'task_spec',
                                   pipeline_spec_pb2.PipelineTaskSpec())
      subgroup_component_spec = getattr(subgroup, 'component_spec',
                                        pipeline_spec_pb2.ComponentSpec())

      is_recursive_subgroup = (
          isinstance(subgroup, dsl.OpsGroup) and subgroup.recursive_ref)
      if is_recursive_subgroup:
        raise NotImplementedError(
            'Recursive subgroup is not supported in v2 yet.')
      else:
        subgroup_key = subgroup.name

      subgroup_task_spec.task_info.name = (
          subgroup_task_spec.task_info.name or
          dsl_utils.sanitize_task_name(subgroup_key))
      # human_name exists for ops only, and is used to de-dupe component spec.
      subgroup_component_name = (
          subgroup_task_spec.component_ref.name or
          dsl_utils.sanitize_component_name(
              getattr(subgroup, 'human_name', subgroup_key)))
      subgroup_task_spec.component_ref.name = subgroup_component_name

      if isinstance(subgroup, dsl.OpsGroup) and subgroup.type == 'graph':
        raise NotImplementedError(
            'dsl.graph_component is not yet supported in KFP v2 compiler.')

      if isinstance(subgroup, dsl.ContainerOp):
        if hasattr(subgroup, 'importer_spec'):
          importer_task_name = subgroup.task_spec.task_info.name
          importer_comp_name = subgroup.task_spec.component_ref.name
          importer_exec_label = subgroup.component_spec.executor_label
          group_component_spec.dag.tasks[importer_task_name].CopyFrom(
              subgroup.task_spec)
          pipeline_spec.components[importer_comp_name].CopyFrom(
              subgroup.component_spec)
          deployment_config.executors[importer_exec_label].importer.CopyFrom(
              subgroup.importer_spec)

        # Task level caching option.
        subgroup.task_spec.caching_options.enable_cache = subgroup.enable_caching

      subgroup_inputs = inputs.get(subgroup.name, [])
      subgroup_params = [param for param, _ in subgroup_inputs]

      tasks_in_current_dag = [
          dsl_utils.sanitize_task_name(subgroup.name) for subgroup in subgroups
      ]

      input_parameters_in_current_dag = [
          input_name
          for input_name in group_component_spec.input_definitions.parameters
      ]
      input_artifacts_in_current_dag = [
          input_name
          for input_name in group_component_spec.input_definitions.artifacts
      ]

      is_parent_component_root = group_component_spec == pipeline_spec.root

      if isinstance(subgroup, dsl.ContainerOp):
        dsl_component_spec.update_task_inputs_spec(
            subgroup_task_spec,
            group_component_spec.input_definitions,
            subgroup_params,
            tasks_in_current_dag,
            input_parameters_in_current_dag,
            input_artifacts_in_current_dag,
        )

      if isinstance(subgroup, dsl.ParallelFor):
        if subgroup.parallelism is not None:
          warnings.warn(
              'Setting parallelism in ParallelFor is not supported yet.'
              'The setting is ignored.')

        # "Punch the hole", adding additional inputs (other than loop arguments
        # which will be handled separately) needed by its subgroup or tasks.
        loop_subgroup_params = []
        for param in subgroup_params:
          if isinstance(
              param, (_for_loop.LoopArguments, _for_loop.LoopArgumentVariable)):
            continue
          loop_subgroup_params.append(param)

        if subgroup.items_is_pipeline_param:
          # This loop_args is a 'withParam' rather than a 'withItems'.
          # i.e., rather than a static list, it is either the output of
          # another task or an input as global pipeline parameters.
          loop_subgroup_params.append(
              subgroup.loop_args.items_or_pipeline_param)

        dsl_component_spec.build_component_inputs_spec(
            component_spec=subgroup_component_spec,
            pipeline_params=loop_subgroup_params,
            is_root_component=False,
        )
        dsl_component_spec.build_task_inputs_spec(
            subgroup_task_spec,
            loop_subgroup_params,
            tasks_in_current_dag,
            is_parent_component_root,
        )

        if subgroup.items_is_pipeline_param:
          input_parameter_name = (
              dsl_component_spec.additional_input_name_for_pipelineparam(
                  subgroup.loop_args.items_or_pipeline_param))
          loop_arguments_item = '{}-{}'.format(
              input_parameter_name, _for_loop.LoopArguments.LOOP_ITEM_NAME_BASE)

          subgroup_component_spec.input_definitions.parameters[
              loop_arguments_item].type = pipeline_spec_pb2.PrimitiveType.STRING
          subgroup_task_spec.parameter_iterator.items.input_parameter = (
              input_parameter_name)
          subgroup_task_spec.parameter_iterator.item_input = (
              loop_arguments_item)

          # If the loop arguments itself is a loop arguments variable, handle
          # the subvar name.
          loop_args_name, subvar_name = (
              dsl_component_spec._exclude_loop_arguments_variables(
                  subgroup.loop_args.items_or_pipeline_param))
          if subvar_name:
            subgroup_task_spec.inputs.parameters[
                input_parameter_name].parameter_expression_selector = (
                    'parseJson(string_value)["{}"]'.format(subvar_name))
            subgroup_task_spec.inputs.parameters[
                input_parameter_name].component_input_parameter = (
                    dsl_component_spec.additional_input_name_for_pipelineparam(
                        loop_args_name))

        else:
          input_parameter_name = (
              dsl_component_spec.additional_input_name_for_pipelineparam(
                  subgroup.loop_args.full_name))
          raw_values = subgroup.loop_args.to_list_for_task_yaml()

          subgroup_component_spec.input_definitions.parameters[
              input_parameter_name].type = pipeline_spec_pb2.PrimitiveType.STRING
          subgroup_task_spec.parameter_iterator.items.raw = json.dumps(
              raw_values, sort_keys=True)
          subgroup_task_spec.parameter_iterator.item_input = (
              input_parameter_name)

      if isinstance(subgroup, dsl.OpsGroup) and subgroup.type == 'condition':

        # "punch the hole", adding inputs needed by its subgroup or tasks.
        dsl_component_spec.build_component_inputs_spec(
            component_spec=subgroup_component_spec,
            pipeline_params=subgroup_params,
            is_root_component=False,
        )
        dsl_component_spec.build_task_inputs_spec(
            subgroup_task_spec,
            subgroup_params,
            tasks_in_current_dag,
            is_parent_component_root,
        )

        condition = subgroup.condition
        operand_values = []

        operand1_value, operand2_value = self._resolve_condition_operands(
            condition.operand1, condition.operand2)

        condition_string = '{} {} {}'.format(operand1_value, condition.operator,
                                             operand2_value)

        subgroup_task_spec.trigger_policy.CopyFrom(
            pipeline_spec_pb2.PipelineTaskSpec.TriggerPolicy(
                condition=condition_string))

      if isinstance(subgroup, dsl.OpsGroup) and subgroup.type == 'exit_handler':

        # "punch the hole", adding inputs needed by its subgroup or tasks.
        dsl_component_spec.build_component_inputs_spec(
            component_spec=subgroup_component_spec,
            pipeline_params=subgroup_params,
            is_root_component=False,
        )
        dsl_component_spec.build_task_inputs_spec(
            subgroup_task_spec,
            subgroup_params,
            tasks_in_current_dag,
            is_parent_component_root,
        )

      # Generate dependencies section for this task.
      if dependencies.get(subgroup.name, None):
        group_dependencies = list(dependencies[subgroup.name])
        group_dependencies.sort()
        subgroup_task_spec.dependent_tasks.extend(
            [dsl_utils.sanitize_task_name(dep) for dep in group_dependencies])

      # Add component spec if not exists
      if subgroup_component_name not in pipeline_spec.components:
        pipeline_spec.components[subgroup_component_name].CopyFrom(
            subgroup_component_spec)

      # Add task spec
      group_component_spec.dag.tasks[
          subgroup_task_spec.task_info.name].CopyFrom(subgroup_task_spec)

      # Add AIPlatformCustomJobSpec, if applicable.
      custom_job_spec = getattr(subgroup, 'custom_job_spec', None)
      if custom_job_spec:
        executor_label = subgroup_component_spec.executor_label
        if executor_label not in deployment_config.executors:
          deployment_config.executors[
              executor_label].custom_job.custom_job.update(custom_job_spec)

      # Add executor spec, if applicable.
      container_spec = getattr(subgroup, 'container_spec', None)
      # Ignore contaienr_spec if custom_job_spec exists.
      if container_spec and not custom_job_spec:
        if compiler_utils.is_v2_component(subgroup):
          compiler_utils.refactor_v2_container_spec(container_spec)
        executor_label = subgroup_component_spec.executor_label

        if executor_label not in deployment_config.executors:
          deployment_config.executors[executor_label].container.CopyFrom(
              container_spec)

    pipeline_spec.deployment_spec.update(
        json_format.MessageToDict(deployment_config))

    # Surface metrics outputs to the top.
    self._populate_metrics_in_dag_outputs(
        group.ops,
        op_to_parent_groups,
        pipeline_spec,
    )

  def _create_pipeline_spec(
      self,
      args: List[dsl.PipelineParam],
      pipeline: dsl.Pipeline,
  ) -> pipeline_spec_pb2.PipelineSpec:
    """Creates the pipeline spec object.

    Args:
      args: The list of pipeline arguments.
      pipeline: The instantiated pipeline object.

    Returns:
      A PipelineSpec proto representing the compiled pipeline.

    Raises:
      NotImplementedError if the argument is of unsupported types.
    """
    compiler_utils.validate_pipeline_name(pipeline.name)

    deployment_config = pipeline_spec_pb2.PipelineDeploymentConfig()
    pipeline_spec = pipeline_spec_pb2.PipelineSpec()

    pipeline_spec.pipeline_info.name = pipeline.name
    pipeline_spec.sdk_version = 'kfp-{}'.format(kfp.__version__)
    # Schema version 2.0.0 is required for kfp-pipeline-spec>0.1.3.1
    pipeline_spec.schema_version = '2.0.0'

    dsl_component_spec.build_component_inputs_spec(
        component_spec=pipeline_spec.root,
        pipeline_params=args,
        is_root_component=True)

    root_group = pipeline.groups[0]
    opsgroups = self._get_groups(root_group)
    op_name_to_parent_groups = self._get_groups_for_ops(root_group)
    opgroup_name_to_parent_groups = self._get_groups_for_opsgroups(root_group)

    condition_params = self._get_condition_params_for_ops(root_group)
    op_name_to_for_loop_op = self._get_for_loop_ops(root_group)
    inputs, outputs = self._get_inputs_outputs(
        pipeline,
        args,
        root_group,
        op_name_to_parent_groups,
        opgroup_name_to_parent_groups,
        condition_params,
        op_name_to_for_loop_op,
    )
    dependencies = self._get_dependencies(
        pipeline,
        root_group,
        op_name_to_parent_groups,
        opgroup_name_to_parent_groups,
        opsgroups,
        condition_params,
    )

    for opsgroup_name in opsgroups.keys():
      self._group_to_dag_spec(
          opsgroups[opsgroup_name],
          inputs,
          outputs,
          dependencies,
          pipeline_spec,
          deployment_config,
          root_group.name,
          op_name_to_parent_groups,
      )

    # Exit Handler
    if pipeline.groups[0].groups:
      first_group = pipeline.groups[0].groups[0]
      if first_group.type == 'exit_handler':
        exit_handler_op = first_group.exit_op

        # Add exit op task spec
        task_name = exit_handler_op.task_spec.task_info.name
        exit_handler_op.task_spec.dependent_tasks.extend(
            pipeline_spec.root.dag.tasks.keys())
        exit_handler_op.task_spec.trigger_policy.strategy = (
            pipeline_spec_pb2.PipelineTaskSpec.TriggerPolicy.TriggerStrategy
            .ALL_UPSTREAM_TASKS_COMPLETED)
        pipeline_spec.root.dag.tasks[task_name].CopyFrom(
            exit_handler_op.task_spec)

        # Add exit op component spec if it does not exist.
        component_name = exit_handler_op.task_spec.component_ref.name
        if component_name not in pipeline_spec.components:
          pipeline_spec.components[component_name].CopyFrom(
              exit_handler_op.component_spec)

        # Add exit op executor spec if it does not exist.
        executor_label = exit_handler_op.component_spec.executor_label
        if executor_label not in deployment_config.executors:
          deployment_config.executors[executor_label].container.CopyFrom(
              exit_handler_op.container_spec)
          pipeline_spec.deployment_spec.update(
              json_format.MessageToDict(deployment_config))

    return pipeline_spec

  def _validate_exit_handler(self, pipeline):
    """Makes sure there is only one global exit handler.

    This is temporary to be compatible with KFP v1.
    """

    def _validate_exit_handler_helper(group, exiting_op_names, handler_exists):
      if group.type == 'exit_handler':
        if handler_exists or len(exiting_op_names) > 1:
          raise ValueError(
              'Only one global exit_handler is allowed and all ops need to be included.'
          )
        handler_exists = True

      if group.ops:
        exiting_op_names.extend([x.name for x in group.ops])

      for g in group.groups:
        _validate_exit_handler_helper(g, exiting_op_names, handler_exists)

    return _validate_exit_handler_helper(pipeline.groups[0], [], False)

  # TODO: Sanitizing beforehand, so that we don't need to sanitize here.
  def _sanitize_and_inject_artifact(self, pipeline: dsl.Pipeline) -> None:
    """Sanitize operator/param names and inject pipeline artifact location. """

    # Sanitize operator names and param names
    sanitized_ops = {}

    for op in pipeline.ops.values():
      sanitized_name = sanitize_k8s_name(op.name)
      op.name = sanitized_name
      for param in op.outputs.values():
        param.name = sanitize_k8s_name(param.name, True)
        if param.op_name:
          param.op_name = sanitize_k8s_name(param.op_name)
      if op.output is not None and not isinstance(
          op.output, dsl._container_op._MultipleOutputsError):
        op.output.name = sanitize_k8s_name(op.output.name, True)
        op.output.op_name = sanitize_k8s_name(op.output.op_name)
      if op.dependent_names:
        op.dependent_names = [
            sanitize_k8s_name(name) for name in op.dependent_names
        ]
      if isinstance(op, dsl.ContainerOp) and op.file_outputs is not None:
        sanitized_file_outputs = {}
        for key in op.file_outputs.keys():
          sanitized_file_outputs[sanitize_k8s_name(key,
                                                   True)] = op.file_outputs[key]
        op.file_outputs = sanitized_file_outputs
      elif isinstance(op, dsl.ResourceOp) and op.attribute_outputs is not None:
        sanitized_attribute_outputs = {}
        for key in op.attribute_outputs.keys():
          sanitized_attribute_outputs[sanitize_k8s_name(key, True)] = \
            op.attribute_outputs[key]
        op.attribute_outputs = sanitized_attribute_outputs
      if isinstance(op, dsl.ContainerOp):
        if op.input_artifact_paths:
          op.input_artifact_paths = {
              sanitize_k8s_name(key, True): value
              for key, value in op.input_artifact_paths.items()
          }
        if op.artifact_arguments:
          op.artifact_arguments = {
              sanitize_k8s_name(key, True): value
              for key, value in op.artifact_arguments.items()
          }
      sanitized_ops[sanitized_name] = op
    pipeline.ops = sanitized_ops

  def _create_pipeline_v2(
      self,
      pipeline_func: Callable[..., Any],
      pipeline_name: Optional[str] = None,
      pipeline_parameters_override: Optional[Mapping[str, Any]] = None,
  ) -> pipeline_spec_pb2.PipelineJob:
    """Creates a pipeline instance and constructs the pipeline spec from it.

    Args:
      pipeline_func: Pipeline function with @dsl.pipeline decorator.
      pipeline_name: The name of the pipeline. Optional.
      pipeline_parameters_override: The mapping from parameter names to values.
        Optional.

    Returns:
      A PipelineJob proto representing the compiled pipeline.
    """

    # Create the arg list with no default values and call pipeline function.
    # Assign type information to the PipelineParam
    pipeline_meta = _python_op._extract_component_interface(pipeline_func)
    pipeline_name = pipeline_name or pipeline_meta.name

    pipeline_root = getattr(pipeline_func, 'pipeline_root', None)

    args_list = []
    signature = inspect.signature(pipeline_func)
    for arg_name in signature.parameters:
      arg_type = None
      for pipeline_input in pipeline_meta.inputs or []:
        if arg_name == pipeline_input.name:
          arg_type = pipeline_input.type
          break
      if not type_utils.is_parameter_type(arg_type):
        raise TypeError(
            'The pipeline argument "{arg_name}" is viewed as an artifact due to '
            'its type "{arg_type}". And we currently do not support passing '
            'artifacts as pipeline inputs. Consider type annotating the argument'
            ' with a primitive type, such as "str", "int", and "float".'.format(
                arg_name=arg_name, arg_type=arg_type))
      args_list.append(
          dsl.PipelineParam(
              sanitize_k8s_name(arg_name, True), param_type=arg_type))

    with dsl.Pipeline(pipeline_name) as dsl_pipeline:
      pipeline_func(*args_list)

    self._validate_exit_handler(dsl_pipeline)
    self._sanitize_and_inject_artifact(dsl_pipeline)

    # Fill in the default values.
    args_list_with_defaults = []
    if pipeline_meta.inputs:
      args_list_with_defaults = [
          dsl.PipelineParam(
              sanitize_k8s_name(input_spec.name, True),
              param_type=input_spec.type,
              value=input_spec.default) for input_spec in pipeline_meta.inputs
      ]

    # Making the pipeline group name unique to prevent name clashes with templates
    pipeline_group = dsl_pipeline.groups[0]
    temp_pipeline_group_name = uuid.uuid4().hex
    pipeline_group.name = temp_pipeline_group_name

    pipeline_spec = self._create_pipeline_spec(
        args_list_with_defaults,
        dsl_pipeline,
    )

    pipeline_parameters = {
        param.name: param for param in args_list_with_defaults
    }
    # Update pipeline parameters override if there were any.
    pipeline_parameters_override = pipeline_parameters_override or {}
    for k, v in pipeline_parameters_override.items():
      if k not in pipeline_parameters:
        raise ValueError('Pipeline parameter {} does not match any known '
                         'pipeline argument.'.format(k))
      pipeline_parameters[k].value = v

    runtime_config = compiler_utils.build_runtime_config_spec(
        output_directory=pipeline_root, pipeline_parameters=pipeline_parameters)
    pipeline_job = pipeline_spec_pb2.PipelineJob(runtime_config=runtime_config)
    pipeline_job.pipeline_spec.update(json_format.MessageToDict(pipeline_spec))

    return pipeline_job

  def compile(self,
              pipeline_func: Callable[..., Any],
              package_path: str,
              pipeline_name: Optional[str] = None,
              pipeline_parameters: Optional[Mapping[str, Any]] = None,
              type_check: bool = True) -> None:
    """Compile the given pipeline function into pipeline job json.

    Args:
      pipeline_func: Pipeline function with @dsl.pipeline decorator.
      package_path: The output pipeline job .json file path. for example,
        "~/pipeline_job.json"
      pipeline_name: The name of the pipeline. Optional.
      pipeline_parameters: The mapping from parameter names to values. Optional.
      type_check: Whether to enable the type check or not, default: True.
    """
    type_check_old_value = kfp.TYPE_CHECK
    compiling_for_v2_old_value = kfp.COMPILING_FOR_V2
    try:
      kfp.TYPE_CHECK = type_check
      kfp.COMPILING_FOR_V2 = True
      pipeline_job = self._create_pipeline_v2(
          pipeline_func=pipeline_func,
          pipeline_name=pipeline_name,
          pipeline_parameters_override=pipeline_parameters)
      self._write_pipeline(pipeline_job, package_path)
    finally:
      kfp.TYPE_CHECK = type_check_old_value
      kfp.COMPILING_FOR_V2 = compiling_for_v2_old_value

  def _write_pipeline(self, pipeline_job: pipeline_spec_pb2.PipelineJob,
                      output_path: str) -> None:
    """Dump pipeline spec into json file.

    Args:
      pipeline_job: IR pipeline job spec.
      ouput_path: The file path to be written.

    Raises:
      ValueError: if the specified output path doesn't end with the acceptable
      extentions.
    """
    json_text = json_format.MessageToJson(pipeline_job, sort_keys=True)

    if output_path.endswith('.json'):
      with open(output_path, 'w') as json_file:
        json_file.write(json_text)
    else:
      raise ValueError(
          'The output path {} should ends with ".json".'.format(output_path))
