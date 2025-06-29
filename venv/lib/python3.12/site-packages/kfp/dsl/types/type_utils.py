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

import inspect
import json
from typing import Any, Callable, Dict, Optional, Type, Union
import warnings

import kfp
from kfp.dsl import task_final_status
from kfp.dsl.types import artifact_types
from kfp.dsl.types import type_annotations

DEFAULT_ARTIFACT_SCHEMA_VERSION = '0.0.1'
PARAMETER_TYPES = Union[str, int, float, bool, dict, list]

# ComponentSpec I/O types to DSL ontology artifact classes mapping.
ARTIFACT_CLASSES_MAPPING = {
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

# pipeline_spec_pb2.ParameterType enum values
NUMBER_DOUBLE = 1
NUMBER_INTEGER = 2
STRING = 3
BOOLEAN = 4
LIST = 5
STRUCT = 6
TASK_FINAL_STATUS = 7
PARAMETER_TYPES_MAPPING = {
    'integer': NUMBER_INTEGER,
    'int': NUMBER_INTEGER,
    'double': NUMBER_DOUBLE,
    'float': NUMBER_DOUBLE,
    'string': STRING,
    'str': STRING,
    'text': STRING,
    'bool': BOOLEAN,
    'boolean': BOOLEAN,
    'dict': STRUCT,
    'list': LIST,
    'jsonobject': STRUCT,
    'jsonarray': LIST,
}


# copied from distutils.util, which was removed in Python 3.12
# https://github.com/pypa/distutils/blob/fb5c5704962cd3f40c69955437da9a88f4b28567/distutils/util.py#L340-L353
def strtobool(val):
    """Convert a string representation of truth to true (1) or false (0).

    True values are 'y', 'yes', 't', 'true', 'on', and '1'; false values
    are 'n', 'no', 'f', 'false', 'off', and '0'.  Raises ValueError if
    'val' is anything else.
    """
    val = val.lower()
    if val in ('y', 'yes', 't', 'true', 'on', '1'):
        return 1
    elif val in ('n', 'no', 'f', 'false', 'off', '0'):
        return 0
    else:
        raise ValueError('invalid truth value %r' % (val,))


def bool_cast_fn(default: Union[str, bool]) -> bool:
    if isinstance(default, str):
        default = strtobool(default) == 1
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
    ) in PARAMETER_TYPES_MAPPING or is_task_final_status_type(type_name)


def bundled_artifact_to_artifact_proto(
        bundled_artifact_str: str) -> 'pipeline_spec_pb2.ArtifactTypeSchema':
    """Gets the IR ArtifactTypeSchema proto for a bundled artifact in form
    `<namespace>.<Name>@x.x.x` (e.g., system.Artifact@0.0.1)."""
    bundled_artifact_str, schema_version = bundled_artifact_str.split('@')

    from kfp.pipeline_spec import pipeline_spec_pb2

    return pipeline_spec_pb2.ArtifactTypeSchema(
        schema_title=bundled_artifact_str,
        schema_version=schema_version,
    )


def get_parameter_type(
    param_type: Optional[Union[Type, str, dict]]
) -> 'pipeline_spec_pb2.ParameterType':
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
    return PARAMETER_TYPES_MAPPING.get(type_name.lower())


def get_parameter_type_name(
        param_type: Optional[Union[Type, str, dict]]) -> str:
    """Gets the parameter type name."""

    from kfp.pipeline_spec import pipeline_spec_pb2
    param_enum_val = get_parameter_type(param_type)
    if param_enum_val is None:
        raise ValueError(
            '`param_type` is not a parameter type. Cannot get ParameterType name.'
        )
    return pipeline_spec_pb2.ParameterType.ParameterTypeEnum.Name(
        param_enum_val)


class InconsistentTypeException(Exception):
    """InconsistencyTypeException is raised when two types are not
    consistent."""


class InconsistentTypeWarning(Warning):
    """InconsistentTypeWarning is issued when two types are not consistent."""


def _get_type_string_from_component_argument(
    argument_value: Union['pipeline_channel.PipelineChannel', str, bool, int,
                          float, dict, list]
) -> str:
    # avoid circular imports
    from kfp.dsl import pipeline_channel
    if isinstance(argument_value, pipeline_channel.PipelineChannel):
        return argument_value.channel_type

    # argument is a constant
    argument_type = type(argument_value)
    if argument_type in _TYPE_TO_TYPE_NAME:
        return _TYPE_TO_TYPE_NAME[argument_type]

    if isinstance(argument_value, artifact_types.Artifact):
        raise ValueError(
            f'Input artifacts are not supported. Got input artifact of type {argument_value.__class__.__name__!r}.'
        )
    raise ValueError(
        f'Constant argument inputs must be one of type {list(_TYPE_TO_TYPE_NAME.values())} Got: {argument_value!r} of type {type(argument_value)!r}.'
    )


