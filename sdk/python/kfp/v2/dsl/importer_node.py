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

from kfp.pipeline_spec import pipeline_spec_pb2

OUTPUT_KEY = 'result'


def build_importer_spec(
    input_type_schema: str,
    pipeline_param_name: str = None,
    constant_value: str = None
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
  assert bool(pipeline_param_name) != bool(constant_value), (
      'importer spec should be built using either pipeline_param_name or '
      'constant_value.')
  importer_spec = pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec()
  importer_spec.type_schema.instance_schema = input_type_schema
  if pipeline_param_name:
    importer_spec.artifact_uri.runtime_parameter = pipeline_param_name
  elif constant_value:
    importer_spec.artifact_uri.constant_value.string_value = constant_value
  return importer_spec


def build_importer_task_spec(
    dependent_task: pipeline_spec_pb2.PipelineTaskSpec,
    input_name: str,
    input_type_schema: str,
) -> pipeline_spec_pb2.PipelineTaskSpec:
  """Builds an importer task spec.

  Args:
    dependent_task: The task requires importer node.
    input_name: The name of the input artifact needs to be imported.
    input_type_schema: The type of the input artifact.

  Returns:
    An importer node task spec.
  """
  dependent_task_name = dependent_task.task_info.name

  task_spec = pipeline_spec_pb2.PipelineTaskSpec()
  task_spec.task_info.name = '{}_{}_importer'.format(dependent_task_name,
                                                     input_name)
  task_spec.outputs.artifacts[OUTPUT_KEY].artifact_type.instance_schema = (
      input_type_schema)
  task_spec.executor_label = task_spec.task_info.name

  return task_spec
