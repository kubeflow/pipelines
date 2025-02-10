# Copyright 2022 The Kubeflow Authors
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

import re
from typing import Callable


def get_param_descr(fn: Callable, param_name: str) -> str:
    """Extracts the description of a parameter from the docstring of a function
    or method. Docstring must conform to Google Style (https://sphinxcontrib-
    napoleon.readthedocs.io/en/latest/example_google.html).

    Args:
        fn (Callable): The function of method with a __doc__ docstring implemented.
        param_name (str): The parameter for which to extract the description.

    Returns:
        str: The description of the parameter.
    """
    docstring = fn.__doc__

    if docstring is None:
        raise ValueError(
            f'Could not find parameter {param_name} in docstring of {fn}')
    lines = docstring.splitlines()

    # Find Args section
    for i, line in enumerate(lines):
        if line.lstrip().startswith('Args:'):
            break
    else:  # No Args section found
        raise ValueError(f'No Args section found in docstring of {fn}')

    lines = lines[i + 1:]
    first_line_args_regex = rf'^{param_name}( \(.*\))?: '
    for i, line in enumerate(lines):
        stripped = line.lstrip()
        match = re.match(first_line_args_regex, stripped)
        if match:
            description = [re.sub(first_line_args_regex, '', stripped)]
            # Collect any additional lines that are more indented
            current_indent = len(line) - len(stripped)
            for next_line in lines[i + 1:]:
                next_indent = len(next_line) - len(next_line.lstrip())
                if next_indent <= current_indent:
                    break
                description.append(next_line.strip())
            return ' '.join(description)
    raise ValueError(
        f'Could not find parameter {param_name} in docstring of {fn}')
