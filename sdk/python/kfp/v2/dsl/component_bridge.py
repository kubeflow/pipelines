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
"""Function for creating ContainerOp instances from component spec."""

import copy
from typing import Any, Mapping

from kfp import dsl
from kfp.components._naming import _sanitize_python_function_name
from kfp.components._naming import generate_unique_name_conversion_table
from kfp.dsl import types
from kfp.v2.components import structures
from kfp.v2.components.components import _default_component_name
from kfp.v2.components.components import _resolve_command_line_and_paths
from kfp.v2.dsl import container_op
from kfp.v2.dsl import importer_node
from kfp.v2.dsl import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2


# TODO: cleanup unused code.
def create_container_op_from_component_and_arguments(
    component_spec: structures.ComponentSpec,
    arguments: Mapping[str, Any],
    component_ref: structures.ComponentReference = None,
) -> container_op.ContainerOp:
  """Instantiates ContainerOp object.

  Args:
    component_spec: The component spec object.
    arguments: The dictionary of component arguments.
    component_ref: The component reference. Optional.

  Returns:
    A ContainerOp instance.
  """

  pipeline_task_spec = pipeline_spec_pb2.PipelineTaskSpec()
  pipeline_task_spec.task_info.name = component_spec.name
  # might need to append suffix to exuector_label to ensure its uniqueness?
  pipeline_task_spec.executor_label = component_spec.name

  # Keep track of auto-injected importer spec.
  importer_spec = {}

  # Check types of the reference arguments and serialize PipelineParams
  arguments = arguments.copy()
  for input_name, argument_value in arguments.items():
    if isinstance(argument_value, dsl.PipelineParam):
      input_type = component_spec._inputs_dict[input_name].type
      reference_type = argument_value.param_type
      types.verify_type_compatibility(
          reference_type, input_type,
          'Incompatible argument passed to the input "{}" of component "{}": '
          .format(input_name, component_spec.name))

      arguments[input_name] = str(argument_value)

      if type_utils.is_parameter_type(input_type):
        if argument_value.op_name:
          pipeline_task_spec.inputs.parameters[
              input_name].task_output_parameter.producer_task = (
                  argument_value.op_name)
          pipeline_task_spec.inputs.parameters[
              input_name].task_output_parameter.output_parameter_key = (
                  argument_value.name)
        else:
          pipeline_task_spec.inputs.parameters[
              input_name].runtime_value.runtime_parameter = argument_value.name
      else:
        if argument_value.op_name:
          pipeline_task_spec.inputs.artifacts[input_name].producer_task = (
              argument_value.op_name)
          pipeline_task_spec.inputs.artifacts[
              input_name].output_artifact_key = (
                  argument_value.name)
        else:
          # argument_value.op_name could be none, in which case an importer node
          # will be inserted later.
          pipeline_task_spec.inputs.artifacts[input_name].producer_task = ''
          type_schema = type_utils.get_input_artifact_type_schema(
              input_name, component_spec.inputs)
          importer_spec[input_name] = importer_node.build_importer_spec(
              input_type_schema=type_schema,
              pipeline_param_name=argument_value.name)
    elif isinstance(argument_value, str):
      input_type = component_spec._inputs_dict[input_name].type
      if type_utils.is_parameter_type(input_type):
        pipeline_task_spec.inputs.parameters[
            input_name].runtime_value.constant_value.string_value = (
                argument_value)
      else:
        # An importer node with constant value artifact_uri will be inserted.
        pipeline_task_spec.inputs.artifacts[input_name].producer_task = ''
        type_schema = type_utils.get_input_artifact_type_schema(
            input_name, component_spec.inputs)
        importer_spec[input_name] = importer_node.build_importer_spec(
            input_type_schema=type_schema, constant_value=argument_value)
    elif isinstance(argument_value, int):
      pipeline_task_spec.inputs.parameters[
          input_name].runtime_value.constant_value.int_value = argument_value
    elif isinstance(argument_value, float):
      pipeline_task_spec.inputs.parameters[
          input_name].runtime_value.constant_value.double_value = argument_value
    elif isinstance(argument_value, dsl.ContainerOp):
      raise TypeError(
          'ContainerOp object {} was passed to component as an input argument. '
          'Pass a single output instead.'.format(input_name))
    else:
      raise NotImplementedError(
          'Input argument supports only the following types: PipelineParam'
          ', str, int, float. Got: "{}".'.format(argument_value))

  for output in component_spec.outputs or []:
    if type_utils.is_parameter_type(output.type):
      pipeline_task_spec.outputs.parameters[
          output.name].type = type_utils.get_parameter_type(output.type)
    else:
      pipeline_task_spec.outputs.artifacts[
          output.name].artifact_type.instance_schema = (
              type_utils.get_artifact_type_schema(output.type))

  inputs_dict = {
      input_spec.name: input_spec for input_spec in component_spec.inputs or []
  }
  outputs_dict = {
      output_spec.name: output_spec
      for output_spec in component_spec.outputs or []
  }

  def _input_artifact_uri_placeholder(input_key: str) -> str:
    if type_utils.is_parameter_type(inputs_dict[input_key].type):
      raise TypeError(
          'Input "{}" with type "{}" cannot be paired with InputUriPlaceholder.'
          .format(input_key, inputs_dict[input_key].type))
    else:
      return "{{{{$.inputs.artifacts['{}'].uri}}}}".format(input_key)

  def _input_artifact_path_placeholder(input_key: str) -> str:
    if type_utils.is_parameter_type(inputs_dict[input_key].type):
      raise TypeError(
          'Input "{}" with type "{}" cannot be paired with InputPathPlaceholder.'
          .format(input_key, inputs_dict[input_key].type))
    else:
      return "{{{{$.inputs.artifacts['{}'].path}}}}".format(input_key)

  def _input_parameter_placeholder(input_key: str) -> str:
    if type_utils.is_parameter_type(inputs_dict[input_key].type):
      return "{{{{$.inputs.parameters['{}']}}}}".format(input_key)
    else:
      raise TypeError(
          'Input "{}" with type "{}" cannot be paired with InputValuePlaceholder.'
          .format(input_key, inputs_dict[input_key].type))

  def _output_artifact_uri_placeholder(output_key: str) -> str:
    if type_utils.is_parameter_type(outputs_dict[output_key].type):
      raise TypeError(
          'Output "{}" with type "{}" cannot be paired with OutputUriPlaceholder.'
          .format(output_key, outputs_dict[output_key].type))
    else:
      return "{{{{$.outputs.artifacts['{}'].uri}}}}".format(output_key)

  def _output_artifact_path_placeholder(output_key: str) -> str:
    return "{{{{$.outputs.artifacts['{}'].path}}}}".format(output_key)

  def _output_parameter_path_placeholder(output_key: str) -> str:
    return "{{{{$.outputs.parameters['{}'].output_file}}}}".format(output_key)

  def _resolve_output_path_placeholder(output_key: str) -> str:
    if type_utils.is_parameter_type(outputs_dict[output_key].type):
      return _output_parameter_path_placeholder(output_key)
    else:
      return _output_artifact_path_placeholder(output_key)

  resolved_cmd = _resolve_command_line_and_paths(
      component_spec=component_spec,
      arguments=arguments,
      input_value_generator=_input_parameter_placeholder,
      input_uri_generator=_input_artifact_uri_placeholder,
      output_uri_generator=_output_artifact_uri_placeholder,
      input_path_generator=_input_artifact_path_placeholder,
      output_path_generator=_resolve_output_path_placeholder,
  )

  container_spec = component_spec.implementation.container

  pipeline_container_spec = (
      pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec())
  pipeline_container_spec.image = container_spec.image
  pipeline_container_spec.command.extend(resolved_cmd.command)
  pipeline_container_spec.args.extend(resolved_cmd.args)

  output_uris_and_paths = resolved_cmd.output_uris.copy()
  output_uris_and_paths.update(resolved_cmd.output_paths)
  input_uris_and_paths = resolved_cmd.input_uris.copy()
  input_uris_and_paths.update(resolved_cmd.input_paths)

  old_warn_value = dsl.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING
  dsl.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING = True
  task = container_op.ContainerOp(
      name=component_spec.name or _default_component_name,
      image=container_spec.image,
      command=resolved_cmd.command,
      arguments=resolved_cmd.args,
      file_outputs=output_uris_and_paths,
      artifact_argument_paths=[
          dsl.InputArgumentPath(
              argument=arguments[input_name],
              input=input_name,
              path=path,
          ) for input_name, path in input_uris_and_paths.items()
      ],
  )

  task.task_spec = pipeline_task_spec
  task.importer_spec = importer_spec
  task.container_spec = pipeline_container_spec
  dsl.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING = old_warn_value

  component_meta = copy.copy(component_spec)
  task._set_metadata(component_meta)
  component_ref_without_spec = copy.copy(component_ref)
  component_ref_without_spec.spec = None
  task._component_ref = component_ref_without_spec

  # Previously, ContainerOp had strict requirements for the output names, so we
  # had to convert all the names before passing them to the ContainerOp
  # constructor. Outputs with non-pythonic names could not be accessed using
  # their original names. Now ContainerOp supports any output names, so we're
  # now using the original output names. However to support legacy pipelines,
  # we're also adding output references with pythonic names.
  # TODO: Add warning when people use the legacy output names.
  output_names = [
      output_spec.name for output_spec in component_spec.outputs or []
  ]  # Stabilizing the ordering
  output_name_to_python = generate_unique_name_conversion_table(
      output_names, _sanitize_python_function_name)
  for output_name in output_names:
    pythonic_output_name = output_name_to_python[output_name]
    # Note: Some component outputs are currently missing from task.outputs
    # (e.g. MLPipeline UI Metadata)
    if pythonic_output_name not in task.outputs and output_name in task.outputs:
      task.outputs[pythonic_output_name] = task.outputs[output_name]

  if component_spec.metadata:
    annotations = component_spec.metadata.annotations or {}
    for key, value in annotations.items():
      task.add_pod_annotation(key, value)
    for key, value in (component_spec.metadata.labels or {}).items():
      task.add_pod_label(key, value)
      # Disabling the caching for the volatile components by default
    if annotations.get('volatile_component', 'false') == 'true':
      task.execution_options.caching_strategy.max_cache_staleness = 'P0D'

  return task
