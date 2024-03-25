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
"""Cancel Job based on the AI Platform SDK."""

import argparse
import logging
import os
import sys

from google_cloud_pipeline_components.container.v1.cancel_job import remote_runner


def _make_parent_dirs_and_return_path(file_path: str):
  os.makedirs(os.path.dirname(file_path), exist_ok=True)
  return file_path


def _parse_args(args):
  """Parse command line arguments."""
  parser = argparse.ArgumentParser(
      prog='Dataflow python job Pipelines service launcher', description=''
  )
  parser.add_argument(
      '--gcp_resources',
      dest='gcp_resources',
      type=_make_parent_dirs_and_return_path,
      required=True,
      default=argparse.SUPPRESS,
  )
  parsed_args, _ = parser.parse_known_args(args)
  return vars(parsed_args)


def main(argv):
  """Main entry.

  Expected input args are as follows:
    gcp_resources - placeholder input for returning job_id.

  Args:
    argv: A list of system arguments.
  """
  logging.info('Job started for type: cancelJob')

  parsed_args = _parse_args(argv)
  gcp_resources = parsed_args['gcp_resources']
  remote_runner.cancel_job(gcp_resources)


if __name__ == '__main__':
  main(sys.argv[1:])
