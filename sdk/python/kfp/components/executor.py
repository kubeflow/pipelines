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
import json
import inspect
from typing import Callable, Dict
from kfp.components._python_op import InputPath, OutputPath
from kfp.dsl.io_types import InputArtifact, OutputArtifact, create_runtime_artifact


class Executor():
  """Executor executes v2-based Python function components."""

  def __init__(self, executor_input: Dict, function_to_execute: Callable):
    self._func = function_to_execute
    self._input = executor_input
    self._input_artifacts: Dict[str, InputArtifact] = {}
    self._output_artifacts: Dict[str, OutputArtifact] = {}

    for name, artifacts in self._input.get('inputs', {}).get('artifacts',
                                                             {}).items():
      artifacts_list = artifacts.get('artifacts')
      if artifacts_list:
        self._input_artifacts[name] = self._make_input_artifact(
            artifacts_list[0])

    for name, artifacts in self._input.get('outputs', {}).get('artifacts',
                                                              {}).items():
      artifacts_list = artifacts.get('artifacts')
      if artifacts_list:
        self._output_artifacts[name] = self._make_output_artifact(
            artifacts_list[0])

  @classmethod
  def _make_input_artifact(self, runtime_artifact: Dict):
    artifact = create_runtime_artifact(runtime_artifact)
    return InputArtifact(artifact_type=type(artifact), artifact=artifact)

  @classmethod
  def _make_output_artifact(self, runtime_artifact: Dict):
    artifact = create_runtime_artifact(runtime_artifact)
    return OutputArtifact(artifact_type=type(artifact), artifact=artifact)

  def _get_input_artifact(self, name: str):
    return self._input_artifacts.get(name)

  def _get_output_artifact(self, name: str):
    return self._output_artifacts.get(name)

  def _get_input_parameter_value(self, parameter_name: str):
    parameter = self._input.get('inputs', {}).get('parameters',
                                                  {}).get(parameter_name, None)
    if parameter is None:
      return None

    if parameter.get('stringValue'):
      return parameter['stringValue']
    elif parameter.get('intValue'):
      return int(parameter['intValue'])
    elif parameter.get('doubleValue'):
      return float(parameter['doubleValue'])

  def _get_output_parameter_path(self, parameter_name: str):
    parameter_name = self._maybe_strip_path_suffix(parameter_name)
    parameter = self._input.get('outputs',
                                {}).get('parameters',
                                        {}).get(parameter_name, None)
    if parameter is None:
      return None

    return parameter.get('outputFile', None)

  def _get_output_artifact_path(self, artifact_name: str):
    artifact_name = self._maybe_strip_path_suffix(artifact_name)
    output_artifact = self._output_artifacts.get(artifact_name)
    if not output_artifact:
      raise ValueError(
          'Failed to get OutputArtifact path for artifact name {}'.format(
              artifact_name))
    return output_artifact.path

  def _get_input_artifact_path(self, artifact_name: str):
    artifact_name = self._maybe_strip_path_suffix(artifact_name)
    input_artifact = self._input_artifacts.get(artifact_name)
    if not input_artifact:
      raise ValueError(
          'Failed to get InputArtifact path for artifact name {}'.format(
              artifact_name))
    return input_artifact.path

  def _write_executor_output(self):
    executor_output = {'artifacts': {}}

    for name, artifact in self._output_artifacts.items():
      runtime_artifact = {
          "name": artifact.get().name,
          "uri": artifact.get().uri,
          "metadata": artifact.get().metadata,
      }
      artifacts_list = {'artifacts': [runtime_artifact]}

      executor_output['artifacts'][name] = artifacts_list

    with open(self._input['outputs']['outputFile'], 'w') as f:
      f.write(json.dumps(executor_output))

  def _maybe_strip_path_suffix(self, name) -> str:
    if name.endswith('_path'):
      name = name[0:-len('_path')]
    if name.endswith('_file'):
      name = name[0:-len('_file')]
    return name

  def execute(self):
    annotations = inspect.getfullargspec(self._func).annotations

    # Function arguments.
    func_kwargs = {}

    for k, v in annotations.items():
      if v in [str, float, int]:
        func_kwargs[k] = self._get_input_parameter_value(k)

      if isinstance(v, OutputPath):
        if v.type in [str, float, int]:
          func_kwargs[k] = self._get_output_parameter_path(k)
        else:
          func_kwargs[k] = self._get_output_artifact_path(k)

      if isinstance(v, InputPath):
        func_kwargs[k] = self._get_input_artifact_path(k)

      if isinstance(v, InputArtifact):
        func_kwargs[k] = self._get_input_artifact(k)

      if isinstance(v, OutputArtifact):
        func_kwargs[k] = self._get_output_artifact(k)

    # TODO(neuromage): Support returned NamedTuple parameters.
    self._func(**func_kwargs)
    self._write_executor_output()
