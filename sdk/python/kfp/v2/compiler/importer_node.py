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

from typing import Tuple
from kfp.ir import pipeline_spec_pb2

_OUTPUT_KEY = 'result'

def build_importer_spec(
    dependent_task_name : str,
    artifact_name : str,
    artifact_type : str,
) -> Tuple[
    pipeline_spec_pb2.PipelineTaskSpec,
    pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec
]:

    task_spec = pipeline_spec_pb2.PipelineTaskSpec();
    task_spec.task_info.name = f'{dependent_task_name}_{artifact_name}_importer'
    task_spec.outputs.artifacts[_OUTPUT_KEY].artifact_type.schema_title = artifact_type
    task_spec.executor_label = task_spec.task_info.name

    importer_spec = pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec()
    importer_spec.artifact_uri.runtime_parameter = artifact_name
    importer_spec.type_schema.schema_title = artifact_type

    return task_spec, importer_spec
