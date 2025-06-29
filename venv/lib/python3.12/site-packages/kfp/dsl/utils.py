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
"""Definitions of utils methods."""

import importlib
import os
import re
import sys
import types
from typing import List

COMPONENT_NAME_PREFIX = 'comp-'
_EXECUTOR_LABEL_PREFIX = 'exec-'


def load_module(module_name: str, module_directory: str) -> types.ModuleType:
    """Dynamically imports the Python module with the given name and package
    path.

    E.g., Assuming there is a file called `my_module.py` under
    `/some/directory/my_module`, we can use::

        load_module('my_module', '/some/directory')

    to effectively `import mymodule`.

    Args:
        module_name: The name of the module.
        package_path: The package under which the specified module resides.
    """
    module_spec = importlib.util.spec_from_file_location(
        name=module_name,
        location=os.path.join(module_directory, f'{module_name}.py'))
    module = importlib.util.module_from_spec(module_spec)
    sys.modules[module_spec.name] = module
    sys.path.insert(0, str(module_directory))
    module_spec.loader.exec_module(module)
    return module


def maybe_rename_for_k8s(name: str) -> str:
    """Cleans and converts a name to be k8s compatible.

    Args:
      name: The original name.

    Returns:
      A sanitized name.
    """
    return re.sub('-+', '-', re.sub('[^-0-9a-z]+', '-',
                                    name.lower())).lstrip('-').rstrip('-')


def sanitize_input_name(name: str) -> str:
    """Sanitizes input name."""
    return re.sub('[^_0-9a-z]+', '_', name.lower()).lstrip('_').rstrip('_')


def sanitize_component_name(name: str) -> str:
    """Sanitizes component name."""
    return COMPONENT_NAME_PREFIX + maybe_rename_for_k8s(name)


def sanitize_task_name(name: str) -> str:
    """Sanitizes task name."""
    return maybe_rename_for_k8s(name)


def sanitize_executor_label(label: str) -> str:
    """Sanitizes executor label."""
    return _EXECUTOR_LABEL_PREFIX + maybe_rename_for_k8s(label)


def make_name_unique_by_adding_index(
    name: str,
    collection: List[str],
    delimiter: str,
) -> str:
    """Makes a unique name by adding index.

    The index starts from 2 and increase by 1 until we find a unique name.

    Args:
        name: The original name.
        collection: The collection of existing names.
        delimiter: The delimiter to connect the original name and an index.

    Returns:
        A unique name composed of name+delimiter+next index
    """
    unique_name = name
    if unique_name in collection:
        for i in range(2, sys.maxsize**10):
            unique_name = name + delimiter + str(i)
            if unique_name not in collection:
                break
    return unique_name


def validate_pipeline_name(name: str) -> None:
    """Validate pipeline name.

    A valid pipeline name should match ^[a-z0-9][a-z0-9-]{0,127}$.

    Args:
        name: The pipeline name.

    Raises:
        ValueError if the pipeline name doesn't conform to the regular expression.
    """
    pattern = re.compile(r'^[a-z0-9][a-z0-9-]{0,127}$')
    if not pattern.match(name):
        raise ValueError(
            'Invalid pipeline name: %s.\n'
            'Please specify a pipeline name that matches the regular '
            'expression "^[a-z0-9][a-z0-9-]{0,127}$" using '
            '`dsl.pipeline(name=...)` decorator.' % name)
