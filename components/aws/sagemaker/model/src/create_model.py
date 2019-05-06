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
  parser.add_argument('--region', type=str, help='The region where the cluster launches.')
  parser.add_argument('--image', type=str, help='The Amazon EC2 Container Registry (Amazon ECR) path where inference code is stored.')
  parser.add_argument('--model_artifact_url', type=str, help='S3 model artifacts url')
  parser.add_argument('--model_name', type=str, help='The name of the new model.')
  parser.add_argument('--role', type=str, help='The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.')
  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_client(args.region)

  logging.info('Submitting model creation request to SageMaker...')
  _utils.create_model(
      client, args.model_artifact_url, args.model_name, args.image, args.role)

  logging.info('Model creation completed.')
  with open('/tmp/model_name.txt', 'w') as f:
    f.write(args.model_name)


if __name__== "__main__":
  main()
