# Copyright 2018 The Kubeflow Authors
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

import collections
import copy
import inspect
import pathlib
from typing import Any, Dict, Mapping, Optional
import warnings

import kfp
from kfp.components import _structures
from kfp.components import _components
from kfp.components import _naming
from kfp import dsl
from kfp.dsl import _container_op
from kfp.dsl import _for_loop
from kfp.dsl import _pipeline_param
from kfp.dsl import component_spec as dsl_component_spec
from kfp.dsl import dsl_utils
from kfp.dsl import legacy_data_passing_adaptor
from kfp.dsl import types
from kfp.dsl import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2

# Placeholder to represent the output directory hosting all the generated URIs.
# Its actual value will be specified during pipeline compilation.
# The format of OUTPUT_DIR_PLACEHOLDER is serialized dsl.PipelineParam, to
# ensure being extracted as a pipeline parameter during compilation.
# Note that we cannot direclty import dsl module here due to circular
# dependencies.
OUTPUT_DIR_PLACEHOLDER = '{{pipelineparam:op=;name=pipeline-output-directory}}'
# Placeholder to represent to UID of the current pipeline at runtime.
# Will be replaced by engine-specific placeholder during compilation.
RUN_ID_PLACEHOLDER = '{{kfp.run_uid}}'
# Format of the Argo parameter used to pass the producer's Pod ID to
# the consumer.
PRODUCER_POD_NAME_PARAMETER = '{}-producer-pod-id-'
# Format of the input output port name placeholder.
INPUT_OUTPUT_NAME_PATTERN = '{{{{kfp.input-output-name.{}}}}}'
# Fixed name for per-task output metadata json file.
OUTPUT_METADATA_JSON = '/tmp/outputs/executor_output.json'
# Executor input placeholder.
_EXECUTOR_INPUT_PLACEHOLDER = '{{$}}'


def _generate_output_uri(port_name: str) -> str:
  """Generates a unique URI for an output.

  Args:
    port_name: The name of the output associated with this URI.

  Returns:
    The URI assigned to this output, which is unique within the pipeline.
  """
  return str(
      pathlib.PurePosixPath(OUTPUT_DIR_PLACEHOLDER, RUN_ID_PLACEHOLDER,
                            '{{pod.name}}', port_name))


def _generate_input_uri(port_name: str) -> str:
  """Generates the URI for an input.

  Args:
    port_name: The name of the input associated with this URI.

  Returns:
    The URI assigned to this input, will be consistent with the URI where
    the actual content is written after compilation.
  """
  return str(
      pathlib.PurePosixPath(
          OUTPUT_DIR_PLACEHOLDER, RUN_ID_PLACEHOLDER,
          '{{{{inputs.parameters.{input}}}}}'.format(
              input=PRODUCER_POD_NAME_PARAMETER.format(port_name)), port_name))


def _generate_output_metadata_path() -> str:
  """Generates the URI to write the output metadata JSON file."""

  return OUTPUT_METADATA_JSON


def _generate_input_metadata_path(port_name: str) -> str:
  """Generates the placeholder for input artifact metadata file."""

  # Return a placeholder for path to input artifact metadata, which will be
  # rewritten during pipeline compilation.
  return str(
      pathlib.PurePosixPath(
          OUTPUT_DIR_PLACEHOLDER, RUN_ID_PLACEHOLDER,
          '{{{{inputs.parameters.{input}}}}}'.format(
              input=PRODUCER_POD_NAME_PARAMETER.format(port_name)),
          OUTPUT_METADATA_JSON))


def _generate_input_output_name(port_name: str) -> str:
  """Generates the placeholder for input artifact's output name."""

  # Return a placeholder for the output port name of the input artifact, which
  # will be rewritten during pipeline compilation.
  return INPUT_OUTPUT_NAME_PATTERN.format(port_name)


def _generate_executor_input() -> str:
  """Generates the placeholder for serialized executor input."""
  return _EXECUTOR_INPUT_PLACEHOLDER


