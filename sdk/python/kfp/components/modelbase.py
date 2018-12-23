# Copyright 2018 Google LLC
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
    'ModelBase',
]

import copy
from collections import abc, OrderedDict
from typing import Any, Callable, Dict, List, Mapping, MutableMapping, MutableSequence, Optional, Sequence, Tuple, Type, TypeVar, Union, cast, get_type_hints


T = TypeVar('T')


def check_type(typ: Type[T], x: Any) -> T:
    if typ is type(None):
        if x is None:
            return x
        else:
            raise TypeError('Error: Object "{}" is not None.'.format(x))

    if typ is Any or type(typ) is TypeVar:
        return x

    try: #isinstance can fail for generics
        if isinstance(x, typ):
            return cast(typ, x)
    except:
        pass

    if hasattr(typ, '__origin__'): #Handling generic types
        if typ.__origin__ is Union: #Optional == Union
            exception_map = {}
            possible_types = typ.__args__
            if type(None) in possible_types and x is None: #Shortcut for Optional[] tests. Can be removed, but the exceptions will be more noisy.
                return x
            for possible_type in possible_types:
                try:
                    check_type(possible_type, x)
                    return x
                except Exception as ex:
                    exception_map[possible_type] = ex
                    pass
            #exception_lines = ['Exception for type {}: {}.'.format(t, e) for t, e in exception_map.items()]
            exception_lines = [str(e) for t, e in exception_map.items()]
            exception_lines.append('Error: Object "{}" is incompatible with type "{}".'.format(x, typ))
            raise TypeError('\n'.join(exception_lines))

        #not Union => not None
        if x is None:
            raise TypeError('Error: None object is incompatible with type {}'.format(typ))

        #assert isinstance(x, typ.__origin__)
        if typ.__origin__ in [list, List, abc.Sequence, abc.MutableSequence]:
            if not isinstance(x, typ.__origin__):
                raise TypeError('Error: Object "{}" is incompatible with type "{}"'.format(x, typ))
            inner_type = typ.__args__[0]
            for item in x:
                check_type(inner_type, item)
            return x

        elif typ.__origin__ in [dict, Dict, abc.Mapping, abc.MutableMapping]:
            if not isinstance(x, typ.__origin__):
                raise TypeError('Error: Object "{}" is incompatible with type "{}"'.format(x, typ))
            inner_key_type = typ.__args__[0]
            inner_value_type = typ.__args__[1]
            for k, v in x.items():
                check_type(inner_key_type, k)
                check_type(inner_value_type, v)
            return x

        else:
            raise TypeError('Error: Unsupported generic type {}'.format(typ))

    raise TypeError('Error: Object "{}" is incompatible with type "{}"'.format(x, typ))


def parse_typed_object_from_struct(typ: Type[T], struct: Any) -> T:
    if typ is type(None):
        if struct is None:
            return None
        else:
            raise TypeError('Error: Structure "{}" is not None.'.format(struct))

    if typ is Any or type(typ) is TypeVar:
        return struct

    try: #isinstance can fail for generics
        #if (isinstance(struct, typ)
        #    and not (typ is Sequence and type(struct) is str) #! str is also Sequence
        #    and not (typ is int and type(struct) is bool) #! bool is int
        #):
        if type(struct) is typ:
            return struct
    except:
        pass

    if hasattr(typ, 'from_struct'):
        try: #More informative errors
            return typ.from_struct(struct)
        except Exception as ex:
            raise TypeError('Error: {}.from_struct(struct={}) failed with exception: {}'.format(typ.__name__, struct, str(ex)))
    if hasattr(typ, '__origin__'): #Handling generic types
        if typ.__origin__ is Union: #Optional == Union
            results = {}
            exception_map = {}
            possible_types = typ.__args__
            #if type(None) in possible_types and struct is None: #Shortcut for Optional[] tests. Can be removed, but the exceptions will be more noisy.
            #    return None
            if struct == True:
                exception_map = exception_map
            for possible_type in possible_types:
                try:
                    obj = parse_typed_object_from_struct(possible_type, struct)
                    results[possible_type] = obj
                except Exception as ex:
                    exception_map[possible_type] = ex
                    pass

            #Single successful parsing.
            if len(results) == 1:
                return list(results.values())[0]

            if len(results) > 1:
                raise TypeError('Error: Structure "{}" is ambiguous. It can be parsed to multiple types: {}.'.format(struct, list(results.keys())))

            exception_lines = [str(e) for t, e in exception_map.items()]
            exception_lines.append('Error: Structure "{}" is incompatible with type "{}" - none of the types in Union are compatible.'.format(struct, typ))
            raise TypeError('\n'.join(exception_lines))
        #not Union => not None
        if struct is None:
            raise TypeError('Error: None structure is incompatible with type {}'.format(typ))

        #assert isinstance(x, typ.__origin__)
        if typ.__origin__ in [list, List, abc.Sequence, abc.MutableSequence] and type(struct) is not str: #! str is also Sequence
            if not isinstance(struct, typ.__origin__):
                raise TypeError('Error: Structure "{}" is incompatible with type "{}" - it does not have list type.'.format(struct, typ))
            inner_type = typ.__args__[0]
            return [parse_typed_object_from_struct(inner_type, item) for item in struct]

        elif typ.__origin__ in [dict, Dict, abc.Mapping, abc.MutableMapping]:
            if not isinstance(struct, typ.__origin__):
                raise TypeError('Error: Structure "{}" is incompatible with type "{}" - it does not have dict type.'.format(struct, typ))
            inner_key_type = typ.__args__[0]
            inner_value_type = typ.__args__[1]
            return {parse_typed_object_from_struct(inner_key_type, k): parse_typed_object_from_struct(inner_value_type, v) for k, v in struct.items()}

        else:
            raise TypeError('Error: Unsupported generic type "{}"'.format(typ))

    raise TypeError('Error: Structure "{}" is incompatible with type "{}". Structure is not the instance of the type, the type does not have .from_struct method and is not generic.'.format(struct, typ))


