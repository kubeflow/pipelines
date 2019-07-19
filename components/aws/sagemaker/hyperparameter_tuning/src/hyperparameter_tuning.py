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
import json
from datetime import datetime
from pathlib2 import Path

from common import _utils

def str_to_bool(s):
  if s.lower() == 'true':
    return True
  elif s.lower() == 'false':
    return False
  else:
    raise argparse.ArgumentTypeError('"True" or "False" expected.')

def str_to_int(s):
  if s:
    return int(s)
  else:
    return 0

def main(argv=None):
  parser = argparse.ArgumentParser(description='SageMaker Hyperparameter Tuning Job')
  # Set required=True only if parameter is required in the request and there is no default value for it, where default values are based on default values in SageMaker UI
  parser.add_argument('--region', type=str, required=True, help='The region where the cluster launches.')
  parser.add_argument('--job_name', type=str, required=True, help='The name of the tuning job. Must be unique within the same AWS account and AWS region.')
  parser.add_argument('--role', type=str, required=True, help='The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.')
  parser.add_argument('--image', type=str, required=False, help='The registry path of the Docker image that contains the training algorithm.')
  parser.add_argument('--algorithm_name', type=str, required=False, help='The name of the resource algorithm to use for the hyperparameter tuning job.')
  parser.add_argument('--training_input_mode', choices=['File', 'Pipe'], required=True, help='The input mode that the algorithm supports. File or Pipe.', default='File')
  parser.add_argument('--metric_definitions', type=json.loads, required=False, help='The dictionary of name-regex pairs specify the metrics that the algorithm emits.', default='{}')
  parser.add_argument('--strategy', choices=['Bayesian', 'Random'], required=False, help='How hyperparameter tuning chooses the combinations of hyperparameter values to use for the training job it launches.', default='Bayesian')
  parser.add_argument('--metric_name', type=str, required=True, help='The name of the metric to use for the objective metric.')
  parser.add_argument('--metric_type', choices=['Maximize', 'Minimize'], required=True, help='Whether to minimize or maximize the objective metric.')
  parser.add_argument('--early_stopping_type', choices=['Off', 'Auto'], required=False, help='Whether to minimize or maximize the objective metric.', default='Off')
  parser.add_argument('--static_parameters', type=json.loads, required=False, help='The values of hyperparameters that do not change for the tuning job.', default='{}')
  parser.add_argument('--integer_parameters', type=json.loads, required=False, help='The array of IntegerParameterRange objects that specify ranges of integer hyperparameters that you want to search.', default='[]')
  parser.add_argument('--continuous_parameters', type=json.loads, required=False, help='The array of ContinuousParameterRange objects that specify ranges of continuous hyperparameters that you want to search.', default='[]')
  parser.add_argument('--categorical_parameters', type=json.loads, required=False, help='The array of CategoricalParameterRange objects that specify ranges of categorical hyperparameters that you want to search.', default='[]')
  parser.add_argument('--channels', type=json.loads, required=True, help='A list of dicts specifying the input channels. Must have at least one.', default='[{}]')
  parser.add_argument('--output_location', type=str, required=True, help='The Amazon S3 path where you want Amazon SageMaker to store the results of the transform job.')
  parser.add_argument('--output_encryption_key', type=str, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts.', default='')
  parser.add_argument('--instance_type', choices=['ml.m4.xlarge', 'ml.m4.2xlarge', 'ml.m4.4xlarge', 'ml.m4.10xlarge', 'ml.m4.16xlarge', 'ml.m5.large', 'ml.m5.xlarge', 'ml.m5.2xlarge', 'ml.m5.4xlarge',
    'ml.m5.12xlarge', 'ml.m5.24xlarge', 'ml.c4.xlarge', 'ml.c4.2xlarge', 'ml.c4.4xlarge', 'ml.c4.8xlarge', 'ml.p2.xlarge', 'ml.p2.8xlarge', 'ml.p2.16xlarge', 'ml.p3.2xlarge', 'ml.p3.8xlarge', 'ml.p3.16xlarge',
    'ml.c5.xlarge', 'ml.c5.2xlarge', 'ml.c5.4xlarge', 'ml.c5.9xlarge', 'ml.c5.18xlarge'], required=False, help='The ML compute instance type.', default='ml.m4.xlarge')
  parser.add_argument('--instance_count', type=str_to_int, required=False, help='The number of ML compute instances to use in each training job.', default=1)
  parser.add_argument('--volume_size', type=str_to_int, required=False, help='The size of the ML storage volume that you want to provision.', default=1)
  parser.add_argument('--max_num_jobs', type=str_to_int, required=True, help='The maximum number of training jobs that a hyperparameter tuning job can launch.')
  parser.add_argument('--max_parallel_jobs', type=str_to_int, required=True, help='The maximum number of concurrent training jobs that a hyperparameter tuning job can launch.')
  parser.add_argument('--max_run_time', type=str_to_int, required=False, help='The maximum run time in seconds per training job.', default=86400)
  parser.add_argument('--resource_encryption_key', type=str, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).', default='')
  parser.add_argument('--vpc_security_group_ids', type=str, required=False, help='The VPC security group IDs, in the form sg-xxxxxxxx.')
  parser.add_argument('--vpc_subnets', type=str, required=False, help='The ID of the subnets in the VPC to which you want to connect your hpo job.')
  parser.add_argument('--network_isolation', type=str_to_bool, required=False, help='Isolates the training container.', default=True)
  parser.add_argument('--traffic_encryption', type=str_to_bool, required=False, help='Encrypts all communications between ML compute instances in distributed training.', default=False)
  parser.add_argument('--warm_start_type', choices=['IdenticalDataAndAlgorithm', 'TransferLearning', ''], required=False, help='Specifies either "IdenticalDataAndAlgorithm" or "TransferLearning"')
  parser.add_argument('--parent_hpo_jobs', type=str, required=False, help='List of previously completed or stopped hyperparameter tuning jobs to be used as a starting point.', default='')
  parser.add_argument('--tags', type=json.loads, required=False, help='An array of key-value pairs, to categorize AWS resources.', default='{}')

  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_client(args.region)
  logging.info('Submitting HyperParameter Tuning Job request to SageMaker...')
  hpo_job_name = _utils.create_hyperparameter_tuning_job(client, vars(args))
  logging.info('HyperParameter Tuning Job request submitted. Waiting for completion...')
  _utils.wait_for_hyperparameter_training_job(client, hpo_job_name)
  best_job, best_hyperparameters = _utils.get_best_training_job_and_hyperparameters(client, hpo_job_name)
  model_artifact_url = _utils.get_model_artifacts_from_job(client, best_job)

  logging.info('HyperParameter Tuning Job completed.')

  with open('/tmp/best_job_name.txt', 'w') as f:
    f.write(best_job)
  with open('/tmp/best_hyperparameters.txt', 'w') as f:
    f.write(json.dumps(best_hyperparameters))
  with open('/tmp/model_artifact_url.txt', 'w') as f:
    f.write(model_artifact_url)


if __name__== "__main__":
  main()
