# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# Usage:
# python transform.py  \
#   --project bradley-playground \
#   --region us-central1 \
#   --cluster ten4 \
#   --output gs://bradley-playground/transform \
#   --train gs://bradley-playground/sfpd/train.csv \
#   --eval gs://bradley-playground/sfpd/eval.csv \
#   --analysis gs://bradley-playground/analysis \
#   --target resolution


import argparse
import os
import subprocess

from common import _utils


def main(argv=None):
  parser = argparse.ArgumentParser(description='ML Transfomer')
  parser.add_argument('--project', type=str, help='Google Cloud project ID to use.')
  parser.add_argument('--region', type=str, help='Which zone to run the analyzer.')
  parser.add_argument('--cluster', type=str, help='The name of the cluster to run job.')
  parser.add_argument('--output', type=str, help='GCS path to use for output.')
  parser.add_argument('--train', type=str, help='GCS path of the training csv file.')
  parser.add_argument('--eval', type=str, help='GCS path of the eval csv file.')
  parser.add_argument('--analysis', type=str, help='GCS path of the analysis results.')
  parser.add_argument('--target', type=str, help='Target column name.')
  args = parser.parse_args()

  # Remove existing [output]/train and [output]/eval if they exist.
  # It should not be done in the run time code because run time code should be portable
  # to on-prem while we need gsutil here.
  _utils.delete_directory_from_gcs(os.path.join(args.output, 'train'))
  _utils.delete_directory_from_gcs(os.path.join(args.output, 'eval'))

  code_path = os.path.dirname(os.path.realpath(__file__))
  runfile_source = os.path.join(code_path, 'transform_run.py')
  dest_files = _utils.copy_resources_to_gcs([runfile_source], args.output)
  try:
    api = _utils.get_client()
    print('Submitting job...')
    spark_args = ['--output', args.output, '--analysis', args.analysis,
                  '--target', args.target]
    if args.train:
      spark_args.extend(['--train', args.train])
    if args.eval:
      spark_args.extend(['--eval', args.eval])

    job_id = _utils.submit_pyspark_job(
        api, args.project, args.region, args.cluster, dest_files[0], spark_args)
    print('Job request submitted. Waiting for completion...')
    _utils.wait_for_job(api, args.project, args.region, job_id)

    with open('/output_train.txt', 'w') as f:
      f.write(os.path.join(args.output, 'train', 'part-*'))
    with open('/output_eval.txt', 'w') as f:
      f.write(os.path.join(args.output, 'eval', 'part-*'))

    print('Job completed.')
  finally:
    _utils.remove_resources_from_gcs(dest_files)


if __name__== "__main__":
  main()