def model_to_struct(self, original_names: Mapping[str, str] = {}):
    """
    Returns the model properties as a dict
    """
    import inspect
    signature = inspect.signature(self.__init__) #Needed for default values
    result = {}
    for python_name, value in self.__dict__.items():
        if python_name.startswith('_'):
            continue
        attr_name = original_names.get(python_name, python_name)
        if hasattr(value, "to_struct"):
            result[attr_name] = value.to_struct()
        elif isinstance(value, list):
            result[attr_name] = [(x.to_struct() if hasattr(x, 'to_struct') else x) for x in value]
        elif isinstance(value, dict):
            result[attr_name] = {k: (v.to_struct() if hasattr(v, 'to_struct') else v) for k, v in value.items()}
        else:
            param = signature.parameters.get(python_name, None)
            if param is None or param.default == inspect.Parameter.empty or value != param.default:
                result[attr_name] = value

    return result


def model_from_struct(cls : Type[T], struct: Mapping, original_names: Mapping[str, str] = {}) -> T:
    """
    Parses the model object from a dict
    """
    parameter_types = get_type_hints(cls.__init__) #Properlty resolves forward references

    original_names_to_pythonic = {v: k for k, v in original_names.items()}
    #If a pythonic name has a different original name, we forbid the pythonic name in the structure. Otherwise, this function would accept "python-styled" structures that should be invalid
    forbidden_struct_keys = set(original_names_to_pythonic.values()).difference(original_names_to_pythonic.keys())
    args = {}
    for original_name, value in struct.items():
        if original_name in forbidden_struct_keys:
            raise ValueError('Use "{}" key instead of pythonic key "{}" in the structure: {}.'.format(original_names[original_name], original_name, struct))
        python_name = original_names_to_pythonic.get(original_name, original_name)
        param_type = parameter_types.get(python_name, None)
        if param_type is not None:
            args[python_name] = parse_typed_object_from_struct(param_type, value)
        else:
            args[python_name] = value

    return cls(**args)


class ModelBase:
    _original_names = {}
    def __init__(self, args):
        parameter_types = get_type_hints(self.__class__.__init__)
        field_values = {k: v for k, v in args.items() if k != 'self' and not k.startswith('_')}
        for k, v in field_values.items():
            parameter_type = parameter_types.get(k, None)
            if parameter_type is not None:
                check_type(parameter_type, v)
        self.__dict__.update(field_values)

    @classmethod
    def from_struct(cls: Type[T], struct: Mapping) -> T:
        return model_from_struct(cls, struct, original_names=cls._original_names)

    def to_struct(self) -> Mapping:
        return model_to_struct(self, original_names=self._original_names)

    def __repr__(self):
        return self.__class__.__name__ + '.from_struct(' + str(self.to_struct()) + ')'

