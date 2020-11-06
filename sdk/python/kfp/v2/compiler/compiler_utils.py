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

from typing import List, Mapping
from kfp.v2 import dsl
from kfp.v2.proto import pipeline_spec_pb2


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
