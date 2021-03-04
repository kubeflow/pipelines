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
"""MLMD artifact ontology in KFP SDK."""

import os
import yaml

from kfp.dsl import artifact
from kfp.pipeline_spec import pipeline_spec_pb2

class Model(artifact.Artifact):

  TYPE_NAME = 'kfp.Model'

  def __init__(self):
    self._instance_schema = self.__class__.get_artifact_type()

    # Calling base class init to setup the instance.
    super().__init__()

  @classmethod
  def get_artifact_type(cls) -> str:
    schema_file_path=os.path.join(
      os.path.dirname(__file__), 'type_schemas', 'model.yaml')

    with open(schema_file_path) as schema_file:
      return schema_file.read()

  @classmethod
  def get_ir_type(cls) -> pipeline_spec_pb2.ArtifactTypeSchema:
    return pipeline_spec_pb2.ArtifactTypeSchema(
        instance_schema=cls.get_artifact_type())

class Dataset(artifact.Artifact):

  TYPE_NAME = 'kfp.Dataset'

  def __init__(self):
    self._instance_schema = self.__class__.get_artifact_type()

    # Calling base class init to setup the instance.
    super().__init__()

  @classmethod
  def get_artifact_type(cls) -> str:
    schema_file_path=os.path.join(
          os.path.dirname(__file__), 'type_schemas', 'dataset.yaml')

    with open(schema_file_path) as schema_file:
      return schema_file.read()

  @classmethod
  def get_ir_type(cls) -> pipeline_spec_pb2.ArtifactTypeSchema:
    return pipeline_spec_pb2.ArtifactTypeSchema(
        instance_schema=cls.get_artifact_type())
