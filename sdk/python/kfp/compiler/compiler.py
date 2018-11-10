# Copyright 2018 Google LLC
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


from collections import defaultdict
import copy
import inspect
import kfp.dsl as dsl
import re
import string
import tarfile
import tempfile
import yaml


class Compiler(object):
  """DSL Compiler. 

  It compiles DSL pipeline functions into workflow yaml. Example usage:
  ```python
  @dsl.pipeline(
    name='name',
    description='description'
  )
  def my_pipeline(a: dsl.PipelineParam, b: dsl.PipelineParam):
    pass

  Compiler().compile(my_pipeline, 'path/to/workflow.yaml')
  ```
  """

  def _sanitize_name(self, name):
    return re.sub('-+', '-', re.sub('[^-0-9a-z]+', '-', name.lower())).lstrip('-').rstrip('-') #from _make_kubernetes_name

  def _param_full_name(self, param):
    if param.op_name:
      return param.op_name + '-' + param.name
    return self._sanitize_name(param.name)

  def _build_conventional_artifact(self, name):
    return {
      'name': name,
      'path': '/' + name + '.json',
      's3': {
        # TODO: parameterize namespace for minio service
        'endpoint': 'minio-service.kubeflow:9000',
        'bucket': 'mlpipeline',
        'key': 'runs/{{workflow.uid}}/{{pod.name}}/' + name + '.tgz',
        'insecure': True,
        'accessKeySecret': {
          'name': 'mlpipeline-minio-artifact',
          'key': 'accesskey',
        },
        'secretKeySecret': {
          'name': 'mlpipeline-minio-artifact',
          'key': 'secretkey'
        }
      },
    }

  def _op_to_template(self, op):
    """Generate template given an operator inherited from dsl.ContainerOp."""

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
      }
    }
    if processed_args:
      template['container']['args'] = processed_args
    if input_parameters:
      template['inputs'] = {'parameters': input_parameters}

    template['outputs'] = {}
    if output_parameters:
      template['outputs'] = {'parameters': output_parameters}

    # Generate artifact for metadata output
    # The motivation of appending the minio info in the yaml
    # is to specify a unique path for the metadata.
    # TODO: after argo addresses the issue that configures a unique path
    # for the artifact output when default artifact repository is configured,
    # this part needs to be updated to use the default artifact repository.
    output_artifacts = []
    output_artifacts.append(self._build_conventional_artifact('mlpipeline-ui-metadata'))
    output_artifacts.append(self._build_conventional_artifact('mlpipeline-metrics'))
    template['outputs']['artifacts'] = output_artifacts
    if op.command:
      template['container']['command'] = op.command

    # Set resources.
    if op.memory_limit or op.cpu_limit or op.memory_request or op.cpu_request:
      template['container']['resources'] = {}
    if op.memory_limit or op.cpu_limit:
      template['container']['resources']['limits'] = {}
      if op.memory_limit:
        template['container']['resources']['limits']['memory'] = op.memory_limit
      if op.cpu_limit:
        template['container']['resources']['limits']['cpu'] = op.cpu_limit

    if op.memory_request or op.cpu_request:
      template['container']['resources']['requests'] = {}
      if op.memory_request:
        template['container']['resources']['requests']['memory'] = op.memory_request
      if op.cpu_request:
        template['container']['resources']['requests']['cpu'] = op.cpu_request

    return template

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
      groups = [group]
      for g in group.groups:
        groups += _get_groups_helper(g)
      return groups

    return _get_groups_helper(root_group)
        
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

  def _get_inputs_outputs(self, pipeline, root_group, op_groups):
    """Get inputs and outputs of each group and op.

    Returns:
      A tuple (inputs, outputs).
      inputs and outputs are dicts with key being the group/op names and values being list of
      tuples (param_name, producing_op_name). producing_op_name is the name of the op that
      produces the param. If the param is a pipeline param (no producer op), then
      producing_op_name is None.
    """
    condition_params = self._get_condition_params_for_ops(root_group)
    inputs = defaultdict(set)
    outputs = defaultdict(set)
    for op in pipeline.ops.values():
      # op's inputs and all params used in conditions for that op are both considered.
      for param in op.inputs + list(condition_params[op.name]):
        # if the value is already provided (immediate value), then no need to expose
        # it as input for its parent groups. 
        if param.value:
          continue

        full_name = self._param_full_name(param)
        if param.op_name:
          upstream_op = pipeline.ops[param.op_name]
          upstream_groups, downstream_groups = self._get_uncommon_ancestors(
              op_groups, upstream_op, op)
          for i, g in enumerate(downstream_groups):
            if i == 0:
              # If it is the first uncommon downstream group, then the input comes from
              # the first uncommon upstream group.
              inputs[g].add((full_name, upstream_groups[0]))
            else:
              # If not the first downstream group, then the input is passed down from
              # its ancestor groups so the upstream group is None.
              inputs[g].add((full_name, None))
          for i, g in enumerate(upstream_groups):
            if i == len(upstream_groups) - 1:
              # If last upstream group, it is an operator and output comes from container.
              outputs[g].add((full_name, None))
            else:
              # If not last upstream group, output value comes from one of its child.
              outputs[g].add((full_name, upstream_groups[i+1]))
        else:
          if not op.is_exit_handler:
            for g in op_groups[op.name]:
              inputs[g].add((full_name, None))
    return inputs, outputs
    
  def _get_condition_params_for_ops(self, root_group):
    """Get parameters referenced in conditions of ops."""

    conditions = defaultdict(set)

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
        _get_condition_params_for_ops_helper(g, new_current_conditions_params)

    _get_condition_params_for_ops_helper(root_group, [])
    return conditions
      
  def _get_dependencies(self, pipeline, root_group, op_groups):
    """Get dependent groups and ops for all ops and groups.

    Returns:
      A dict. Key is group/op name, value is a list of dependent groups/ops.
      The dependencies are calculated in the following way: if op2 depends on op1,
      and their ancestors are [root, G1, G2, op1] and [root, G1, G3, G4, op2],
      then G3 is dependent on G2. Basically dependency only exists in the first uncommon
      ancesters in their ancesters chain. Only sibling groups/ops can have dependencies.
    """
    condition_params = self._get_condition_params_for_ops(root_group)
    dependencies = defaultdict(set)
    for op in pipeline.ops.values():
      unstream_op_names = set()
      for param in op.inputs + list(condition_params[op.name]):
        if param.op_name:
          unstream_op_names.add(param.op_name)
      unstream_op_names |= set(op.dependent_op_names)

      for op_name in unstream_op_names:
        upstream_op = pipeline.ops[op_name]
        upstream_groups, downstream_groups = self._get_uncommon_ancestors(
            op_groups, upstream_op, op)
        dependencies[downstream_groups[0]].add(upstream_groups[0])
    return dependencies

  def _create_condition(self, condition):
    left = ('{{inputs.parameters.%s}}' % self._param_full_name(condition.operand1)
            if isinstance(condition.operand1, dsl.PipelineParam)
            else str(condition.operand1))
    right = ('{{inputs.parameters.%s}}' % self._param_full_name(condition.operand2)
             if isinstance(condition.operand2, dsl.PipelineParam)
             else str(condition.operand2))
    return ('%s == %s' % (left, right))

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

    if group.type == 'condition':
      # This is a workaround for the fact that argo does not support conditions in DAG mode.
      # Basically, we insert an extra group that contains only the original group. The extra group
      # operates in "step" mode where condition is supported.
      only_child = group.groups[0]
      step = {
          'name': only_child.name,
          'template': only_child.name,
      }
      if inputs.get(only_child.name, None):
        arguments = []
        for param_name, dependent_name in inputs[only_child.name]:
          arguments.append({
              'name': param_name,
              'value': '{{inputs.parameters.%s}}' % param_name
          })
        arguments.sort(key=lambda x: x['name'])
        step['arguments'] = {'parameters': arguments}
        step['when'] = self._create_condition(group.condition)
      template['steps'] = [[step]]
    else:
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

  def _create_new_groups(self, root_group):
    """Create a copy of the input group, and insert extra groups for conditions."""

    new_group = copy.deepcopy(root_group)
    
    def _insert_group_for_condition_helper(group):
      for i, g in enumerate(group.groups):
        if g.type == 'condition':
          child_condition_group = dsl.OpsGroup('condition-child', g.name + '-child')
          child_condition_group.ops = g.ops
          child_condition_group.groups = g.groups
          g.groups = [child_condition_group]
          g.ops = list()
          _insert_group_for_condition_helper(child_condition_group)
        else:
          _insert_group_for_condition_helper(g)

    _insert_group_for_condition_helper(new_group)
    return new_group
    
  def _create_templates(self, pipeline):
    """Create all groups and ops templates in the pipeline."""

    # This is needed only because Argo does not support condition in DAG mode.
    # Revisit when https://github.com/argoproj/argo/issues/921 is fixed.
    new_root_group = self._create_new_groups(pipeline.groups[0])

    op_groups = self._get_groups_for_ops(new_root_group)
    inputs, outputs = self._get_inputs_outputs(pipeline, new_root_group, op_groups)
    dependencies = self._get_dependencies(pipeline, new_root_group, op_groups)
    groups = self._get_groups(new_root_group)

    templates = []
    for g in groups:
      templates.append(self._group_to_template(g, inputs, outputs, dependencies))

    for op in pipeline.ops.values():
      templates.append(self._op_to_template(op))
    return templates

  def _create_pipeline_workflow(self, args, pipeline):
    """Create workflow for the pipeline."""

    input_params = []
    for arg in args:
      param = {'name': arg.name}
      if arg.value is not None:
        param['value'] = str(arg.value)
      input_params.append(param)

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
        'arguments': {'parameters': input_params},
        'serviceAccountName': 'pipeline-runner'
      }
    }
    if exit_handler:
      workflow['spec']['onExit'] = exit_handler.name
    return workflow

  def _validate_args(self, argspec):
    if argspec.defaults:
      for value in argspec.defaults:
        if not issubclass(type(value), dsl.PipelineParam):
          raise ValueError(
              'Default values of argument has to be type dsl.PipelineParam or its child.')

  def _validate_exit_handler(self, pipeline):
    """Makes sure there is only one global exit handler.

    Note this is a temporary workaround until argo supports local exit handler.
    """

    def _validate_exit_handler_helper(group, exiting_op_names, handler_exists):
      if group.type == 'exit_handler':
        if handler_exists or len(exiting_op_names) > 1:
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

    registered_pipeline_functions = dsl.Pipeline.get_pipeline_functions()
    if pipeline_func not in registered_pipeline_functions:
      raise ValueError('Please use a function with @dsl.pipeline decorator.')

    pipeline_name, _ = dsl.Pipeline.get_pipeline_functions()[pipeline_func]
    pipeline_name = self._sanitize_name(pipeline_name)

    # Create the arg list with no default values and call pipeline function.
    args_list = [dsl.PipelineParam(self._sanitize_name(arg_name))
                 for arg_name in argspec.args]
    with dsl.Pipeline(pipeline_name) as p:
      pipeline_func(*args_list)

    # Remove when argo supports local exit handler.    
    self._validate_exit_handler(p)

    # Fill in the default values.
    args_list_with_defaults = [dsl.PipelineParam(self._sanitize_name(arg_name))
                               for arg_name in argspec.args]
    if argspec.defaults:
      for arg, default in zip(reversed(args_list_with_defaults), reversed(argspec.defaults)):
        arg.value = default.value

    workflow = self._create_pipeline_workflow(args_list_with_defaults, p)
    return workflow

  def compile(self, pipeline_func, package_path):
    """Compile the given pipeline function into workflow yaml.

    Args:
      pipeline_func: pipeline functions with @dsl.pipeline decorator.
      package_path: the output workflow tar.gz file path. for example, "~/a.tar.gz"
    """
    workflow = self._compile(pipeline_func)
    yaml.Dumper.ignore_aliases = lambda *args : True
    yaml_text = yaml.dump(workflow, default_flow_style=False)

    from contextlib import closing
    from io import BytesIO
    with tarfile.open(package_path, "w:gz") as tar:
      with closing(BytesIO(yaml_text.encode())) as yaml_file:
        tarinfo = tarfile.TarInfo('pipeline.yaml')
        tarinfo.size = len(yaml_file.getvalue())
        tar.addfile(tarinfo, fileobj=yaml_file)