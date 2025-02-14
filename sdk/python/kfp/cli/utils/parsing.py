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

    Raises:
        ValueError: If docstring is not found or parameter is not found in docstring.

    Returns:
        str: The description of the parameter.
    """
    docstring = fn.__doc__

    if docstring is None:
        raise ValueError(
            f'Could not find parameter {param_name} in docstring of {fn}')
    lines = docstring.splitlines()

    # collect all lines beginning after args, also get indentation space_chars
    for i, line in enumerate(lines):
        if line.lstrip().startswith('Args:'):
            break

    lines = lines[i + 1:]

    first_already_found = False
    return_lines = []

    # allow but don't require type in docstring
    first_line_args_regex = rf'^{param_name}( \(.*\))?: '
    for line in lines:
        if not first_already_found and re.match(first_line_args_regex,
                                                line.lstrip()):
            new_line = re.sub(first_line_args_regex, '', line.strip())
            return_lines.append(new_line)
            first_already_found = True
            first_indentation_level = len(line) - len(line.lstrip())
            continue

        if first_already_found:
            indentation_level = len(line) - len(line.lstrip())
            if indentation_level <= first_indentation_level:
                return ' '.join(return_lines)
            else:
                return_lines.append(line.strip())
    raise ValueError(
        f'Could not find parameter {param_name} in docstring of {fn}')
