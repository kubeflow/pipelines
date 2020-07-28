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
import signal

from common import _utils

def create_parser():
  parser = argparse.ArgumentParser(description='SageMaker Training Job')
  _utils.add_default_client_arguments(parser)
  
  parser.add_argument('--job_name', type=str, required=False, help='The name of the training job.', default='')
  parser.add_argument('--role', type=str, required=True, help='The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.')
  parser.add_argument('--image', type=str, required=False, help='The registry path of the Docker image that contains the training algorithm.', default='')
  parser.add_argument('--algorithm_name', type=str, required=False, help='The name of the resource algorithm to use for the training job.', default='')
  parser.add_argument('--metric_definitions', type=_utils.yaml_or_json_str, required=False, help='The dictionary of name-regex pairs specify the metrics that the algorithm emits.', default={})
  parser.add_argument('--training_input_mode', type=str, required=False, choices=['File', 'Pipe'], help='The input mode that the algorithm supports. File or Pipe.', default='File')
  parser.add_argument('--hyperparameters', type=_utils.yaml_or_json_str, required=False, help='Dictionary of hyperparameters for the the algorithm.', default={})
  parser.add_argument('--channels', type=_utils.yaml_or_json_str, required=True, help='A list of dicts specifying the input channels. Must have at least one.')
  parser.add_argument('--instance_type', type=str, required=False, help='The ML compute instance type.', default='ml.m4.xlarge')
  parser.add_argument('--instance_count', type=int, required=True, help='The registry path of the Docker image that contains the training algorithm.', default=1)
  parser.add_argument('--volume_size', type=int, required=True, help='The size of the ML storage volume that you want to provision.', default=30)
  parser.add_argument('--resource_encryption_key', type=str, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).', default='')
  parser.add_argument('--max_run_time', type=int, required=True, help='The maximum run time in seconds for the training job.', default=86400)
  parser.add_argument('--model_artifact_path', type=str, required=True, help='Identifies the S3 path where you want Amazon SageMaker to store the model artifacts.')
  parser.add_argument('--output_encryption_key', type=str, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts.', default='')
  parser.add_argument('--vpc_security_group_ids', type=str, required=False, help='The VPC security group IDs, in the form sg-xxxxxxxx.')
  parser.add_argument('--vpc_subnets', type=str, required=False, help='The ID of the subnets in the VPC to which you want to connect your hpo job.')
  parser.add_argument('--network_isolation', type=_utils.str_to_bool, required=False, help='Isolates the training container.', default=True)
  parser.add_argument('--traffic_encryption', type=_utils.str_to_bool, required=False, help='Encrypts all communications between ML compute instances in distributed training.', default=False)
  parser.add_argument('--debug_hook_config', type=_utils.yaml_or_json_str, required=False, help='Configuration information for the debug hook parameters, collection configuration, and storage paths.', default={})
  parser.add_argument('--debug_rule_config', type=_utils.yaml_or_json_str, required=False, help='Configuration information for debugging rules.', default=[])
  parser.add_argument('--tensorboard_output_config', type=_utils.yaml_or_json_str, required=False, help='Configuration of storage locations for TensorBoard output.', default={})

  ### Start spot instance support
  parser.add_argument('--spot_instance', type=_utils.str_to_bool, required=False, help='Use managed spot training.', default=False)
  parser.add_argument('--max_wait_time', type=int, required=False, help='The maximum time in seconds you are willing to wait for a managed spot training job to complete.', default=86400)
  parser.add_argument('--checkpoint_config', type=_utils.yaml_or_json_str, required=False, help='Dictionary of information about the output location for managed spot training checkpoint data.', default={})
  ### End spot instance support

  parser.add_argument('--tags', type=_utils.yaml_or_json_str, required=False, help='An array of key-value pairs, to categorize AWS resources.', default={})

  ### Start outputs
  parser.add_argument('--model_artifact_url_output_path', type=str, default='/tmp/model-artifact-url', help='Local output path for the file containing the model artifacts URL.')
  parser.add_argument('--job_name_output_path', type=str, default='/tmp/job-name', help='Local output path for the file containing the training job name.')
  parser.add_argument('--training_image_output_path', type=str, default='/tmp/training-image', help='Local output path for the file containing the registry path of the Docker image that contains the training algorithm.')
  ### End outputs

  return parser

def main(argv=None):
  parser = create_parser()
  args = parser.parse_args(argv)

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_sagemaker_client(args.region, args.endpoint_url)

  logging.info('Submitting Training Job to SageMaker...')
  job_name = _utils.create_training_job(client, vars(args))

  def signal_term_handler(signalNumber, frame):
    _utils.stop_training_job(client, job_name)
    logging.info(f"Training Job: {job_name} request submitted to Stop")
  signal.signal(signal.SIGTERM, signal_term_handler)

  logging.info('Job request submitted. Waiting for completion...')
  try:
    _utils.wait_for_training_job(client, job_name)
    _utils.wait_for_debug_rules(client, job_name)
  except:
    raise
  finally:
    cw_client = _utils.get_cloudwatch_client(args.region)
    _utils.print_logs_for_job(cw_client, '/aws/sagemaker/TrainingJobs', job_name)

  image = _utils.get_image_from_job(client, job_name)
  model_artifact_url = _utils.get_model_artifacts_from_job(client, job_name)
  logging.info('Get model artifacts %s from training job %s.', model_artifact_url, job_name)

  _utils.write_output(args.model_artifact_url_output_path, model_artifact_url)
  _utils.write_output(args.job_name_output_path, job_name)
  _utils.write_output(args.training_image_output_path, image)

  logging.info('Job completed.')


if __name__== "__main__":
  main(sys.argv[1:])
