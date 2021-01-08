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

from absl import logging
import importlib
import tensorflow as tf
from typing import Callable, Dict
from google.protobuf import json_format

from kfp.components import _python_op
from kfp.pipeline_spec import pipeline_spec_pb2
from kfp.dsl import artifact



def get_parameter_from_output(file_path: str, param_name: str):
  """Gets a parameter value by its name from output metadata JSON."""
  output = pipeline_spec_pb2.ExecutorOutput()
  json_format.Parse(
      text=tf.io.gfile.GFile(file_path, 'r').read(),
      message=output)
  value = output.parameters[param_name]
  return getattr(value, value.Whichone('value'))


def get_artifact_from_output(
    file_path: str, output_name: str) -> artifact.Artifact:
  """Gets an artifact object from output metadata JSON."""
  output = pipeline_spec_pb2.ExecutorOutput()
  json_format.Parse(
      text=tf.io.gfile.GFile(file_path, 'r').read(),
      message=output
  )
  # Currently we bear the assumption that each output contains only one artifact
  json_str = json_format.MessageToJson(
      output.artifacts[output_name][0], sort_keys=True)

  # Convert runtime_artifact to Python artifact
  return artifact.Artifact.deserialize(json_str)


def import_func_from_source(source_path: str, fn_name: str) -> Callable:
  """Imports a function from a Python file."""
  # TODO(numerology): Implement this.
  pass


def get_output_artifacts(
    fn: Callable, output_uris: Dict[str, str]) -> Dict[str, artifact.Artifact]:
  """Gets the output artifacts from function signature and provided URIs.

  Args:
    fn: A user-provided function, whose signature annotates the type of output
      artifacts.
    output_uris: The mapping from output artifact name to its URI.

  Returns:
    A mapping from output artifact name to Python artifact objects.
  """
  # Inspect the function signature to determine the set of output artifact.
  spec = _python_op._extract_component_interface(fn)

  result = {} # Mapping from output name to artifacts.
  for output in spec.outputs:
    if (getattr(output, '_passing_style', None) == _python_op.OutputArtifact):
      # Creates an artifact according to its name
      type_name = getattr(output, 'type', None)
      if not type_name:
        continue

      try:
        artifact_cls = getattr(
            importlib.import_module(artifact.KFP_ARTIFACT_ONTOLOGY_MODULE),
            type_name)

      except (AttributeError, ImportError, ValueError):
        logging.warning((
            'Could not load artifact class %s.%s; using fallback deserialization'
            ' for the relevant artifact. Please make sure that any artifact '
            'classes can be imported within your container or environment.'),
            artifact.KFP_ARTIFACT_ONTOLOGY_MODULE, type_name)
        artifact_cls = artifact.Artifact

      art = artifact_cls()
      art.uri = output_uris[output.name]
      result[output.name] = art

  return result
