# Copyright 2020 Google LLC
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
"""KFP DSL v2 compiler implementation."""

import datetime
import json
from collections import defaultdict
from deprecated import deprecated
import inspect
import tarfile
import uuid
import zipfile
from typing import Callable, Set, List, Text, Dict, Tuple, Any, Union, Optional

import kfp
from kfp.dsl import _for_loop

from kfp import dsl
from kfp.compiler._k8s_helper import sanitize_k8s_name
from kfp.compiler._op_to_template import _op_to_template

from kfp.components.structures import ComponentSpec, InputSpec
from kfp.components._yaml_utils import dump_yaml
from kfp.dsl._metadata import _extract_pipeline_metadata
from kfp.dsl._ops_group import OpsGroup
from kfp.dsl import ir_types
from kfp.v2.proto import pipeline_spec_pb2
from google.protobuf.json_format import MessageToJson


class Compiler(object):
  """DSL Compiler that compiles pipeline functions into workflow yaml.

  Example:
    How to use the compiler to construct workflow yaml::

      @dsl.pipeline(
        name='name',
        description='description'
      )
      def my_pipeline(a: int = 1, b: str = "default value"):
        ...

      Compiler().compile(my_pipeline, 'path/to/workflow.yaml')
  """

  def _pipelineparam_full_name(self, param):
    """_pipelineparam_full_name converts the names of pipeline parameters
      to unique names in the argo yaml

    Args:
      param(PipelineParam): pipeline parameter
      """
    if param.op_name:
      return param.op_name + '-' + param.name
    return param.name


  def _get_groups_for_ops(self, root_group):
    """Helper function to get belonging groups for each op.

    Each pipeline has a root group. Each group has a list of operators (leaf) and groups.
    This function traverse the tree and get all ancestor groups for all operators.

    Returns:
      A dict. Key is the operator's name. Value is a list of ancestor groups including the
              op itself. The list of a given operator is sorted in a way that the farthest
              group is the first and operator itself is the last.
    """
    def _get_op_groups_helper(current_groups, ops_to_groups):
      root_group = current_groups[-1]
      for g in root_group.groups:
        # Add recursive opsgroup in the ops_to_groups
        # such that the i/o dependency can be propagated to the ancester opsgroups
        if g.recursive_ref:
          raise NotImplementedError('Recursive group is not supported yet in v2.')
        current_groups.append(g)
        _get_op_groups_helper(current_groups, ops_to_groups)
        del current_groups[-1]
      for op in root_group.ops:
        ops_to_groups[op.name] = [x.name for x in current_groups] + [op.name]

    ops_to_groups = {}
    current_groups = [root_group]
    _get_op_groups_helper(current_groups, ops_to_groups)
    return ops_to_groups


  def _get_groups(self, root_group):
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

  def _get_uncommon_ancestors(self, op_groups, op1, op2):
    """Helper function to get unique ancestors between two ops.

    For example, op1's ancestor groups are [root, G1, G2, G3, op1], op2's ancestor groups are
    [root, G1, G4, op2], then it returns a tuple ([G2, G3, op1], [G4, op2]).
    """
    #TODO: extract a function for the following two code module
    if op1.name in op_groups:
      op1_groups = op_groups[op1.name]
    else:
      raise ValueError(op1.name + ' does not exist.')

    if op2.name in op_groups:
      op2_groups = op_groups[op2.name]
    else:
      raise ValueError(op2.name + ' does not exist.')

    both_groups = [op1_groups, op2_groups]
    common_groups_len = sum(1 for x in zip(*both_groups) if x==(x[0],)*len(x))
    group1 = op1_groups[common_groups_len:]
    group2 = op2_groups[common_groups_len:]
    return (group1, group2)

  def _get_next_group_or_op(cls, to_visit: List, already_visited: Set):
    """Get next group or op to visit."""
    if len(to_visit) == 0:
      return None
    next = to_visit.pop(0)
    while next in already_visited:
      next = to_visit.pop(0)
    already_visited.add(next)
    return next

  def _get_all_subgroups_and_ops(self, op):
    """Get all ops and groups contained within this group."""
    subgroups = []
    if hasattr(op, 'ops'):
      subgroups.extend(op.ops)
    if hasattr(op, 'groups'):
      subgroups.extend(op.groups)
    return subgroups

  def _get_inputs_outputs(
      self,
      pipeline,
      root_group,
      op_groups,
      pipeline_spec: pipeline_spec_pb2.PipelineSpec,
  ):
    """Get inputs and outputs of each group and op.

    Returns:
      A tuple (inputs, outputs).
      inputs and outputs are dicts with key being the group/op names and values being list of
      tuples (param_name, producing_op_name). producing_op_name is the name of the op that
      produces the param. If the param is a pipeline param (no producer op), then
      producing_op_name is None.
    """
    inputs = defaultdict(set)
    outputs = defaultdict(set)

    for op in pipeline.ops.values():
      from kfp.components.structures import ComponentSpec
      for param in op.inputs:
        # if the value is already provided (immediate value), then no need to expose
        # it as input for its parent groups.
        if param.value:
          continue
        if param.op_name:
          upstream_op = pipeline.ops[param.op_name]
          upstream_groups, downstream_groups = \
            self._get_uncommon_ancestors(op_groups, upstream_op, op)
          for i, group_name in enumerate(downstream_groups):
            if i == 0:
              # If it is the first uncommon downstream group, then the input comes from
              # the first uncommon upstream group.
              inputs[group_name].add((param.full_name, upstream_groups[0]))
            else:
              # If not the first downstream group, then the input is passed down from
              # its ancestor groups so the upstream group is None.
              inputs[group_name].add((param.full_name, None))
          for i, group_name in enumerate(upstream_groups):
            if i == len(upstream_groups) - 1:
              # If last upstream group, it is an operator and output comes from container.
              outputs[group_name].add((param.full_name, None))
            else:
              # If not last upstream group, output value comes from one of its child.
              outputs[group_name].add((param.full_name, upstream_groups[i+1]))

    return inputs, outputs

  def _get_dependencies(self, pipeline, root_group, op_groups, opsgroups):
    """Get dependent groups and ops for all ops and groups.

    Returns:
      A dict. Key is group/op name, value is a list of dependent groups/ops.
      The dependencies are calculated in the following way: if op2 depends on op1,
      and their ancestors are [root, G1, G2, op1] and [root, G1, G3, G4, op2],
      then G3 is dependent on G2. Basically dependency only exists in the first uncommon
      ancesters in their ancesters chain. Only sibling groups/ops can have dependencies.
    """
    dependencies = defaultdict(set)
    for op in pipeline.ops.values():
      upstream_op_names = set()
      for param in op.inputs:
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

        upstream_groups, downstream_groups = self._get_uncommon_ancestors(op_groups, upstream_op, op)
        dependencies[downstream_groups[0]].add(upstream_groups[0])

    return dependencies


  def _resolve_value_or_reference(self, value_or_reference, potential_references):
    """_resolve_value_or_reference resolves values and PipelineParams, which could be task parameters or input parameters.

    Args:
      value_or_reference: value or reference to be resolved. It could be basic python types or PipelineParam
      potential_references(dict{str->str}): a dictionary of parameter names to task names
      """
    if isinstance(value_or_reference, dsl.PipelineParam):
      parameter_name = self._pipelineparam_full_name(value_or_reference)
      task_names = [task_name for param_name, task_name in potential_references if param_name == parameter_name]
      if task_names:
        task_name = task_names[0]
        # When the task_name is None, the parameter comes directly from ancient ancesters
        # instead of parents. Thus, it is resolved as the input parameter in the current group.
        if task_name is None:
          return '{{inputs.parameters.%s}}' % parameter_name
        else:
          return '{{tasks.%s.outputs.parameters.%s}}' % (task_name, parameter_name)
      else:
        return '{{inputs.parameters.%s}}' % parameter_name
    else:
      return str(value_or_reference)


  def get_arguments_for_sub_group(
      self,
      sub_group: Union[OpsGroup, dsl._container_op.BaseOp],
      is_recursive_subgroup: Optional[bool],
      inputs: Dict[Text, Tuple[Text, Text]],
  ):
    arguments = []
    for param_name, dependent_name in inputs[sub_group.name]:
      if is_recursive_subgroup:
        raise NotImplementedError('Recursive group not supported yet in v2.')
        for input_name, input in sub_group.arguments.items():
          if param_name == self._pipelineparam_full_name(input):
            break
        referenced_input = sub_group.recursive_ref.arguments[input_name]
        argument_name = self._pipelineparam_full_name(referenced_input)
      else:
        argument_name = param_name

      # Preparing argument. It can be pipeline input reference, task output reference or loop item (or loop item attribute
      if dependent_name:
        argument_value = '{{tasks.%s.outputs.parameters.%s}}' % (dependent_name, param_name)
      else:
        argument_value = '{{inputs.parameters.%s}}' % param_name

      arguments.append({
          'name': argument_name,
          'value': argument_value,
      })

    arguments.sort(key=lambda x: x['name'])
    return arguments


  def _create_pipeline_spec(self,
                            args: List[dsl.PipelineParam],
                            pipeline: dsl.Pipeline,
                            ) -> pipeline_spec_pb2.PipelineSpec:
    """Create the pipeline spec object."""

    pipeline_spec = pipeline_spec_pb2.PipelineSpec()
    pipeline_spec.pipeline_info.name = pipeline.name or 'Pipeline'
    pipeline_spec.sdk_version = "kfp.v2"
    pipeline_spec.schema_version = "dummy.v1"

    # Pipeline Parameters
    for arg in args:
      if arg.value is not None:
        if isinstance(arg.value, int):
          pipeline_spec.runtime_parameters[arg.name].type = pipeline_spec_pb2.PrimitiveType().INT
          pipeline_spec.runtime_parameters[arg.name].default_value.int_value = arg.value
        elif isinstance(arg.value, float):
          pipeline_spec.runtime_parameters[arg.name].type = pipeline_spec_pb2.PrimitiveType().DOUBLE
          pipeline_spec.runtime_parameters[arg.name].default_value.double_value = arg.value
        elif isinstance(arg.value, str):
          pipeline_spec.runtime_parameters[arg.name].type = pipeline_spec_pb2.PrimitiveType().STRING
          pipeline_spec.runtime_parameters[arg.name].default_value.string_value = arg.value
        else:
            raise NotImplementedError('Unexpected parameter type with: "{}".'.format(str(arg.value)))

    root_group = pipeline.groups[0]
    opsgroups = self._get_groups(root_group)
    op_name_to_parent_groups = self._get_groups_for_ops(root_group)
    inputs, outputs = self._get_inputs_outputs(
        pipeline,
        root_group,
        op_name_to_parent_groups,
        pipeline_spec,
    )
    dependencies = self._get_dependencies(
        pipeline,
        root_group,
        op_name_to_parent_groups,
        opsgroups,
    )

    deployment_config = pipeline_spec_pb2.PipelineDeploymentConfig()
    importer_tasks = []


    def _get_input_artifact_type(
        input_name: str,
        component_spec: ComponentSpec,
    ) -> str:
      """Find the input artifact type by input name."""
      for input in component_spec.inputs:
        if input.name == input_name:
          return ir_types._artifact_types_mapping.get(input.type.lower()) or ''
      return ''


    for op in pipeline.ops.values():
      component_spec = op._metadata
      task = pipeline_spec.tasks.add()
      task.CopyFrom(op.task_spec)
      deployment_config.executors[task.task_info.name].container.CopyFrom(op.container_spec)

      # Check if need to insert importer node
      for input_name in task.inputs.artifacts:
        if task.inputs.artifacts[input_name].producer_task == '':
          artifact_type = _get_input_artifact_type(input_name, component_spec)

          from kfp.v2.compiler import importer_node
          importer_task, importer_spec = importer_node.build_importer_spec(
              task, input_name, artifact_type)
          importer_tasks.append(importer_task)

          task.inputs.artifacts[input_name].producer_task = importer_task.task_info.name
          task.inputs.artifacts[input_name].output_artifact_key = importer_node._OUTPUT_KEY

          deployment_config.executors[importer_task.executor_label].importer.CopyFrom(importer_spec)

    pipeline_spec.deployment_config.Pack(deployment_config)

    if len(importer_tasks) > 0:
      pipeline_spec.tasks.extend(importer_tasks)

    return pipeline_spec


  def _sanitize_and_inject_artifact(self, pipeline: dsl.Pipeline):
    """Sanitize operator/param names and inject pipeline artifact location."""

    # Sanitize operator names and param names
    sanitized_ops = {}

    for op in pipeline.ops.values():
      sanitized_name = sanitize_k8s_name(op.name)
      op.name = sanitized_name
      for param in op.outputs.values():
        param.name = sanitize_k8s_name(param.name, True)
        if param.op_name:
          param.op_name = sanitize_k8s_name(param.op_name)
      if op.output is not None and not isinstance(op.output, dsl._container_op._MultipleOutputsError):
        op.output.name = sanitize_k8s_name(op.output.name, True)
        op.output.op_name = sanitize_k8s_name(op.output.op_name)
      if op.dependent_names:
        op.dependent_names = [sanitize_k8s_name(name) for name in op.dependent_names]
      if isinstance(op, dsl.ContainerOp) and op.file_outputs is not None:
        sanitized_file_outputs = {}
        for key in op.file_outputs.keys():
          sanitized_file_outputs[sanitize_k8s_name(key, True)] = op.file_outputs[key]
        op.file_outputs = sanitized_file_outputs
      elif isinstance(op, dsl.ResourceOp) and op.attribute_outputs is not None:
        sanitized_attribute_outputs = {}
        for key in op.attribute_outputs.keys():
          sanitized_attribute_outputs[sanitize_k8s_name(key, True)] = \
            op.attribute_outputs[key]
        op.attribute_outputs = sanitized_attribute_outputs
      if isinstance(op, dsl.ContainerOp):
        if op.input_artifact_paths:
          op.input_artifact_paths = {sanitize_k8s_name(key, True): value for key, value in op.input_artifact_paths.items()}
        if op.artifact_arguments:
          op.artifact_arguments = {sanitize_k8s_name(key, True): value for key, value in op.artifact_arguments.items()}
      sanitized_ops[sanitized_name] = op
    pipeline.ops = sanitized_ops


  def _create_pipeline(self,
                       pipeline_func: Callable,
                       pipeline_name: Text=None,
                       pipeline_description: Text=None,
                       params_list: List[dsl.PipelineParam]=None,
                       ) -> pipeline_spec_pb2.PipelineSpec:
    """ Internal implementation of create_pipeline."""
    params_list = params_list or []

    # Create the arg list with no default values and call pipeline function.
    # Assign type information to the PipelineParam
    pipeline_meta = _extract_pipeline_metadata(pipeline_func)
    pipeline_meta.name = pipeline_name or pipeline_meta.name
    pipeline_meta.description = pipeline_description or pipeline_meta.description
    pipeline_name = sanitize_k8s_name(pipeline_meta.name)

    # Need to first clear the default value of dsl.PipelineParams. Otherwise, it
    # will be resolved immediately in place when being to each component.
    default_param_values = {}
    for param in params_list:
      default_param_values[param.name] = param.value
      param.value = None

    # Currently only allow specifying pipeline params at one place.
    if params_list and pipeline_meta.inputs:
      raise ValueError('Either specify pipeline params in the pipeline function, or in "params_list", but not both.')

    args_list = []
    signature = inspect.signature(pipeline_func)
    for arg_name in signature.parameters:
      arg_type = None
      for input in pipeline_meta.inputs or []:
        if arg_name == input.name:
          arg_type = input.type
          break
      args_list.append(dsl.PipelineParam(sanitize_k8s_name(arg_name, True), param_type=arg_type))

    with dsl.Pipeline(pipeline_name) as dsl_pipeline:
      pipeline_func(*args_list)

    # need to sanitize unless we remove sanitization from everywhere e.g.  kfp/dsl/...
    self._sanitize_and_inject_artifact(dsl_pipeline)

    # Fill in the default values.
    args_list_with_defaults = []
    if pipeline_meta.inputs:
      args_list_with_defaults = [
          dsl.PipelineParam(sanitize_k8s_name(input_spec.name, True), value=input_spec.default)
        for input_spec in pipeline_meta.inputs
      ]
    elif params_list:
      # Or, if args are provided by params_list, fill in pipeline_meta.
      for param in params_list:
        param.value = default_param_values[param.name]

      args_list_with_defaults = params_list
      pipeline_meta.inputs = [
          InputSpec(
              name=param.name,
              type=param.param_type,
              default=param.value) for param in params_list]

    pipeline_spec = self._create_pipeline_spec(
        args_list_with_defaults,
        dsl_pipeline,
    )

    print('###proto###\n', MessageToJson(pipeline_spec))
    return pipeline_spec

  def compile(self, pipeline_func, package_path, type_check=True):
    """Compile the given pipeline function into workflow yaml.

    Args:
      pipeline_func: Pipeline functions with @dsl.pipeline decorator.
      package_path: The output workflow tar.gz file path. for example, "~/a.tar.gz"
      type_check: Whether to enable the type check or not, default: True.
    """
    import kfp
    type_check_old_value = kfp.TYPE_CHECK
    try:
      kfp.TYPE_CHECK = type_check
      self._create_and_write_pipeline_spec(
          pipeline_func=pipeline_func,
          package_path=package_path)
    finally:
      kfp.TYPE_CHECK = type_check_old_value

  @staticmethod
  def _write_pipeline(pipeline_spec: pipeline_spec_pb2.PipelineSpec, package_path: Text = None):
    """Dump pipeline workflow into yaml spec and write out in the format specified by the user.

    Args:
      workflow: Workflow spec of the pipline, dict.
      package_path: file path to be written. If not specified, a yaml_text string will be returned.
    """
    json_text = MessageToJson(pipeline_spec)

    if package_path is None:
      return json_text

    if package_path.endswith('.tar.gz') or package_path.endswith('.tgz'):
      from contextlib import closing
      from io import BytesIO
      with tarfile.open(package_path, "w:gz") as tar:
          with closing(BytesIO(json_text.encode())) as json_file:
            tarinfo = tarfile.TarInfo('pipeline.json')
            tarinfo.size = len(json_file.getvalue())
            tar.addfile(tarinfo, fileobj=json_file)
    elif package_path.endswith('.zip'):
      with zipfile.ZipFile(package_path, "w") as zip:
        zipinfo = zipfile.ZipInfo('pipeline.json')
        zipinfo.compress_type = zipfile.ZIP_DEFLATED
        zip.writestr(zipinfo, json_text)
    elif package_path.endswith('.json'):
      with open(package_path, 'w') as json_file:
        json_file.write(json_text)
    else:
      raise ValueError(
          'The output path '+ package_path +
          ' should ends with one of the following formats: '
          '[.tar.gz, .tgz, .zip, .json]')

  def _create_and_write_pipeline_spec(
      self,
      pipeline_func: Callable,
      pipeline_name: Text=None,
      pipeline_description: Text=None,
      params_list: List[dsl.PipelineParam]=None,
      package_path: Text=None
  ) -> None:
    """Compile the given pipeline function and dump it to specified file format."""
    pipeline = self._create_pipeline(
        pipeline_func,
        pipeline_name,
        pipeline_description,
        params_list)
    self._write_pipeline(pipeline, package_path)
