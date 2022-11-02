# Copyright 2022 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the 'License');
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an 'AS IS' BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import collections
from collections import abc
import dataclasses
import inspect
import pprint
from typing import (Any, ForwardRef, Iterable, Iterator, Mapping,
                    MutableMapping, MutableSequence, Optional, OrderedDict,
                    Sequence, Tuple, Type, TypeVar, Union)

PRIMITIVE_TYPES = {int, str, float, bool}

# typing.Optional.__origin__ is typing.Union
UNION_TYPES = {Union}

# do not need typing.List, because __origin__ is list
ITERABLE_TYPES = {
    list,
    abc.Sequence,
    abc.MutableSequence,
    Sequence,
    MutableSequence,
    Iterable,
}

# do not need typing.Dict, because __origin__ is dict
MAPPING_TYPES = {
    dict, abc.Mapping, abc.MutableMapping, Mapping, MutableMapping, OrderedDict,
    collections.OrderedDict
}

OTHER_SUPPORTED_TYPES = {type(None), Any}
SUPPORTED_TYPES = PRIMITIVE_TYPES | UNION_TYPES | ITERABLE_TYPES | MAPPING_TYPES | OTHER_SUPPORTED_TYPES

BaseModelType = TypeVar('BaseModelType', bound='BaseModel')


class BaseModel:
    """BaseModel for structures. Subclasses are converted to dataclasses at
    object construction time, with.

    Subclasses are dataclasses with methods to support for converting to
    and from dict, type enforcement, and user-defined validation logic.
    """

    # this quiets the mypy "Unexpected keyword argument..." errors on subclass construction
    # TODO: find a way to propogate type info to subclasses
    def __init__(self, *args, **kwargs):
        pass

    def __init_subclass__(cls) -> None:
        """Hook called at subclass definition time and at instance construction
        time.

        Validates that the field to type mapping provided at subclass
        definition time are supported by BaseModel.
        """
        cls = dataclasses.dataclass(cls)
        # print(inspect.signature(cls.__init__))
        for field in dataclasses.fields(cls):
            cls._recursively_validate_type_is_supported(field.type)

    @classmethod
    def _recursively_validate_type_is_supported(cls, type_: type) -> None:
        """Walks the type definition (generics and subtypes) and checks if it
        is supported by downstream BaseModel operations.

        Args:
            type_ (type): Type to check.

        Raises:
            TypeError: If type is unsupported.
        """
        if isinstance(type_, ForwardRef):
            return

        if type_ in SUPPORTED_TYPES or _is_basemodel(type_):
            return

        if _get_origin_py37(type_) not in SUPPORTED_TYPES:
            raise TypeError(
                f'Type {type_} is not a supported type fields on child class of {BaseModel.__name__}: {cls.__name__}.'
            )

        args = _get_args_py37(type_) or [Any, Any]
        for arg in args:
            cls._recursively_validate_type_is_supported(arg)

    def __post_init__(self) -> None:
        """Hook called after object is instantiated from BaseModel.

        Transforms data and validates data using user-defined logic by
        calling all methods prefixed with `_transform_`, then all
        methods prefixed with `_validate_`.
        """
        validate_methods = [
            method for method in dir(self)
            if method.startswith('_transform_') and
            callable(getattr(self, method))
        ]
        for method in validate_methods:
            getattr(self, method)()

        validate_methods = [
            method for method in dir(self) if
            method.startswith('_validate_') and callable(getattr(self, method))
        ]
        for method in validate_methods:
            getattr(self, method)()

    def __str__(self) -> str:
        """Returns a readable representation of the BaseModel subclass."""
        return base_model_format(self)


def base_model_format(x: BaseModelType) -> str:
    """Formats a BaseModel object for improved readability.

    Args:
        x (BaseModelType): The subclass of BaseModel to format.
        chars (int, optional): Indentation size. Defaults to 0.

    Returns:
        str: Readable string representation of the object.
    """

    CHARS = 0

    def first_level_indent(string: str, chars: int = 1) -> str:
        return '\n'.join(' ' * chars + p for p in string.split('\n'))

    def body_level_indent(string: str, chars=4) -> str:
        a, *b = string.split('\n')
        return a + '\n' + first_level_indent(
            '\n'.join(b),
            chars=chars,
        ) if b else a

    def parts() -> Iterator[str]:
        if dataclasses.is_dataclass(x):
            yield type(x).__name__ + '('

            def fields() -> Iterator[str]:
                for field in dataclasses.fields(x):
                    nindent = CHARS + len(field.name) + 4
                    value = getattr(x, field.name)
                    rep_value = base_model_format(value)
                    yield (' ' * (CHARS + 3) + body_level_indent(
                        f'{field.name}={rep_value}', chars=nindent))

            yield ',\n'.join(fields())
            yield ' ' * CHARS + ')'
        else:
            yield pprint.pformat(x)

    return '\n'.join(parts())


def _is_basemodel(obj: Any) -> bool:
    """Checks if object is a subclass of BaseModel.

    Args:
        obj (Any): Any object

    Returns:
        bool: Is a subclass of BaseModel.
    """
    return inspect.isclass(obj) and issubclass(obj, BaseModel)


def _get_origin_py37(type_: Type) -> Optional[Type]:
    """typing.get_origin is introduced in Python 3.8, but we need a get_origin
    that is compatible with 3.7.

    Args:
        type_ (Type): A type.

    Returns:
        Type: The origin of `type_`.
    """
    # uses typing for types
    return type_.__origin__ if hasattr(type_, '__origin__') else None


def _get_args_py37(type_: Type) -> Tuple[Type]:
    """typing.get_args is introduced in Python 3.8, but we need a get_args that
    is compatible with 3.7.

    Args:
        type_ (Type): A type.

    Returns:
        Tuple[Type]: The type arguments of `type_`.
    """
    # uses typing for types
    return type_.__args__ if hasattr(type_, '__args__') else tuple()
