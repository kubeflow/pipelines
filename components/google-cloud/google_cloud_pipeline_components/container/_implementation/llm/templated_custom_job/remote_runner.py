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

import base64
import json
import logging
import sys
from typing import Any, Callable, Dict, List, Optional
import google_cloud_pipeline_components.google_cloud_pipeline_components.container.v1.custom_job.remote_runner as custom_job_remote_runner
import jinja2
from jinja2 import sandbox

# Note that type annotations need to match the python version in the GCPC docker
# image in addition to the internal python version.

ParamListType = Optional[List[List[str]]]


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
    integer_params: ParamListType,
) -> Dict[str, Optional[int]]:
  """Converts a list of (key, [integer]) pairs to a dictionary."""
  return convert_key_value_param_list(
      param_list=integer_params, type_cast=int, cmd_flag='--set_integer'
  )


def convert_string_params(
    string_params: ParamListType,
) -> Dict[str, Optional[str]]:
  """Converts a list of (key, [string]) pairs to a dictionary."""
  return convert_key_value_param_list(
      param_list=string_params, type_cast=str, cmd_flag='--set_string'
  )


def convert_float_params(
    float_params: ParamListType,
) -> Dict[str, Optional[float]]:
  """Converts a list of (key, [float]) pairs to a dictionary."""
  return convert_key_value_param_list(
      param_list=float_params, type_cast=float, cmd_flag='--set_float'
  )


def convert_boolean_params(
    boolean_params: ParamListType,
) -> Dict[str, Optional[bool]]:
  """Converts a list of (key, [boolean]) pairs to a dictionary."""
  return convert_key_value_param_list(
      param_list=boolean_params,
      type_cast=lambda x: x.lower() in ['1', 'true', 'y', 'yes'],
      cmd_flag='--set_boolean',
  )


def convert_json_params(
    json_params: ParamListType,
) -> Dict[str, Any]:
  """Converts a list of (key, [json objects]) pairs to a dictionary."""
  return convert_key_value_param_list(
      param_list=json_params, type_cast=json.loads, cmd_flag='--set_json'
  )


# This method will also be used for unit tests.
def decode_and_render_payload(
    payload: str,
    int_params: ParamListType = None,
    string_params: ParamListType = None,
    float_params: ParamListType = None,
    boolean_params: ParamListType = None,
    json_params: ParamListType = None,
) -> str:
  """Decodes base64-encoded Jinja2 payload and renders it."""
  params = convert_integer_params(int_params)
  params.update(convert_string_params(string_params))
  params.update(convert_float_params(float_params))
  params.update(convert_boolean_params(boolean_params))
  params.update(convert_json_params(json_params))

  return render_payload(base64.b64decode(payload).decode('utf-8'), params)


def create_templated_custom_job(
    type: str,  # pylint: disable=redefined-builtin
    project: str,
    location: str,
    payload: str,
    gcp_resources: str,
    dry_run: bool = False,
    set_integer: ParamListType = None,
    set_string: ParamListType = None,
    set_float: ParamListType = None,
    set_boolean: ParamListType = None,
    set_json: ParamListType = None,
) -> None:
  """Creates and polls a Custom Job."""
  rendered_payload = decode_and_render_payload(
      payload=payload,
      int_params=set_integer,
      string_params=set_string,
      float_params=set_float,
      boolean_params=set_boolean,
      json_params=set_json,
  )

  # Call json.loads() to validate that the payload is a valid JSON.
  # Call json.dumps() instead of using rendered_payload to remove redundant
  # blank spaces in the payload.
  try:
    payload_str = json.dumps(json.loads(rendered_payload))
  except json.JSONDecodeError as e:
    logging.error(
        'Cannot deserialize the rendered payload to JSON: %r', rendered_payload
    )
    raise ValueError(
        'The rendered payload is an invalid JSON. Please see the error log for '
        'details.'
    ) from e

  if dry_run:
    logging.info(
        'Log rendered payload for dry run and exit with error code 1: %s',
        payload_str,
    )
    sys.exit(1)

  custom_job_remote_runner.create_custom_job(
      type=type,
      project=project,
      location=location,
      payload=payload_str,
      gcp_resources=gcp_resources,
  )
