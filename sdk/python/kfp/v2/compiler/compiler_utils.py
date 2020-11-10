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

from typing import Any, List, Mapping, Optional, Union
from kfp.v2 import dsl
from kfp.pipeline_spec import pipeline_spec_pb2


def build_runtime_parameter_spec(
    pipeline_params: List[dsl.PipelineParam]
) -> Mapping[str, pipeline_spec_pb2.PipelineSpec.RuntimeParameter]:
  """Converts pipeine parameters to runtime parameters mapping.

  Args:
    pipeline_params: The list of pipeline parameters.

  Returns:
    A map of pipeline parameter name to its runtime parameter message.
  """

  def to_message(param: dsl.PipelineParam):
    result = pipeline_spec_pb2.PipelineSpec.RuntimeParameter()
    if param.param_type == 'Integer' or (param.param_type is None and
                                         isinstance(param.value, int)):

      result.type = pipeline_spec_pb2.PrimitiveType.INT
      if param.value is not None:
        result.default_value.int_value = int(param.value)
    elif param.param_type == 'Float' or (param.param_type is None and
                                         isinstance(param.value, float)):
      result.type = pipeline_spec_pb2.PrimitiveType.DOUBLE
      if param.value is not None:
        result.default_value.double_value = float(param.value)
    elif param.param_type == 'String' or param.param_type is None:
      result.type = pipeline_spec_pb2.PrimitiveType.STRING
      if param.value is not None:
        result.default_value.string_value = str(param.value)
    else:
      raise TypeError('Unsupported type "{}" for argument "{}".'.format(
          param.param_type, param.name))
    return result

  return {param.name: to_message(param) for param in pipeline_params}


def build_runtime_config_spec(
    pipeline_root: str,
    pipeline_parameters: Optional[Mapping[str, Any]] = None,
) -> pipeline_spec_pb2.PipelineJob.RuntimeConfig:
  """Converts pipeine parameters to runtime parameters mapping.

  Args:
    pipeline_root: The root of pipeline outputs.
    pipeline_parameters: The mapping from parameter names to values. Optional.

  Returns:
    A pipeline job RuntimeConfig object.
  """

  def _get_value(
      value: Optional[Union[int, float,
                            str]]) -> Optional[pipeline_spec_pb2.Value]:
    if value is None:
      return None

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
      gcs_output_directory=pipeline_root,
      parameters={k: _get_value(v) for k, v in parameter_values.items()})
