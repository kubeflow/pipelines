# Copyright 2020-2022 The Kubeflow Authors
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

from distutils import util
import inspect
import json
from typing import Any, Callable, Dict, Optional, Type, Union
import warnings

import kfp
from kfp.components import task_final_status
from kfp.components.types import artifact_types
from kfp.components.types import type_annotations
from kfp.pipeline_spec import pipeline_spec_pb2

DEFAULT_ARTIFACT_SCHEMA_VERSION = '0.0.1'
PARAMETER_TYPES = Union[str, int, float, bool, dict, list]

# ComponentSpec I/O types to DSL ontology artifact classes mapping.
_ARTIFACT_CLASSES_MAPPING = {
    'artifact': artifact_types.Artifact,
    'model': artifact_types.Model,
    'dataset': artifact_types.Dataset,
    'metrics': artifact_types.Metrics,
    'classificationmetrics': artifact_types.ClassificationMetrics,
    'slicedclassificationmetrics': artifact_types.SlicedClassificationMetrics,
    'html': artifact_types.HTML,
    'markdown': artifact_types.Markdown,
}

_GOOGLE_TYPES_PATTERN = r'^google.[A-Za-z]+$'
_GOOGLE_TYPES_VERSION = DEFAULT_ARTIFACT_SCHEMA_VERSION

# ComponentSpec I/O types to (IR) PipelineTaskSpec I/O types mapping.
# The keys are normalized (lowercased). These are types viewed as Parameters.
# The values are the corresponding IR parameter primitive types.
_PARAMETER_TYPES_MAPPING = {
    'integer': pipeline_spec_pb2.ParameterType.NUMBER_INTEGER,
    'int': pipeline_spec_pb2.ParameterType.NUMBER_INTEGER,
    'double': pipeline_spec_pb2.ParameterType.NUMBER_DOUBLE,
    'float': pipeline_spec_pb2.ParameterType.NUMBER_DOUBLE,
    'string': pipeline_spec_pb2.ParameterType.STRING,
    'str': pipeline_spec_pb2.ParameterType.STRING,
    'text': pipeline_spec_pb2.ParameterType.STRING,
    'bool': pipeline_spec_pb2.ParameterType.BOOLEAN,
    'boolean': pipeline_spec_pb2.ParameterType.BOOLEAN,
    'dict': pipeline_spec_pb2.ParameterType.STRUCT,
    'list': pipeline_spec_pb2.ParameterType.LIST,
    'jsonobject': pipeline_spec_pb2.ParameterType.STRUCT,
    'jsonarray': pipeline_spec_pb2.ParameterType.LIST,
}


def bool_cast_fn(default: Union[str, bool]) -> bool:
    if isinstance(default, str):
        default = util.strtobool(default) == 1
    return default


def try_loading_json(default: str) -> Union[dict, list, str]:
    try:
        return json.loads(default)
    except:
        return default


_V1_DEFAULT_DESERIALIZER_MAPPING: Dict[str, Callable] = {
    'integer': int,
    'int': int,
    'double': float,
    'float': float,
    'string': str,
    'str': str,
    'text': str,
    'bool': bool_cast_fn,
    'boolean': bool_cast_fn,
    'dict': try_loading_json,
    'list': try_loading_json,
    'jsonobject': try_loading_json,
    'jsonarray': try_loading_json,
}


def deserialize_v1_component_yaml_default(type_: str, default: Any) -> Any:
    """Deserializes v1 default values to correct in-memory types.

    Typecasts for primitive types. Tries to load JSON for arrays and
    structs.
    """
    if default is None:
        return default
    if isinstance(type_, str):
        cast_fn = _V1_DEFAULT_DESERIALIZER_MAPPING.get(type_.lower(),
                                                       lambda x: x)
        return cast_fn(default)
    return default


# Mapping primitive types to their IR message field names.
# This is used in constructing condition strings.
_PARAMETER_TYPES_VALUE_REFERENCE_MAPPING = {
    pipeline_spec_pb2.ParameterType.NUMBER_INTEGER: 'number_value',
    pipeline_spec_pb2.ParameterType.NUMBER_DOUBLE: 'number_value',
    pipeline_spec_pb2.ParameterType.STRING: 'string_value',
    pipeline_spec_pb2.ParameterType.BOOLEAN: 'bool_value',
    pipeline_spec_pb2.ParameterType.STRUCT: 'struct_value',
    pipeline_spec_pb2.ParameterType.LIST: 'list_value',
}


def is_task_final_status_type(type_name: Optional[Union[str, dict]]) -> bool:
    """Check if a ComponentSpec I/O type is PipelineTaskFinalStatus.

    Args:
      type_name: type name of the ComponentSpec I/O type.

    Returns:
      True if the type name is 'PipelineTaskFinalStatus'.
    """
    return isinstance(type_name, str) and (
        type_name == task_final_status.PipelineTaskFinalStatus.__name__)


