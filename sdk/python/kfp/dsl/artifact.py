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
"""Base class for MLMD artifact ontology in KFP SDK."""
from typing import Any, Dict, Optional

from absl import logging
import enum
import importlib
from google.protobuf import json_format
import yaml

from kfp.pipeline_spec import pipeline_spec_pb2
from kfp.dsl import serialization_utils

_KFP_ARTIFACT_TITLE_PATTERN = 'kfp.{}'
KFP_ARTIFACT_ONTOLOGY_MODULE = 'kfp.dsl.ontology_artifacts'
DEFAULT_ARTIFACT_SCHEMA = 'title: kfp.Artifact\ntype: object\nproperties:\n'


# Enum for property types.
# This is introduced to decouple the MLMD ontology with Python built-in types.
class PropertyType(enum.Enum):
  INT = 1
  DOUBLE = 2
  STRING = 3


class Property(object):
  """Property specified for an Artifact."""

  # Mapping from Python enum to primitive type in the IR proto.
  _ALLOWED_PROPERTY_TYPES = {
      PropertyType.INT: pipeline_spec_pb2.PrimitiveType.INT,
      PropertyType.DOUBLE: pipeline_spec_pb2.PrimitiveType.DOUBLE,
      PropertyType.STRING: pipeline_spec_pb2.PrimitiveType.STRING,
  }

  def __init__(self, type: PropertyType, description: Optional[str] = None):
    if type not in Property._ALLOWED_PROPERTY_TYPES:
      raise ValueError('Property type must be one of %s.' %
                       list(Property._ALLOWED_PROPERTY_TYPES.keys()))
    self.type = type
    self.description = description

  @classmethod
  def from_dict(cls, dict_data: Dict[str, Any]) -> 'Property':
    """Deserializes the Property object from YAML dict."""
    if not dict_data.get('type'):
      raise TypeError('Missing type keyword in property dict.')
    if dict_data['type'] == 'string':
      kind = PropertyType.STRING
    elif dict_data['type'] == 'int':
      kind = PropertyType.INT
    elif dict_data['type'] == 'double':
      kind = PropertyType.DOUBLE
    else:
      raise TypeError('Got unknown type: %s' % dict_data['type'])

    return Property(
        type=kind,
        description=dict_data['description']
    )

  def get_ir_type(self):
    """Gets the IR primitive type."""
    return Property._ALLOWED_PROPERTY_TYPES[self.type]

  def get_type_name(self):
    """Gets the type name used in YAML instance."""
    if self.type == PropertyType.INT:
      return 'int'
    elif self.type == PropertyType.DOUBLE:
      return 'double'
    elif self.type == PropertyType.STRING:
      return 'string'
    else:
      raise TypeError('Unexpected property type: %s' % self.type)


