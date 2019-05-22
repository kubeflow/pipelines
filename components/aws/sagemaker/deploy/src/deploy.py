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
  parser.add_argument('--model_name', type=str, help='The name of the new model.')
  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_client(args.region)
  logging.info('Submitting Endpoint request to SageMaker...')
  endpoint_name = _utils.deploy_model(client, args.model_name)

  with open('/tmp/endpoint_name.txt', 'w') as f:
    f.write(endpoint_name)

  logging.info('Endpoint creation completed.')


if __name__== "__main__":
  main()