def is_parameter_type(type_name: Optional[Union[str, dict]]) -> bool:
    """Check if a ComponentSpec I/O type is considered as a parameter type.

    Args:
      type_name: type name of the ComponentSpec I/O type.

    Returns:
      True if the type name maps to a parameter type else False.
    """
    if isinstance(type_name, str):
        type_name = type_annotations.get_short_type_name(type_name)
    elif isinstance(type_name, dict):
        type_name = list(type_name.keys())[0]
    else:
        return False

    return type_name.lower(
    ) in _PARAMETER_TYPES_MAPPING or is_task_final_status_type(type_name)


def bundled_artifact_to_artifact_proto(
        bundled_artifact_str: str) -> pipeline_spec_pb2.ArtifactTypeSchema:
    """Gets the IR ArtifactTypeSchema proto for a bundled artifact in form
    `<namespace>.<Name>@x.x.x` (e.g., system.Artifact@0.0.1)."""
    bundled_artifact_str, schema_version = bundled_artifact_str.split('@')
    return pipeline_spec_pb2.ArtifactTypeSchema(
        schema_title=bundled_artifact_str,
        schema_version=schema_version,
    )


def get_parameter_type(
    param_type: Optional[Union[Type, str, dict]]
) -> pipeline_spec_pb2.ParameterType:
    """Get the IR I/O parameter type for the given ComponentSpec I/O type.

    Args:
      param_type: type of the ComponentSpec I/O type. Can be a primitive Python
        builtin type or a type name.

    Returns:
      The enum value of the mapped IR I/O primitive type.

    Raises:
      AttributeError: if type_name is not a string type.
    """
    # Special handling for PipelineTaskFinalStatus, treat it as Dict type.
    if is_task_final_status_type(param_type):
        param_type = 'dict'
    if type(param_type) == type:
        type_name = param_type.__name__
    elif isinstance(param_type, dict):
        type_name = list(param_type.keys())[0]
    else:
        type_name = type_annotations.get_short_type_name(str(param_type))
    return _PARAMETER_TYPES_MAPPING.get(type_name.lower())


def get_parameter_type_name(
        param_type: Optional[Union[Type, str, dict]]) -> str:
    """Gets the parameter type name."""
    return pipeline_spec_pb2.ParameterType.ParameterTypeEnum.Name(
        get_parameter_type(param_type))


def get_parameter_type_field_name(type_name: Optional[str]) -> Optional[str]:
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


class InconsistentTypeException(Exception):
    """InconsistencyTypeException is raised when two types are not
    consistent."""


class InconsistentTypeWarning(Warning):
    """InconsistentTypeWarning is issued when two types are not consistent."""


def verify_type_compatibility(
    given_type: str,
    expected_type: str,
    error_message_prefix: str,
) -> bool:
    """Verifies the given argument type is compatible with the expected type.

    Args:
        given_type: The type of the argument passed to the input.
        expected_type: The declared type of the input.
        error_message_prefix: The prefix for the error message.

    Returns:
        True if types are compatible, and False if otherwise.

    Raises:
        InconsistentTypeException if types are incompatible and TYPE_CHECK==True.
    """

    types_are_compatible = False
    is_parameter = is_parameter_type(str(given_type))

    # handle parameters
    if is_parameter:
        # Normalize parameter type names.
        if is_parameter_type(given_type):
            given_type = get_parameter_type_name(given_type)
        if is_parameter_type(expected_type):
            expected_type = get_parameter_type_name(expected_type)

        types_are_compatible = check_parameter_type_compatibility(
            given_type, expected_type)
    else:
        # handle artifacts
        given_schema_title, given_schema_version = given_type.split('@')
        expected_schema_title, expected_schema_version = expected_type.split(
            '@')
        if artifact_types.Artifact.schema_title in {
                given_schema_title, expected_schema_title
        }:
            types_are_compatible = True
        else:
            schema_title_compatible = given_schema_title == expected_schema_title
            schema_version_compatible = given_schema_version.split(
                '.')[0] == expected_schema_version.split('.')[0]
            types_are_compatible = schema_title_compatible and schema_version_compatible

    # maybe raise, maybe warn, return bool
    if not types_are_compatible:
        error_text = error_message_prefix + f'Argument type "{given_type}" is incompatible with the input type "{expected_type}"'
        if kfp.TYPE_CHECK:
            raise InconsistentTypeException(error_text)
        else:
            warnings.warn(InconsistentTypeWarning(error_text))

    return types_are_compatible


def check_parameter_type_compatibility(
    given_type: Union[str, dict],
    expected_type: Union[str, dict],
):
    if isinstance(given_type, str):
        given_type = {given_type: {}}
    if isinstance(expected_type, str):
        expected_type = {expected_type: {}}
    return _check_dict_types(given_type, expected_type)


def _check_dict_types(
    given_type: dict,
    expected_type: dict,
):
    given_type_name, _ = list(given_type.items())[0]
    expected_type_name, _ = list(expected_type.items())[0]
    if given_type_name == '' or expected_type_name == '':
        # If the type name is empty, it matches any types
        return True
    if given_type_name != expected_type_name:
        print('type name ' + str(given_type_name) +
              ' is different from expected: ' + str(expected_type_name))
        return False
    type_name = given_type_name
    for type_property in given_type[type_name]:
        if type_property not in expected_type[type_name]:
            print(type_name + ' has a property ' + str(type_property) +
                  ' that the latter does not.')
            return False
        if given_type[type_name][type_property] != expected_type[type_name][
                type_property]:
            print(type_name + ' has a property ' + str(type_property) +
                  ' with value: ' + str(given_type[type_name][type_property]) +
                  ' and ' + str(expected_type[type_name][type_property]))
            return False
    return True


