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
"""Utility function for building Importer Node spec."""

from typing import Optional

from kfp.dsl import dsl_utils
from kfp.pipeline_spec import pipeline_spec_pb2

OUTPUT_KEY = 'result'


def build_importer_spec(
    input_type_schema: pipeline_spec_pb2.ArtifactTypeSchema,
    pipeline_param_name: Optional[str] = None,
    constant_value: Optional[str] = None
) -> pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec:
  """Builds an importer executor spec.

  Args:
    input_type_schema: The type of the input artifact.
    pipeline_param_name: The name of the pipeline parameter if the importer gets
      its artifacts_uri via a pipeline parameter. This argument is mutually
      exclusive with constant_value.
    constant_value: The value of artifact_uri in case a contant value is passed
      directly into the compoent op. This argument is mutually exclusive with
      pipeline_param_name.

  Returns:
    An importer spec.
  """
  assert bool(pipeline_param_name is None) != bool(constant_value is None), (
      'importer spec should be built using either pipeline_param_name or '
      'constant_value.')
  importer_spec = pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec()
  importer_spec.type_schema.CopyFrom(input_type_schema)
  # TODO: subject to IR change on artifact_uri message type.
  if pipeline_param_name:
    importer_spec.artifact_uri.runtime_parameter = pipeline_param_name
  elif constant_value:
    importer_spec.artifact_uri.constant_value.string_value = constant_value
  return importer_spec


def build_importer_task_spec(
    importer_base_name: str,
) -> pipeline_spec_pb2.PipelineTaskSpec:
  """Builds an importer task spec.

  Args:
    importer_base_name: The base name of the importer node.

  Returns:
    An importer node task spec.
  """
  result = pipeline_spec_pb2.PipelineTaskSpec()
  result.task_info.name = dsl_utils.sanitize_task_name(importer_base_name)
  result.component_ref.name = dsl_utils.sanitize_component_name(
      importer_base_name)

  return result


def build_importer_component_spec(
    importer_base_name: str,
    input_name: str,
    input_type_schema: pipeline_spec_pb2.ArtifactTypeSchema,
) -> pipeline_spec_pb2.ComponentSpec:
  """Builds an importer component spec.

  Args:
    importer_base_name: The base name of the importer node.
    dependent_task: The task requires importer node.
    input_name: The name of the input artifact needs to be imported.
    input_type_schema: The type of the input artifact.

  Returns:
    An importer node component spec.
  """
  result = pipeline_spec_pb2.ComponentSpec()
  result.executor_label = dsl_utils.sanitize_executor_label(importer_base_name)
  result.input_definitions.parameters[
      input_name].type = pipeline_spec_pb2.PrimitiveType.STRING
  result.output_definitions.artifacts[
      OUTPUT_KEY].artifact_type.CopyFrom(input_type_schema)

  return result


def generate_importer_base_name(dependent_task_name: str,
                                input_name: str) -> str:
  """Generates the base name of an importer node.

  The base name is formed by connecting the dependent task name and the input
  artifact name. It's used to form task name, component ref, and executor label.

  Args:
    dependent_task_name: The name of the task requires importer node.
    input_name: The name of the input artifact needs to be imported.

  Returns:
    A base importer node name.
  """
  return 'importer-{}-{}'.format(dependent_task_name, input_name)
