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

from common import _utils

def create_parser():
  parser = argparse.ArgumentParser(description='SageMaker Hyperparameter Tuning Job')
  _utils.add_default_client_arguments(parser)
  
  parser.add_argument('--team_name', type=str.strip, required=True, help='The name of your work team.')
  parser.add_argument('--description', type=str.strip, required=True, help='A description of the work team.')
  parser.add_argument('--user_pool', type=str.strip, required=False, help='An identifier for a user pool. The user pool must be in the same region as the service that you are calling.', default='')
  parser.add_argument('--user_groups', type=str.strip, required=False, help='A list of identifiers for user groups separated by commas.', default='')
  parser.add_argument('--client_id', type=str.strip, required=False, help='An identifier for an application client. You must create the app client ID using Amazon Cognito.', default='')
  parser.add_argument('--sns_topic', type=str.strip, required=False, help='The ARN for the SNS topic to which notifications should be published.', default='')
  parser.add_argument('--tags', type=_utils.str_to_json_dict, required=False, help='An array of key-value pairs, to categorize AWS resources.', default='{}')

  return parser

def main(argv=None):
  parser = create_parser()
  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_sagemaker_client(args.region, args.endpoint_url)
  logging.info('Submitting a create workteam request to SageMaker...')
  workteam_arn = _utils.create_workteam(client, vars(args))

  logging.info('Workteam created.')

  with open('/tmp/workteam_arn.txt', 'w') as f:
    f.write(workteam_arn)


if __name__== "__main__":
  main()
