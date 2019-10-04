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
from pathlib2 import Path

from common import _utils

try:
  unicode
except NameError:
  unicode = str


def create_parser():
  parser = argparse.ArgumentParser(description='SageMaker Batch Transformation Job')
  _utils.add_default_client_arguments(parser)
  
  parser.add_argument('--job_name', type=str.strip, required=False, help='The name of the transform job.', default='')
  parser.add_argument('--model_name', type=str.strip, required=True, help='The name of the model that you want to use for the transform job.')
  parser.add_argument('--max_concurrent', type=_utils.str_to_int, required=False, help='The maximum number of parallel requests that can be sent to each instance in a transform job.', default='0')
  parser.add_argument('--max_payload', type=_utils.str_to_int, required=False, help='The maximum allowed size of the payload, in MB.', default='6')
  parser.add_argument('--batch_strategy', choices=['MultiRecord', 'SingleRecord', ''], type=str.strip, required=False, help='The number of records to include in a mini-batch for an HTTP inference request.', default='')
  parser.add_argument('--environment', type=_utils.str_to_json_dict, required=False, help='The dictionary of the environment variables to set in the Docker container. Up to 16 key-value entries in the map.', default='{}')
  parser.add_argument('--input_location', type=str.strip, required=True, help='The S3 location of the data source that is associated with a channel.')
  parser.add_argument('--data_type', choices=['ManifestFile', 'S3Prefix', 'AugmentedManifestFile', ''], type=str.strip, required=False, help='Data type of the input. Can be ManifestFile, S3Prefix, or AugmentedManifestFile.', default='S3Prefix')
  parser.add_argument('--content_type', type=str.strip, required=False, help='The multipurpose internet mail extension (MIME) type of the data.', default='')
  parser.add_argument('--split_type', choices=['None', 'Line', 'RecordIO', 'TFRecord', ''], type=str.strip, required=False, help='The method to use to split the transform job data files into smaller batches.', default='None')
  parser.add_argument('--compression_type', choices=['None', 'Gzip', ''], type=str.strip, required=False, help='If the transform data is compressed, the specification of the compression type.', default='None')
  parser.add_argument('--output_location', type=str.strip, required=True, help='The Amazon S3 path where you want Amazon SageMaker to store the results of the transform job.')
  parser.add_argument('--accept', type=str.strip, required=False, help='The MIME type used to specify the output data.')
  parser.add_argument('--assemble_with', choices=['None', 'Line', ''], type=str.strip, required=False, help='Defines how to assemble the results of the transform job as a single S3 object. Either None or Line.')
  parser.add_argument('--output_encryption_key', type=str.strip, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts.', default='')
  parser.add_argument('--input_filter', type=str.strip, required=False, help='A JSONPath expression used to select a portion of the input data to pass to the algorithm.', default='')
  parser.add_argument('--output_filter', type=str.strip, required=False, help='A JSONPath expression used to select a portion of the joined dataset to save in the output file for a batch transform job.', default='')
  parser.add_argument('--join_source', choices=['None', 'Input', ''], type=str.strip, required=False, help='Specifies the source of the data to join with the transformed data.', default='None')
  parser.add_argument('--instance_type', choices=['ml.m4.xlarge', 'ml.m4.2xlarge', 'ml.m4.4xlarge', 'ml.m4.10xlarge', 'ml.m4.16xlarge', 'ml.m5.large', 'ml.m5.xlarge', 'ml.m5.2xlarge', 'ml.m5.4xlarge',
    'ml.m5.12xlarge', 'ml.m5.24xlarge', 'ml.c4.xlarge', 'ml.c4.2xlarge', 'ml.c4.4xlarge', 'ml.c4.8xlarge', 'ml.p2.xlarge', 'ml.p2.8xlarge', 'ml.p2.16xlarge', 'ml.p3.2xlarge', 'ml.p3.8xlarge', 'ml.p3.16xlarge',
    'ml.c5.xlarge', 'ml.c5.2xlarge', 'ml.c5.4xlarge', 'ml.c5.9xlarge', 'ml.c5.18xlarge'], type=str.strip, required=True, help='The ML compute instance type for the transform job.', default='ml.m4.xlarge')
  parser.add_argument('--instance_count', type=_utils.str_to_int, required=False, help='The number of ML compute instances to use in the transform job.')
  parser.add_argument('--resource_encryption_key', type=str.strip, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).', default='')
  parser.add_argument('--tags', type=_utils.str_to_json_dict, required=False, help='An array of key-value pairs, to categorize AWS resources.', default='{}')
  parser.add_argument('--output_location_file', type=str.strip, required=True, help='File path where the program will write the Amazon S3 URI of the transform job results.')

  return parser

def main(argv=None):
  parser = create_parser()
  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_sagemaker_client(args.region, args.endpoint_url)
  logging.info('Submitting Batch Transformation request to SageMaker...')
  batch_job_name = _utils.create_transform_job(client, vars(args))
  logging.info('Batch Job request submitted. Waiting for completion...')
  _utils.wait_for_transform_job(client, batch_job_name)

  Path(args.output_location_file).parent.mkdir(parents=True, exist_ok=True)
  Path(args.output_location_file).write_text(unicode(args.output_location))

  logging.info('Batch Transformation creation completed.')


if __name__== "__main__":
  main()
