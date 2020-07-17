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
  parser = argparse.ArgumentParser(description='SageMaker Batch Transformation Job')
  _utils.add_default_client_arguments(parser)
  
  parser.add_argument('--job_name', type=str, required=False, help='The name of the transform job.', default='')
  parser.add_argument('--model_name', type=str, required=True, help='The name of the model that you want to use for the transform job.')
  parser.add_argument('--max_concurrent', type=int, required=False, help='The maximum number of parallel requests that can be sent to each instance in a transform job.', default='0')
  parser.add_argument('--max_payload', type=int, required=False, help='The maximum allowed size of the payload, in MB.', default='6')
  parser.add_argument('--batch_strategy', choices=['MultiRecord', 'SingleRecord', ''], type=str, required=False, help='The number of records to include in a mini-batch for an HTTP inference request.', default='')
  parser.add_argument('--environment', type=_utils.yaml_or_json_str, required=False, help='The dictionary of the environment variables to set in the Docker container. Up to 16 key-value entries in the map.', default={})
  parser.add_argument('--input_location', type=str, required=True, help='The S3 location of the data source that is associated with a channel.')
  parser.add_argument('--data_type', choices=['ManifestFile', 'S3Prefix', 'AugmentedManifestFile', ''], type=str, required=False, help='Data type of the input. Can be ManifestFile, S3Prefix, or AugmentedManifestFile.', default='S3Prefix')
  parser.add_argument('--content_type', type=str, required=False, help='The multipurpose internet mail extension (MIME) type of the data.', default='')
  parser.add_argument('--split_type', choices=['None', 'Line', 'RecordIO', 'TFRecord', ''], type=str, required=False, help='The method to use to split the transform job data files into smaller batches.', default='None')
  parser.add_argument('--compression_type', choices=['None', 'Gzip', ''], type=str, required=False, help='If the transform data is compressed, the specification of the compression type.', default='None')
  parser.add_argument('--output_location', type=str, required=True, help='The Amazon S3 path where you want Amazon SageMaker to store the results of the transform job.')
  parser.add_argument('--accept', type=str, required=False, help='The MIME type used to specify the output data.')
  parser.add_argument('--assemble_with', choices=['None', 'Line', ''], type=str, required=False, help='Defines how to assemble the results of the transform job as a single S3 object. Either None or Line.')
  parser.add_argument('--output_encryption_key', type=str, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt the model artifacts.', default='')
  parser.add_argument('--input_filter', type=str, required=False, help='A JSONPath expression used to select a portion of the input data to pass to the algorithm.', default='')
  parser.add_argument('--output_filter', type=str, required=False, help='A JSONPath expression used to select a portion of the joined dataset to save in the output file for a batch transform job.', default='')
  parser.add_argument('--join_source', choices=['None', 'Input', ''], type=str, required=False, help='Specifies the source of the data to join with the transformed data.', default='None')
  parser.add_argument('--instance_type', type=str, required=False, help='The ML compute instance type for the transform job.', default='ml.m4.xlarge')
  parser.add_argument('--instance_count', type=int, required=False, help='The number of ML compute instances to use in the transform job.')
  parser.add_argument('--resource_encryption_key', type=str, required=False, help='The AWS KMS key that Amazon SageMaker uses to encrypt data on the storage volume attached to the ML compute instance(s).', default='')
  parser.add_argument('--tags', type=_utils.yaml_or_json_str, required=False, help='An array of key-value pairs, to categorize AWS resources.', default={})
  parser.add_argument('--output_location_output_path', type=str, default='/tmp/output-location', help='Local output path for the file containing the Amazon S3 URI of the transform job results.')

  return parser

def main(argv=None):
  parser = create_parser()
  args = parser.parse_args(argv)

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_sagemaker_client(args.region, args.endpoint_url)
  logging.info('Submitting Batch Transformation request to SageMaker...')
  batch_job_name = _utils.create_transform_job(client, vars(args))

  def signal_term_handler(signalNumber, frame):
    _utils.stop_transform_job(client, batch_job_name)
    logging.info(f"Transform job: {batch_job_name} request submitted to Stop")
  signal.signal(signal.SIGTERM, signal_term_handler)

  logging.info('Batch Job request submitted. Waiting for completion...')

  try:
    _utils.wait_for_transform_job(client, batch_job_name)
  except:
    raise
  finally:
    cw_client = _utils.get_cloudwatch_client(args.region)
    _utils.print_logs_for_job(cw_client, '/aws/sagemaker/TransformJobs', batch_job_name)

  _utils.write_output(args.output_location_output_path, args.output_location)

  logging.info('Batch Transformation creation completed.')


if __name__== "__main__":
  main(sys.argv[1:])
