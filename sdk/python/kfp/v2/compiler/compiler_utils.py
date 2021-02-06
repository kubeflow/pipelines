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
"""KFP v2 DSL compiler utility functions."""

import re
from typing import Any, Mapping, Optional, Union

from kfp.containers import _component_builder
from kfp.dsl import _container_op
from kfp.pipeline_spec import pipeline_spec_pb2

# Alias for PipelineContainerSpec
PipelineContainerSpec = pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec


def build_runtime_config_spec(
    output_directory: str,
    pipeline_parameters: Optional[Mapping[str, Any]] = None,
) -> pipeline_spec_pb2.PipelineJob.RuntimeConfig:
  """Converts pipeine parameters to runtime parameters mapping.

  Args:
    output_directory: The root of pipeline outputs.
    pipeline_parameters: The mapping from parameter names to values. Optional.

  Returns:
    A pipeline job RuntimeConfig object.
  """

  def _get_value(value: Union[int, float, str]) -> pipeline_spec_pb2.Value:
    assert value is not None, 'None values should be filterd out.'

    result = pipeline_spec_pb2.Value()
    if isinstance(value, int):
      result.int_value = value
    elif isinstance(value, float):
      result.double_value = value
    elif isinstance(value, str):
      result.string_value = value
    else:
      raise TypeError('Got unknown type of value: {}'.format(value))

    return result

  parameter_values = pipeline_parameters or {}
  return pipeline_spec_pb2.PipelineJob.RuntimeConfig(
      gcs_output_directory=output_directory,
      parameters={
          k: _get_value(v) for k, v in parameter_values.items() if v is not None
      })


def validate_pipeline_name(name: str) -> None:
  """Validate pipeline name.

  A valid pipeline name should match ^[a-z0-9][a-z0-9-]{0,127}$.

  Args:
    name: The pipeline name.

  Raises:
    ValueError if the pipeline name doesn't conform to the regular expression.
  """
  pattern = re.compile(r'^[a-z0-9][a-z0-9-]{0,127}$')
  if not pattern.match(name):
    raise ValueError('Invalid pipeline name: %s.\n'
                     'Please specify a pipeline name that matches the regular '
                     'expression "^[a-z0-9][a-z0-9-]{0,127}$" using '
                     '`dsl.pipeline(name=...)` decorator.' % name)

def is_v2_component(op: _container_op.ContainerOp) -> bool:
  """Determines whether a component is a KFP v2 component."""
  if not op._metadata or not op._metadata.metadata:
    return False
  if not (op._metadata.metadata.annotations
          or _component_builder.V2_COMPONENT_ANNOTATION
          not in op._metadata.metadata.annotations):
    return False
  return bool(
      op._metadata.metadata.annotations[
        _component_builder.V2_COMPONENT_ANNOTATION])


def refactor_v2_container_spec(
    container_spec: PipelineContainerSpec) -> None:
  """Refactor the container spec for a v2 component."""
  if not '--function_name' in container_spec.args:
    raise RuntimeError('V2 component is expected to have function_name as a '
                       'command line arg.')
  fn_name_idx = list(container_spec.args).index('--function_name') + 1
  fn_name = container_spec.args[fn_name_idx]
  container_spec.ClearField('command')
  container_spec.ClearField('args')
  container_spec.command.extend(['python', '-m', 'kfp.container.entrypoint'])
  container_spec.args.extend(
      ['--executor_input_str','{{$}}', '--function_name', fn_name])