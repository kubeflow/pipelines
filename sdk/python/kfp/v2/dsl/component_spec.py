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
"""Functions for creating IR ComponentSpec instance."""

from typing import List

from kfp.components import _structures as structures
from kfp.v2 import dsl
from kfp.v2.dsl import dsl_utils
from kfp.v2.dsl import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2


def build_component_spec_from_structure(
    component_spec: structures.ComponentSpec,
) -> pipeline_spec_pb2.ComponentSpec:
  """Builds an IR ComponentSpec instance from structures.ComponentSpec.

  Args:
    component_spec: The structure component spec.

  Returns:
    An instance of IR ComponentSpec.
  """
  result = pipeline_spec_pb2.ComponentSpec()
  result.executor_label = dsl_utils.sanitize_executor_label(component_spec.name)

  for input_spec in component_spec.inputs or []:
    if type_utils.is_parameter_type(input_spec.type):
      result.input_definitions.parameters[
          input_spec.name].type = type_utils.get_parameter_type(input_spec.type)
    else:
      result.input_definitions.artifacts[
          input_spec.name].artifact_type.instance_schema = (
              type_utils.get_artifact_type_schema(input_spec.type))

  for output_spec in component_spec.outputs or []:
    if type_utils.is_parameter_type(output_spec.type):
      result.output_definitions.parameters[
          output_spec.name].type = type_utils.get_parameter_type(
              output_spec.type)
    else:
      result.output_definitions.artifacts[
          output_spec.name].artifact_type.instance_schema = (
              type_utils.get_artifact_type_schema(output_spec.type))

  return result


def build_root_spec_from_pipeline_params(
    pipeline_params: List[dsl.PipelineParam],
) -> pipeline_spec_pb2.ComponentSpec:
  """Builds the root component spec instance from pipeline params.

  This is useful when building the component spec for a pipeline (aka piipeline
  root). Such a component spec doesn't need output_definitions, and its
  implementation field will be filled in later.

  Args:
    pipeline_params: The list of pipeline params.

  Returns:
    An instance of IR ComponentSpec.
  """
  result = pipeline_spec_pb2.ComponentSpec()
  for param in pipeline_params or []:
    if type_utils.is_parameter_type(param.param_type):
      result.input_definitions.parameters[
          param.name].type = type_utils.get_parameter_type(param.param_type)
    else:
      result.input_definitions.artifacts[
          param.name].artifact_type.instance_schema = (
              type_utils.get_artifact_type_schema(param.param_type))

  return result
