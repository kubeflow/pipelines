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

import logging
import sys

from google_cloud_pipeline_components.container.v1.cancel_job import remote_runner
# from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import parser_util


def main(argv):
  """Main entry.

  Expected input args are as follows:
    gcp_resources - placeholder input for returning job_id.

  Args:
    argv: A list of system arguments.
  """
  logging.info('Job started for type: cancelJob')

  remote_runner.cancel_job(argv)


if __name__ == '__main__':
  main(sys.argv[1:])
