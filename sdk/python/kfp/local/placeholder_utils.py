# Copyright 2023 The Kubeflow Authors
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
"""Utilities for working with placeholders."""
import json
import random
import re
from typing import Any, Dict, List, Optional, Union

from kfp import dsl


def make_random_id():
    """Makes a random 8 digit integer."""
    return str(random.randint(0, 99999999))


def replace_placeholders(
    full_command: List[str],
    executor_input_dict: Dict[str, Any],
    pipeline_resource_name: str,
    task_resource_name: str,
    pipeline_root: str,
) -> List[str]:
    """Iterates over each element in the command and replaces placeholders."""
    unique_pipeline_id = make_random_id()
    unique_task_id = make_random_id()
    provided_inputs = get_provided_inputs(executor_input_dict)
    full_command = [
        resolve_struct_placeholders(
            placeholder,
            provided_inputs,
        ) for placeholder in full_command
    ]
    full_command = flatten_list(full_command)
    resolved_command = []
    for el in full_command:
        resolved_el = resolve_individual_placeholder(
            element=el,
            executor_input_dict=executor_input_dict,
            pipeline_resource_name=pipeline_resource_name,
            task_resource_name=task_resource_name,
            pipeline_root=pipeline_root,
            pipeline_job_id=unique_pipeline_id,
            pipeline_task_id=unique_task_id,
        )
        if resolved_el is None:
            continue
        elif isinstance(resolved_el, str):
            resolved_command.append(resolved_el)
        elif isinstance(resolved_el, list):
            resolved_command.extend(resolved_el)
        else:
            raise ValueError(
                f'Got unknown command element {resolved_el} of type {type(resolved_el)}.'
            )
    return resolved_command


def flatten_list(l: List[Union[str, list, None]]) -> List[str]:
    """Iteratively flattens arbitrarily deeply nested lists, filtering out
    elements that are None."""
    result = []
    stack = l.copy()
    while stack:
        element = stack.pop(0)
        if isinstance(element, list):
            stack = element + stack
        elif element is not None:
            result.append(element)
    return result


def get_provided_inputs(executor_input_dict: Dict[str, Any]) -> Dict[str, Any]:
    params = executor_input_dict.get('inputs', {}).get('parameterValues', {})
    pkeys = [k for k, v in params.items() if v is not None]
    artifacts = executor_input_dict.get('inputs', {}).get('artifacts', {})
    akeys = [k for k, v in artifacts.items() if v is not None]
    return pkeys + akeys


def get_value_using_path(
    dictionary: Dict[str, Any],
    path: List[str],
) -> Optional[Any]:
    list_or_dict = dictionary
    if not path:
        raise ValueError('path cannot be empty.')
    try:
        for p in path:
            list_or_dict = list_or_dict[p]
        return list_or_dict
    except KeyError:
        return None


def convert_placeholder_parts_to_path(parts: List[str]) -> List[str]:
    # if inputs, parameters --> parameterValues
    if parts[0] == 'inputs' and parts[1] == 'parameters':
        parts[1] = 'parameterValues'

    # if outputs, parameter output_file --> outputFile
    if parts[0] == 'outputs' and parts[1] == 'parameters' and parts[
            3] == 'output_file':
        parts[3] = 'outputFile'

    # if artifacts...
    if parts[1] == 'artifacts':

        # ...need to get nested artifact object...
        parts.insert(3, 'artifacts')
        # ...and first entry in list with index 0
        parts.insert(4, 0)

        # for local, path is the uri
        if parts[5] == 'path':
            parts[5] = 'uri'

    return parts


