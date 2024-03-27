# Copyright 2024 The Kubeflow Authors. All Rights Reserved.
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
"""GCP launcher for data labeling jobs based on the AI Platform SDK."""

import argparse
import logging
import os
import sys
from typing import Any, Dict

from google_cloud_pipeline_components.container._implementation.model_evaluation.data_labeling_job import remote_runner


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
      prog='Dataflow python job Pipelines service launcher', description=''
  )
  parser.add_argument(
      '--type',
      dest='type',
      type=str,
      required=True,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--project',
      dest='project',
      type=str,
      required=True,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--location',
      dest='location',
      type=str,
      required=True,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--gcp_resources',
      dest='gcp_resources',
      type=_make_parent_dirs_and_return_path,
      required=True,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--job_display_name',
      dest='job_display_name',
      type=str,
      required=True,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--dataset_name',
      dest='dataset_name',
      type=str,
      required=True,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--instruction_uri',
      dest='instruction_uri',
      type=str,
      required=True,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--inputs_schema_uri',
      dest='inputs_schema_uri',
      type=str,
      required=True,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--annotation_spec',
      dest='annotation_spec',
      type=str,
      required=True,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--labeler_count',
      dest='labeler_count',
      type=int,
      required=True,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--annotation_label',
      dest='annotation_label',
      type=str,
      required=True,
      default=argparse.SUPPRESS,
  )
  parsed_args, _ = parser.parse_known_args(args)
  return vars(parsed_args)


def main(argv):
  """Main entry.

  Expected input args are as follows:
    Project - Required. The project of which the resource will be launched.
    Region - Required. The region of which the resource will be launched.
    Type - Required. GCP launcher is a single container. This Enum will
        specify which resource to be launched.
    Request payload - Required. The full serialized json of the resource spec.
        Note this can contain the Pipeline Placeholders.
    gcp_resources - placeholder output for returning job_id.

  Args:
    argv: A list of system arguments.
  """
  parsed_args = _parse_args(argv)
  job_type = parsed_args['type']

  if job_type != 'DataLabelingJob':
    raise ValueError('Incorrect job type: ' + job_type)

  logging.info(
      'Starting DataLabelingJob using the following arguments: %s',
      parsed_args,
  )

  remote_runner.create_data_labeling_job(**parsed_args)


if __name__ == '__main__':
  main(sys.argv[1:])
