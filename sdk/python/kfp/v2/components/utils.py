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
