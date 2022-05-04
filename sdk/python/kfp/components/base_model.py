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
import dataclasses
import inspect
import json
import pprint
from collections import abc
from typing import (Any, Dict, ForwardRef, Iterable, Iterator, Mapping,
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

    _aliases: Dict[str, str] = {}  # will be used by subclasses

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

    def to_dict(self, by_alias: bool = False) -> Dict[str, Any]:
        """Recursively converts to a dictionary.

        Args:
            by_alias (bool, optional): Whether to use attribute name to alias field mapping provided by cls._aliases when converting to dictionary. Defaults to False.

        Returns:
            Dict[str, Any]: Dictionary representation of the object.
        """
        return convert_object_to_dict(self, by_alias=by_alias)

    def to_json(self, by_alias: bool = False) -> str:
        """Recursively converts to a JSON string.

        Args:
            by_alias (bool, optional): Whether to use attribute name to alias field mapping provided by cls._aliases when converting to JSON. Defaults to False.

        Returns:
            str: JSON representation of the object.
        """
        return json.dumps(self.to_dict(by_alias=by_alias))

    @classmethod
    def from_dict(cls,
                  data: Dict[str, Any],
                  by_alias: bool = False) -> BaseModelType:
        """Recursively loads object from a dictionary.

        Args:
            data (Dict[str, Any]): Dictionary representation of the object.
            by_alias (bool, optional): Whether to use attribute name to alias field mapping provided by cls._aliases when reading in dictionary. Defaults to False.

        Returns:
            BaseModelType: Subclass of BaseModel.
        """
        return _load_basemodel_helper(cls, data, by_alias=by_alias)

    @classmethod
    def from_json(cls, text: str, by_alias: bool = False) -> 'BaseModel':
        """Recursively loads object from a JSON string.

        Args:
            text (str): JSON representation of the object.
            by_alias (bool, optional): Whether to use attribute name to alias field mapping provided by cls._aliases when reading in JSON. Defaults to False.

        Returns:
            BaseModelType: Subclass of BaseModel.
        """
        return _load_basemodel_helper(cls, json.loads(text), by_alias=by_alias)

    @property
    def types(self) -> Dict[str, type]:
        """Dictionary mapping field names to types."""
        return {field.name: field.type for field in dataclasses.fields(self)}

    @property
    def fields(self) -> Tuple[dataclasses.Field, ...]:
        """The dataclass fields."""
        return dataclasses.fields(self)

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
        calling all methods prefixed with `transform_`, then all methods
        prefixed with `validate_`.
        """
        validate_methods = [
            method for method in dir(self) if
            method.startswith('transform_') and callable(getattr(self, method))
        ]
        for method in validate_methods:
            getattr(self, method)()

        validate_methods = [
            method for method in dir(self) if method.startswith('validate_') and
            callable(getattr(self, method))
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


def convert_object_to_dict(obj: BaseModelType,
                           by_alias: bool) -> Dict[str, Any]:
    """Recursion helper function for converting a BaseModel and data structures
    therein to a dictionary. Converts all fields that do not start with an
    underscore.

    Args:
        obj (BaseModelType): The object to convert to a dictionary. Initially called with subclass of BaseModel.
        by_alias (bool): Whether to use the attribute name to alias field mapping provided by cls._aliases when converting to dictionary.

    Raises:
        ValueError: If a field is missing a required value. In pracice, this should never be raised, but is included to help with debugging.

    Returns:
        Dict[str, Any]: The dictionary representation of the object.
    """
    signature = inspect.signature(obj.__init__)

    result = {}
    for attr_name in signature.parameters:
        if attr_name.startswith('_'):
            continue

        field_name = attr_name
        value = getattr(obj, attr_name)
        param = signature.parameters.get(attr_name, None)

        if by_alias and hasattr(obj, '_aliases'):
            field_name = obj._aliases.get(attr_name, attr_name)

        if hasattr(value, 'to_dict'):
            result[field_name] = value.to_dict(by_alias=by_alias)

        elif isinstance(value, list):
            result[field_name] = [
                (x.to_dict(by_alias=by_alias) if hasattr(x, 'to_dict') else x)
                for x in value
            ]
        elif isinstance(value, dict):
            result[field_name] = {
                k:
                (v.to_dict(by_alias=by_alias) if hasattr(v, 'to_dict') else v)
                for k, v in value.items()
            }
        elif (param is not None):
            result[
                field_name] = value if value != param.default else param.default
        else:
            raise ValueError(
                f'Cannot serialize {obj}. No value for {attr_name}.')
    return result


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


def _load_basemodel_helper(type_: Any, data: Any, by_alias: bool) -> Any:
    """Helper function for recursively loading a BaseModel.

    Args:
        type_ (Any): The type of the object to load. Typically an instance of `type`, `BaseModel` or `Any`.
        data (Any): The data to load.

    Returns:
        Any: The loaded object.
    """
    if isinstance(type_, str):
        raise TypeError(
            'Please do not use built-in collection types as generics (e.g., list[int]) and do not include the import line `from __future__ import annotations`. Please use the corresponding generic from typing (e.g., List[int]).'
        )

    # catch unsupported types early
    type_or_generic = _get_origin_py37(type_) or type_
    if type_or_generic not in SUPPORTED_TYPES and not _is_basemodel(type_):
        raise TypeError(
            f'Unsupported type: {type_}. Cannot load data into object.')

    # if don't have any helpful type information, return data as is
    if type_ is Any:
        return data

    # if type is NoneType and data is None, return data/None
    if type_ is type(None):
        if data is None:
            return data
        else:
            raise TypeError(
                f'Expected value None for type NoneType. Got: {data}')

    # handle primitives, with typecasting
    if type_ in PRIMITIVE_TYPES:
        return type_(data)

    # simple types are handled, now handle for container types
    origin = _get_origin_py37(type_)
    args = _get_args_py37(type_) or [
        Any, Any
    ]  # if there is an inner type in the generic, use it, else use Any
    # recursively load iterable objects
    if origin in ITERABLE_TYPES:
        for arg in args:  # TODO handle errors
            return [
                _load_basemodel_helper(arg, element, by_alias=by_alias)
                for element in data
            ]

    # recursively load mapping objects
    if origin in MAPPING_TYPES:
        if len(args) != 2:
            raise TypeError(
                f'Expected exactly 2 type arguments for mapping type {type_}.')
        return {
            _load_basemodel_helper(args[0], k, by_alias=by_alias):
            _load_basemodel_helper(
                args[1],  # type: ignore
                # length check a few lines up ensures index 1 exists
                v,
                by_alias=by_alias) for k, v in data.items()
        }

    # if the type is a Union, try to load the data into each of the types,
    # greedily accepting the first annotation arg that works --> the developer
    # can indicate which types are preferred based on the annotation arg order
    if origin in UNION_TYPES:
        # don't try to cast none if the union type is optional
        if type(None) in args and data is None:
            return None
        for arg in args:
            return _load_basemodel_helper(args[0], data, by_alias=by_alias)

    # finally, handle the cases where the type is an instance of baseclass
    if _is_basemodel(type_):
        fields = dataclasses.fields(type_)
        basemodel_kwargs = {}
        for field in fields:
            attr_name = field.name
            data_field_name = attr_name
            if by_alias and hasattr(type_, '_aliases'):
                data_field_name = type_._aliases.get(attr_name, attr_name)
            value = data.get(data_field_name)
            if value is None:
                if field.default is dataclasses.MISSING and field.default_factory is dataclasses.MISSING:
                    raise ValueError(
                        f'Missing required field: {data_field_name}')
                value = field.default if field.default is not dataclasses.MISSING else field.default_factory(
                )
            else:
                value = _load_basemodel_helper(
                    field.type, value, by_alias=by_alias)
            basemodel_kwargs[attr_name] = value
        return type_(**basemodel_kwargs)

    raise TypeError(
        f'Unknown error when loading data: {data} into type {type_}')
