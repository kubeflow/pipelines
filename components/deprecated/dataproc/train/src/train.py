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


# A program to perform training of an XGBoost model through a dataproc cluster.
# Usage:
# python train.py  \
#   --project bradley-playground \
#   --region us-central1 \
#   --cluster ten4 \
#   --package gs://bradley-playground/xgboost4j-example-0.8-SNAPSHOT-jar-with-dependencies.jar \
#   --output gs://bradley-playground/train/model \
#   --conf gs://bradley-playground/trainconf.json \
#   --rounds 300 \
#   --workers 2 \
#   --train gs://bradley-playground/transform/train/part-* \
#   --eval gs://bradley-playground/transform/eval/part-* \
#   --analysis gs://bradley-playground/analysis \
#   --target resolution


import argparse
import logging
from pathlib import Path

from common import _utils


def main(argv=None):
  parser = argparse.ArgumentParser(description='ML Trainer')
  parser.add_argument('--project', type=str, help='Google Cloud project ID to use.')
  parser.add_argument('--region', type=str, help='Which zone to run the analyzer.')
  parser.add_argument('--cluster', type=str, help='The name of the cluster to run job.')
  parser.add_argument('--package', type=str,
                      help='GCS Path of XGBoost distributed trainer package.')
  parser.add_argument('--output', type=str, help='GCS path to use for output.')
  parser.add_argument('--conf', type=str, help='GCS path of the training json config file.')
  parser.add_argument('--rounds', type=int, help='Number of rounds to train.')
  parser.add_argument('--workers', type=int, help='Number of workers to use for training.')
  parser.add_argument('--train', type=str, help='GCS path of the training libsvm file pattern.')
  parser.add_argument('--eval', type=str, help='GCS path of the eval libsvm file pattern.')
  parser.add_argument('--analysis', type=str, help='GCS path of the analysis input.')
  parser.add_argument('--target', type=str, help='Target column name.')
  parser.add_argument('--output-dir-uri-output-path',
                      type=str,
                      default='/output.txt',
                      help='Local output path for the file containing the output dir URI.')

  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  api = _utils.get_client()
  logging.info('Submitting job...')
  spark_args = [args.conf, str(args.rounds), str(args.workers), args.analysis, args.target,
                args.train, args.eval, args.output]
  job_id = _utils.submit_spark_job(
      api, args.project, args.region, args.cluster, [args.package],
      'ml.dmlc.xgboost4j.scala.example.spark.XGBoostTrainer', spark_args)
  logging.info('Job request submitted. Waiting for completion...')
  _utils.wait_for_job(api, args.project, args.region, job_id)
  Path(args.output_dir_uri_output_path).parent.mkdir(parents=True, exist_ok=True)
  Path(args.output_dir_uri_output_path).write_text(args.output)

  logging.info('Job completed.')


if __name__== "__main__":
  main()
