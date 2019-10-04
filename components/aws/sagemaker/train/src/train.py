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
  
  parser.add_argument('--job_name', type=str.strip, required=False, help='The name of the training job.', default='')
  parser.add_argument('--role', type=str.strip, required=True, help='The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.')
  parser.add_argument('--image', type=str.strip, required=True, help='The registry path of the Docker image that contains the training algorithm.', default='')
  parser.add_argument('--algorithm_name', type=str.strip, required=False, help='The name of the resource algorithm to use for the training job.', default='')
  parser.add_argument('--metric_definitions', type=_utils.str_to_json_dict, required=False, help='The dictionary of name-regex pairs specify the metrics that the algorithm emits.', default='{}')
  parser.add_argument('--training_input_mode', choices=['File', 'Pipe'], type=str.strip, help='The input mode that the algorithm supports. File or Pipe.', default='File')
  parser.add_argument('--hyperparameters', type=_utils.str_to_json_dict, help='Dictionary of hyperparameters for the the algorithm.', default='{}')
  parser.add_argument('--channels', type=_utils.str_to_json_list, required=True, help='A list of dicts specifying the input channels. Must have at least one.')
  parser.add_argument('--data_location_1', type=str.strip, required=False, help='The S3 URI of the input data source for channel 1.', default='')
  parser.add_argument('--data_location_2', type=str.strip, required=False, help='The S3 URI of the input data source for channel 2.', default='')
  parser.add_argument('--data_location_3', type=str.strip, required=False, help='The S3 URI of the input data source for channel 3.', default='')
  parser.add_argument('--data_location_4', type=str.strip, required=False, help='The S3 URI of the input data source for channel 4.', default='')
  parser.add_argument('--data_location_5', type=str.strip, required=False, help='The S3 URI of the input data source for channel 5.', default='')
  parser.add_argument('--data_location_6', type=str.strip, required=False, help='The S3 URI of the input data source for channel 6.', default='')
  parser.add_argument('--data_location_7', type=str.strip, required=False, help='The S3 URI of the input data source for channel 7.', default='')
  parser.add_argument('--data_location_8', type=str.strip, required=False, help='The S3 URI of the input data source for channel 8.', default='')
  parser.add_argument('--instance_type', required=True, choices=['ml.m4.xlarge', 'ml.m4.2xlarge', 'ml.m4.4xlarge', 'ml.m4.10xlarge', 'ml.m4.16xlarge', 'ml.m5.large', 'ml.m5.xlarge', 'ml.m5.2xlarge', 'ml.m5.4xlarge',
    'ml.m5.12xlarge', 'ml.m5.24xlarge', 'ml.c4.xlarge', 'ml.c4.2xlarge', 'ml.c4.4xlarge', 'ml.c4.8xlarge', 'ml.p2.xlarge', 'ml.p2.8xlarge', 'ml.p2.16xlarge', 'ml.p3.2xlarge', 'ml.p3.8xlarge', 'ml.p3.16xlarge',
    'ml.c5.xlarge', 'ml.c5.2xlarge', 'ml.c5.4xlarge', 'ml.c5.9xlarge', 'ml.c5.18xlarge'], type=str.strip, help='The ML compute instance type.', default='ml.m4.xlarge')
  parser.add_argument('--instance_count', required=True, type=_utils.str_to_int, help='The registry path of the Docker image that contains the training algorithm.', default=1)
  parser.add_argument('--volume_size', type=_utils.str_to_int, required=True, help='The size of the ML storage volume that you want to provision.', default=1)
  parser.add_argument('--resource_encryption_key', type=str.strip, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).', default='')
  parser.add_argument('--max_run_time', type=_utils.str_to_int, required=True, help='The maximum run time in seconds for the training job.', default=86400)
  parser.add_argument('--model_artifact_path', type=str.strip, required=True, help='Identifies the S3 path where you want Amazon SageMaker to store the model artifacts.')
  parser.add_argument('--output_encryption_key', type=str.strip, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts.', default='')
  parser.add_argument('--vpc_security_group_ids', type=str.strip, required=False, help='The VPC security group IDs, in the form sg-xxxxxxxx.')
  parser.add_argument('--vpc_subnets', type=str.strip, required=False, help='The ID of the subnets in the VPC to which you want to connect your hpo job.')
  parser.add_argument('--network_isolation', type=_utils.str_to_bool, required=False, help='Isolates the training container.', default=True)
  parser.add_argument('--traffic_encryption', type=_utils.str_to_bool, required=False, help='Encrypts all communications between ML compute instances in distributed training.', default=False)

  ### Start spot instance support
  parser.add_argument('--spot_instance', type=_utils.str_to_bool, required=False, help='Use managed spot training.', default=False)
  parser.add_argument('--max_wait_time', type=_utils.str_to_int, required=False, help='The maximum time in seconds you are willing to wait for a managed spot training job to complete.', default=86400)
  parser.add_argument('--checkpoint_config', type=_utils.str_to_json_dict, required=False, help='Dictionary of information about the output location for managed spot training checkpoint data.', default='{}')
  ### End spot instance support

  parser.add_argument('--tags', type=_utils.str_to_json_dict, required=False, help='An array of key-value pairs, to categorize AWS resources.', default='{}')

  return parser

def main(argv=None):
  parser = create_parser()
  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_sagemaker_client(args.region, args.endpoint_url)

  logging.info('Submitting Training Job to SageMaker...')
  job_name = _utils.create_training_job(client, vars(args))
  logging.info('Job request submitted. Waiting for completion...')
  _utils.wait_for_training_job(client, job_name)

  image = _utils.get_image_from_job(client, job_name)
  model_artifact_url = _utils.get_model_artifacts_from_job(client, job_name)
  logging.info('Get model artifacts %s from training job %s.', model_artifact_url, job_name)

  with open('/tmp/model_artifact_url.txt', 'w') as f:
    f.write(model_artifact_url)
  with open('/tmp/job_name.txt', 'w') as f:
    f.write(job_name)
  with open('/tmp/training_image.txt', 'w') as f:
    f.write(image)

  logging.info('Job completed.')


if __name__== "__main__":
  main()
