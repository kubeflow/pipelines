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
  parser = argparse.ArgumentParser(description='SageMaker Training Job')
  _utils.add_default_client_arguments(parser)
  
  parser.add_argument('--endpoint_config_name', type=str.strip, required=False, help='The name of the endpoint configuration.', default='')
  parser.add_argument('--variant_name_1', type=str.strip, required=False, help='The name of the production variant.', default='variant-name-1')
  parser.add_argument('--model_name_1', type=str.strip, required=True, help='The model name used for endpoint deployment.')
  parser.add_argument('--initial_instance_count_1', type=_utils.str_to_int, required=False, help='Number of instances to launch initially.', default=1)
  parser.add_argument('--instance_type_1', choices=['ml.m4.xlarge', 'ml.m4.2xlarge', 'ml.m4.4xlarge', 'ml.m4.10xlarge', 'ml.m4.16xlarge', 'ml.m5.large', 'ml.m5.xlarge', 'ml.m5.2xlarge', 'ml.m5.4xlarge',
    'ml.m5.12xlarge', 'ml.m5.24xlarge', 'ml.c4.xlarge', 'ml.c4.2xlarge', 'ml.c4.4xlarge', 'ml.c4.8xlarge', 'ml.p2.xlarge', 'ml.p2.8xlarge', 'ml.p2.16xlarge', 'ml.p3.2xlarge', 'ml.p3.8xlarge', 'ml.p3.16xlarge',
    'ml.c5.xlarge', 'ml.c5.2xlarge', 'ml.c5.4xlarge', 'ml.c5.9xlarge', 'ml.c5.18xlarge', ''], type=str.strip, required=False, help='The ML compute instance type.', default='ml.m4.xlarge')
  parser.add_argument('--initial_variant_weight_1', type=_utils.str_to_float, required=False, help='Determines initial traffic distribution among all of the models that you specify in the endpoint configuration.', default=1.0)
  parser.add_argument('--accelerator_type_1', choices=['ml.eia1.medium', 'ml.eia1.large', 'ml.eia1.xlarge', ''], type=str.strip, required=False, help='The size of the Elastic Inference (EI) instance to use for the production variant.', default='')
  parser.add_argument('--variant_name_2', type=str.strip, required=False, help='The name of the production variant.', default='variant-name-2')
  parser.add_argument('--model_name_2', type=str.strip, required=False, help='The model name used for endpoint deployment.', default='')
  parser.add_argument('--initial_instance_count_2', type=_utils.str_to_int, required=False, help='Number of instances to launch initially.', default=1)
  parser.add_argument('--instance_type_2', choices=['ml.m4.xlarge', 'ml.m4.2xlarge', 'ml.m4.4xlarge', 'ml.m4.10xlarge', 'ml.m4.16xlarge', 'ml.m5.large', 'ml.m5.xlarge', 'ml.m5.2xlarge', 'ml.m5.4xlarge',
    'ml.m5.12xlarge', 'ml.m5.24xlarge', 'ml.c4.xlarge', 'ml.c4.2xlarge', 'ml.c4.4xlarge', 'ml.c4.8xlarge', 'ml.p2.xlarge', 'ml.p2.8xlarge', 'ml.p2.16xlarge', 'ml.p3.2xlarge', 'ml.p3.8xlarge', 'ml.p3.16xlarge',
    'ml.c5.xlarge', 'ml.c5.2xlarge', 'ml.c5.4xlarge', 'ml.c5.9xlarge', 'ml.c5.18xlarge', ''], type=str.strip, required=False, help='The ML compute instance type.', default='ml.m4.xlarge')
  parser.add_argument('--initial_variant_weight_2', type=_utils.str_to_float, required=False, help='Determines initial traffic distribution among all of the models that you specify in the endpoint configuration.', default=1.0)
  parser.add_argument('--accelerator_type_2', choices=['ml.eia1.medium', 'ml.eia1.large', 'ml.eia1.xlarge', ''], type=str.strip, required=False, help='The size of the Elastic Inference (EI) instance to use for the production variant.', default='')
  parser.add_argument('--variant_name_3', type=str.strip, required=False, help='The name of the production variant.', default='variant-name-3')
  parser.add_argument('--model_name_3', type=str.strip, required=False, help='The model name used for endpoint deployment.', default='')
  parser.add_argument('--initial_instance_count_3', type=_utils.str_to_int, required=False, help='Number of instances to launch initially.', default=1)
  parser.add_argument('--instance_type_3', choices=['ml.m4.xlarge', 'ml.m4.2xlarge', 'ml.m4.4xlarge', 'ml.m4.10xlarge', 'ml.m4.16xlarge', 'ml.m5.large', 'ml.m5.xlarge', 'ml.m5.2xlarge', 'ml.m5.4xlarge',
    'ml.m5.12xlarge', 'ml.m5.24xlarge', 'ml.c4.xlarge', 'ml.c4.2xlarge', 'ml.c4.4xlarge', 'ml.c4.8xlarge', 'ml.p2.xlarge', 'ml.p2.8xlarge', 'ml.p2.16xlarge', 'ml.p3.2xlarge', 'ml.p3.8xlarge', 'ml.p3.16xlarge',
    'ml.c5.xlarge', 'ml.c5.2xlarge', 'ml.c5.4xlarge', 'ml.c5.9xlarge', 'ml.c5.18xlarge', ''], type=str.strip, required=False, help='The ML compute instance type.', default='ml.m4.xlarge')
  parser.add_argument('--initial_variant_weight_3', type=_utils.str_to_float, required=False, help='Determines initial traffic distribution among all of the models that you specify in the endpoint configuration.', default=1.0)
  parser.add_argument('--accelerator_type_3', choices=['ml.eia1.medium', 'ml.eia1.large', 'ml.eia1.xlarge', ''], type=str.strip, required=False, help='The size of the Elastic Inference (EI) instance to use for the production variant.', default='')
  parser.add_argument('--resource_encryption_key', type=str.strip, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).', default='')
  parser.add_argument('--endpoint_config_tags', type=_utils.str_to_json_dict, required=False, help='An array of key-value pairs, to categorize AWS resources.', default='{}')

  parser.add_argument('--endpoint_name', type=str.strip, required=False, help='The name of the endpoint.', default='')
  parser.add_argument('--endpoint_tags', type=_utils.str_to_json_dict, required=False, help='An array of key-value pairs, to categorize AWS resources.', default='{}')

  return parser

def main(argv=None):
  parser = create_parser()
  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_sagemaker_client(args.region, args.endpoint_url)
  logging.info('Submitting Endpoint request to SageMaker...')
  endpoint_name = _utils.deploy_model(client, vars(args))
  logging.info('Endpoint creation request submitted. Waiting for completion...')
  _utils.wait_for_endpoint_creation(client, endpoint_name)

  with open('/tmp/endpoint_name.txt', 'w') as f:
    f.write(endpoint_name)

  logging.info('Endpoint creation completed.')


if __name__== "__main__":
  main()