def resolve_io_placeholders(
    executor_input: Dict[str, Any],
    command: str,
) -> str:
    placeholders = re.findall(r'\{\{\$\.(.*?)\}\}', command)

    # e.g., placeholder = "inputs.parameters[''text'']"
    for placeholder in placeholders:
        if 'json_escape' in placeholder:
            raise ValueError('JSON escape placeholders are not supported.')

        # e.g., parts = ['inputs', 'parameters', '', 'text', '', '']
        parts = re.split(r'\.|\[|\]|\'\'|\'', placeholder)

        # e.g., nonempty_parts = ['inputs', 'parameters', 'text']
        nonempty_parts = [part for part in parts if part]

        # e.g., path = ['inputs', 'parameterValues', 'text']
        path = convert_placeholder_parts_to_path(nonempty_parts)

        # e.g., path = ['inputs', 'parameterValues', 'text']
        value = get_value_using_path(executor_input, path)
        if not isinstance(value, str):
            # even if value is None, should json.dumps to null
            # and still resolve placeholder
            value = json.dumps(value)
        command = command.replace('{{$.' + placeholder + '}}', value)

    return command


def resolve_struct_placeholders(
    placeholder: str,
    provided_inputs: List[str],
) -> List[Any]:
    """Resolves IfPresent and Concat placeholders to an arbitrarily deeply
    nested list of strings, which may contain None."""

    # throughout, filter out None for the case where IfPresent False and no else
    def filter_none(l: List[Any]) -> List[Any]:
        return [e for e in l if e is not None]

    def recursively_resolve_struct(placeholder: Dict[str, Any]) -> str:
        if isinstance(placeholder, str):
            return placeholder
        elif isinstance(placeholder, list):
            raise ValueError(
                f"You have an incorrectly nested {dsl.IfPresentPlaceholder!r} with a list provided for 'then' or 'else'."
            )

        first_key = list(placeholder.keys())[0]
        if first_key == 'Concat':
            concat = [
                recursively_resolve_struct(p) for p in placeholder['Concat']
            ]
            return ''.join(filter_none(concat))
        elif first_key == 'IfPresent':
            inner_struct = placeholder['IfPresent']
            if inner_struct['InputName'] in provided_inputs:
                then = inner_struct['Then']
                if isinstance(then, str):
                    return then
                elif isinstance(then, list):
                    return filter_none(
                        [recursively_resolve_struct(p) for p in then])
                elif isinstance(then, dict):
                    return recursively_resolve_struct(then)
            else:
                else_ = inner_struct.get('Else')
                if else_ is None:
                    return else_
                if isinstance(else_, str):
                    return else_
                elif isinstance(else_, list):
                    return filter_none(
                        [recursively_resolve_struct(p) for p in else_])
                elif isinstance(else_, dict):
                    return recursively_resolve_struct(else_)
        else:
            raise ValueError

    if placeholder.startswith('{"Concat": ') or placeholder.startswith(
            '{"IfPresent": '):
        des_placeholder = json.loads(placeholder)
        return recursively_resolve_struct(des_placeholder)
    else:
        return placeholder


def resolve_individual_placeholder(
    element: str,
    executor_input_dict: Dict[str, Any],
    pipeline_resource_name: str,
    task_resource_name: str,
    pipeline_root: str,
    pipeline_job_id: str,
    pipeline_task_id: str,
) -> str:
    """Replaces placeholders for a single element."""
    # match on literal for constant placeholders
    PLACEHOLDERS = {
        r'{{$.outputs.output_file}}':
            executor_input_dict['outputs']['outputFile'],
        r'{{$.outputMetadataUri}}':
            executor_input_dict['outputs']['outputFile'],
        r'{{$}}':
            json.dumps(executor_input_dict),
        dsl.PIPELINE_JOB_NAME_PLACEHOLDER:
            pipeline_resource_name,
        dsl.PIPELINE_JOB_ID_PLACEHOLDER:
            pipeline_job_id,
        dsl.PIPELINE_TASK_NAME_PLACEHOLDER:
            task_resource_name,
        dsl.PIPELINE_TASK_ID_PLACEHOLDER:
            pipeline_task_id,
        dsl.PIPELINE_ROOT_PLACEHOLDER:
            pipeline_root,
    }
    for placeholder, value in PLACEHOLDERS.items():
        element = element.replace(placeholder, value)

    # match non-constant placeholders (i.e., have key(s))
    return resolve_io_placeholders(executor_input_dict, element)
