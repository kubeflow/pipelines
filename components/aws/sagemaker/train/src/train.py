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

from common import _utils

def main(argv=None):
  parser = argparse.ArgumentParser(description='SageMaker Training Job')
  parser.add_argument('--region', type=str, help='The region where the training job launches.')
  parser.add_argument('--image', type=str, help='The registry path of the Docker image that contains the training algorithm.')
  parser.add_argument('--instance_type', type=str, help='The ML compute instance type.')
  parser.add_argument('--instance_count', type=int, help='The registry path of the Docker image that contains the training algorithm.')
  parser.add_argument('--volume_size', type=int, help='The size of the ML storage volume that you want to provision.')
  parser.add_argument('--dataset_path', type=str, help='The S3 location of the data source that is associated with a channel.')
  parser.add_argument('--model_artifact_path', type=str, help='Identifies the S3 path where you want Amazon SageMaker to store the model artifacts.')
  parser.add_argument('--role', type=str, help='The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.')
  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_client(args.region)

  logging.info('Submitting Training Job to SageMaker...')
  job_name = _utils.create_training_job(
      client, args.image, args.instance_type, args.instance_count, args.volume_size, args.dataset_path, args.model_artifact_path, args.role)
  logging.info('Job request submitted. Waiting for completion...')
  _utils.wait_for_training_job(client, job_name)

  model_artifact_url = _utils.get_model_artifacts_from_job(client, job_name)
  logging.info('Get model artifacts %s from training job %s.', model_artifact_url, job_name)

  with open('/tmp/model_artifact_url.txt', 'w') as f:
    f.write(model_artifact_url)
  with open('/tmp/job_name.txt', 'w') as f:
    f.write(job_name)

  logging.info('Job completed.')


if __name__== "__main__":
  main()
