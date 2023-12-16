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
from typing import Any, Dict, List

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
    for placeholder, value in PLACEHOLDERS.items():
        element = element.replace(placeholder, value)

    return element