class ExtraPlaceholderResolver:

  def __init__(self):
    self.input_uris = {}
    self.input_metadata_paths = {}
    self.output_uris = {}

  def resolve_placeholder(
      self,
      arg,
      component_spec: _structures.ComponentSpec,
      arguments: dict,
  ) -> str:
    inputs_dict = {
        input_spec.name: input_spec
        for input_spec in component_spec.inputs or []
    }

    if isinstance(arg, _structures.InputUriPlaceholder):
      input_name = arg.input_name
      if input_name in arguments:
        input_uri = _generate_input_uri(input_name)
        self.input_uris[input_name] = input_uri
        return input_uri
      else:
        input_spec = inputs_dict[input_name]
        if input_spec.optional:
          return None
        else:
          raise ValueError('No value provided for input {}'.format(input_name))

    elif isinstance(arg, _structures.OutputUriPlaceholder):
      output_name = arg.output_name
      output_uri = _generate_output_uri(output_name)
      self.output_uris[output_name] = output_uri
      return output_uri

    elif isinstance(arg, _structures.InputMetadataPlaceholder):
      input_name = arg.input_name
      if input_name in arguments:
        input_metadata_path = _generate_input_metadata_path(input_name)
        self.input_metadata_paths[input_name] = input_metadata_path
        return input_metadata_path
      else:
        input_spec = inputs_dict[input_name]
        if input_spec.optional:
          return None
        else:
          raise ValueError('No value provided for input {}'.format(input_name))

    elif isinstance(arg, _structures.InputOutputPortNamePlaceholder):
      input_name = arg.input_name
      if input_name in arguments:
        return _generate_input_output_name(input_name)
      else:
        input_spec = inputs_dict[input_name]
        if input_spec.optional:
          return None
        else:
          raise ValueError('No value provided for input {}'.format(input_name))

    elif isinstance(arg, _structures.OutputMetadataPlaceholder):
      # TODO: Consider making the output metadata per-artifact.
      return _generate_output_metadata_path()
    elif isinstance(arg, _structures.ExecutorInputPlaceholder):
      return _generate_executor_input()

    return None


