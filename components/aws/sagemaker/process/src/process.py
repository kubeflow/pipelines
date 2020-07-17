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
  parser = argparse.ArgumentParser(description='SageMaker Processing Job')
  _utils.add_default_client_arguments(parser)
  
  parser.add_argument('--job_name', type=str, required=False, help='The name of the processing job.', default='')
  parser.add_argument('--role', type=str, required=True, help='The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.')
  parser.add_argument('--image', type=str, required=True, help='The registry path of the Docker image that contains the processing container.', default='')
  parser.add_argument('--instance_type', required=True, type=str, help='The ML compute instance type.', default='ml.m4.xlarge')
  parser.add_argument('--instance_count', required=True, type=int, help='The number of ML compute instances to use in each processing job.', default=1)
  parser.add_argument('--volume_size', type=int, required=False, help='The size of the ML storage volume that you want to provision.', default=30)
  parser.add_argument('--resource_encryption_key', type=str, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).', default='')
  parser.add_argument('--output_encryption_key', type=str, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt the processing artifacts.', default='')
  parser.add_argument('--max_run_time', type=int, required=False, help='The maximum run time in seconds for the processing job.', default=86400)
  parser.add_argument('--environment', type=_utils.yaml_or_json_str, required=False, help='The dictionary of the environment variables to set in the Docker container. Up to 16 key-value entries in the map.', default={})
  parser.add_argument('--container_entrypoint', type=_utils.yaml_or_json_str, required=False, help='The entrypoint for the processing job. This is in the form of a list of strings that make a command.', default=[])
  parser.add_argument('--container_arguments', type=_utils.yaml_or_json_str, required=False, help='A list of string arguments to be passed to a processing job.', default=[])
  parser.add_argument('--input_config', type=_utils.yaml_or_json_str, required=False, help='Parameters that specify Amazon S3 inputs for a processing job.', default=[])
  parser.add_argument('--output_config', type=_utils.yaml_or_json_str, required=True, help='Parameters that specify Amazon S3 outputs for a processing job.', default=[])
  parser.add_argument('--vpc_security_group_ids', type=str, required=False, help='The VPC security group IDs, in the form sg-xxxxxxxx.')
  parser.add_argument('--vpc_subnets', type=str, required=False, help='The ID of the subnets in the VPC to which you want to connect your hpo job.')
  parser.add_argument('--network_isolation', type=_utils.str_to_bool, required=False, help='Isolates the processing container.', default=True)
  parser.add_argument('--traffic_encryption', type=_utils.str_to_bool, required=False, help='Encrypts all communications between ML compute instances in distributed training.', default=False)
  parser.add_argument('--tags', type=_utils.yaml_or_json_str, required=False, help='An array of key-value pairs, to categorize AWS resources.', default={})
  parser.add_argument('--job_name_output_path', type=str, default='/tmp/job-name', help='Local output path for the file containing the name of the processing job.')
  parser.add_argument('--output_artifacts_output_path', type=str, default='/tmp/output-artifacts', help='Local output path for the file containing the dictionary describing the output S3 artifacts.')

  return parser

def main(argv=None):
  parser = create_parser()
  args = parser.parse_args(argv)

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_sagemaker_client(args.region, args.endpoint_url)

  logging.info('Submitting Processing Job to SageMaker...')
  job_name = _utils.create_processing_job(client, vars(args))

  def signal_term_handler(signalNumber, frame):
    logging.info(f"Stopping Processing Job: {job_name}")
    _utils.stop_processing_job(client, job_name)
    logging.info(f"Processing Job: {job_name} request submitted to Stop")
  signal.signal(signal.SIGTERM, signal_term_handler)

  logging.info('Job request submitted. Waiting for completion...')

  try:
    _utils.wait_for_processing_job(client, job_name)
  except:
    raise
  finally:
    cw_client = _utils.get_cloudwatch_client(args.region)
    _utils.print_logs_for_job(cw_client, '/aws/sagemaker/ProcessingJobs', job_name)

  outputs = _utils.get_processing_job_outputs(client, job_name)

  _utils.write_output(args.job_name_output_path, job_name)
  _utils.write_output(args.output_artifacts_output_path, outputs, json_encode=True)

  logging.info('Job completed.')


if __name__== "__main__":
  main(sys.argv[1:])
