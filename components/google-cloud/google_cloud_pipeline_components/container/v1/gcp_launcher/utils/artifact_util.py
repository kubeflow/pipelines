# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from typing import Dict
import json
import os
from kfp.v2 import dsl


def update_output_artifact(executor_input: str,
                           target_artifact_name: str,
                           uri: str,
                           metadata: dict = {}):
  """Updates the output artifact with the new uri and metadata."""
  executor_input_json = json.loads(executor_input)
  executor_output = {'artifacts': {}}
  for name, artifacts in executor_input_json.get('outputs',
                                                 {}).get('artifacts',
                                                         {}).items():
    artifacts_list = artifacts.get('artifacts')
    if name == target_artifact_name and artifacts_list:
      updated_runtime_artifact = artifacts_list[0]
      updated_runtime_artifact['uri'] = uri
      updated_runtime_artifact['metadata'] = metadata
      artifacts_list = {'artifacts': [updated_runtime_artifact]}

    executor_output['artifacts'][name] = artifacts_list

  # update the output artifacts.
  os.makedirs(
      os.path.dirname(executor_input_json['outputs']['outputFile']),
      exist_ok=True)
  with open(executor_input_json['outputs']['outputFile'], 'w') as f:
    f.write(json.dumps(executor_output))


# Writes a list of Artifacts to the executor output file.
def update_output_artifacts(executor_input: str, artifacts: list):
  """Updates a list of Artifacts to the executor output file."""
  executor_input_json = json.loads(executor_input)
  executor_output = {'artifacts': {}}
  output_artifacts = executor_input_json.get('outputs', {}).get('artifacts', {})
  # This assumes that no other output artifact exists.
  for artifact in artifacts:
    if artifact.name in output_artifacts.keys():
        # Converts the artifact into executor output artifact
        # https://github.com/kubeflow/pipelines/blob/master/api/v2alpha1/pipeline_spec.proto#L878
        artifacts_list = output_artifacts[artifact.name].get('artifacts')
        if artifacts_list:
          updated_runtime_artifact = artifacts_list[0]
          updated_runtime_artifact['uri'] = artifact.uri
          updated_runtime_artifact['metadata'] = artifact.metadata
          artifacts_list = {'artifacts': [updated_runtime_artifact]}
        executor_output['artifacts'][artifact.name] = artifacts_list

  # update the output artifacts.
  os.makedirs(
      os.path.dirname(executor_input_json['outputs']['outputFile']),
      exist_ok=True)
  with open(executor_input_json['outputs']['outputFile'], 'w') as f:
    f.write(json.dumps(executor_output))