def verify_type_compatibility(
    given_value: Union['pipeline_channel.PipelineChannel', str, bool, int,
                       float, dict, list],
    expected_spec: Union['structures.InputSpec', 'structures.OutputSpec'],
    error_message_prefix: str,
    checks_input: bool = True,
    raise_on_error: bool = True,
) -> bool:
    """Verifies the given argument type is compatible with the expected type.

    Args:
        given_value: The channel or constant provided as an argument.
        expected_spec: The InputSpec or OutputSpec that describes the expected type of given_value.
        error_message_prefix: The prefix for the error message.
        checks_input: True if checks an argument (given_value) against a component/pipeline input type (expected_spec). False if checks a component output (argument_value) against the pipeline output type (expected_spec).
        raise_on_error: Whether to raise on type compatibility error. Should be passed kfp.TYPE_CHECK.

    Returns:
        True if types are compatible, and False if otherwise.

    Raises:
        InconsistentTypeException if raise_on_error=True.
    """
    # extract and normalize types
    expected_type = expected_spec.type
    given_type = _get_type_string_from_component_argument(given_value)

    # avoid circular imports
    from kfp.dsl import for_loop

    # Workaround for potential type-checking issues during ParallelFor compilation: When LoopArgument or LoopArgumentVariable are involved and the expected type is 'String', we temporarily relax type enforcement to avoid blocking compilation. This is necessary due to potential information loss during the compilation step.
    if isinstance(given_value,
                  (for_loop.LoopParameterArgument,
                   for_loop.LoopArgumentVariable)) and given_type == 'String':
        return True

    given_is_param = is_parameter_type(str(given_type))
    if given_is_param:
        given_type = get_parameter_type_name(given_type)
        given_is_artifact_list = False
    else:
        given_is_artifact_list = given_value.is_artifact_list

    expected_is_param = is_parameter_type(expected_type)
    if expected_is_param:
        expected_type = get_parameter_type_name(expected_type)
        expected_is_artifact_list = False
    else:
        expected_is_artifact_list = expected_spec.is_artifact_list

    # compare the normalized types
    if given_is_param != expected_is_param:
        types_are_compatible = False
    elif given_is_param and expected_is_param:
        types_are_compatible = check_parameter_type_compatibility(
            given_type, expected_type)
    else:
        types_are_compatible = check_artifact_type_compatibility(
            given_type=given_type,
            given_is_artifact_list=given_is_artifact_list,
            expected_type=expected_type,
            expected_is_artifact_list=expected_is_artifact_list)

    # maybe raise, maybe warn, return bool
    if not types_are_compatible:
        # update the types for lists of artifacts for error message
        given_type = f'List[{given_type}]' if given_is_artifact_list else given_type
        expected_type = f'List[{expected_type}]' if expected_is_artifact_list else expected_type
        if checks_input:
            error_message_suffix = f'Argument type {given_type!r} is incompatible with the input type {expected_type!r}'
        else:
            error_message_suffix = f'Output of type {given_type!r} cannot be surfaced as pipeline output type {expected_type!r}'
        error_text = error_message_prefix + error_message_suffix
        if raise_on_error:
            raise InconsistentTypeException(error_text)
        else:
            warnings.warn(InconsistentTypeWarning(error_text))

    return types_are_compatible


def check_artifact_type_compatibility(given_type: str,
                                      given_is_artifact_list: bool,
                                      expected_type: str,
                                      expected_is_artifact_list: bool) -> bool:
    given_schema_title, given_schema_version = given_type.split('@')
    expected_schema_title, expected_schema_version = expected_type.split('@')
    same_list_of_artifacts_status = expected_is_artifact_list == given_is_artifact_list
    if not same_list_of_artifacts_status:
        return False
    elif artifact_types.Artifact.schema_title in {
            given_schema_title, expected_schema_title
    }:
        return True
    else:
        schema_title_compatible = given_schema_title == expected_schema_title
        schema_version_compatible = given_schema_version.split(
            '.')[0] == expected_schema_version.split('.')[0]

        return schema_title_compatible and schema_version_compatible


def check_parameter_type_compatibility(given_type: str,
                                       expected_type: str) -> bool:
    if isinstance(given_type, str) and isinstance(expected_type, str):
        return given_type == expected_type
    else:
        return check_v1_struct_parameter_type_compatibility(
            given_type, expected_type)


def check_v1_struct_parameter_type_compatibility(
    given_type: Union[str, dict],
    expected_type: Union[str, dict],
) -> bool:
    if isinstance(given_type, str):
        given_type = {given_type: {}}
    if isinstance(expected_type, str):
        expected_type = {expected_type: {}}
    return _check_dict_types(given_type, expected_type)


def _check_dict_types(
    given_type: dict,
    expected_type: dict,
) -> bool:
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
    'TASK_FINAL_STATUS': task_final_status.PipelineTaskFinalStatus.__name__,
}

IR_TYPE_TO_COMMENT_TYPE_STRING = {
    'STRING': str.__name__,
    'NUMBER_INTEGER': int.__name__,
    'NUMBER_DOUBLE': float.__name__,
    'LIST': list.__name__,
    'STRUCT': dict.__name__,
    'BOOLEAN': bool.__name__,
    'TASK_FINAL_STATUS': task_final_status.PipelineTaskFinalStatus.__name__,
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
    if not isinstance(type_name, str):
        return type_name

    if type_name.startswith('typing.'):
        type_name = type_name.lstrip('typing.')

    if type_name.lower().startswith('list') or type_name.lower().startswith(
            'dict'):
        return type_name.split('[')[0]

    else:
        return type_name


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