def _create_container_op_from_component_and_arguments(
    component_spec: _structures.ComponentSpec,
    arguments: Mapping[str, Any],
    component_ref: Optional[_structures.ComponentReference] = None,
) -> _container_op.ContainerOp:
  """Instantiates ContainerOp object.

  Args:
    component_spec: The component spec object.
    arguments: The dictionary of component arguments.
    component_ref: (only for v1) The component references.

  Returns:
    A ContainerOp instance.
  """

  # Add component inputs with default value to the arguments dict if they are not
  # in the arguments dict already.
  arguments = arguments.copy()
  for input_spec in component_spec.inputs or []:
    if input_spec.name not in arguments and input_spec.default is not None:
      default_value = input_spec.default
      if input_spec.type == 'Integer':
        default_value = int(default_value)
      elif input_spec.type == 'Float':
        default_value = float(default_value)
      arguments[input_spec.name] = default_value

  # Check types of the reference arguments and serialize PipelineParams
  original_arguments = arguments
  arguments = arguments.copy()
  for input_name, argument_value in arguments.items():
    if isinstance(argument_value, _pipeline_param.PipelineParam):
      input_type = component_spec._inputs_dict[input_name].type
      argument_type = argument_value.param_type
      types.verify_type_compatibility(
          argument_type, input_type,
          'Incompatible argument passed to the input "{}" of component "{}": '
          .format(input_name, component_spec.name))

      arguments[input_name] = str(argument_value)
    if isinstance(argument_value, _container_op.ContainerOp):
      raise TypeError(
          'ContainerOp object was passed to component as an input argument. '
          'Pass a single output instead.')
  placeholder_resolver = ExtraPlaceholderResolver()
  resolved_cmd = _components._resolve_command_line_and_paths(
      component_spec=component_spec,
      arguments=arguments,
      placeholder_resolver=placeholder_resolver.resolve_placeholder,
  )

  container_spec = component_spec.implementation.container

  old_warn_value = _container_op.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING
  _container_op.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING = True

  output_paths_and_uris = collections.OrderedDict(resolved_cmd.output_paths or
                                                  {})
  output_paths_and_uris.update(placeholder_resolver.output_uris)
  input_paths_and_uris = collections.OrderedDict(resolved_cmd.input_paths or {})
  input_paths_and_uris.update(placeholder_resolver.input_uris)

  artifact_argument_paths = [
      dsl.InputArgumentPath(
          argument=arguments[input_name],
          input=input_name,
          path=path_or_uri,
      ) for input_name, path_or_uri in input_paths_and_uris.items()
  ]

  task = _container_op.ContainerOp(
      name=component_spec.name or _components._default_component_name,
      image=container_spec.image,
      command=resolved_cmd.command,
      arguments=resolved_cmd.args,
      file_outputs=output_paths_and_uris,
      artifact_argument_paths=artifact_argument_paths,
  )
  _container_op.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING = old_warn_value

  component_meta = copy.copy(component_spec)
  task._set_metadata(component_meta)
  component_ref_without_spec = copy.copy(component_ref)
  component_ref_without_spec.spec = None
  task._component_ref = component_ref_without_spec

  task._parameter_arguments = resolved_cmd.inputs_consumed_by_value

  # Previously, ContainerOp had strict requirements for the output names, so we
  # had to convert all the names before passing them to the ContainerOp
  # constructor.
  # Outputs with non-pythonic names could not be accessed using their original
  # names. Now ContainerOp supports any output names, so we're now using the
  # original output names. However to support legacy pipelines, we're also
  # adding output references with pythonic names.
  # TODO: Add warning when people use the legacy output names.
  output_names = [
      output_spec.name for output_spec in component_spec.outputs or []
  ]  # Stabilizing the ordering
  output_name_to_python = _naming.generate_unique_name_conversion_table(
      output_names, _naming._sanitize_python_function_name)
  for output_name in output_names:
    pythonic_output_name = output_name_to_python[output_name]
    # Note: Some component outputs are currently missing from task.outputs
    # (e.g. MLPipeline UI Metadata)
    if pythonic_output_name not in task.outputs and output_name in task.outputs:
      task.outputs[pythonic_output_name] = task.outputs[output_name]

  if container_spec.env:
    from kubernetes import client as k8s_client
    for name, value in container_spec.env.items():
      task.container.add_env_variable(
          k8s_client.V1EnvVar(name=name, value=value))

  if component_spec.metadata:
    annotations = component_spec.metadata.annotations or {}
    for key, value in annotations.items():
      task.add_pod_annotation(key, value)
    for key, value in (component_spec.metadata.labels or {}).items():
      task.add_pod_label(key, value)
    # Disabling the caching for the volatile components by default
    if annotations.get('volatile_component', 'false') == 'true':
      task.execution_options.caching_strategy.max_cache_staleness = 'P0D'

  _attach_v2_specs(task, component_spec, original_arguments)

  return task


