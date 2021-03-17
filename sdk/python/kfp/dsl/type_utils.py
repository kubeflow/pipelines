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

from typing import Dict, List, Optional, Type, Union
from kfp.components import structures
from kfp.pipeline_spec import pipeline_spec_pb2
from kfp.dsl import artifact
from kfp.dsl import ontology_artifacts

# ComponentSpec I/O types to (IR) PipelineTaskSpec I/O types mapping.
# The keys are normalized (lowercased). These are types viewed as Artifacts.
# The values are the corresponding IR artifact ontology types.
_ARTIFACT_TYPES_MAPPING = {
    'model':
        ontology_artifacts.Model.get_artifact_type(),
    'dataset':
        ontology_artifacts.Dataset.get_artifact_type(),
    'metrics':
        ontology_artifacts.Metrics.get_artifact_type(),
    'classificationmetrics':
        ontology_artifacts.ClassificationMetrics.get_artifact_type(),
    'slicedclassificationmetrics':
        ontology_artifacts.SlicedClassificationMetrics.get_artifact_type(),
}

# ComponentSpec I/O types to DSL ontology artifact classes mapping.
_ARTIFACT_CLASSES_MAPPING = {
    'model':
        ontology_artifacts.Model,
    'dataset':
        ontology_artifacts.Dataset,
    'metrics':
        ontology_artifacts.Metrics,
    'classificationmetrics':
        ontology_artifacts.ClassificationMetrics,
    'slicedclassificationmetrics':
        ontology_artifacts.SlicedClassificationMetrics,
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
    'text': pipeline_spec_pb2.PrimitiveType.STRING,
}

# Mapping primitive types to their IR message field names.
# This is used in constructing condition strings.
_PARAMETER_TYPES_VALUE_REFERENCE_MAPPING = {
    pipeline_spec_pb2.PrimitiveType.INT: 'int_value',
    pipeline_spec_pb2.PrimitiveType.DOUBLE: 'double_value',
    pipeline_spec_pb2.PrimitiveType.STRING: 'string_value',
}


def is_parameter_type(type_name: Optional[str]) -> bool:
  """Check if a ComponentSpec I/O type is considered as a parameter type.

  Args:
    type_name: type name of the ComponentSpec I/O type.

  Returns:
    True if the type name maps to a parameter type else False.
  """
  if isinstance(type_name, str):
    return type_name.lower() in _PARAMETER_TYPES_MAPPING
  else:
    return False


def get_artifact_type_schema(type_name: Union[str, Dict, List]) -> str:
  """Gets the IR I/O artifact type for the given ComponentSpec I/O type.

  Args:
    type_name: type name of the ComponentSpec I/O type.

  Returns:
     The string value of artifact type schema. Defaults to generic artifact.
  """
  if isinstance(type_name, str):
    return _ARTIFACT_TYPES_MAPPING.get(type_name.lower(),
                                       artifact.Artifact.get_artifact_type())
  else:
    return artifact.Artifact.get_artifact_type()


def get_artifact_type_schema_message(
    type_name: str) -> pipeline_spec_pb2.ArtifactTypeSchema:
  """Gets the IR I/O artifact type msg for the given ComponentSpec I/O type."""
  if isinstance(type_name, str):
    return _ARTIFACT_CLASSES_MAPPING.get(type_name.lower(),
                                         artifact.Artifact).get_ir_type()
  else:
    return artifact.Artifact.get_ir_type()


def get_parameter_type(
    param_type: Optional[Union[Type, str]]) -> pipeline_spec_pb2.PrimitiveType:
  """Get the IR I/O parameter type for the given ComponentSpec I/O type.

  Args:
    param_type: type of the ComponentSpec I/O type. Can be a primitive Python
      builtin type or a type name.

  Returns:
    The enum value of the mapped IR I/O primitive type.

  Raises:
    AttributeError: if type_name is not a string type.
  """
  if type(param_type) == type:
    if param_type not in (str, float, int):
      raise TypeError('Got illegal parameter type. Currently only support: '
                      'str, int and float. Got %s' % param_type)
    param_type = param_type.__name__
  return _PARAMETER_TYPES_MAPPING.get(param_type.lower())


def get_parameter_type_field_name(type_name: Optional[str]) -> str:
  """Get the IR field name for the given primitive type.

  For example: 'str' -> 'string_value', 'double' -> 'double_value', etc.

  Args:
    type_name: type name of the ComponentSpec I/O primitive type.

  Returns:
    The IR value reference field name.

  Raises:
    AttributeError: if type_name is not a string type.
  """
  return _PARAMETER_TYPES_VALUE_REFERENCE_MAPPING.get(
      get_parameter_type(type_name))


def get_input_artifact_type_schema(
    input_name: str,
    inputs: List[structures.InputSpec],
) -> Optional[str]:
  """Find the input artifact type by input name.

  Args:
    input_name: The name of the component input.
    inputs: The list of InputSpec

  Returns:
    The artifact type schema of the input.

  Raises:
    AssertionError if input not found, or input found but not an artifact type.
  """
  for component_input in inputs:
    if component_input.name == input_name:
      assert not is_parameter_type(
          component_input.type), 'Input is not an artifact type.'
      return get_artifact_type_schema(component_input.type)
  assert False, 'Input not found.'
