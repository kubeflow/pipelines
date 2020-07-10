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
  parser = argparse.ArgumentParser(description='SageMaker Hyperparameter Tuning Job')
  _utils.add_default_client_arguments(parser)

  parser.add_argument('--job_name', type=str, required=False, help='The name of the tuning job. Must be unique within the same AWS account and AWS region.')
  parser.add_argument('--role', type=str, required=True, help='The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.')
  parser.add_argument('--image', type=str, required=False, help='The registry path of the Docker image that contains the training algorithm.', default='')
  parser.add_argument('--algorithm_name', type=str, required=False, help='The name of the resource algorithm to use for the hyperparameter tuning job.', default='')
  parser.add_argument('--training_input_mode', choices=['File', 'Pipe'], type=str, required=False, help='The input mode that the algorithm supports. File or Pipe.', default='File')
  parser.add_argument('--metric_definitions', type=_utils.yaml_or_json_str, required=False, help='The dictionary of name-regex pairs specify the metrics that the algorithm emits.', default={})
  parser.add_argument('--strategy', choices=['Bayesian', 'Random'], type=str, required=False, help='How hyperparameter tuning chooses the combinations of hyperparameter values to use for the training job it launches.', default='Bayesian')
  parser.add_argument('--metric_name', type=str, required=True, help='The name of the metric to use for the objective metric.')
  parser.add_argument('--metric_type', choices=['Maximize', 'Minimize'], type=str, required=True, help='Whether to minimize or maximize the objective metric.')
  parser.add_argument('--early_stopping_type', choices=['Off', 'Auto'], type=str, required=False, help='Whether to minimize or maximize the objective metric.', default='Off')
  parser.add_argument('--static_parameters', type=_utils.yaml_or_json_str, required=False, help='The values of hyperparameters that do not change for the tuning job.', default={})
  parser.add_argument('--integer_parameters', type=_utils.yaml_or_json_str, required=False, help='The array of IntegerParameterRange objects that specify ranges of integer hyperparameters that you want to search.', default=[])
  parser.add_argument('--continuous_parameters', type=_utils.yaml_or_json_str, required=False, help='The array of ContinuousParameterRange objects that specify ranges of continuous hyperparameters that you want to search.', default=[])
  parser.add_argument('--categorical_parameters', type=_utils.yaml_or_json_str, required=False, help='The array of CategoricalParameterRange objects that specify ranges of categorical hyperparameters that you want to search.', default=[])
  parser.add_argument('--channels', type=_utils.yaml_or_json_str, required=True, help='A list of dicts specifying the input channels. Must have at least one.')
  parser.add_argument('--output_location', type=str, required=True, help='The Amazon S3 path where you want Amazon SageMaker to store the results of the transform job.')
  parser.add_argument('--output_encryption_key', type=str, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts.', default='')
  parser.add_argument('--instance_type', type=str, required=False, help='The ML compute instance type.', default='ml.m4.xlarge')
  parser.add_argument('--instance_count', type=int, required=False, help='The number of ML compute instances to use in each training job.', default=1)
  parser.add_argument('--volume_size', type=int, required=False, help='The size of the ML storage volume that you want to provision.', default=30)
  parser.add_argument('--max_num_jobs', type=int, required=True, help='The maximum number of training jobs that a hyperparameter tuning job can launch.')
  parser.add_argument('--max_parallel_jobs', type=int, required=True, help='The maximum number of concurrent training jobs that a hyperparameter tuning job can launch.')
  parser.add_argument('--max_run_time', type=int, required=False, help='The maximum run time in seconds per training job.', default=86400)
  parser.add_argument('--resource_encryption_key', type=str, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).', default='')
  parser.add_argument('--vpc_security_group_ids', type=str, required=False, help='The VPC security group IDs, in the form sg-xxxxxxxx.')
  parser.add_argument('--vpc_subnets', type=str, required=False, help='The ID of the subnets in the VPC to which you want to connect your hpo job.')
  parser.add_argument('--network_isolation', type=_utils.str_to_bool, required=False, help='Isolates the training container.', default=True)
  parser.add_argument('--traffic_encryption', type=_utils.str_to_bool, required=False, help='Encrypts all communications between ML compute instances in distributed training.', default=False)
  parser.add_argument('--warm_start_type', choices=['IdenticalDataAndAlgorithm', 'TransferLearning', ''], type=str, required=False, help='Specifies either "IdenticalDataAndAlgorithm" or "TransferLearning"')
  parser.add_argument('--parent_hpo_jobs', type=str, required=False, help='List of previously completed or stopped hyperparameter tuning jobs to be used as a starting point.', default='')

  ### Start spot instance support
  parser.add_argument('--spot_instance', type=_utils.str_to_bool, required=False, help='Use managed spot training.', default=False)
  parser.add_argument('--max_wait_time', type=int, required=False, help='The maximum time in seconds you are willing to wait for a managed spot training job to complete.', default=86400)
  parser.add_argument('--checkpoint_config', type=_utils.yaml_or_json_str, required=False, help='Dictionary of information about the output location for managed spot training checkpoint data.', default={})
  ### End spot instance support

  parser.add_argument('--tags', type=_utils.yaml_or_json_str, required=False, help='An array of key-value pairs, to categorize AWS resources.', default={})

  ### Start outputs
  parser.add_argument('--hpo_job_name_output_path', type=str, default='/tmp/hpo-job-name', help='Local output path for the file containing the name of the hyper parameter tuning job')
  parser.add_argument('--model_artifact_url_output_path', type=str, default='/tmp/artifact-url', help='Local output path for the file containing the model artifacts url')
  parser.add_argument('--best_job_name_output_path', type=str, default='/tmp/best-job-name', help='Local output path for the file containing the name of the best training job in the hyper parameter tuning job')
  parser.add_argument('--best_hyperparameters_output_path', type=str, default='/tmp/best-hyperparams', help='Local output path for the file containing the final tuned hyperparameters')
  parser.add_argument('--training_image_output_path', type=str, default='/tmp/training-image', help='Local output path for the file containing the registry path of the Docker image that contains the training algorithm')
  ### End outputs

  return parser

def main(argv=None):
  parser = create_parser()
  args = parser.parse_args(argv)

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_sagemaker_client(args.region)
  logging.info('Submitting HyperParameter Tuning Job request to SageMaker...')
  hpo_job_name = _utils.create_hyperparameter_tuning_job(client, vars(args))

  def signal_term_handler(signalNumber, frame):
    _utils.stop_hyperparameter_tuning_job(client, hpo_job_name)
    logging.info(f"HyperParameter Tuning Job: {hpo_job_name} request submitted to Stop")
  signal.signal(signal.SIGTERM, signal_term_handler)

  logging.info('HyperParameter Tuning Job request submitted. Waiting for completion...')
  _utils.wait_for_hyperparameter_training_job(client, hpo_job_name)
  best_job, best_hyperparameters = _utils.get_best_training_job_and_hyperparameters(client, hpo_job_name)
  model_artifact_url = _utils.get_model_artifacts_from_job(client, best_job)
  image = _utils.get_image_from_job(client, best_job)

  logging.info('HyperParameter Tuning Job completed.')

  _utils.write_output(args.hpo_job_name_output_path, hpo_job_name)
  _utils.write_output(args.model_artifact_url_output_path, model_artifact_url)
  _utils.write_output(args.best_job_name_output_path, best_job)
  _utils.write_output(args.best_hyperparameters_output_path, best_hyperparameters, json_encode=True)
  _utils.write_output(args.training_image_output_path, image)

if __name__== "__main__":
  main(sys.argv[1:])
