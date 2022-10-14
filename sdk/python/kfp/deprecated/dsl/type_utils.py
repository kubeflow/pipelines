# Copyright 2021 The Kubeflow Authors
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
"""Deprecated. See kfp.components.types.type_utils instead.

This module will be removed in KFP v2.0.
"""
import warnings
from kfp.components.types import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2
import inspect
from typing import Union, Type
import re
from typing import List, Optional
from kfp.deprecated.components import _structures
from kfp.components.types import artifact_types

warnings.warn(
    'Module kfp.dsl.type_utils is deprecated and will be removed'
    ' in KFP v2.0. Please use from kfp.components.types.type_utils instead.',
    category=FutureWarning)

is_parameter_type = type_utils.is_parameter_type
get_parameter_type = type_utils.get_parameter_type

# copying a lot of code from v2 to here to avoid certain dependencies of deprecated on v2, since making any changes to this code in v2 would result in breaks in deprecated/
_GOOGLE_TYPES_PATTERN = r'^google.[A-Za-z]+$'
_GOOGLE_TYPES_VERSION = '0.0.1'
_ARTIFACT_CLASSES_MAPPING = {
    'model': artifact_types.Model,
    'dataset': artifact_types.Dataset,
    'metrics': artifact_types.Metrics,
    'classificationmetrics': artifact_types.ClassificationMetrics,
    'slicedclassificationmetrics': artifact_types.SlicedClassificationMetrics,
    'html': artifact_types.HTML,
    'markdown': artifact_types.Markdown,
}


def get_artifact_type_schema(
    artifact_class_or_type_name: Optional[Union[str,
                                                Type[artifact_types.Artifact]]]
) -> pipeline_spec_pb2.ArtifactTypeSchema:
    """Gets the IR I/O artifact type msg for the given ComponentSpec I/O
    type."""
    artifact_class = artifact_types.Artifact
    if isinstance(artifact_class_or_type_name, str):
        if re.match(_GOOGLE_TYPES_PATTERN, artifact_class_or_type_name):
            return pipeline_spec_pb2.ArtifactTypeSchema(
                schema_title=artifact_class_or_type_name,
                schema_version=_GOOGLE_TYPES_VERSION,
            )
        artifact_class = _ARTIFACT_CLASSES_MAPPING.get(
            artifact_class_or_type_name.lower(), artifact_types.Artifact)
    elif inspect.isclass(artifact_class_or_type_name) and issubclass(
            artifact_class_or_type_name, artifact_types.Artifact):
        artifact_class = artifact_class_or_type_name

    return pipeline_spec_pb2.ArtifactTypeSchema(
        schema_title=artifact_class.schema_title,
        schema_version=artifact_class.schema_version)


def get_input_artifact_type_schema(
    input_name: str,
    inputs: List[_structures.InputSpec],
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
            return get_artifact_type_schema_old(component_input.type)
    assert False, 'Input not found.'
