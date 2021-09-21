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

# A program to perform training through a EMR cluster.
# Usage:
# python train.py  \
#   --region us-west-2 \
#   --jobflow_id j-xsdsadsadsa \
#   --job_name traing_job \
#   --jar_path s3://kubeflow-pipeline/jars/py_workcount.py\
#   --main_class org.apache.spark.examples.JavaWordCount \
#   --input s3://kubeflow-pipeline/datasets/words.txt \
#   --output s3://kubeflow-pipeline/datasets/output/ \

import argparse
import logging
import random
from datetime import datetime
from pathlib2 import Path

from common import _utils

try:
  unicode
except NameError:
  unicode = str


def main(argv=None):
  parser = argparse.ArgumentParser(description='Submit PySpark Job')
  parser.add_argument('--region', type=str, help='The region where the cluster launches.')
  parser.add_argument('--jobflow_id', type=str, help='The name of the cluster to run job.')
  parser.add_argument('--job_name', type=str, help='The name of spark job.')
  parser.add_argument('--py_file', type=str, help='A path to a pyspark file run during the step')
  parser.add_argument('--input', type=str, help='File path of the dataset.')
  parser.add_argument('--output', type=str, help='Output path of the result files.')
  parser.add_argument('--output_file', type=str, help='S3 URI of the training job results.')

  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_client(args.region)
  logging.info('Submitting job to %s...', args.jobflow_id)
  spark_args = [args.input, args.output]
  step_id = _utils.submit_pyspark_job(
      client, args.jobflow_id, args.job_name, args.py_file, spark_args)
  logging.info('Job request submitted. Waiting for completion...')
  _utils.wait_for_job(client, args.jobflow_id, step_id)

  Path('/output.txt').write_text(unicode(step_id))
  Path(args.output_file).parent.mkdir(parents=True, exist_ok=True)
  Path(args.output_file).write_text(unicode(args.output))
  logging.info('Job completed.')

if __name__== "__main__":
  main()
