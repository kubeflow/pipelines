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
# python analyze.py  \
#   --project bradley-playground \
#   --region us-central1 \
#   --cluster ten4 \
#   --output gs://bradley-playground/analysis \
#   --train gs://bradley-playground/sfpd/train.csv \
#   --schema gs://bradley-playground/schema.json \


import argparse
import os

from common import _utils


def main(argv=None):
  parser = argparse.ArgumentParser(description='ML Analyzer')
  parser.add_argument('--project', type=str, help='Google Cloud project ID to use.')
  parser.add_argument('--region', type=str, help='Which zone to run the analyzer.')
  parser.add_argument('--cluster', type=str, help='The name of the cluster to run job.')
  parser.add_argument('--output', type=str, help='GCS path to use for output.')
  parser.add_argument('--train', type=str, help='GCS path of the training csv file.')
  parser.add_argument('--schema', type=str, help='GCS path of the json schema file.')
  args = parser.parse_args()

  code_path = os.path.dirname(os.path.realpath(__file__))
  runfile_source = os.path.join(code_path, 'analyze_run.py')
  dest_files = _utils.copy_resources_to_gcs([runfile_source], args.output)
  try:
    api = _utils.get_client()
    print('Submitting job...')
    spark_args = ['--output', args.output, '--train', args.train, '--schema', args.schema]
    job_id = _utils.submit_pyspark_job(
        api, args.project, args.region, args.cluster, dest_files[0], spark_args)
    print('Job request submitted. Waiting for completion...')
    _utils.wait_for_job(api, args.project, args.region, job_id)
    with open('/output.txt', 'w') as f:
      f.write(args.output)

    print('Job completed.')
  finally:
    _utils.remove_resources_from_gcs(dest_files)


if __name__== "__main__":
  main()
