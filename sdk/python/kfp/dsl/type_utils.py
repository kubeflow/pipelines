# Copyright 2020 The Kubeflow Authors
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
import inspect
from typing import Dict, List, Optional, Type, Union
from kfp.components import structures
from kfp.components import type_annotation_utils
from kfp.pipeline_spec import pipeline_spec_pb2
from kfp.dsl import artifact_utils
from kfp.dsl import io_types

# ComponentSpec I/O types to DSL ontology artifact classes mapping.
_ARTIFACT_CLASSES_MAPPING = {
    'model': io_types.Model,
    'dataset': io_types.Dataset,
    'metrics': io_types.Metrics,
    'classificationmetrics': io_types.ClassificationMetrics,
    'slicedclassificationmetrics': io_types.SlicedClassificationMetrics,
    'html': io_types.HTML,
    'markdown': io_types.Markdown,
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
    'bool': pipeline_spec_pb2.PrimitiveType.STRING,
    'boolean': pipeline_spec_pb2.PrimitiveType.STRING,
    'dict': pipeline_spec_pb2.PrimitiveType.STRING,
    'list': pipeline_spec_pb2.PrimitiveType.STRING,
    'jsonobject': pipeline_spec_pb2.PrimitiveType.STRING,
    'jsonarray': pipeline_spec_pb2.PrimitiveType.STRING,
}

# Mapping primitive types to their IR message field names.
# This is used in constructing condition strings.
_PARAMETER_TYPES_VALUE_REFERENCE_MAPPING = {
    pipeline_spec_pb2.PrimitiveType.INT: 'int_value',
    pipeline_spec_pb2.PrimitiveType.DOUBLE: 'double_value',
    pipeline_spec_pb2.PrimitiveType.STRING: 'string_value',
}


def is_parameter_type(type_name: Optional[Union[str, dict]]) -> bool:
  """Check if a ComponentSpec I/O type is considered as a parameter type.

  Args:
    type_name: type name of the ComponentSpec I/O type.

  Returns:
    True if the type name maps to a parameter type else False.
  """
  if isinstance(type_name, str):
    type_name = type_annotation_utils.get_short_type_name(type_name)
  elif isinstance(type_name, dict):
    type_name = list(type_name.keys())[0]
  else:
    return False

  return type_name.lower() in _PARAMETER_TYPES_MAPPING


def get_artifact_type_schema(
    artifact_class_or_type_name: Optional[Union[str, Type[io_types.Artifact]]]
) -> pipeline_spec_pb2.ArtifactTypeSchema:
  """Gets the IR I/O artifact type msg for the given ComponentSpec I/O type."""
  artifact_class = io_types.Artifact
  if isinstance(artifact_class_or_type_name, str):
    artifact_class = _ARTIFACT_CLASSES_MAPPING.get(
        artifact_class_or_type_name.lower(), io_types.Artifact)
  elif inspect.isclass(artifact_class_or_type_name) and issubclass(
      artifact_class_or_type_name, io_types.Artifact):
    artifact_class = artifact_class_or_type_name

  return pipeline_spec_pb2.ArtifactTypeSchema(
      schema_title=artifact_class.TYPE_NAME)


def get_parameter_type(
    param_type: Optional[Union[Type, str, dict]]
) -> pipeline_spec_pb2.PrimitiveType:
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
    type_name = param_type.__name__
  elif isinstance(param_type, dict):
    type_name = list(param_type.keys())[0]
  else:
    type_name = type_annotation_utils.get_short_type_name(str(param_type))
  return _PARAMETER_TYPES_MAPPING.get(type_name.lower())


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
