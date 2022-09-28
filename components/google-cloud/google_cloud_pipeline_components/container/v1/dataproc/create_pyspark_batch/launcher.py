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
"""GCP launcher for Dataproc Create Pyspark Batch."""

import argparse
import logging
import sys

from . import remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import parser_util


def _parse_args(args):
  """Parse command line arguments."""
  parser, parsed_args = parser_util.parse_default_args(args)
  # Parse the conditionally required arguments
  parser.add_argument(
      '--batch_id',
      dest='batch_id',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
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

  if job_type != 'DataprocPySparkBatch':
    raise ValueError('Incorrect job type: ' + job_type)

  logging.info('Job started for type: ' + job_type)

  remote_runner.create_pyspark_batch(**parsed_args)


if __name__ == '__main__':
  main(sys.argv[1:])
