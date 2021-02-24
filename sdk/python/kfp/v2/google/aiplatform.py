# Copyright 2021 Google LLC
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
"""Connector components of Google AI Platform (Unified) services."""
from typing import Any, Dict, List, Optional, Type, Union

from absl import logging
import collections
from kfp import dsl
from kfp.components import _structures
from kfp.dsl import artifact
from kfp.pipeline_spec import pipeline_spec_pb2
from kfp.v2.dsl import dsl_utils
from kfp.v2.dsl import type_utils

_AIPlatformCustomJobSpec = pipeline_spec_pb2.PipelineDeploymentConfig.AIPlatformCustomJobSpec
_DUMMY_CONTAINER_OP_IMAGE = 'dummy/image'
_DUMMY_PATH = 'dummy/path'
_MAX_PACKAGE_URIS = 100
_DEFAULT_CUSTOM_JOB_MACHINE_TYPE = 'n1-standard-2'

_ValueOrPipelineParam = Union[dsl.PipelineParam, str, float, int]

# TODO: Support all declared types in
# components._structures.CommandlineArgumenType
_CommandlineArgumentType = Union[
  str, int, float,
  _structures.InputValuePlaceholder,
  _structures.InputPathPlaceholder,
  _structures.OutputPathPlaceholder,
  _structures.InputUriPlaceholder,
  _structures.OutputUriPlaceholder,
]


# TODO: extract this to a utils module, and share with dsl.component_bridge
def _input_artifact_uri_placeholder(input_key: str) -> str:
  return "{{{{$.inputs.artifacts['{}'].uri}}}}".format(input_key)


def _input_artifact_path_placeholder(input_key: str) -> str:
  return "{{{{$.inputs.artifacts['{}'].path}}}}".format(input_key)


def _input_parameter_placeholder(input_key: str) -> str:
  return "{{{{$.inputs.parameters['{}']}}}}".format(input_key)


def _output_artifact_uri_placeholder(output_key: str) -> str:
  return "{{{{$.outputs.artifacts['{}'].uri}}}}".format(output_key)


def _output_artifact_path_placeholder(output_key: str) -> str:
  return "{{{{$.outputs.artifacts['{}'].path}}}}".format(output_key)


def _output_parameter_path_placeholder(output_key: str) -> str:
  return "{{{{$.outputs.parameters['{}'].output_file}}}}".format(output_key)


class AiPlatformCustomJobOp(dsl.ContainerOp):
  """V2 AiPlatformCustomJobOp class.

  This class inherits V1 ContainerOp class so that it can be correctly picked
  by compiler. The implementation of the task is an AiPlatformCustomJobSpec
  proto message.
  """

  def __init__(self,
      name: str,
      custom_job_spec: Dict[str, Any],
      component_spec: pipeline_spec_pb2.ComponentSpec,
      task_spec: pipeline_spec_pb2.PipelineTaskSpec,
      task_inputs: Optional[List[dsl.InputArgumentPath]] = None,
      task_outputs: Optional[Dict[str, str]] = None):
    """Instantiates the AiPlatformCustomJobOp object.

    Args:
      name: Name of the task.
      custom_job_spec: JSON struct of the CustomJob spec, representing the job
        that will be submitted to AI Platform (Unified) service. See
        https://cloud.google.com/ai-platform-unified/docs/reference/rest/v1beta1/CustomJobSpec
        for detailed reference.
      task_inputs: Optional. List of InputArgumentPath of this task. Each
        InputArgumentPath object has 3 attributes: input, path and argument
        we actually only care about the input, which will be translated to the
        input name of the component spec.
        Path and argument are tied to artifact argument in Argo, which is not
        used in this case.
      task_outputs: Optional. Mapping of task outputs to its URL.
    """
    old_warn_value = dsl.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING
    dsl.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING = True
    super().__init__(
        name=name,
        image=_DUMMY_CONTAINER_OP_IMAGE,
        artifact_argument_paths=task_inputs,
        file_outputs=task_outputs
    )
    self.component_spec = component_spec
    self.task_spec = task_spec
    self.custom_job_spec = custom_job_spec
    dsl.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING = old_warn_value


