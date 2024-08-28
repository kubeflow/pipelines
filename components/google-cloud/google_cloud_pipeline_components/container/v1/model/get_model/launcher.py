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
"""GCP launcher for Get Model based on the AI Platform SDK."""

import argparse
import sys

from google_cloud_pipeline_components.container.v1.model.get_model import remote_runner


def _parse_args(args):
  """Parse command line arguments."""
  parser = argparse.ArgumentParser(
      prog='Vertex Pipelines get model launcher', description=''
  )
  parser.add_argument(
      '--executor_input',
      dest='executor_input',
      type=str,
      # executor_input is only needed for components that emit output artifacts.
      required=True,
      default=argparse.SUPPRESS,
  )
  parser.add_argument('--project', dest='project', type=str)
  parser.add_argument('--location', dest='location', type=str)
  parser.add_argument('--model_name', dest='model_name', type=str)
  parsed_args, _ = parser.parse_known_args(args)
  return vars(parsed_args)


def main(argv):
  """Main entry.

  Expected input args are as follows:
    model_name - Required. Provided string resource name to create a model
    artifact.
    project - Required. Project to get this Model from.
    location - Required. Location to get this Model from.

  Args:
    argv: A list of system arguments.
  """
  parsed_args = _parse_args(argv)
  remote_runner.get_model(**parsed_args)


if __name__ == '__main__':
  main(sys.argv[1:])
