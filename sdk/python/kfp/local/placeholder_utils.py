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
from typing import Any, Dict, List, Optional

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
    return [
        replace_placeholder_for_element(
            element=el,
            executor_input_dict=executor_input_dict,
            pipeline_resource_name=pipeline_resource_name,
            task_resource_name=task_resource_name,
            pipeline_root=pipeline_root,
            pipeline_job_id=unique_pipeline_id,
            pipeline_task_id=unique_task_id,
        ) for el in full_command
    ]


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
        if value is not None:
            if not isinstance(value, str):
                value = json.dumps(value)
            command = command.replace('{{$.' + placeholder + '}}', value)

    return command


# TODO: support concat and if-present placeholders
def replace_placeholder_for_element(
    element: str,
    executor_input_dict: Dict[str, Any],
    pipeline_resource_name: str,
    task_resource_name: str,
    pipeline_root: str,
    pipeline_job_id: str,
    pipeline_task_id: str,
) -> str:
    """Replaces placeholders for a single element."""
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

    # match on literal for constant placeholders
    for placeholder, value in PLACEHOLDERS.items():
        element = element.replace(placeholder, value)

    # match differently for non-constant placeholders (i.e., have key(s))
    return resolve_io_placeholders(executor_input_dict, element)
