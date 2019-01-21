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
    '_normalize_identifier_name',
    '_sanitize_kubernetes_resource_name',
    '_sanitize_python_function_name',
    '_sanitize_file_name',
    '_convert_to_human_name',
    '_generate_unique_suffix',
    'generate_unique_name_conversion_table',
]


import re
import sys
from typing import Callable, Sequence, Mapping


def _normalize_identifier_name(name):
    import re
    normalized_name = name.lower()
    normalized_name = re.sub(r'[\W_]', ' ', normalized_name)           #No non-word characters
    normalized_name = re.sub(' +', ' ', normalized_name).strip()    #No double spaces, leading or trailing spaces
    if re.match(r'\d', normalized_name):
        normalized_name = 'n' + normalized_name                     #No leading digits
    return normalized_name


def _sanitize_kubernetes_resource_name(name):
    return _normalize_identifier_name(name).replace(' ', '-')


def _sanitize_python_function_name(name):
    return _normalize_identifier_name(name).replace(' ', '_')


def _sanitize_file_name(name):
    import re
    return re.sub('[^-_.0-9a-zA-Z]+', '_', name)


def _convert_to_human_name(name: str):
    '''Converts underscore or dash delimited name to space-delimited name that starts with a capital letter.
    Does not handle "camelCase" names.
    '''
    return name.replace('_', ' ').replace('-', ' ').strip().capitalize()


def _generate_unique_suffix(data):
    import time
    import hashlib
    string_data = str( (data, time.time()) )
    return hashlib.sha256(string_data.encode()).hexdigest()[0:8]


def _convert_name_and_make_it_unique_by_adding_number(name: str, used_converted_names, conversion_func: Callable[[str], str]):
    converted_name = conversion_func(name)
    if converted_name in used_converted_names:
        for i in range(2, sys.maxsize ** 10): #Starting indices from 2: "Something", "Something_2", ...
            converted_name = conversion_func(name + ' ' + str(i))
            if converted_name not in used_converted_names:
                break
    return converted_name


def generate_unique_name_conversion_table(names: Sequence[str], conversion_func: Callable[[str], str]) -> Mapping[str, str]:
    '''Given a list of names and conversion_func, this function generates a map from original names to converted names that are made unique by adding numbers.
    '''
    forward_map = {}
    reverse_map = {}
    for name in names:
        if name in forward_map:
            raise ValueError('Original name {} is not unique.'.format(name))
        converted_name = _convert_name_and_make_it_unique_by_adding_number(name, reverse_map, conversion_func)
        forward_map[name] = converted_name
        reverse_map[converted_name] = name
    return forward_map
