# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""GCP remote runner for templated custom jobs based on the AI Platform SDK."""

import json
from typing import Any, Callable, Dict, List, Optional
import jinja2
from jinja2 import sandbox

# Note that type annotations need to match the python version in the GCPC docker
# image in addition to the internal python version.


def _json_escape_filter(value: str) -> str:
  """A Jinja2 filter for JSON escaping."""
  return json.dumps(value)[1:-1]


def render_payload(payload_str: str, params: Dict[str, Any]) -> str:
  """Renders a base64-encoded Custom Job payload in Jinja2 format."""
  env = sandbox.ImmutableSandboxedEnvironment(
      autoescape=False, undefined=jinja2.StrictUndefined
  )

  # We add an additional `json_dumps` filter because the builtin filter
  # `tojson`, which is implemented as `htmlsafe_json_dumps`, converts special
  # symbols to `\u` format, e.g., `&` -> `\u0026`, `<` -> `\u003c`.
  env.filters['json_dumps'] = json.dumps
  env.filters['json_escape'] = _json_escape_filter

  template = env.from_string(payload_str)
  return template.render(**params)


def convert_key_value_param_list(
    param_list: Optional[List[List[str]]],
    type_cast: Callable[[str], Any],
    cmd_flag: str,
) -> Dict[str, Any]:
  """Converts a list of (key, [value]) pairs to a dictionary.

  Args:
      param_list: A list of (key, [value]) pairs of string type.
      type_cast: A function to cast `value`, if exists, from string to a
        specific type.
      cmd_flag: The command-line flag for this list of parameters. This is used
        to provide better error message when raising an exception.

  Returns:
      A dictionary of the converted (key, value) pairs.
  """
  params = {}
  if not param_list:
    return params
  for key_value in param_list:
    if 1 <= len(key_value) <= 2:
      key = key_value[0]
      if 1 == len(key_value):
        params[key] = None
      else:
        try:
          params[key] = type_cast(key_value[1])
        except json.JSONDecodeError as e:
          raise ValueError(
              f'Cannot decode value for [{key}]: {key_value[1]}'
          ) from e
    else:
      raise ValueError(
          f'{cmd_flag} can only take 1 or 2 params, but found {key_value}'
      )
  return params


def convert_integer_params(
    integer_params: Optional[List[List[str]]],
) -> Dict[str, Optional[int]]:
  """Converts a list of (key, [integer]) pairs to a dictionary."""
  return convert_key_value_param_list(
      param_list=integer_params, type_cast=int, cmd_flag='--set_integer'
  )


def convert_string_params(
    string_params: Optional[List[List[str]]],
) -> Dict[str, Optional[str]]:
  """Converts a list of (key, [string]) pairs to a dictionary."""
  return convert_key_value_param_list(
      param_list=string_params, type_cast=str, cmd_flag='--set_string'
  )


def convert_float_params(
    float_params: Optional[List[List[str]]],
) -> Dict[str, Optional[float]]:
  """Converts a list of (key, [float]) pairs to a dictionary."""
  return convert_key_value_param_list(
      param_list=float_params, type_cast=float, cmd_flag='--set_float'
  )


def convert_boolean_params(
    boolean_params: Optional[List[List[str]]],
) -> Dict[str, Optional[bool]]:
  """Converts a list of (key, [boolean]) pairs to a dictionary."""
  return convert_key_value_param_list(
      param_list=boolean_params,
      type_cast=lambda x: x.lower() in ['1', 'true', 'y', 'yes'],
      cmd_flag='--set_boolean',
  )


def convert_json_params(
    json_params: Optional[List[List[str]]],
) -> Dict[str, Any]:
  """Converts a list of (key, [json objects]) pairs to a dictionary."""
  return convert_key_value_param_list(
      param_list=json_params, type_cast=json.loads, cmd_flag='--set_json'
  )
