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
"""GCP launcher for AutoML image training jobs based on the AI Platform SDK."""

import argparse
import json
import logging
import sys
from typing import List

from google_cloud_pipeline_components.container.v1.automl_training_job.image import remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import parser_util


def _parse_args(args: List[str]):
  """Parse command line arguments."""
  args.append('--payload')
  args.append('"{}"')  # Unused but required by parser_util.
  parser, _ = parser_util.parse_default_args(args)
  # Parse the conditionally required arguments.
  parser.add_argument(
      '--executor_input',
      dest='executor_input',
      type=str,
      # executor_input is only needed for components that emit output artifacts.
      required=True,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--display_name',
      dest='display_name',
      type=str,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--prediction_type',
      dest='prediction_type',
      type=str,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--multi_label',
      dest='multi_label',
      type=parser_util.parse_bool,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--model_type',
      dest='model_type',
      type=str,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--labels',
      dest='labels',
      type=json.loads,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--dataset',
      dest='dataset',
      type=str,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--disable_early_stopping',
      dest='disable_early_stopping',
      type=parser_util.parse_bool,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--training_encryption_spec_key_name',
      dest='training_encryption_spec_key_name',
      type=str,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--model_encryption_spec_key_name',
      dest='model_encryption_spec_key_name',
      type=str,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--model_display_name',
      dest='model_display_name',
      type=str,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--training_fraction_split',
      dest='training_fraction_split',
      type=float,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--validation_fraction_split',
      dest='validation_fraction_split',
      type=float,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--test_fraction_split',
      dest='test_fraction_split',
      type=float,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--budget_milli_node_hours',
      dest='budget_milli_node_hours',
      type=int,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--training_filter_split',
      dest='training_filter_split',
      type=str,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--validation_filter_split',
      dest='validation_filter_split',
      type=str,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--test_filter_split',
      dest='test_filter_split',
      type=str,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--base_model',
      dest='base_model',
      type=str,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--incremental_train_base_model',
      dest='incremental_train_base_model',
      type=str,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--parent_model',
      dest='parent_model',
      type=str,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--is_default_version',
      dest='is_default_version',
      type=parser_util.parse_bool,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--model_version_aliases',
      dest='model_version_aliases',
      type=json.loads,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--model_version_description',
      dest='model_version_description',
      type=str,
      required=False,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--model_labels',
      dest='model_labels',
      type=json.loads,
      required=False,
      default=argparse.SUPPRESS,
  )
  parsed_args, _ = parser.parse_known_args(args)
  args_dict = vars(parsed_args)
  del args_dict['payload']
  return args_dict


def main(argv: List[str]):
  """Main entry.

  Expected input args are as follows:
    Project - Required. The project of which the resource will be launched.
    Region - Required. The region of which the resource will be launched.
    Type - Required. GCP launcher is a single container. This Enum will
        specify which resource to be launched.
    gcp_resources - placeholder output for returning job_id.
    Extra arguments - For constructing request payload. See remote_runner.py for
    more information.

  Args:
    argv: A list of system arguments.
  """
  parsed_args = _parse_args(argv)
  job_type = parsed_args['type']

  if job_type != 'AutoMLImageTrainingJob':
    raise ValueError('Incorrect job type: ' + job_type)

  logging.info(
      'Starting AutoMLImageTrainingJob using the following arguments: %s',
      parsed_args,
  )

  remote_runner.create_pipeline(**parsed_args)


if __name__ == '__main__':
  main(sys.argv[1:])