def _get_custom_job_op(
    task_name: str,
    job_spec: Dict[str, Any],
    input_artifacts: Optional[Dict[str, dsl.PipelineParam]] = None,
    input_parameters: Optional[Dict[str, _ValueOrPipelineParam]] = None,
    output_artifacts: Optional[Dict[str, Type[artifact.Artifact]]] = None,
    output_parameters: Optional[Dict[str, Any]] = None,
) -> AiPlatformCustomJobOp:
  """Gets an AiPlatformCustomJobOp from job spec and I/O definition."""
  pipeline_task_spec = pipeline_spec_pb2.PipelineTaskSpec()
  pipeline_component_spec = pipeline_spec_pb2.ComponentSpec()

  pipeline_task_spec.task_info.CopyFrom(
      pipeline_spec_pb2.PipelineTaskInfo(name=task_name))

  # Iterate through the inputs/outputs declaration to get pipeline component
  # spec.
  for input_name, param in input_parameters.items():
    if isinstance(param, dsl.PipelineParam):
      pipeline_component_spec.input_definitions.parameters[
        input_name].type = type_utils.get_parameter_type(param.param_type)
    else:
      pipeline_component_spec.input_definitions.parameters[
        input_name].type = type_utils.get_parameter_type(type(param))

  for input_name, art in input_artifacts.items():
    if not isinstance(art, dsl.PipelineParam):
      raise RuntimeError(
          'Get unresolved input artifact for input %s. Input '
          'artifacts must be connected to a producer task.' % input_name)
    pipeline_component_spec.input_definitions.artifacts[
      input_name].artifact_type.CopyFrom(
        type_utils.get_artifact_type_schema_message(art.param_type))

  for output_name, param_type in output_parameters.items():
    pipeline_component_spec.output_definitions.parameters[
      output_name].type = type_utils.get_parameter_type(param_type)

  for output_name, artifact_type in output_artifacts.items():
    pipeline_component_spec.output_definitions.artifacts[
      output_name].artifact_type.CopyFrom(artifact_type.get_ir_type())

  pipeline_component_spec.executor_label = dsl_utils.sanitize_executor_label(
      task_name)

  # Iterate through the inputs/outputs specs to get pipeline task spec.
  for input_name, param in input_parameters.items():
    if isinstance(param, dsl.PipelineParam) and param.op_name:
      # If the param has a valid op_name, this should be a pipeline parameter
      # produced by an upstream task.
      pipeline_task_spec.inputs.parameters[input_name].CopyFrom(
          pipeline_spec_pb2.TaskInputsSpec.InputParameterSpec(
              task_output_parameter=pipeline_spec_pb2.TaskInputsSpec.InputParameterSpec.TaskOutputParameterSpec(
                  producer_task='task-{}'.format(param.op_name),
                  output_parameter_key=param.name
              )))
    elif isinstance(param, dsl.PipelineParam) and not param.op_name:
      # If a valid op_name is missing, this should be a pipeline parameter.
      pipeline_task_spec.inputs.parameters[input_name].CopyFrom(
          pipeline_spec_pb2.TaskInputsSpec.InputParameterSpec(
              component_input_parameter=param.name))
    else:
      # If this is not a pipeline param, then it should be a value.
      pipeline_task_spec.inputs.parameters[input_name].CopyFrom(
          pipeline_spec_pb2.TaskInputsSpec.InputParameterSpec(
              runtime_value=pipeline_spec_pb2.ValueOrRuntimeParameter(
                  constant_value=dsl_utils.get_value(param))))

  for input_name, art in input_artifacts.items():
    if art.op_name:
      # If the param has a valid op_name, this should be an artifact produced
      # by an upstream task.
      pipeline_task_spec.inputs.artifacts[input_name].CopyFrom(
          pipeline_spec_pb2.TaskInputsSpec.InputArtifactSpec(
              task_output_artifact=pipeline_spec_pb2.TaskInputsSpec.InputArtifactSpec.TaskOutputArtifactSpec(
                  producer_task='task-{}'.format(art.op_name),
                  output_artifact_key=art.name)))
    else:
      # Otherwise, this should be from the input of the subdag.
      pipeline_task_spec.inputs.artifacts[input_name].CopyFrom(
          pipeline_spec_pb2.TaskInputsSpec.InputArtifactSpec(
              component_input_artifact=art.name
          ))

  # TODO: Add task dependencies/trigger policies/caching/iterator
  pipeline_task_spec.component_ref.name = dsl_utils.sanitize_component_name(
      task_name)

  # Construct dummy I/O declaration for the op.
  # TODO: resolve name conflict instead of raising errors.
  dummy_outputs = collections.OrderedDict()
  for output_name, _ in output_artifacts.items():
    dummy_outputs[output_name] = _DUMMY_PATH

  for output_name, _ in output_parameters.items():
    if output_name in dummy_outputs:
      raise KeyError('Got name collision for output key %s. Consider renaming '
                     'either output parameters or output '
                     'artifacts.' % output_name)
    dummy_outputs[output_name] = _DUMMY_PATH

  dummy_inputs = collections.OrderedDict()
  for input_name, art in input_artifacts.items():
    dummy_inputs[input_name] = _DUMMY_PATH
  for input_name, param in input_parameters.items():
    if input_name in dummy_inputs:
      raise KeyError('Got name collision for input key %s. Consider renaming '
                     'either input parameters or input '
                     'artifacts.' % input_name)
    dummy_inputs[input_name] = _DUMMY_PATH

  # Construct the AIP (Unified) custom job op.
  return AiPlatformCustomJobOp(
      name=task_name,
      custom_job_spec=job_spec,
      component_spec=pipeline_component_spec,
      task_spec=pipeline_task_spec,
      task_inputs=[
          dsl.InputArgumentPath(
              argument=dummy_inputs[input_name],
              input=input_name,
              path=path,
          ) for input_name, path in dummy_inputs.items()
      ],
      task_outputs=dummy_outputs
  )


