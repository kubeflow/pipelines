# pytype: skip-file
# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Module for creating pipeline components based on AI Platform SDK."""

import collections
import inspect
import json
from typing import Any, Callable, Dict, Optional, Sequence, Union

from google.cloud import aiplatform
from google.cloud import aiplatform_v1beta1

PROTO_PLUS_CLASS_TYPES = {
    aiplatform_v1beta1.types.explanation_metadata.ExplanationMetadata: (
        'ExplanationMetadata'
    ),
    aiplatform_v1beta1.types.explanation.ExplanationParameters: (
        'ExplanationParameters'
    ),
}


def get_forward_reference(
    annotation: Any,
) -> Optional[aiplatform.base.VertexAiResourceNoun]:
  """Resolves forward references to AiPlatform Class."""

  def get_aiplatform_class_by_name(_annotation):
    """Resolves str annotation to AiPlatfrom Class."""
    if isinstance(_annotation, str):
      return getattr(aiplatform, _annotation, None)

  ai_platform_class = get_aiplatform_class_by_name(annotation)
  if ai_platform_class:
    return ai_platform_class

  try:
    # Python 3.7+
    from typing import ForwardRef

    if isinstance(annotation, ForwardRef):
      annotation = annotation.__forward_arg__
      ai_platform_class = get_aiplatform_class_by_name(annotation)
      if ai_platform_class:
        return ai_platform_class

  except ImportError:
    pass


# This is the Union of all typed datasets.
# Relying on the annotation defined in the SDK
# as additional typed Datasets may be added in the future.
dataset_annotation = (
    inspect.signature(aiplatform.CustomTrainingJob.run)
    .parameters['dataset']
    .annotation
)


def resolve_annotation(annotation: Any) -> Any:
  """Resolves annotation type against a MB SDK type.

  Use this for Optional, Union, Forward References

  Args:
      annotation: Annotation to resolve

  Returns:
      Direct annotation
  """

  # handle forward reference string

  # if this is an Ai Platform resource noun
  if inspect.isclass(annotation):
    if issubclass(annotation, aiplatform.base.VertexAiResourceNoun):
      return annotation

  # if this is a union of all typed datasets annotation
  if annotation is dataset_annotation:
    # return the catch all Dataset class
    return aiplatform.datasets.dataset._Dataset

  # handle forward references
  resolved_annotation = get_forward_reference(annotation)
  if resolved_annotation:
    return resolved_annotation

  # handle optional types
  if getattr(annotation, '__origin__', None) is Union:
    # assume optional type
    # TODO check for optional type
    resolved_annotation = get_forward_reference(annotation.__args__[0])
    if resolved_annotation:
      return resolved_annotation
    else:
      return annotation.__args__[0]

  if annotation is inspect._empty:
    return None

  return annotation


def is_serializable_to_json(annotation: Any) -> bool:
  """Checks if the type is serializable.

  Args:
      annotation: parameter annotation

  Returns:
      True if serializable to json.
  """
  serializable_types = (dict, list, collections.abc.Sequence, Dict, Sequence)
  return getattr(annotation, '__origin__', None) in serializable_types


def is_mb_sdk_resource_noun_type(mb_sdk_type: Any) -> bool:
  """Determines if type passed in should be a metadata type.

  Args:
      mb_sdk_type: Type to check

  Returns:
      True if this is a resource noun
  """
  if inspect.isclass(mb_sdk_type):
    return issubclass(mb_sdk_type, aiplatform.base.VertexAiResourceNoun)
  return False


def get_proto_plus_class(annotation: Any) -> Optional[Callable]:
  """Get Proto Plus Class for this annotation.

  Args:
      annotation: parameter annotation

  Returns:
      Proto Plus Class for annotation type
  """
  if annotation in PROTO_PLUS_CLASS_TYPES:
    return annotation


def get_proto_plus_deserializer(
    annotation: Any,
) -> Optional[Callable[..., str]]:
  """Get deserializer for objects to pass them as strings.

  Remote runner will deserialize.

  Args:
      annotation: parameter annotation

  Returns:
      deserializer for annotation type if it's a Proto Plus class
  """
  proto_plus_class = get_proto_plus_class(annotation)
  if proto_plus_class:
    return proto_plus_class.from_json


def get_deserializer(annotation: Any) -> Optional[Callable[..., str]]:
  """Get deserializer for objects to pass them as strings.

  Remote runner will deserialize.

  Args:
      annotation: parameter annotation

  Returns:
      deserializer for annotation type
  """
  proto_plus_deserializer = get_proto_plus_deserializer(annotation)
  if proto_plus_deserializer:
    return proto_plus_deserializer

  if is_serializable_to_json(annotation):
    return json.loads
