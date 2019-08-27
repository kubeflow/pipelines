# Copyright 2019 Google LLC
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

__all__ = [
    'type_to_type_name',
    'type_name_to_type',
    'type_to_deserializer',
    'type_name_to_deserializer',
    'type_name_to_serializer',
]


import inspect
from typing import Any, Callable, NamedTuple, Sequence
import warnings


Converter = NamedTuple('Converter', [
    ('types', Sequence[str]),
    ('type_names', Sequence[str]),
    ('serializer', Callable[[Any], str]),
    ('deserializer_code', str),
    ('definitions', str),
])


def _deserialize_bool(s) -> bool:
    from distutils.util import strtobool
    return strtobool(s) == 1


_bool_deserializer_definitions = inspect.getsource(_deserialize_bool)
_bool_deserializer_code = _deserialize_bool.__name__


def _serialize_json(obj) -> str:
    import json
    return json.dumps(obj)


def _serialize_base64_pickle(obj) -> str:
    import base64
    import pickle
    return base64.b64encode(pickle.dumps(obj)).decode('ascii')


def _deserialize_base64_pickle(s):
    import base64
    import pickle
    return pickle.loads(base64.b64decode(s))


_deserialize_base64_pickle_definitions = inspect.getsource(_deserialize_base64_pickle)
_deserialize_base64_pickle_code = _deserialize_base64_pickle.__name__


_converters = [
    Converter([str], ['String', 'str'], str, 'str', None),
    Converter([int], ['Integer', 'int'], str, 'int', None),
    Converter([float], ['Float', 'float'], str, 'float', None),
    Converter([bool], ['Boolean', 'bool'], str, _bool_deserializer_code, _bool_deserializer_definitions),
    Converter([list], ['JsonArray', 'List', 'list'], _serialize_json, 'json.loads', 'import json'), # ! JSON map keys are always strings. Python converts all keys to strings without warnings
    Converter([dict], ['JsonObject', 'Dictionary', 'Dict', 'dict'], _serialize_json, 'json.loads', 'import json'), # ! JSON map keys are always strings. Python converts all keys to strings without warnings
    Converter([], ['Json'], _serialize_json, 'json.loads', 'import json'),
    Converter([], ['Base64Pickle'], _serialize_base64_pickle, _deserialize_base64_pickle_code, _deserialize_base64_pickle_definitions),
]


type_to_type_name = {typ: converter.type_names[0] for converter in _converters for typ in converter.types}
type_name_to_type = {type_name: converter.types[0] for converter in _converters for type_name in converter.type_names if converter.types}
type_to_deserializer = {typ: (converter.deserializer_code, converter.definitions) for converter in _converters for typ in converter.types}
type_name_to_deserializer = {type_name: (converter.deserializer_code, converter.definitions) for converter in _converters for type_name in converter.type_names}
type_name_to_serializer = {type_name: converter.serializer for converter in _converters for type_name in converter.type_names}


def serialize_value(value, type_name: str) -> str:
    '''serialize_value converts the passed value to string based on the serializer associated with the passed type_name'''
    if isinstance(value, str):
        return value # The value is supposedly already serialized

    if type_name is None:
        type_name = type_to_type_name.get(type(value), type(value).__name__)
        warnings.warn('Missing type name was inferred as "{}" based on the value "{}".'.format(type_name, str(value)))

    serializer = type_name_to_serializer.get(type_name, None)
    if serializer:
        try:
            serialized_value = serializer(value)
            if not isinstance(serialized_value, str):
                raise TypeError('Serializer {} returned result of type "{}" instead of string.'.format(serializer, type(serialized_value)))
            return serialized_value
        except Exception as e:
            raise ValueError('Failed to serialize the value "{}" of type "{}" to type "{}". Exception: {}'.format(
                str(value),
                str(type(value).__name__),
                str(type_name),
                str(e),
            ))

    serialized_value = str(value)
    warnings.warn('There are no registered serializers from type "{}" to type "{}", so the value will be serializers as string "{}".'.format(
        str(type(value).__name__),
        str(type_name),
        serialized_value),
    )
    return serialized_value
