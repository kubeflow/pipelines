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
"""Classes for input/output type annotations in KFP SDK.

These are only compatible with v2 Pipelines.
"""

import re
from typing import Any, List, Optional, Type, TypeVar, Union

from kfp.dsl.types import artifact_types
from kfp.dsl.types import type_annotations
from kfp.dsl.types import type_utils


class OutputPath:
    """Type annotation used in component definitions for indicating a parameter
    is a path to an output. The path parameter typed with this annotation can
    be treated as a locally accessible filepath within the component body.

    The argument typed with this annotation is provided at runtime by the executing backend and does not need to be passed as an input by the pipeline author (see example).


    Args:
        type: The type of the value written to the output path.

    Example:
      ::

        @dsl.component
        def create_parameter(
                message: str,
                output_parameter_path: OutputPath(str),
        ):
            with open(output_parameter_path, 'w') as f:
                f.write(message)


        @dsl.component
        def consume_parameter(message: str):
            print(message)


        @dsl.pipeline(name='my-pipeline', pipeline_root='gs://my-bucket')
        def my_pipeline(message: str = 'default message'):
            create_param_op = create_parameter(message=message)
            consume_parameter(message=create_param_op.outputs['output_parameter_path'])
    """

    def __init__(self, type=None):
        self.type = construct_type_for_inputpath_or_outputpath(type)

    def __eq__(self, other):
        return isinstance(other, OutputPath) and self.type == other.type


class InputPath:
    """Type annotation used in component definitions for indicating a parameter
    is a path to an input.

    Example:
      ::

        @dsl.component
        def create_dataset(dataset_path: OutputPath('Dataset'),):
            import json
            dataset = {'my_dataset': [[1, 2, 3], [4, 5, 6]]}
            with open(dataset_path, 'w') as f:
                json.dump(dataset, f)


        @dsl.component
        def consume_dataset(dataset: InputPath('Dataset')):
            print(dataset)


        @dsl.pipeline(name='my-pipeline', pipeline_root='gs://my-bucket')
        def my_pipeline():
            create_dataset_op = create_dataset()
            consume_dataset(dataset=create_dataset_op.outputs['dataset_path'])
    """

    def __init__(self, type=None):
        self.type = construct_type_for_inputpath_or_outputpath(type)

    def __eq__(self, other):
        return isinstance(other, InputPath) and self.type == other.type


def construct_type_for_inputpath_or_outputpath(
        type_: Union[str, Type, None]) -> Optional[str]:
    if type_annotations.is_artifact_class(type_):
        return type_utils.create_bundled_artifact_type(type_.schema_title,
                                                       type_.schema_version)
    elif isinstance(
            type_,
            str) and type_.lower() in type_utils.ARTIFACT_CLASSES_MAPPING:
        # v1 artifact backward compat, e.g. dsl.OutputPath('Dataset')
        return type_utils.create_bundled_artifact_type(
            type_utils.ARTIFACT_CLASSES_MAPPING[type_.lower()].schema_title)
    elif type_utils.get_parameter_type(type_):
        return type_
    else:
        # v1 unknown type dsl.OutputPath('MyCustomType')
        return type_utils.create_bundled_artifact_type(
            artifact_types.Artifact.schema_title)


class InputAnnotation:
    """Marker type for input artifacts."""


class OutputAnnotation:
    """Marker type for output artifacts."""


def is_Input_Output_artifact_annotation(typ) -> bool:
    if not hasattr(typ, '__metadata__'):
        return False

    if typ.__metadata__[0] not in [InputAnnotation, OutputAnnotation]:
        return False

    return True


def is_artifact_wrapped_in_Input(typ: Any) -> bool:
    """Returns True if typ is of type Input[T]."""
    if not is_Input_Output_artifact_annotation(typ):
        return False

    return typ.__metadata__[0] == InputAnnotation


