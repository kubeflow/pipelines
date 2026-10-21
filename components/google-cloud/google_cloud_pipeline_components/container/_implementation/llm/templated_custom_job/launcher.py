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
"""Jinja2-based templated launcher for custom jobs.

Note: Whenever possible, please prefer the vanilla custom_job launcher instead
of this Jinja2-based templated custom job launcher. This launcher does not take
advantage of the Vertex Pipeline backend optimization and will thus launch a
Custom Job that runs another Custom Job.
"""

import argparse
import logging
import sys
from typing import Any, Dict, List

from google_cloud_pipeline_components.container._implementation.llm.templated_custom_job import remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import parser_util


def _parse_args(args: List[str]) -> Dict[str, Any]:
  """Parse command line arguments."""
  parser, _ = parser_util.parse_default_args(args)
  parser.add_argument(
      '--dry_run',
      dest='dry_run',
      action='store_true',
      help=(
          'If set, log the rendered payload for the Custom Job and exit with '
          'error code 1'
      ),
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--set_integer',
      dest='set_integer',
      action='append',
      nargs='+',
      help='(key, [value]) pairs. If `value` is missing, the value is None',
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--set_string',
      dest='set_string',
      action='append',
      nargs='+',
      help='(key, [value]) pairs. If `value` is missing, the value is None',
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--set_float',
      dest='set_float',
      action='append',
      nargs='+',
      help='(key, [value]) pairs. If `value` is missing, the value is None',
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--set_boolean',
      dest='set_boolean',
      action='append',
      nargs='+',
      help='(key, [value]) pairs. If `value` is missing, the value is None',
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--set_json',
      dest='set_json',
      action='append',
      nargs='+',
      help='(key, [value]) pairs. If `value` is missing, the value is None',
      default=argparse.SUPPRESS,
  )
  parsed_args, _ = parser.parse_known_args(args)
  return vars(parsed_args)


def main(argv: List[str]) -> None:
  """Main entry.

  Expected input args are as follows:
    project - Required. The project of which the resource will be launched.
    location - Required. The region of which the resource will be launched.
    type - Required. GCPC launcher is a single container. This Enum will specify
      which resource to be launched.
    payload - Required. The base64-encoded Jinja2 payload that will be rendered
      to a full serialized JSON of the resource spec. This payload normally
      doesn't contain Pipeline Placehoders.
    gcp_resources - Required. Placeholder output for returning job_id.
    dry_run - If set, log the rendered payload for the Custom Job and exit with
      error code 1.
    set_integer - A list of `(key, [value])` pairs of strings that'll be used to
      render the payload. The `value` will be cast to an integer. If `value` is
      missing, it'll be `None`. Note that `value` can contain Pipeline
      Placeholders.
    set_string - A list of (key, [value]) pairs of strings that'll be used to
      render the payload. If `value` is missing, it'll be `None`. Note that
      `value` can contain Pipeline Placeholders.
    set_float - A list of `(key, [value])` pairs of strings that'll be used to
      render the payload. The `value` will be cast to a float number. If `value`
      is missing, it'll be `None`. Note that `value` can contain Pipeline
      Placeholders.
    set_boolean - A list of `(key, [value])` pairs of strings that'll be used to
      render the payload. The `value` will be cast to a boolean value. If
      `value` is missing, it'll be `None`. Note that `value` can contain
      Pipeline Placeholders.
    set_json - A list of `(key, [value])` pairs of strings that'll be used to
      render the payload. The `value` will be cast to an object by calling
      `json.loads()`. If `value` is missing, it'll be `None`. Note that `value`
      can contain Pipeline Placeholders.

  Args:
    argv: A list of system arguments.
  """
  parsed_args = _parse_args(argv)
  logging.basicConfig(
      format='[%(asctime)s] [%(levelname)s]: %(message)s', level=logging.INFO
  )
  job_type = parsed_args['type']
  if job_type != 'TemplatedCustomJob':
    raise ValueError('Incorrect job type: ' + job_type)

  logging.info('Job started for type: %s', job_type)

  remote_runner.create_templated_custom_job(**parsed_args)


if __name__ == '__main__':
  main(sys.argv[1:])
