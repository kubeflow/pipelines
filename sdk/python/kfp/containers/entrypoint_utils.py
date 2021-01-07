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

import tensorflow as tf
from google.protobuf import json_format
from kfp.pipeline_spec import pipeline_spec_pb2


def get_parameter_from_output(file_path: str, param_name: str):
  """Gets a parameter value by its name from output metadata JSON."""
  output = pipeline_spec_pb2.ExecutorOutput()
  json_format.Parse(
      text=tf.io.gfile.GFile(file_path, 'r').read(),
      message=output)
  value = output.parameters[param_name]
  return getattr(value, value.Whichone('value'))

def get_artifact_from_output(file_path: str, output_name: str):
  """Gets an artifact object from output metadata JSON."""
  output = pipeline_spec_pb2.ExecutorOutput()
  json_format.Parse(
      text=tf.io.gfile.GFile(file_path, 'r').read(),
      message=output
  )
  # Currently we bear the assumption that each output contains only one artifact
  runtime_artifact = output.artifacts[output_name][0]

  # Convert runtime_artifact to Python artifact
  # TODO(numerology): Implement this

def get_artifact(runtime_artifact: pipeline_spec_pb2.RuntimeArtifact):
  pass
