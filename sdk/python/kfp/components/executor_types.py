import json
import inspect
from typing import Callable, Dict
from kfp.components._python_op import RuntimeArtifact, InputArtifact, OutputArtifact, InputPath, OutputPath


class Executor():

  def __init__(self, executor_input: Dict, func: Callable):
    self._func = func
    self._input = executor_input
    self._input_artifacts = {}
    self._output_artifacts = {}

    for name, artifacts in self._input.get('inputs', {}).get('artifacts',
                                                             {}).items():
      artifacts_list = artifacts.get('artifacts')
      if artifacts_list:
        self._input_artifacts[name] = RuntimeArtifact(artifacts_list[0])

    for name, artifacts in self._input.get('outputs', {}).get('artifacts',
                                                              {}).items():
      artifacts_list = artifacts.get('artifacts')
      if artifacts_list:
        self._output_artifacts[name] = RuntimeArtifact(artifacts_list[0])

  def _get_input_artifact(self, name: str):
    return self._input_artifacts.get(name)

  def _get_output_artifact(self, name: str):
    return self._output_artifacts.get(name)

  def _get_input_parameter_value(self, parameter_name: str):
    parameter = self._input.get('inputs', {}).get('parameters',
                                                  {}).get(parameter_name, None)
    if parameter is None:
      return None  # or default?

    if parameter.get('stringValue'):
      return parameter['stringValue']
    elif parameter.get('intValue'):
      return parameter['intValue']
    elif parameter.get('doubleValue'):
      return parameter['doubleValue']

  def _get_output_parameter_path(self, parameter_name: str):
    parameter = self._input.get('outputs',
                                {}).get('parameters',
                                        {}).get(parameter_name, None)
    if parameter is None:
      return None  # or default?

    return parameter.get('outputFile', None)

  def _write_executor_output(self):
    executor_output = {'artifacts': {}}

    for name, artifact in self._output_artifacts.items():
      runtime_artifact = {
          "name": artifact.name,
          "uri": artifact.uri,
          "metadata": artifact.metadata,
      }
      artifacts_list = {'artifacts': [runtime_artifact]}

      executor_output['artifacts'][name] = artifacts_list

    with open(self._input['outputs']['outputFile'], 'w') as f:
      f.write(json.dumps(executor_output))

  def execute(self):
    annotations = inspect.getfullargspec(self._func).annotations

    # Function arguments.
    func_kwargs = {}

    for k, v in annotations.items():
      if v in [str, float, int]:
        func_kwargs[k] = self._get_input_parameter_value(k)

      if isinstance(v, OutputPath):
        func_kwargs[k] = self._get_output_parameter_path(k)

      if isinstance(v, InputArtifact):
        func_kwargs[k] = self._get_input_artifact(k)

      if isinstance(v, OutputArtifact):
        func_kwargs[k] = self._get_output_artifact(k)

    print(func_kwargs)
    self._func(**func_kwargs)
    self._write_executor_output()