def _attach_v2_specs(
    task: _container_op.ContainerOp,
    component_spec: _structures.ComponentSpec,
    arguments: Mapping[str, Any],
) -> None:
  """Attaches v2 specs to a ContainerOp object.

  Attach v2_specs to the ContainerOp object regardless whether the pipeline is
  being compiled to v1 (Argo yaml) or v2 (IR json).
  However, there're different behaviors for the two cases. Namely, resolved
  commands and arguments, error handling, etc.
  Regarding the difference in error handling, v2 has a stricter requirement on
  input type annotation. For instance, an input without any type annotation is
  viewed as an artifact, and if it's paired with InputValuePlaceholder, an
  error will be thrown at compile time. However, we cannot raise such an error
  in v1, as it wouldn't break existing pipelines.

  Args:
    task: The ContainerOp object to attach IR specs.
    component_spec: The component spec object.
    arguments: The dictionary of component arguments.
  """

  if not kfp.COMPILING_FOR_V2:
    return

  task.legacy_data_passing_adaptors: Dict[
      str, legacy_data_pasing_adaptor.BaseAdaptor] = {}

  def _resolve_commands_and_args_v2(
      component_spec: _structures.ComponentSpec,
      arguments: Mapping[str, Any],
  ) -> _components._ResolvedCommandLineAndPaths:
    """Resolves the command line argument placeholders for v2 (IR).

    Args:
      component_spec: The component spec object.
      arguments: The dictionary of component arguments.

    Returns:
      A named tuple: _components._ResolvedCommandLineAndPaths.
    """
    inputs_dict = {
        input_spec.name: input_spec
        for input_spec in component_spec.inputs or []
    }
    outputs_dict = {
        output_spec.name: output_spec
        for output_spec in component_spec.outputs or []
    }

    def _resolve_output_path_placeholder(output_key: str) -> str:
      if type_utils.is_parameter_type(outputs_dict[output_key].type):
        return dsl_utils.output_parameter_path_placeholder(output_key)
      else:
        return dsl_utils.output_artifact_path_placeholder(output_key)

    placeholder_resolver = ExtraPlaceholderResolver()

    def _resolve_input_path_placeholder(input_name: str) -> str:
      input_spec = inputs_dict[input_name]

      if input_name in arguments:
        need_adaptor = False

        # InputPathPlaceholder can only be used for artifacts.
        # If the input type isn't an artifact type, convert it to an artifact.
        if type_utils.is_parameter_type(input_spec.type):
          warnings.warn(
              f'Input "{input_name}" with type "{input_spec.type}" should not'
              ' be paired with InputPathPlaceholder.'
              ' Removing the type to convert the input to a generic artifact.'
              ' If the intention is to pass a parameter rather than an artifact,'
              ' use InputValuePlaceholder instead.')
          input_spec.type = None

        # The input must be an artifact.
        if not isinstance(arguments[input_name], dsl.PipelineParam) or \
             arguments[input_name].op_name is None or \
             type_utils.is_parameter_type(arguments[input_name].param_type):

          warnings.warn(
              f'Input "{input_name}" with type "{input_spec.type}" is treated as'
              ' an artifact. But it gets value from a parameter.'
              ' A legacy adaptor component will be injected.')

          if not isinstance(arguments[input_name], dsl.PipelineParam):
            from_type = type(arguments[input_name])
          elif type_utils.is_parameter_type(arguments[input_name].param_type):
            from_type = arguments[input_name].param_type
          else:
            from_type = str

          task.legacy_data_passing_adaptors[input_name] = (
              legacy_data_passing_adaptor.ParameterToArtifactAdaptor(
                  input_type=from_type,
                  output_type=input_spec.type,
              ))

        input_path = dsl_utils.input_artifact_path_placeholder(input_name)
        return input_path
      else:
        if input_spec.optional:
          return None
        else:
          raise ValueError('No value provided for input {}'.format(input_name))

      return dsl_utils.input_artifact_path_placeholder(input_name)

    def _resolve_ir_placeholders_v2(
        arg,
        component_spec: _structures.ComponentSpec,
        arguments: dict,
    ) -> str:

      if isinstance(arg, _structures.InputValuePlaceholder):
        input_name = arg.input_name
        input_spec = inputs_dict[input_name]

        if input_name in arguments:

          # InputValuePlaceholder can be used for both parameters and artifacts,
          # although for artifacts, it may result runtime failures.
          # The original input type determines whether the input is an artifact
          # or a parameter.
          if type_utils.is_parameter_type(input_spec.type):
            # The input should be a parameter.
            if isinstance(arguments[input_name], dsl.PipelineParam) and \
               not type_utils.is_parameter_type(arguments[input_name].param_type) and \
               arguments[input_name].op_name is not None and \
               not isinstance(
                   arguments[input_name],
                   (_for_loop.LoopArguments, _for_loop.LoopArgumentVariable)):
              warnings.warn(
                  f'Input "{input_name}" with type "{input_spec.type}" is '
                  'treated as a parameter. But it gets value from an artifact.'
                  ' A legacy adaptor component will be injected.')

              task.legacy_data_passing_adaptors[input_name] = (
                  legacy_data_passing_adaptor.ArtifactToParameterAdaptor(
                      input_type=arguments[input_name].param_type,
                      output_type=input_spec.type,
                  ))
          else:
            # The input should be an artifact.
            if not isinstance(arguments[input_name], dsl.PipelineParam) or \
             arguments[input_name].op_name is None or \
             type_utils.is_parameter_type(arguments[input_name].param_type):
              warnings.warn(
                  f'Input "{input_name}" with type "{input_spec.type}" is '
                  'treated as an artifact. But it gets value from a parameter.'
                  ' A legacy adaptor component will be injected.')

              if not isinstance(arguments[input_name], dsl.PipelineParam):
                from_type = type(arguments[input_name])
              elif type_utils.is_parameter_type(
                  arguments[input_name].param_type):
                from_type = arguments[input_name].param_type
              else:
                from_type = str

              task.legacy_data_passing_adaptors[input_name] = (
                  legacy_data_passing_adaptor.ParameterToArtifactAdaptor(
                      input_type=from_type,
                      output_type=input_spec.type,
                  ))

          if type_utils.is_parameter_type(input_spec.type):
            return dsl_utils.input_parameter_placeholder(input_name)
          else:
            return dsl_utils.input_artifact_value_placeholder(input_name)
        else:
          if input_spec.optional:
            return None
          else:
            raise ValueError(
                'No value provided for input {}'.format(input_name))

      elif isinstance(arg, _structures.InputUriPlaceholder):
        input_name = arg.input_name
        input_spec = inputs_dict[input_name]

        if input_name in arguments:

          # InputUriPlaceholder can only be used for artifacts.
          # If the input type isn't an artifact type, convert it to an artifact.
          if type_utils.is_parameter_type(input_spec.type):
            warnings.warn(
                f'Input "{input_name}" with type "{input_spec.type}" should not'
                ' be paired with InputPathPlaceholder.'
                ' Removing the type to convert the input to a generic artifact.'
            )
            input_spec.type = None

          # The input must be an artifact.
          if not isinstance(arguments[input_name], dsl.PipelineParam) or \
               arguments[input_name].op_name is None or \
               type_utils.is_parameter_type(arguments[input_name].param_type):

            # Input is either a constant value or a pipeline input
            warnings.warn(
                f'Input "{input_name}" with type "{input_spec.type}" is treated'
                ' as an artifact. But it gets its value from a parameter.'
                ' A legacy adaptor component will be injected.')

            if not isinstance(arguments[input_name], dsl.PipelineParam):
              from_type = type(arguments[input_name])
            elif type_utils.is_parameter_type(arguments[input_name].param_type):
              from_type = arguments[input_name].param_type
            else:
              from_type = str

            task.legacy_data_passing_adaptors[input_name] = (
                legacy_data_passing_adaptor.ParameterToArtifactAdaptor(
                    input_type=from_type,
                    output_type=input_spec.type,
                ))

          input_uri = dsl_utils.input_artifact_uri_placeholder(input_name)
          return input_uri
        else:
          input_spec = inputs_dict[input_name]
          if input_spec.optional:
            return None
          else:
            raise ValueError(
                'No value provided for input {}'.format(input_name))

      elif isinstance(arg, _structures.OutputUriPlaceholder):

        output_name = arg.output_name
        output_spec = outputs_dict[output_name]
        # OutputUriPlaceholder can only be used for artifacts.
        # If the output type isn't an artifact type, convert it to an artifact.
        if type_utils.is_parameter_type(output_spec.type):
          warnings.warn(
              f'Output "{output_name}" with type "{output_spec.type}" should not'
              ' be paired with InputPathPlaceholder.'
              ' Removing the type to convert the output to a generic artifact.'
          )
          output_spec.type = None


        output_uri = dsl_utils.output_artifact_uri_placeholder(output_name)
        return output_uri

      return placeholder_resolver.resolve_placeholder(
          arg=arg,
          component_spec=component_spec,
          arguments=arguments,
      )

    resolved_cmd = _components._resolve_command_line_and_paths(
        component_spec=component_spec,
        arguments=arguments,
        input_path_generator=_resolve_input_path_placeholder,
        output_path_generator=_resolve_output_path_placeholder,
        placeholder_resolver=_resolve_ir_placeholders_v2,
    )
    return resolved_cmd

  pipeline_task_spec = pipeline_spec_pb2.PipelineTaskSpec()

  # Check types of the reference arguments and serialize PipelineParams
  arguments = arguments.copy()

  resolved_cmd = _resolve_commands_and_args_v2(
      component_spec=component_spec, arguments=arguments)

  # Preserve input params for ContainerOp.inputs
  input_params_set = set([
      param for param in arguments.values()
      if isinstance(param, _pipeline_param.PipelineParam)
  ])

  for input_name, argument_value in arguments.items():
    input_type = component_spec._inputs_dict[input_name].type
    argument_type = None

    if isinstance(argument_value, _pipeline_param.PipelineParam):
      argument_type = argument_value.param_type

      types.verify_type_compatibility(
          argument_type, input_type,
          'Incompatible argument passed to the input "{}" of component "{}": '
          .format(input_name, component_spec.name))

      # Loop arguments defaults to 'String' type if type is unknown.
      # This has to be done after the type compatiblity check.
      if argument_type is None and isinstance(
          argument_value,
          (_for_loop.LoopArguments, _for_loop.LoopArgumentVariable)):
        argument_type = 'String'

      arguments[input_name] = str(argument_value)

      if type_utils.is_parameter_type(argument_type):
        if argument_value.op_name:
          pipeline_task_spec.inputs.parameters[
              input_name].task_output_parameter.producer_task = (
                  dsl_utils.sanitize_task_name(argument_value.op_name))
          pipeline_task_spec.inputs.parameters[
              input_name].task_output_parameter.output_parameter_key = (
                  argument_value.name)
        else:
          pipeline_task_spec.inputs.parameters[
              input_name].component_input_parameter = argument_value.name
      else:
        if argument_value.op_name:
          pipeline_task_spec.inputs.artifacts[
              input_name].task_output_artifact.producer_task = (
                  dsl_utils.sanitize_task_name(argument_value.op_name))
          pipeline_task_spec.inputs.artifacts[
              input_name].task_output_artifact.output_artifact_key = (
                  argument_value.name)
        else:
          warnings.warn(
              f'The pipeline input "${argument_value.name}" should be type'
              ' annotated with one of the parameter types: str, int, float,'
              ' bool, dict, or list. Unrecognized types will be default to'
              ' string type.')
          pipeline_task_spec.inputs.parameters[
              input_name].component_input_parameter = argument_value.name

    elif isinstance(argument_value, str):
      argument_type = 'String'
      pipeline_params = _pipeline_param.extract_pipelineparams_from_any(
          argument_value)
      if pipeline_params and kfp.COMPILING_FOR_V2:
        # argument_value contains PipelineParam placeholders which needs to be
        # replaced. And the input needs to be added to the task spec.
        for param in pipeline_params:
          # Form the name for the compiler injected input, and make sure it
          # doesn't collide with any existing input names.
          additional_input_name = (
              dsl_component_spec.additional_input_name_for_pipelineparam(param))
          for existing_input_name, _ in arguments.items():
            if existing_input_name == additional_input_name:
              raise ValueError('Name collision between existing input name '
                               '{} and compiler injected input name {}'.format(
                                   existing_input_name, additional_input_name))

          # Add the additional param to the input params set. Otherwise, it will
          # not be included when the params set is not empty.
          input_params_set.add(param)
          additional_input_placeholder = dsl_utils.input_parameter_placeholder(
              additional_input_name)
          argument_value = argument_value.replace(param.pattern,
                                                  additional_input_placeholder)

          # The output references are subject to change -- the producer task may
          # not be whitin the same DAG.
          if param.op_name:
            pipeline_task_spec.inputs.parameters[
                additional_input_name].task_output_parameter.producer_task = (
                    dsl_utils.sanitize_task_name(param.op_name))
            pipeline_task_spec.inputs.parameters[
                additional_input_name].task_output_parameter.output_parameter_key = (
                    param.name)
          else:
            pipeline_task_spec.inputs.parameters[
                additional_input_name].component_input_parameter = param.full_name

      input_type = component_spec._inputs_dict[input_name].type
      pipeline_task_spec.inputs.parameters[
          input_name].runtime_value.constant_value.string_value = (
              argument_value)
    elif isinstance(argument_value, int):
      argument_type = 'Integer'
      pipeline_task_spec.inputs.parameters[
          input_name].runtime_value.constant_value.int_value = argument_value
    elif isinstance(argument_value, float):
      argument_type = 'Float'
      pipeline_task_spec.inputs.parameters[
          input_name].runtime_value.constant_value.double_value = argument_value
    elif isinstance(argument_value, _container_op.ContainerOp):
      raise TypeError(
          'ContainerOp object {} was passed to component as an input argument. '
          'Pass a single output instead.'.format(input_name))
    else:
      if kfp.COMPILING_FOR_V2:
        raise NotImplementedError(
            'Input argument supports only the following types: PipelineParam'
            ', str, int, float. Got: "{}".'.format(argument_value))

    # Wire in the legacy data passing adaptor if applicable
    if input_name in task.legacy_data_passing_adaptors:
      adaptor = task.legacy_data_passing_adaptors[input_name]

      if isinstance(adaptor,
                    legacy_data_passing_adaptor.ParameterToArtifactAdaptor):

        adaptor.task_spec.inputs.parameters[adaptor.INPUT_KEY].CopyFrom(
            pipeline_task_spec.inputs.parameters[input_name])
        pipeline_task_spec.inputs.parameters.pop(input_name, None)
        pipeline_task_spec.inputs.artifacts[
            input_name].task_output_artifact.producer_task = (
                adaptor.task_spec.task_info.name)
        pipeline_task_spec.inputs.artifacts[
            input_name].task_output_artifact.output_artifact_key = (
                adaptor.OUTPUT_KEY)

      elif isinstance(adaptor,
                      legacy_data_passing_adaptor.ArtifactToParameterAdaptor):

        adaptor.task_spec.inputs.artifacts[adaptor.INPUT_KEY].CopyFrom(
            pipeline_task_spec.inputs.artifacts[input_name])
        pipeline_task_spec.inputs.artifacts.pop(input_name, None)
        pipeline_task_spec.inputs.parameters[
            input_name].task_output_parameter.producer_task = (
                adaptor.task_spec.task_info.name)
        pipeline_task_spec.inputs.parameters[
            input_name].task_output_parameter.output_parameter_key = (
                adaptor.OUTPUT_KEY)

      else:
        raise RuntimeError(
            f'Expect a valid legacy data passing adaptor. Got {type(adaptor)}.')

      # Remove the original input PipelineParam from the inputs set, so that
      # later when we build the task dependencies, the task that produces the
      # PipelineParam will not be counted as a direct depdentent task.
      if isinstance(argument_value, dsl.PipelineParam):
        input_params_set.remove(argument_value)

    argument_is_parameter_type = type_utils.is_parameter_type(argument_type)
    input_is_parameter_type = type_utils.is_parameter_type(input_type)
    if kfp.COMPILING_FOR_V2 and (argument_is_parameter_type !=
                                 input_is_parameter_type):
      if isinstance(argument_value, dsl.PipelineParam):
        param_or_value_msg = 'PipelineParam "{}"'.format(
            argument_value.full_name)
      else:
        param_or_value_msg = 'value "{}"'.format(argument_value)

      warnings.warn(
          'Passing '
          '{param_or_value} with type "{arg_type}" (as "{arg_category}") to '
          'component input '
          '"{input_name}" with type "{input_type}" (as "{input_category}") is '
          'incompatible. Please fix the type of the component input.'.format(
              param_or_value=param_or_value_msg,
              arg_type=argument_type,
              arg_category='Parameter'
              if argument_is_parameter_type else 'Artifact',
              input_name=input_name,
              input_type=input_type,
              input_category='Paramter'
              if input_is_parameter_type else 'Artifact',
          ))

  if not component_spec.name:
    component_spec.name = _components._default_component_name

  # task.name is unique at this point.
  pipeline_task_spec.task_info.name = (dsl_utils.sanitize_task_name(task.name))

  task.container_spec = (
      pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec(
          image=component_spec.implementation.container.image,
          command=resolved_cmd.command,
          args=resolved_cmd.args))

  # TODO(chensun): dedupe IR component_spec and contaienr_spec
  pipeline_task_spec.component_ref.name = (
      dsl_utils.sanitize_component_name(task.name))
  executor_label = dsl_utils.sanitize_executor_label(task.name)

  task.component_spec = dsl_component_spec.build_component_spec_from_structure(
      component_spec, executor_label, arguments.keys())

  task.task_spec = pipeline_task_spec

  # Override command and arguments if compiling to v2.
  if kfp.COMPILING_FOR_V2:
    task.command = resolved_cmd.command
    task.arguments = resolved_cmd.args

    # limit this to v2 compiling only to avoid possible behavior change in v1.
    task.inputs = list(input_params_set)