def is_artifact_wrapped_in_Output(typ: Any) -> bool:
    """Returns True if typ is of type Output[T]."""
    if not is_Input_Output_artifact_annotation(typ):
        return False

    return typ.__metadata__[0] == OutputAnnotation


def get_io_artifact_class(typ):
    from kfp.dsl import Input
    from kfp.dsl import Output
    if not is_Input_Output_artifact_annotation(typ):
        return None
    if typ == Input or typ == Output:
        return None

    # extract inner type from list of artifacts
    inner = strip_Input_or_Output_marker(typ)
    if hasattr(inner, '__origin__') and inner.__origin__ == list:
        return inner.__args__[0]

    return inner


def strip_Input_or_Output_marker(typ: Any) -> artifact_types.Artifact:
    return typ.__args__[0]


def get_input_or_output_marker(
        typ) -> Optional[Union[InputAnnotation, OutputAnnotation]]:
    if not is_Input_Output_artifact_annotation(typ):
        return None

    return typ.__metadata__[0]


T = TypeVar('T')


def maybe_strip_optional_from_annotation(annotation: T) -> T:
    """Strips 'Optional' from 'Optional[<type>]' if applicable.

    For example::
      Optional[str] -> str
      str -> str
      List[int] -> List[int]

    Args:
      annotation: The original type annotation which may or may not has
        `Optional`.

    Returns:
      The type inside Optional[] if Optional exists, otherwise the original type.
    """
    if getattr(annotation, '__origin__',
               None) is Union and annotation.__args__[1] is type(None):
        return annotation.__args__[0]
    return annotation


def maybe_strip_optional_from_annotation_string(annotation: str) -> str:
    if annotation.startswith('Optional[') and annotation.endswith(']'):
        return annotation.lstrip('Optional[').rstrip(']')
    return annotation


def get_short_type_name(type_name: str) -> str:
    """Extracts the short form type name.

    This method is used for looking up serializer for a given type.

    For example::
      typing.List -> List
      typing.List[int] -> List
      typing.Dict[str, str] -> Dict
      List -> List
      str -> str

    Args:
      type_name: The original type name.

    Returns:
      The short form type name or the original name if pattern doesn't match.
    """
    match = re.match(r'(typing\.)?(?P<type>\w+)(?:\[.+\])?', type_name)
    return match['type'] if match else type_name


def is_artifact_class(artifact_class_or_instance: Type) -> bool:
    # we do not yet support non-pre-registered custom artifact types with instance_schema attribute
    return hasattr(artifact_class_or_instance, 'schema_title') and hasattr(
        artifact_class_or_instance, 'schema_version')


def is_list_of_artifacts(
    annotation: Union[Type[List[artifact_types.Artifact]],
                      Type[artifact_types.Artifact]]
) -> bool:
    """Checks if an object is a list of list of artifacts annotation (e.g.,
    List[Artifact], List[Dataset])"""
    return is_generic_list(annotation) and issubclass_of_artifact(
        get_inner_type(annotation))


def get_inner_type(annotation: Any) -> Optional[Any]:
    """Returns the inner type of a generic annotation.

    For Union or Optional types with multiple inner types, a tuple of
    types is returned.
    """
    # Check if the annotation has '__args__' attribute
    if hasattr(annotation, '__args__'):
        if len(annotation.__args__) == 1:
            return annotation.__args__[0]
        else:
            return tuple(annotation.__args__)
    return None


def issubclass_of_artifact(obj: Any) -> bool:
    """Checks if an object is a class and a subclass of a dsl.Artifact."""
    return type(obj) == type and issubclass(obj, artifact_types.Artifact)


def is_generic_list(annotation: Any) -> bool:
    # handles generics from the typing module for python<3.9
    typing_generic_list = getattr(annotation, '__origin__',
                                  None) is list or getattr(
                                      annotation, '__origin__', None) is List
    # handles built-in generics for python>=3.9
    built_in_generic_list = annotation == list
    return typing_generic_list or built_in_generic_list
