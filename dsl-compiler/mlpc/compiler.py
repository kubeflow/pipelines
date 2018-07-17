# Copyright 2018 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
# in compliance with the License. You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software distributed under the License
# is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
# or implied. See the License for the specific language governing permissions and limitations under
# the License.


from collections import defaultdict
import inspect
import mlp
import re
import string
import yaml


class Compiler(object):
  """DSL Compiler. 

  It compiles DSL pipeline functions into workflow yaml. Example usage:
  ```python
  @mlp.pipeline(
    name='name',
    description='description'
  )
  def my_pipeline(a: mlp.PipelineParam, b: mlp.PipelineParam):
    pass

  Compiler().compile(my_pipeline, 'path/to/workflow.yaml')
  ```
  """

  def _sanitize_name(self, name):
    return re.sub(r"[^\w]", '', name).lower()

  def _param_full_name(self, param):
    if param.op_name:
      return param.op_name + '-' + param.name
    return self._sanitize_name(param.name)

  def _op_to_template(self, op):
    """Generate template given an operator inherited from mlp.ContainerOp."""

    processed_args = None
    if op.arguments:
      processed_args = list(map(str, op.arguments))
      for i, _ in enumerate(processed_args):
        if op.argument_inputs:
          for param in op.argument_inputs:
            full_name = self._param_full_name(param)
            processed_args[i] = re.sub(str(param), '{{inputs.parameters.%s}}' % full_name,
                                       processed_args[i])

    input_parameters = []
    for param in op.inputs:
      one_parameter = {'name': self._param_full_name(param)}
      if param.value:
        one_parameter['value'] = str(param.value)
      input_parameters.append(one_parameter)
    # Sort to make the results deterministic.
    input_parameters.sort(key=lambda x: x['name'])

    output_parameters = []
    for param in op.outputs.values():
      output_parameters.append({
          'name': self._param_full_name(param),
          'valueFrom': {'path': op.file_outputs[param.name]}
      })
    output_parameters.sort(key=lambda x: x['name'])

    template = {
      'name': op.name,
      'container': {
        'image': op.image,
        'command': op.command
      }
    }
    if processed_args:
      template['container']['args'] = processed_args
    if input_parameters:
      template['inputs'] = {'parameters': input_parameters}
    if output_parameters:
      template['outputs'] = {'parameters': output_parameters}
    return template

  def _get_groups_for_ops(self, pipeline):
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
        current_groups.append(g)
        _get_op_groups_helper(current_groups, ops_to_groups)
        del current_groups[-1]
      for op in root_group.ops:
        ops_to_groups[op.name] = [x.name for x in current_groups] + [op.name]

    ops_to_groups = {}
    current_groups = [pipeline.groups[0]]
    _get_op_groups_helper(current_groups, ops_to_groups)
    return ops_to_groups

  def _get_groups(self, pipeline):
    """Helper function to get all groups (not including ops) in a pipeline."""

    def _get_groups_helper(group):
      groups = [group]
      for g in group.groups:
        groups += _get_groups_helper(g)
      return groups

    return _get_groups_helper(pipeline.groups[0])
        
  def _get_uncommon_ancestors(self, op_groups, op1, op2):
    """Helper function to get unique ancestors between two ops.

    For example, op1's ancestor groups are [root, G1, G2, G3, op1], op2's ancestor groups are
    [root, G1, G4, op2], then it returns a tuple ([G2, G3, op1], [G4, op2]).
    """
    both_groups = [op_groups[op1.name], op_groups[op2.name]]
    common_groups_len = sum(1 for x in zip(*both_groups) if x==(x[0],)*len(x))
    group1 = op_groups[op1.name][common_groups_len:]
    group2 = op_groups[op2.name][common_groups_len:]
    return (group1, group2)

  def _get_inputs_outputs(self, pipeline, op_groups):
    """Get inputs and outputs of each group and op.

    Returns:
      A tuple (inputs, outputs).
      inputs and outputs are dicts with key being the group/op names and values being list of
      tuples (param_name, producing_op_name). producing_op_name is the name of the op that
      produces the param. If the param is a pipeline param (no producer op), then
      producing_op_name is None.
    """
    inputs = defaultdict(list)
    outputs = defaultdict(list)
    for op in pipeline.ops.values():
      for param in op.inputs:
        full_name = self._param_full_name(param)
        if param.op_name:
          upstream_op = pipeline.ops[param.op_name]
          upstream_groups, downstream_groups = self._get_uncommon_ancestors(
              op_groups, upstream_op, op)
          for i, g in enumerate(downstream_groups):
            if i == 0:
              # If it is the first uncommon downstream group, then the input comes from
              # the first uncommon upstream group.
              inputs[g].append((full_name, upstream_groups[0]))
            else:
              # If not the first downstream group, then the input is passed down from
              # its ancestor groups so the upstream group is None.
              inputs[g].append(full_name, None)
          for i, g in enumerate(upstream_groups):
            if i == len(upstream_groups) - 1:
              # If last upstream group, it is an operator and output comes from container.
              outputs[g].append((full_name, None))
            else:
              # If not last upstream group, output value comes from one of its child.
              outputs[g].append((full_name, upstream_groups[i+1]))
        elif param.value is None:
          for g in op_groups[op.name]:
            inputs[g].append((full_name, None))
    return inputs, outputs
    
  def _get_dependencies(self, pipeline, op_groups):
    """Get dependent groups and ops for all ops and groups.

    Returns:
      A dict. Key is group/op name, value is a list of dependent groups/ops.
      The dependencies are calculated in the following way: if op2 depends on op1,
      and their ancestors are [root, G1, G2, op1] and [root, G1, G3, G4, op2],
      then G3 is dependent on G2. Basically dependency only exists in the first uncommon
      ancesters in their ancesters chain. Only sibling groups/ops can have dependencies.
    """
    dependencies = defaultdict(list)
    for op in pipeline.ops.values():
      unstream_op_names = set()
      for param in op.inputs:
        if param.op_name:
          unstream_op_names.add(param.op_name)
      unstream_op_names |= set(op.dependent_op_names)

      for op_name in unstream_op_names:
        upstream_op = pipeline.ops[op_name]
        upstream_groups, downstream_groups = self._get_uncommon_ancestors(
            op_groups, upstream_op, op)
        dependencies[downstream_groups[0]].append(upstream_groups[0])
    return dependencies

  def _group_to_template(self, group, inputs, outputs, dependencies):
    """Generate template given an OpsGroup.
    
    inputs, outputs, dependencies are all helper dicts.
    """
    template = {'name': group.name}

    # Generate inputs section.
    if inputs.get(group.name, None):
      template_inputs = [{'name': x[0]} for x in inputs[group.name]]
      template_inputs.sort(key=lambda x: x['name'])
      template['inputs'] = {
        'parameters': template_inputs
      }

    # Generate outputs section.
    if outputs.get(group.name, None):
      template_outputs = []
      for param_name, depentent_name in outputs[group.name]:
        template_outputs.append({
          'name': param_name,
          'valueFrom': {
            'parameter': '{{tasks.%s.outputs.parameters.%s}}' % (depentent_name, param_name)
          }
        })
      template_outputs.sort(key=lambda x: x['name'])
      template['outputs'] = {'parameters': template_outputs}

    # Generate tasks section.
    tasks = []
    for sub_group in group.groups + group.ops:
      task = {
        'name': sub_group.name,
        'template': sub_group.name,
      }
      # Generate dependencies section for this task.
      if dependencies.get(sub_group.name, None):
        group_dependencies = list(dependencies[sub_group.name])
        group_dependencies.sort()
        task['dependencies'] = group_dependencies

      # Generate arguments section for this task.
      if inputs.get(sub_group.name, None):
        arguments = []
        for param_name, dependent_name in inputs[sub_group.name]:
          if dependent_name:
            # The value comes from an upstream sibling.
            arguments.append({
              'name': param_name,
              'value': '{{tasks.%s.outputs.parameters.%s}}' % (dependent_name, param_name)
            })
          else:
            # The value comes from its parent.
            arguments.append({
              'name': param_name,
              'value': '{{inputs.parameters.%s}}' % param_name
            })
        arguments.sort(key=lambda x: x['name'])
        task['arguments'] = {'parameters': arguments}
      tasks.append(task)
    tasks.sort(key=lambda x: x['name'])
    template['dag'] = {'tasks': tasks}
    return template     

  def _create_templates(self, pipeline):
    """Create all groups and ops templates in the pipeline."""

    op_groups = self._get_groups_for_ops(pipeline)
    inputs, outputs = self._get_inputs_outputs(pipeline, op_groups)
    dependencies = self._get_dependencies(pipeline, op_groups)
    groups = self._get_groups(pipeline)

    templates = []
    for g in groups:
      templates.append(self._group_to_template(g, inputs, outputs, dependencies))

    for op in pipeline.ops.values():
      templates.append(self._op_to_template(op))
    return templates

  def _create_pipeline_workflow(self, args, pipeline):
    """Create workflow for the pipeline."""

    input_params = [{'name': arg} for arg in args]
    templates = self._create_templates(pipeline)
    templates.sort(key=lambda x: x['name'])

    exit_handler = None
    if pipeline.groups[0].groups:
      first_group = pipeline.groups[0].groups[0]
      if first_group.type == 'exit_handler':
        exit_handler = first_group.exit_op

    workflow = {
      'apiVersion': 'argoproj.io/v1alpha1',
      'kind': 'Workflow',
      'metadata': {'generateName': pipeline.name + '-'},
      'spec': {
        'entrypoint': pipeline.name,
        'templates': templates,
        'arguments': {'parameters': input_params}
      }
    }
    if exit_handler:
      workflow['spec']['onExit'] = exit_handler.name
    return workflow

  def _validate_args(self, argspec): 
    for arg in argspec.args:
      if arg not in argspec.annotations:
        raise ValueError('There is no annotation for argument "%s".' % arg)
      if not issubclass(argspec.annotations[arg], mlp.PipelineParam):
        raise ValueError(
            'Annotation of argument "%s" has to be type mlp.PipelineParam or its child.' % arg)

  def _validate_exit_handler(self, pipeline):
    """Makes sure there is only one global exit handler.

    Note this is a temporary workaround until argo supports local exit handler.
    """

    def _validate_exit_handler_helper(group, exiting_op_names, handler_exists):
      if group.type == 'exit_handler':
        if (handler_exists or len(exiting_op_names) > 1
            or group.exit_op.name != exiting_op_names[0]):
          raise ValueError('Only one global exit_handler is allowed and all ops need to be included.')
        handler_exists = True

      if group.ops:
        exiting_op_names.extend([x.name for x in group.ops])

      for g in group.groups:
        _validate_exit_handler_helper(g, exiting_op_names, handler_exists)

    return _validate_exit_handler_helper(pipeline.groups[0], [], False)

  def _compile(self, pipeline_func):
    """Compile the given pipeline function into workflow."""

    argspec = inspect.getfullargspec(pipeline_func)
    self._validate_args(argspec)

    args_list = [mlp.PipelineParam(arg_name) for arg_name in argspec.args]
    registered_pipeline_functions = mlp.Pipeline.get_pipeline_functions()
    if pipeline_func not in registered_pipeline_functions:
      raise ValueError('Please use a function with @mlp.pipeline decorator.')

    pipeline_name, _ = mlp.Pipeline.get_pipeline_functions()[pipeline_func]
    pipeline_name = self._sanitize_name(pipeline_name)
    with mlp.Pipeline(pipeline_name) as p:
      pipeline_func(*args_list)

    # Remove when argo supports local exit handler.    
    self._validate_exit_handler(p)

    workflow = self._create_pipeline_workflow(argspec.args, p)
    return workflow

  def compile(self, pipeline_func, package_path):
    """Compile the given pipeline function into workflow yaml.

    Args:
      pipeline_func: pipeline functions with @mlp.pipeline decorator.
      package_path: the output workflow yaml file path.
    """
    workflow = self._compile(pipeline_func)
    yaml.Dumper.ignore_aliases = lambda *args : True
    with open(package_path, 'w') as f:
      yaml.dump(workflow, f, default_flow_style=False)
