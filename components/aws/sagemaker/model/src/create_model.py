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

import sys
import argparse
import logging

from common import _utils

def create_parser():
  parser = argparse.ArgumentParser(description='SageMaker Training Job')
  _utils.add_default_client_arguments(parser)
  
  parser.add_argument('--model_name', type=str, required=True, help='The name of the new model.')
  parser.add_argument('--role', type=str, required=True, help='The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.')
  parser.add_argument('--container_host_name', type=str, required=False, help='When a ContainerDefinition is part of an inference pipeline, this value uniquely identifies the container for the purposes of logging and metrics.', default='')
  parser.add_argument('--image', type=str, required=False, help='The Amazon EC2 Container Registry (Amazon ECR) path where inference code is stored.', default='')
  parser.add_argument('--model_artifact_url', type=str, required=False, help='S3 path where Amazon SageMaker to store the model artifacts.', default='')
  parser.add_argument('--environment', type=_utils.yaml_or_json_str, required=False, help='The dictionary of the environment variables to set in the Docker container. Up to 16 key-value entries in the map.', default={})
  parser.add_argument('--model_package', type=str, required=False, help='The name or Amazon Resource Name (ARN) of the model package to use to create the model.', default='')
  parser.add_argument('--secondary_containers', type=_utils.yaml_or_json_str, required=False, help='A list of dicts that specifies the additional containers in the inference pipeline.', default={})
  parser.add_argument('--vpc_security_group_ids', type=str, required=False, help='The VPC security group IDs, in the form sg-xxxxxxxx.', default='')
  parser.add_argument('--vpc_subnets', type=str, required=False, help='The ID of the subnets in the VPC to which you want to connect your hpo job.', default='')
  parser.add_argument('--network_isolation', type=_utils.str_to_bool, required=False, help='Isolates the training container.', default=True)
  parser.add_argument('--tags', type=_utils.yaml_or_json_str, required=False, help='An array of key-value pairs, to categorize AWS resources.', default={})
  parser.add_argument('--model_name_output_path', type=str, default='/tmp/model-name', help='Local output path for the file containing the name of the model SageMaker created.')

  return parser

def main(argv=None):
  parser = create_parser()
  args = parser.parse_args(argv)

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_sagemaker_client(args.region, args.endpoint_url)

  logging.info('Submitting model creation request to SageMaker...')
  _utils.create_model(client, vars(args))

  logging.info('Model creation completed.')

  _utils.write_output(args.model_name_output_path, args.model_name)


if __name__== "__main__":
  main(sys.argv[1:])
