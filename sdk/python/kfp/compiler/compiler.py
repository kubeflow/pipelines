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
import inspect
import re
import tarfile
import yaml

from .. import dsl
from ._k8s_helper import K8sHelper

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
    """From _make_kubernetes_name
    _sanitize_name cleans and converts the names in the workflow.
    """
    return re.sub('-+', '-', re.sub('[^-0-9a-z]+', '-', name.lower())).lstrip('-').rstrip('-')

  def _pipelineparam_full_name(self, param):
    """_pipelineparam_full_name

    Args:
      param(PipelineParam): pipeline parameter
      """
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

  def _process_args(self, raw_args, argument_inputs):
    if not raw_args:
      return []

    processed_args = list(map(str, raw_args))
    for i, _ in enumerate(processed_args):
      for param in argument_inputs:
        full_name = self._pipelineparam_full_name(param)
        processed_args[i] = re.sub(
            str(param), '{{inputs.parameters.%s}}' % full_name, str(processed_args[i]))

    return processed_args
  
  def _op_to_template(self, op):
    """Generate template given an operator inherited from dsl.ContainerOp."""
    
    input_parameters = []
    for param in op.inputs:
      one_parameter = {'name': self._pipelineparam_full_name(param)}
      if param.value:
        one_parameter['value'] = str(param.value)
      input_parameters.append(one_parameter)
    # Sort to make the results deterministic.
    input_parameters.sort(key=lambda x: x['name'])

    output_parameters = []
    for param in op.outputs.values():
      output_parameters.append({
          'name': self._pipelineparam_full_name(param),
          'valueFrom': {'path': op.file_outputs[param.name]}
      })
    output_parameters.sort(key=lambda x: x['name'])

    template = {
      'name': op.name,
      'container': {
        'image': op.image,
      }
    }
    processed_arguments = self._process_args(op.arguments, op.argument_inputs)
    processed_command = self._process_args(op.command, op.argument_inputs)
    if processed_arguments:
      template['container']['args'] = processed_arguments
    if processed_command:
      template['container']['command'] = processed_command
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


    # Set resources.
    if op.resource_limits or op.resource_requests:
      template['container']['resources'] = {}
    if op.resource_limits:
      template['container']['resources']['limits'] = op.resource_limits
    if op.resource_requests:
      template['container']['resources']['requests'] = op.resource_requests

    # Set nodeSelector.
    if op.node_selector:
      template['nodeSelector'] = op.node_selector

    if op.env_variables:
      template['container']['env'] = list(map(K8sHelper.convert_k8s_obj_to_json, op.env_variables))
    if op.volume_mounts:
      template['container']['volumeMounts'] = list(map(K8sHelper.convert_k8s_obj_to_json, op.volume_mounts))

    if op.pod_annotations or op.pod_labels:
      template['metadata'] = {}
      if op.pod_annotations:
        template['metadata']['annotations'] = op.pod_annotations
      if op.pod_labels:
        template['metadata']['labels'] = op.pod_labels
    
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

        full_name = self._pipelineparam_full_name(param)
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
        return '{{tasks.%s.outputs.parameters.%s}}' % (task_name, parameter_name)
      else:
        return '{{inputs.parameters.%s}}' % parameter_name
    else:
      return str(value_or_reference)

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

      if isinstance(sub_group, dsl.OpsGroup) and sub_group.type == 'condition':
        subgroup_inputs = inputs.get(sub_group.name, [])
        condition = sub_group.condition
        operand1_value = self._resolve_value_or_reference(condition.operand1, subgroup_inputs)
        operand2_value = self._resolve_value_or_reference(condition.operand2, subgroup_inputs)
        task['when'] = '{} {} {}'.format(operand1_value, condition.operator, operand2_value)

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

    new_root_group = pipeline.groups[0]

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

  def _create_volumes(self, pipeline):
    """Create volumes required for the templates"""
    volumes = []
    volume_name_set = set()
    for op in pipeline.ops.values():
      if op.volumes:
        for v in op.volumes:
          # Remove volume duplicates which have the same name
          #TODO: check for duplicity based on the serialized volumes instead of just name.
          if v.name not in volume_name_set:
            volume_name_set.add(v.name)
            volumes.append(K8sHelper.convert_k8s_obj_to_json(v))
    volumes.sort(key=lambda x: x['name'])
    return volumes

  def _create_pipeline_workflow(self, args, pipeline):
    """Create workflow for the pipeline."""

    # Input Parameters
    input_params = []
    for arg in args:
      param = {'name': arg.name}
      if arg.value is not None:
        param['value'] = str(arg.value)
      input_params.append(param)

    # Templates
    templates = self._create_templates(pipeline)
    templates.sort(key=lambda x: x['name'])

    # Exit Handler
    exit_handler = None
    if pipeline.groups[0].groups:
      first_group = pipeline.groups[0].groups[0]
      if first_group.type == 'exit_handler':
        exit_handler = first_group.exit_op

    # Volumes
    volumes = self._create_volumes(pipeline)

    # The whole pipeline workflow
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
    if volumes:
      workflow['spec']['volumes'] = volumes
    return workflow

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
        arg.value = default.value if isinstance(default, dsl.PipelineParam) else default

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
