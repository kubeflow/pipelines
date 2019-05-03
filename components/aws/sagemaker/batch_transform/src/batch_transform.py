# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import argparse
import logging
import random
from datetime import datetime
from pathlib2 import Path

from common import _utils


def main(argv=None):
  parser = argparse.ArgumentParser(description='SageMaker Batch Transformation Job')
  parser.add_argument('--region', type=str, help='The region where the cluster launches.')
  parser.add_argument('--model_name', type=str, help='The name of the model that you want to use for the transform job.')
  parser.add_argument('--input_location', type=str, help='The S3 location of the data source that is associated with a channel.')
  parser.add_argument('--output_location', type=str, help='The Amazon S3 path where you want Amazon SageMaker to store the results of the transform job.')
  parser.add_argument('--output_location_file', type=str, help='File path where the program will write the Amazon S3 URI of the transform job results.')

  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_client(args.region)
  logging.info('Submitting Batch Transformation request to SageMaker...')
  batch_job_name = _utils.create_transform_job(
      client, args.model_name, args.input_location, args.output_location)
  logging.info('Batch Job request submitted. Waiting for completion...')
  _utils.wait_for_transform_job(client, batch_job_name)
  _utils.print_tranformation_job_result(args.output_location)

  Path(args.output_location_file).parent.mkdir(parents=True, exist_ok=True)
  Path(args.output_location_file).write_text(unicode(args.output_location))

  logging.info('Batch Transformation creation completed.')


if __name__== "__main__":
  main()