class Artifact(object):
  """KFP Artifact Python class.

  Artifact Python class/object mainly serves following purposes in different
  period of its lifecycle.

  1. During compile time, users can use Artifact class to annotate I/O types of
     their components.
  2. At runtime, Artifact objects provide helper function/utilities to access
     the underlying RuntimeArtifact pb message, and provide additional layers
     of validation to ensure type compatibility.
  """

  # Name of the Artifact type.
  TYPE_NAME = 'Artifact'
  # Property schema.
  # Example usage:
  #
  # PROPERTIES = {
  #   'span': Property(type=PropertyType.INT),
  #   # Comma separated of splits for an artifact. Empty string means artifact
  #   # has no split.
  #   'split_names': Property(type=PropertyType.STRING),
  # }
  PROPERTIES = None

  # Initialization flag to support setattr / getattr behavior.
  _initialized = False

  def __init__(self, instance_schema: Optional[str] = None):
    """Constructs an instance of Artifact"""
    if self.__class__ == Artifact:
      if not instance_schema:
        raise ValueError(
            'The "instance_schema" argument must be set.')
      schema_yaml = yaml.safe_load(instance_schema)
      if 'properties' not in schema_yaml:
        raise ValueError('Invalid instance_schema, properties must be present. '
                         'Got %s' % instance_schema)
      schema = schema_yaml['properties'] or {}
      if 'title' not in schema_yaml:
        raise ValueError('Invalid instance_schema, title must be present. '
                         'Got %s' % instance_schema)
      self.TYPE_NAME = schema_yaml['title']
      self.PROPERTIES = {}
      for k, v in schema.items():
        self.PROPERTIES[k] = Property.from_dict(v)
    else:
      if instance_schema:
        raise ValueError(
            'The "instance_schema" argument must not be passed for '
            'Artifact subclass %s.' % self.__class__)
      instance_schema = self.get_artifact_type()

    # MLMD artifact type schema string.
    self._type_schema = instance_schema
    # Instantiate a RuntimeArtifact pb message as the POD data structure.
    self._artifact = pipeline_spec_pb2.RuntimeArtifact()
    self._artifact.type.CopyFrom(pipeline_spec_pb2.ArtifactTypeSchema(
        instance_schema=instance_schema
    ))
    # Initialization flag to prevent recursive getattr / setattr errors.
    self._initialized = True

  @classmethod
  def get_artifact_type(cls) -> str:
    """Gets the instance_schema according to the Python schema spec."""
    title = _KFP_ARTIFACT_TITLE_PATTERN.format(cls.TYPE_NAME)
    schema_map = None
    if cls.PROPERTIES:
      schema_map = {}
      for k, v in cls.PROPERTIES.items():
        schema_map[k] = {
            'type': v.get_type_name(),
            'description': v.description
        }
    result_map = {
        'title': title,
        'type': 'object',
        'properties': schema_map
    }
    return serialization_utils.yaml_dump(result_map)

  @property
  def type_schema(self) -> str:
    """Gets the instance_schema specified for this Artifact object."""
    return self._type_schema

  def __repr__(self) -> str:
    return 'Artifact(artifact: {}, type_schema: {})'.format(
        str(self._artifact), str(self.type_schema))

  def __getattr__(self, name: str) -> Any:
    """Custom __getattr__ to allow access to artifact properties."""
    if name == '_artifact_type':
      # Prevent infinite recursion when used with copy.deepcopy().
      raise AttributeError()
    properties = self.PROPERTIES or {}
    if name not in properties:
      raise AttributeError(
          '%s artifact has no property %r.' % (self.TYPE_NAME, name))
    property_type = properties[name].type
    if property_type == PropertyType.STRING:
      if name not in self._artifact.properties:
        # Avoid populating empty property protobuf with the [] operator.
        return ''
      return self._artifact.properties[name].string_value
    elif property_type == PropertyType.INT:
      if name not in self._artifact.properties:
        # Avoid populating empty property protobuf with the [] operator.
        return 0
      return self._artifact.properties[name].int_value
    elif property_type == PropertyType.DOUBLE:
      if name not in self._artifact.properties:
        # Avoid populating empty property protobuf with the [] operator.
        return 0.0
      return self._artifact.properties[name].double_value
    else:
      raise Exception('Unknown MLMD type %r for property %r.' %
                      (property_type, name))

  def __setattr__(self, name: str, value: Any):
    """Custom __setattr__ to allow access to artifact properties."""
    if not self._initialized:
      object.__setattr__(self, name, value)
      return
    properties = self.PROPERTIES or {}
    if name not in properties:
      if (name in self.__dict__ or
          any(name in c.__dict__ for c in self.__class__.mro())):
        # Use any provided getter / setter if available.
        object.__setattr__(self, name, value)
        return
      # In the case where we do not handle this via an explicit getter /
      # setter, we assume that the user implied an artifact attribute store,
      # and we raise an exception since such an attribute was not explicitly
      # defined in the Artifact PROPERTIES dictionary.
      raise AttributeError('Cannot set unknown property %r on artifact %r.' %
                           (name, self))
    property_type = properties[name].type
    if property_type == PropertyType.STRING:
      if not isinstance(value, str):
        raise Exception(
            'Expected string value for property %r; got %r instead.' %
            (name, value))
      self._artifact.properties[name].string_value = value
    elif property_type == PropertyType.INT:
      if not isinstance(value, int):
        raise Exception(
            'Expected integer value for property %r; got %r instead.' %
            (name, value))
      self._artifact.properties[name].int_value = value
    elif property_type == PropertyType.DOUBLE:
      if not isinstance(value, float):
        raise Exception(
            'Expected integer value for property %r; got %r instead.' %
            (name, value))
      self._artifact.properties[name].double_value = value
    else:
      raise Exception('Unknown property type %r for property %r.' %
                      (property_type, name))

  @property
  def type(self):
    return self.__class__

  @property
  def type_name(self):
    return self.TYPE_NAME

  @property
  def runtime_artifact(self) -> pipeline_spec_pb2.RuntimeArtifact:
    return self._artifact

  @runtime_artifact.setter
  def runtime_artifact(self, artifact: pipeline_spec_pb2.RuntimeArtifact):
    self._artifact = artifact

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

  # Custom property accessors.
  def set_string_custom_property(self, key: str, value: str):
    """Sets a custom property of string type."""
    self._artifact.custom_properties[key].string_value = value

  def set_int_custom_property(self, key: str, value: int):
    """Sets a custom property of int type."""
    self._artifact.custom_properties[key].int_value = value

  def set_float_custom_property(self, key: str, value: float):
    """Sets a custom property of float type."""
    self._artifact.custom_properties[key].double_value = value

  def has_custom_property(self, key: str) -> bool:
    return key in self._artifact.custom_properties

  def get_string_custom_property(self, key: str) -> str:
    """Gets a custom property of string type."""
    if key not in self._artifact.custom_properties:
      return ''
    return self._artifact.custom_properties[key].string_value

  def get_int_custom_property(self, key: str) -> int:
    """Gets a custom property of int type."""
    if key not in self._artifact.custom_properties:
      return 0
    return self._artifact.custom_properties[key].int_value

  def get_float_custom_property(self, key: str) -> float:
    """Gets a custom property of float type."""
    if key not in self._artifact.custom_properties:
      return 0.0
    return self._artifact.custom_properties[key].double_value

  @classmethod
  def deserialize(cls, data: str) -> Any:
    """Deserializes an Artifact object from JSON dict."""
    artifact = pipeline_spec_pb2.RuntimeArtifact()
    json_format.Parse(data, artifact, ignore_unknown_fields=True)
    instance_schema = yaml.safe_load(artifact.type.instance_schema)
    type_name = instance_schema['title'][len('kfp.'):]
    result = None
    try:
      artifact_cls = getattr(
          importlib.import_module(KFP_ARTIFACT_ONTOLOGY_MODULE), type_name)
      # TODO(numerology): Add deserialization tests for first party classes.
      result = artifact_cls()
    except (AttributeError, ImportError, ValueError):
      logging.warning((
          'Could not load artifact class %s.%s; using fallback deserialization '
          'for the relevant artifact. Please make sure that any artifact '
          'classes can be imported within your container or environment.'),
          KFP_ARTIFACT_ONTOLOGY_MODULE, type_name)
    if not result:
      # Otherwise generate a generic Artifact object.
      result = Artifact(instance_schema=artifact.type.instance_schema)
    result.runtime_artifact = artifact
    return result

  def serialize(self) -> str:
    """Serializes an Artifact to JSON dict format."""
    return json_format.MessageToJson(self._artifact, sort_keys=True)
