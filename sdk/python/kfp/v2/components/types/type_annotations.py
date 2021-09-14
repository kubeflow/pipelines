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
from typing import TypeVar, Union

T = TypeVar('T')


class OutputPath:
    """Annotation for indicating a variable is a path to an output."""

    def __init__(self, type=None):
        self.type = type

    def __eq__(self, other):
        if isinstance(other, OutputPath):
            return self.type == other.type
        return False


class InputPath:
    """Annotation for indicating a variable is a path to an input."""

    def __init__(self, type=None):
        self.type = type

    def __eq__(self, other):
        if isinstance(other, InputPath):
            return self.type == other.type
        return False


class InputAnnotation():
    """Marker type for input artifacts."""
    pass


class OutputAnnotation():
    """Marker type for output artifacts."""
    pass


# TODO: Use typing.Annotated instead of this hack.
# With typing.Annotated (Python 3.9+ or typing_extensions package), the
# following would look like:
# Input = typing.Annotated[T, InputAnnotation]
# Output = typing.Annotated[T, OutputAnnotation]

# Input represents an Input artifact of type T.
Input = Union[T, InputAnnotation]

# Output represents an Output artifact of type T.
Output = Union[T, OutputAnnotation]


def is_artifact_annotation(typ) -> bool:
    if hasattr(typ, '_subs_tree'):  # Python 3.6
        subs_tree = typ._subs_tree()
        return len(
            subs_tree) == 3 and subs_tree[0] == Union and subs_tree[2] in [
                InputAnnotation, OutputAnnotation
            ]

    if not hasattr(typ, '__origin__'):
        return False

    if typ.__origin__ != Union and type(typ.__origin__) != type(Union):
        return False

    if not hasattr(typ, '__args__') or len(typ.__args__) != 2:
        return False

    if typ.__args__[1] not in [InputAnnotation, OutputAnnotation]:
        return False

    return True


def is_input_artifact(typ) -> bool:
    """Returns True if typ is of type Input[T]."""
    if not is_artifact_annotation(typ):
        return False

    if hasattr(typ, '_subs_tree'):  # Python 3.6
        subs_tree = typ._subs_tree()
        return len(subs_tree) == 3 and subs_tree[2] == InputAnnotation

    return typ.__args__[1] == InputAnnotation


def is_output_artifact(typ) -> bool:
    """Returns True if typ is of type Output[T]."""
    if not is_artifact_annotation(typ):
        return False

    if hasattr(typ, '_subs_tree'):  # Python 3.6
        subs_tree = typ._subs_tree()
        return len(subs_tree) == 3 and subs_tree[2] == OutputAnnotation

    return typ.__args__[1] == OutputAnnotation


def get_io_artifact_class(typ):
    if not is_artifact_annotation(typ):
        return None
    if typ == Input or typ == Output:
        return None

    if hasattr(typ, '_subs_tree'):  # Python 3.6
        subs_tree = typ._subs_tree()
        if len(subs_tree) != 3:
            return None
        return subs_tree[1]

    return typ.__args__[0]


def get_io_artifact_annotation(typ):
    if not is_artifact_annotation(typ):
        return None

    if hasattr(typ, '_subs_tree'):  # Python 3.6
        subs_tree = typ._subs_tree()
        if len(subs_tree) != 3:
            return None
        return subs_tree[2]

    return typ.__args__[1]


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
    match = re.match('(typing\.)?(?P<type>\w+)(?:\[.+\])?', type_name)
    if match:
        return match.group('type')
    else:
        return type_name
