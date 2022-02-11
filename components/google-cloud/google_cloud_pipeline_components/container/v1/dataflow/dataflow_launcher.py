# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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
"""Launcher module for dataflow componeonts."""

import argparse
import os
import sys
from typing import Dict, Any
from . import dataflow_python_job_remote_runner


def _make_parent_dirs_and_return_path(file_path: str):
  os.makedirs(os.path.dirname(file_path), exist_ok=True)
  return file_path


def _parse_args(args) -> Dict[str, Any]:
  """Parse command line arguments.

  Args:
    args: A list of arguments.

  Returns:
    A tuple containing an argparse.Namespace class instance holding parsed args,
    and a list containing all unknonw args.
  """
  parser = argparse.ArgumentParser(
      prog='Dataflow python job Pipelines service launcher', description='')
  parser.add_argument(
      '--project',
      dest='project',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--location',
      dest='location',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--python_module_path',
      dest='python_module_path',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--temp_location',
      dest='temp_location',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--requirements_file_path',
      dest='requirements_file_path',
      type=str,
      required=False,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--args',
      dest='args',
      type=str,
      required=False,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--gcp_resources',
      dest='gcp_resources',
      type=_make_parent_dirs_and_return_path,
      required=True,
      default=argparse.SUPPRESS)
  parsed_args, _ = parser.parse_known_args(args)
  return vars(parsed_args)


def main(argv):
  """Main entry for Dataflow python job launcher.

  expected input args are as follows:
  project - Required. The project of which the resource will be launched.
  Location - Required. The region of which the resource will be launched.
  python_module_path - The gcs path to the python file to run.
  temp_location - A GCS path for Dataflow to stage temporary job files created
  during the execution of the pipeline.
  requirements_file_path - The gcs or local path to the requirements file.
  args - The list of args to pass to the python file.
  gcp_resources - A placeholder output for returning the gcp_resouces proto.

  Args:
    argv: A list of system arguments.
  """

  parsed_args = _parse_args(argv)
  dataflow_python_job_remote_runner.create_python_job(**parsed_args)


if __name__ == '__main__':
  main(sys.argv[1:])