def custom_job(
    name: str,
    input_artifacts: Optional[Dict[str, dsl.PipelineParam]] = None,
    input_parameters: Optional[Dict[str, _ValueOrPipelineParam]] = None,
    output_artifacts: Optional[Dict[str, Type[artifact.Artifact]]] = None,
    output_parameters: Optional[Dict[str, Type[Union[str, float, int]]]] = None,
    # Custom container training specs.
    image_uri: Optional[str] = None,
    commands: Optional[List[str]] = None,
    # Custom Python training spec.
    executor_image_uri: Optional[str] = None,
    package_uris: Optional[List[str]] = None,
    python_module: Optional[str] = None,
    # Command line args of the user program.
    args: Optional[List[Any]] = None,
    machine_type: Optional[str] = None,
    # Full-fledged custom job API spec. For details please see:
    # https://cloud.google.com/ai-platform-unified/docs/reference/rest/v1beta1/CustomJobSpec
    additional_job_spec: Optional[Dict[str, Any]] = None
) -> AiPlatformCustomJobOp:
  """DSL representation of a AI Platform (Unified) custom training job.

  For detailed doc of the service, please refer to
  https://cloud.google.com/ai-platform-unified/docs/training/create-custom-job

  Args:
    name: The name of this task.
    input_artifacts: The input artifact specification. Should be a mapping from
      input name to output from upstream tasks.
    input_parameters: The input parameter specification. Should be a mapping
      from input name to one of the following three:
      - output from upstream tasks, or
      - pipeline parameter, or
      - constant value
    output_artifacts: The output artifact declaration. Should be a mapping from
      output name to a type subclassing artifact.Artifact.
    output_parameters: The output parameter declaration. Should be a mapping
      from output name to one of 1) str, 2) float, or 3) int.
    image_uri: The URI of the container image containing the user training
      program. Applicable for custom container training.
    commands: The container command/entrypoint. Applicable for custom container
      training.
    executor_image_uri: The URI of the container image containing the
      dependencies of user training program. Applicable for custom Python
      training.
    package_uris: The Python packages that are expected to be running on the
      executor container. Applicable for custom Python training.
    python_module: The entrypoint of user training program. Applicable for
      custom Python training.
    args: The command line arguments of user training program. This is expected
      to be a list of either 1) constant string, or 2) KFP DSL placeholders, to
      connect the user program with the declared component I/O.
    machine_type: The machine type used to run the training program. The value
      of this field will be propagated to all worker pools if not specified
      otherwise in additional_job_spec.
    additional_job_spec: Full-fledged custom job API spec. The value specified
      in this field will override the defaults provided through other function
      parameters.

      For details please see:
      https://cloud.google.com/ai-platform-unified/docs/reference/rest/v1beta1/CustomJobSpec

  Returns:
    A KFP ContainerOp object represents the launcher container job, from which
    the user training program will be submitted to AI Platform (Unified) Custom
    Job service.

  Raises:
    KeyError on name collision between parameter and artifact I/O declaration.
    ValueError when:
      1. neither or both image_uri and executor_image_uri are provided; or
      2. no valid package_uris and python_module is provided for custom Python
         training.
  """
  # Check the sanity of the provided parameters.
  input_artifacts = input_artifacts or {}
  input_parameters = input_parameters or {}
  output_artifacts = output_artifacts or {}
  output_parameters = output_parameters or {}
  if bool(set(input_artifacts.keys()) & set(input_parameters.keys())):
    raise KeyError('Input key conflict between input parameters and artifacts.')
  if bool(set(output_artifacts.keys()) & set(output_parameters.keys())):
    raise KeyError('Output key conflict between output parameters and '
                   'artifacts.')

  if not additional_job_spec and  bool(image_uri) == bool(executor_image_uri):
    raise ValueError('The user program needs to be either a custom container '
                     'training job, or a custom Python training job')

  # For Python custom training job, package URIs and modules are also required.
  if executor_image_uri:
    if not package_uris or not python_module or len(
        package_uris) > _MAX_PACKAGE_URIS:
      raise ValueError('For custom Python training, package_uris with length < '
                       '100 and python_module are expected.')

  # Check and scaffold the parameters to form the custom job request spec.
  custom_job_spec = additional_job_spec or {}
  if not custom_job_spec.get('workerPoolSpecs'):
    # Single node training, deriving job spec from top-level parameters.
    if image_uri:
      # Single node custom container training
      worker_pool_spec = {
          "machineSpec": {
              "machineType": machine_type or _DEFAULT_CUSTOM_JOB_MACHINE_TYPE
          },
          "replicaCount": "1",
          "containerSpec": {
              "imageUri": image_uri,
          }
      }
      if commands:
        worker_pool_spec['containerSpec']['command'] = commands
      if args:
        worker_pool_spec['containerSpec']['args'] = args
      custom_job_spec['workerPoolSpecs'] = [worker_pool_spec]
    if executor_image_uri:
      worker_pool_spec = {
          "machineSpec": {
              "machineType": machine_type or _DEFAULT_CUSTOM_JOB_MACHINE_TYPE
          },
          "replicaCount": "1",
          "pythonPackageSpec": {
              "executorImageUri": executor_image_uri,
              "packageUris": package_uris,
              "pythonModule": python_module,
              "args": args
          }
      }
      custom_job_spec['workerPoolSpecs'] = [worker_pool_spec]
  else:
    # If the full-fledged job spec is provided. We'll use it as much as
    # possible, and patch some top-level parameters.
    for spec in custom_job_spec['workerPoolSpecs']:
      if image_uri:
        if (not spec.get('pythonPackageSpec')
            and not spec.get('containerSpec', {}).get('imageUri')):
          spec['containerSpec'] = spec.get('containerSpec', {})
          spec['containerSpec']['imageUri'] = image_uri
      if commands:
        if (not spec.get('pythonPackageSpec')
            and not spec.get('containerSpec', {}).get('command')):
          spec['containerSpec'] = spec.get('containerSpec', {})
          spec['containerSpec']['command'] = commands
      if executor_image_uri:
        if (not spec.get('containerSpec')
            and not spec.get('pythonPackageSpec', {}).get('executorImageUri')):
          spec['pythonPackageSpec'] = spec.get('pythonPackageSpec', {})
          spec['pythonPackageSpec']['executorImageUri'] = executor_image_uri
      if package_uris:
        if (not spec.get('containerSpec')
            and not spec.get('pythonPackageSpec', {}).get('packageUris')):
          spec['pythonPackageSpec'] = spec.get('pythonPackageSpec', {})
          spec['pythonPackageSpec']['packageUris'] = package_uris
      if python_module:
        if (not spec.get('containerSpec')
            and not spec.get('pythonPackageSpec', {}).get('pythonModule')):
          spec['pythonPackageSpec'] = spec.get('pythonPackageSpec', {})
          spec['pythonPackageSpec']['pythonModule'] = python_module
      if args:
        if spec.get('containerSpec') and not spec['containerSpec'].get('args'):
          spec['containerSpec']['args'] = args
        if (spec.get('pythonPackageSpec')
            and not spec['pythonPackageSpec'].get('args')):
          spec['pythonPackageSpec']['args'] = args

  # Resolve the custom job spec by wiring it with the I/O spec.
  def _resolve_output_path_placeholder(output_key: str) -> str:
    if output_key in output_parameters:
      return _output_parameter_path_placeholder(output_key)
    else:
      return _output_artifact_path_placeholder(output_key)

  def _resolve_cmd(cmd: Optional[_CommandlineArgumentType]) -> Optional[str]:
    """Resolves a single command line cmd/arg."""
    if cmd is None:
      return None
    elif isinstance(cmd, (str, float, int)):
      return str(cmd)
    elif isinstance(cmd, _structures.InputValuePlaceholder):
      return _input_parameter_placeholder(cmd.input_name)
    elif isinstance(cmd, _structures.InputPathPlaceholder):
      return _input_artifact_path_placeholder(cmd.input_name)
    elif isinstance(cmd, _structures.InputUriPlaceholder):
      return _input_artifact_uri_placeholder(cmd.input_name)
    elif isinstance(cmd, _structures.OutputPathPlaceholder):
      return _resolve_output_path_placeholder(cmd.output_name)
    elif isinstance(cmd, _structures.OutputUriPlaceholder):
      return _output_artifact_uri_placeholder(cmd.output_name)
    else:
      raise TypeError('Got unexpected placeholder type for %s' % cmd)

  def _resolve_cmd_lines(cmds: Optional[List[_CommandlineArgumentType]]) -> None:
    """Resolves a list of commands/args."""
    if not cmds:
      return
    for idx, cmd in enumerate(cmds):
      cmds[idx] = _resolve_cmd(cmd)

  for wp_spec in custom_job_spec['workerPoolSpecs']:
    if 'containerSpec' in wp_spec:
      # For custom container training, resolve placeholders in commands and
      # program args.
      container_spec = wp_spec['containerSpec']
      if 'command' in container_spec:
        _resolve_cmd_lines(container_spec['command'])
      if 'args' in container_spec:
        _resolve_cmd_lines(container_spec['args'])
    else:
      assert 'pythonPackageSpec' in wp_spec
      # For custom Python training, resolve placeholders in args only.
      python_spec = wp_spec['pythonPackageSpec']
      if 'args' in python_spec:
        _resolve_cmd_lines(python_spec['args'])

  job_spec = {
      'name': name,
      'jobSpec': custom_job_spec
  }

  return _get_custom_job_op(
      task_name=name,
      job_spec=job_spec,
      input_artifacts=input_artifacts,
      input_parameters=input_parameters,
      output_artifacts=output_artifacts,
      output_parameters=output_parameters
  )
