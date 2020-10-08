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

from typing import Tuple
from kfp.v2.proto import pipeline_spec_pb2

OUTPUT_KEY = 'result'


def build_importer_spec(
    dependent_task: pipeline_spec_pb2.PipelineTaskSpec,
    input_name: str,
    input_type_schema: str,
) -> Tuple[pipeline_spec_pb2.PipelineTaskSpec,
           pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec]:
  """Build importer task spec and importer executor spec.

  Args:
    dependent_task: the task requires importer node.
    input_name: the name of the input artifact needs to be imported.
    input_type_schema: the type of the input artifact.

  Returns:
    a tuple of task_spec and importer_spec
  """
  dependent_task_name = dependent_task.task_info.name
  pipeline_parameter_name = (
      dependent_task.inputs.artifacts[input_name].output_artifact_key)

  task_spec = pipeline_spec_pb2.PipelineTaskSpec()
  task_spec.task_info.name = '{}_{}_importer'.format(dependent_task_name,
                                                     input_name)
  task_spec.outputs.artifacts[OUTPUT_KEY].artifact_type.instance_schema = (
      input_type_schema)
  task_spec.executor_label = task_spec.task_info.name

  importer_spec = pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec()
  importer_spec.artifact_uri.runtime_parameter = pipeline_parameter_name
  importer_spec.type_schema.instance_schema = input_type_schema

  return task_spec, importer_spec