_TYPE_TO_TYPE_NAME = {
    str: 'String',
    int: 'Integer',
    float: 'Float',
    bool: 'Boolean',
    list: 'List',
    dict: 'Dict',
}


def get_canonical_type_name_for_type(typ: Type) -> Optional[str]:
    """Find the canonical type name for a given type.

    Args:
        typ: The type to search for.

    Returns:
        The canonical name of the type found.
    """
    return _TYPE_TO_TYPE_NAME.get(typ, None)


class TypeCheckManager:
    """Context manager to set a type check mode within context, then restore
    mode to original value upon exiting the context."""

    def __init__(self, enable: bool) -> None:
        """TypeCheckManager constructor.

        Args:
            enable: Type check mode used within context.
        """
        self._enable = enable

    def __enter__(self) -> 'TypeCheckManager':
        """Set type check mode to self._enable.

        Returns:
            TypeCheckManager: Returns itself.
        """
        self._prev = kfp.TYPE_CHECK
        kfp.TYPE_CHECK = self._enable
        return self

    def __exit__(self, *unused_args) -> None:
        """Restore type check mode to its previous state."""
        kfp.TYPE_CHECK = self._prev


# for reading in IR back to in-memory data structures
IR_TYPE_TO_IN_MEMORY_SPEC_TYPE = {
    'STRING': 'String',
    'NUMBER_INTEGER': 'Integer',
    'NUMBER_DOUBLE': 'Float',
    'LIST': 'List',
    'STRUCT': 'Dict',
    'BOOLEAN': 'Boolean',
}

IN_MEMORY_SPEC_TYPE_TO_IR_TYPE = {
    v: k for k, v in IR_TYPE_TO_IN_MEMORY_SPEC_TYPE.items()
}


def get_canonical_name_for_outer_generic(type_name: Any) -> str:
    """Maps a complex/nested type name back to a canonical type.

    E.g.
        get_canonical_name_for_outer_generic('typing.List[str]')
        'List'

        get_canonical_name_for_outer_generic('typing.Dict[typing.List[str], str]')
        'Dict'

    Args:
        type_name (Any): The type. Returns input if not a string.

    Returns:
        str: The canonical type.
    """
    if not isinstance(type_name, str) or not type_name.startswith('typing.'):
        return type_name

    return type_name.lstrip('typing.').split('[')[0]


def create_bundled_artifact_type(schema_title: str,
                                 schema_version: Optional[str] = None) -> str:
    if not isinstance(schema_title, str):
        raise ValueError
    return schema_title + '@' + (
        schema_version or DEFAULT_ARTIFACT_SCHEMA_VERSION)


def validate_schema_version(schema_version: str) -> None:
    split_schema_version = schema_version.split('.')
    if len(split_schema_version) != 3:
        raise TypeError(
            f'Artifact schema_version must use three-part semantic versioning. Got: {schema_version}'
        )


def validate_schema_title(schema_title: str) -> None:
    split_schema_title = schema_title.split('.')
    if len(split_schema_title) != 2:
        raise TypeError(
            f'Artifact schema_title must have both a namespace and a name, separated by a `.`. Got: {schema_title}'
        )
    namespace, _ = split_schema_title
    if namespace not in {'system', 'google'}:
        raise TypeError(
            f'Artifact schema_title must belong to `system` or `google` namespace. Got: {schema_title}'
        )


def validate_bundled_artifact_type(type_: str) -> None:
    split_type = type_.split('@')
    # two parts and neither are empty strings
    if len(split_type) != 2 or not all(split_type):
        raise TypeError(
            f'Artifacts must have both a schema_title and a schema_version, separated by `@`. Got: {type_}'
        )
    schema_title, schema_version = split_type
    validate_schema_title(schema_title)
    validate_schema_version(schema_version)


def _annotation_to_type_struct(annotation):
    if not annotation or annotation == inspect.Parameter.empty:
        return None
    if hasattr(annotation, 'to_dict'):
        annotation = annotation.to_dict()
    if isinstance(annotation, dict):
        return annotation
    if isinstance(annotation, type):
        type_struct = get_canonical_type_name_for_type(annotation)
        if type_struct:
            return type_struct
        elif type_annotations.is_artifact_class(annotation):
            schema_title = annotation.schema_title
        else:
            schema_title = str(annotation.__name__)
    elif hasattr(annotation, '__forward_arg__'):
        schema_title = str(annotation.__forward_arg__)
    else:
        schema_title = str(annotation)
    type_struct = get_canonical_type_name_for_type(schema_title)
    return type_struct or schema_title


def is_typed_named_tuple_annotation(annotation: Any) -> bool:
    return hasattr(annotation, '_fields') and hasattr(annotation,
                                                      '__annotations__')
