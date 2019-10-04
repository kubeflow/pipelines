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
  parser = argparse.ArgumentParser(description='SageMaker Ground Truth Job')
  _utils.add_default_client_arguments(parser)
  
  parser.add_argument('--role', type=str.strip, required=True, help='The Amazon Resource Name (ARN) that Amazon SageMaker assumes to perform tasks on your behalf.')
  parser.add_argument('--job_name', type=str.strip, required=True, help='The name of the labeling job.')
  parser.add_argument('--label_attribute_name', type=str.strip, required=False, help='The attribute name to use for the label in the output manifest file. Default is the job name.', default='')
  parser.add_argument('--manifest_location', type=str.strip, required=True, help='The Amazon S3 location of the manifest file that describes the input data objects.')
  parser.add_argument('--output_location', type=str.strip, required=True, help='The Amazon S3 location to write output data.')
  parser.add_argument('--output_encryption_key', type=str.strip, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts.', default='')
  parser.add_argument('--task_type', type=str.strip, required=True, help='Built in image classification, bounding box, text classification, or semantic segmentation, or custom. If custom, please provide pre- and post-labeling task lambda functions.')
  parser.add_argument('--worker_type', type=str.strip, required=True, help='The workteam for data labeling, either public, private, or vendor.')
  parser.add_argument('--workteam_arn', type=str.strip, required=False, help='The ARN of the work team assigned to complete the tasks.')
  parser.add_argument('--no_adult_content', type=_utils.str_to_bool, required=False, help='If true, your data is free of adult content.', default='False')
  parser.add_argument('--no_ppi', type=_utils.str_to_bool, required=False, help='If true, your data is free of personally identifiable information.', default='False')
  parser.add_argument('--label_category_config', type=str.strip, required=False, help='The S3 URL of the JSON structured file that defines the categories used to label the data objects.', default='')
  parser.add_argument('--max_human_labeled_objects', type=_utils.str_to_int, required=False, help='The maximum number of objects that can be labeled by human workers.', default=0)
  parser.add_argument('--max_percent_objects', type=_utils.str_to_int, required=False, help='The maximum percentatge of input data objects that should be labeled.', default=0)
  parser.add_argument('--enable_auto_labeling', type=_utils.str_to_bool, required=False, help='Enables auto-labeling, only for bounding box, text classification, and image classification.', default=False)
  parser.add_argument('--initial_model_arn', type=str.strip, required=False, help='The ARN of the final model used for a previous auto-labeling job.', default='')
  parser.add_argument('--resource_encryption_key', type=str.strip, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).', default='')
  parser.add_argument('--ui_template', type=str.strip, required=True, help='The Amazon S3 bucket location of the UI template.')
  parser.add_argument('--pre_human_task_function', type=str.strip, required=False, help='The ARN of a Lambda function that is run before a data object is sent to a human worker.', default='')
  parser.add_argument('--post_human_task_function', type=str.strip, required=False, help='The ARN of a Lambda function implements the logic for annotation consolidation.', default='')
  parser.add_argument('--task_keywords', type=str.strip, required=False, help='Keywords used to describe the task so that workers on Amazon Mechanical Turk can discover the task.', default='')
  parser.add_argument('--title', type=str.strip, required=True, help='A title for the task for your human workers.')
  parser.add_argument('--description', type=str.strip, required=True, help='A description of the task for your human workers.')
  parser.add_argument('--num_workers_per_object', type=_utils.str_to_int, required=True, help='The number of human workers that will label an object.')
  parser.add_argument('--time_limit', type=_utils.str_to_int, required=True, help='The amount of time that a worker has to complete a task in seconds')
  parser.add_argument('--task_availibility', type=_utils.str_to_int, required=False, help='The length of time that a task remains available for labelling by human workers.', default=0)
  parser.add_argument('--max_concurrent_tasks', type=_utils.str_to_int, required=False, help='The maximum number of data objects that can be labeled by human workers at the same time.', default=0)
  parser.add_argument('--workforce_task_price', type=_utils.str_to_float, required=False, help='The price that you pay for each task performed by a public worker in USD. Specify to the tenth fractions of a cent. Format as "0.000".', default=0.000)
  parser.add_argument('--tags', type=_utils.str_to_json_dict, required=False, help='An array of key-value pairs, to categorize AWS resources.', default='{}')

  return parser

def main(argv=None):
  parser = create_parser()
  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_sagemaker_client(args.region, args.endpoint_url)
  logging.info('Submitting Ground Truth Job request to SageMaker...')
  _utils.create_labeling_job(client, vars(args))
  logging.info('Ground Truth labeling job request submitted. Waiting for completion...')
  _utils.wait_for_labeling_job(client, args.job_name)
  output_manifest, active_learning_model_arn = _utils.get_labeling_job_outputs(client, args.job_name, args.enable_auto_labeling)

  logging.info('Ground Truth Labeling Job completed.')

  with open('/tmp/output_manifest_location.txt', 'w') as f:
    f.write(output_manifest)
  with open('/tmp/active_learning_model_arn.txt', 'w') as f:
    f.write(active_learning_model_arn)


if __name__== "__main__":
  main()
