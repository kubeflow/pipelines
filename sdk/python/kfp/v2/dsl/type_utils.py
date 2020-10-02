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
"""Utilities for component I/O type mapping."""

from kfp.v2.proto import pipeline_spec_pb2

# ComponentSpec I/O types to (IR) PipelineTaskSpec I/O types mapping.
# The keys are normalized (lowercased). These are types viewed as Artifacts.
# The values are the corresponding IR artifact types.
_ARTIFACT_TYPES_MAPPING = {
    # TODO: support more artifact types
    'gcspath': 'mlpipeline.Artifact',
    'model': 'mlpipeline.Model',
}

# ComponentSpec I/O types to (IR) PipelineTaskSpec I/O types mapping.
# The keys are normalized (lowercased). These are types viewed as Parameters.
# The values are the corresponding IR parameter primitive types.
_PARAMETER_TYPES_MAPPING = {
    'integer': pipeline_spec_pb2.PrimitiveType.INT,
    'int': pipeline_spec_pb2.PrimitiveType.INT,
    'double': pipeline_spec_pb2.PrimitiveType.DOUBLE,
    'float': pipeline_spec_pb2.PrimitiveType.DOUBLE,
    'string': pipeline_spec_pb2.PrimitiveType.STRING,
    'str': pipeline_spec_pb2.PrimitiveType.STRING,
}


def is_artifact_type(type_name: str) -> bool:
  """Check if a ComponentSpec I/O type is considered as an artifact type.

  Args:
    type_name: type name of the ComponentSpec I/O type.

  Returns:
    True if the type name maps to an artifact type else False.

  Raises:
    AttributeError: if type_name os not a string type.
  """
  return type_name.lower() in _ARTIFACT_TYPES_MAPPING


def is_parameter_type(type_name: str) -> bool:
  """Check if a ComponentSpec I/O type is considered as a parameter type.

  Args:
    type_name: type name of the ComponentSpec I/O type.

  Returns:
    True if the type name maps to a parameter type else False.

  Raises:
    AttributeError: if type_name os not a string type.
  """
  return type_name.lower() in _PARAMETER_TYPES_MAPPING


def get_artifact_type(type_name: str) -> str:
  """Get the IR I/O artifact type for the given ComponentSpec I/O type.

  Args:
    type_name: type name of the ComponentSpec I/O type.

  Returns:
     The string value of artifact type.

  Raises:
    AttributeError: if type_name os not a string type.
    KeyError: if type_name doesn't match any predefined key.
  """
  return _ARTIFACT_TYPES_MAPPING.get(type_name.lower())


def get_parameter_type(type_name: str) -> pipeline_spec_pb2.PrimitiveType:
  """Get the IR I/O parameter type for the given ComponentSpec I/O type.

  Args:
    type_name: type name of the ComponentSpec I/O type.

  Returns:
    The enum value of the mapped IR I/O primitive type.

  Raises:
    AttributeError: if type_name os not a string type.
    KeyError: if type_name doesn't match any predefined key.
  """
  return _PARAMETER_TYPES_MAPPING.get(type_name.lower())
