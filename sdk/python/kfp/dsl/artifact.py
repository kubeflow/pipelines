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
"""Base class for MLMD artifact in KFP SDK."""

from typing import Any, Dict, Optional

from absl import logging
import enum
import importlib
import jsonschema
import yaml

from google.protobuf import json_format
from google.protobuf import struct_pb2
from kfp.pipeline_spec import pipeline_spec_pb2
from kfp.dsl import serialization_utils

KFP_ARTIFACT_ONTOLOGY_MODULE = 'kfp.dsl.ontology_artifacts'
DEFAULT_ARTIFACT_SCHEMA = 'title: kfp.Artifact\ntype: object\nproperties:\n'

class MetadataFieldType(enum.Enum):
  """Supported metadata field types."""
  NUMBER = 'number'
  STRING = 'string'
  BOOL = 'bool'
  OBJECT = 'object'
  ARRAY = 'array'


class Artifact(object):
  """KFP Artifact Python class.

  Artifact Python class/object mainly serves following purposes in different
  period of its lifecycle.

  1. During compile time, users can use Artifact class to annotate I/O types of
     their components.
  2. At runtime, Artifact objects provide helper function/utilities to access
     the underlying RuntimeArtifact pb message, and provide additional layers
     of validation to ensure type compatibility for fields specified in the 
     instance schema.
  """

  TYPE_NAME="kfp.Artifact"

  # Initialization flag to support setattr / getattr behavior.
  _initialized = False

  def _parse_instance_schema(self):
    """ Parses the instance schema of the Artifact.

    Ensures that schema is well-formed and sets up self._metadata_fields map
    for type-checking.

    Expects _instance_schema to be set.

    Raises:
      RuntimeError if _instance schema is not set.
      ValueError if title field is not set in instance_schema.
    """

    if not self._instance_schema:
      raise RuntimeError("instance_schema not set for the Artifact")

    schema = yaml.full_load(self._instance_schema)
    if 'title' not in schema.keys():
      raise ValueError('Invalid instance_schema, title must be present. Got: {}'.format(self._instance_schema))

    self.TYPE_NAME = schema['title']
    self._metadata_fields = {}
    if 'properties' in schema.keys():
      metadata_schema = schema['properties'] or {}
      for property_name, property_def in metadata_schema.items():
        try:
          self._metadata_fields[property_name] = MetadataFieldType(property_def['type'])
        except ValueError:
          raise ValueError('Unsupported type:{} specified for field: {} in instance schema'.format(property_def['type'], property_name))

  def __init__(self, instance_schema: Optional[str] = None):
    """Constructs an instance of Artifact.

    Setups up self._metadata_fields to perform type checking and initialize RuntimeArtifact.
    """
    if self.__class__ == Artifact:
      if not instance_schema:
        raise ValueError(
            'The "instance_schema" argument must be set for Artifact.')
      self._instance_schema = instance_schema
    else:
      if instance_schema:
        raise ValueError(
            'The "instance_schema" argument must not be passed for Artifact subclass: {}'.format(self.__class__))

    # setup self._metadata_fields
    self._parse_instance_schema()

    # Instantiate a RuntimeArtifact pb message as the POD data structure.
    self._artifact = pipeline_spec_pb2.RuntimeArtifact()

    # Stores the metadata for the Artifact.
    self.metadata = {}

    self._artifact.type.CopyFrom(pipeline_spec_pb2.ArtifactTypeSchema(
        instance_schema=self._instance_schema
    ))

    self._initialized = True

  @property
  def type_schema(self) -> str:
    """Gets the instance_schema for this Artifact object."""
    return self._instance_schema

  def __getattr__(self, name: str) -> Any:
    """Custom __getattr__ to allow access to artifact metadata."""

    if name not in self._metadata_fields:
      raise AttributeError(
          'No metadata field: {} in artifact.'.format(name))

    return self.metadata[name]

  def __setattr__(self, name: str, value: Any):
    """Custom __setattr__ to allow access to artifact metadata."""

    if not self._initialized:
      object.__setattr__(self, name, value)
      return
    
    metadata_fields = {}
    if self._metadata_fields:
      metadata_fields = self._metadata_fields

    if name not in self._metadata_fields:
      if (name in self.__dict__ or
          any(name in c.__dict__ for c in self.__class__.mro())):
        # Use any provided getter / setter if available.
        object.__setattr__(self, name, value)
        return
      # In the case where we do not handle this via an explicit getter /
      # setter, we assume that the user implied an artifact attribute store,
      # and we raise an exception since such an attribute was not explicitly
      # defined in the Artifact PROPERTIES dictionary.
      raise AttributeError('Cannot set an unspecified metadata field:{} on artifact. Only fields specified in instance schema can be set.'.format(name))

    field_type = metadata_fields[name]
    if field_type == MetadataFieldType.STRING:
      if not isinstance(value, str):
        raise RuntimeError('Expected string value for property: {} got:{} instead'.format(name, value))
      self.metadata[name] = value
    elif field_type == MetadataFieldType.NUMBER:
      if not isinstance(value, int) and not isinstance(value, float):
        raise RuntimeError('Expected number value for property: {} got:{} instead'.format(name, value))
      self.metadata[name] = value
    elif field_type == MetadataFieldType.BOOL:
      if not isinstance(value, bool):
        raise RuntimeError('Expected number value for property: {} got:{} instead'.format(name, value))
      self.metadata[name] = value
    elif field_type == MetadataFieldType.ARRAY:
      if not isinstance(value, list):
        raise RuntimeError('Expected number value for property: {} got:{} instead'.format(name, value))
      self.metadata[name] = value
    else:
      # Treating everything else as object.
      self.metadata[name] = value

  @property
  def type(self):
    return self.__class__

  @property
  def type_name(self):
    return self.TYPE_NAME

  @property
  def uri(self) -> str:
    return self._artifact.uri

  @uri.setter
  def uri(self, uri: str) -> None:
    self._artifact.uri = uri

  @property
  def name(self) -> str:
    return self._artifact.name

  @name.setter
  def name(self, name: str) -> None:
    self._artifact.name = name

  @property
  def runtime_artifact(self) -> pipeline_spec_pb2.RuntimeArtifact:
    if len(self.metadata) != 0:
      metadata_protobuf_struct = struct_pb2.Struct()
      metadata_protobuf_struct.update(self.metadata)
      self._artifact.metadata.CopyFrom(metadata_protobuf_struct)

    return self._artifact

  @runtime_artifact.setter
  def runtime_artifact(self, artifact: pipeline_spec_pb2.RuntimeArtifact):
    self._artifact = artifact

  def serialize(self) -> str:
    """Serializes an Artifact to JSON dict format."""

    if len(self._metadata_fields) != 0:
      try:
        jsonschema.validate(instance=self.metadata, schema=yaml.full_load(self._instance_schema))
      except jsonschema.exceptions.ValidationError:
        raise RuntimeError('Invalid instance schema set for artifact: {} instance schema: {}'.format(self.name, self._instance_schema))
      except jsonschema.exceptions.ValidationError:
        raise RuntimeError('metadata:{} type check failed for artifact: {} against instance schema: {}'.format(self.metadata, self.name, self._instance_schema))

      metadata_protobuf_struct = struct_pb2.Struct()
      metadata_protobuf_struct.update(self.metadata)
      self._artifact.metadata.CopyFrom(metadata_protobuf_struct)

    return json_format.MessageToJson(self._artifact, sort_keys=True)

  @classmethod
  def get_artifact_type(cls) -> str:
    """Gets the instance_schema according to the Python schema spec."""
    result_map = {
        'title': cls.TYPE_NAME,
        'type': 'object'
    }

    return serialization_utils.yaml_dump(result_map)

  @classmethod
  def get_from_runtime_artifact(
      cls, artifact: pipeline_spec_pb2.RuntimeArtifact) -> Any:
    """Deserializes an Artifact object from RuntimeArtifact message."""
    instance_schema = yaml.safe_load(artifact.type.instance_schema)
    type_name = instance_schema['title'][len('kfp.'):]
    result = None
    try:
      artifact_cls = getattr(
          importlib.import_module(KFP_ARTIFACT_ONTOLOGY_MODULE), type_name)
      result = artifact_cls()
    except (AttributeError, ImportError, ValueError) as err:
      logging.warning('Failed to instantiate Ontology Artifact:{} instance'.format(type_name))

    if not result:
      # Otherwise generate a generic Artifact object.
      result = Artifact(instance_schema=artifact.type.instance_schema)
    result.runtime_artifact = artifact
    result.metadata = json_format.MessageToDict(artifact.metadata)
    return result

  @classmethod
  def deserialize(cls, data: str) -> Any:
    """Deserializes an Artifact object from JSON dict."""
    artifact = pipeline_spec_pb2.RuntimeArtifact()
    json_format.Parse(data, artifact, ignore_unknown_fields=True)
    return cls.get_from_runtime_artifact(artifact)
