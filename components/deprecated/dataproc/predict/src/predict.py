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


# A program to perform prediction with an XGBoost model through a dataproc cluster.
# 
# Usage:
# python predict.py  \
#   --project bradley-playground \
#   --region us-central1 \
#   --cluster my-cluster \
#   --package gs://bradley-playground/xgboost4j-example-0.8-SNAPSHOT-jar-with-dependencies.jar \
#   --model gs://bradley-playground/model \
#   --output gs://bradley-playground/predict/ \
#   --workers 2 \
#   --predict gs://bradley-playground/transform/eval/part-* \
#   --analysis gs://bradley-playground/analysis \
#   --target resolution


import argparse
import json
import os
from pathlib import Path

from common import _utils
import logging
from tensorflow.python.lib.io import file_io


def main(argv=None):
  parser = argparse.ArgumentParser(description='ML Predictor')
  parser.add_argument('--project', type=str, help='Google Cloud project ID to use.')
  parser.add_argument('--region', type=str, help='Which zone to run the analyzer.')
  parser.add_argument('--cluster', type=str, help='The name of the cluster to run job.')
  parser.add_argument('--package', type=str,
                      help='GCS Path of XGBoost distributed trainer package.')
  parser.add_argument('--model', type=str, help='GCS path of the model file.')
  parser.add_argument('--output', type=str, help='GCS path to use for output.')
  parser.add_argument('--predict', type=str, help='GCS path of prediction libsvm file.')
  parser.add_argument('--analysis', type=str, help='GCS path of the analysis input.')
  parser.add_argument('--target', type=str, help='Target column name.')
  parser.add_argument('--prediction-results-uri-pattern-output-path',
                      type=str,
                      default='/output.txt',
                      help='Local output path for the file containing prediction results URI pattern.')
  parser.add_argument('--ui-metadata-output-path',
                      type=str,
                      default='/mlpipeline-ui-metadata.json',
                      help='Local output path for the file containing UI metadata JSON structure.')

  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  api = _utils.get_client()
  logging.info('Submitting job...')
  spark_args = [args.model, args.predict, args.analysis, args.target, args.output]
  job_id = _utils.submit_spark_job(
      api, args.project, args.region, args.cluster, [args.package],
      'ml.dmlc.xgboost4j.scala.example.spark.XGBoostPredictor', spark_args)
  logging.info('Job request submitted. Waiting for completion...')
  _utils.wait_for_job(api, args.project, args.region, job_id)
  prediction_results_uri_pattern = os.path.join(args.output, 'part-*.csv')
  Path(args.prediction_results_uri_pattern_output_path).parent.mkdir(parents=True, exist_ok=True)
  Path(args.prediction_results_uri_pattern_output_path).write_text(prediction_results_uri_pattern)

  with file_io.FileIO(os.path.join(args.output, 'schema.json'), 'r') as f:
    schema = json.load(f)

  metadata = {
    'outputs' : [{
      'type': 'table',
      'storage': 'gcs',
      'format': 'csv',
      'header': [x['name'] for x in schema],
      'source': prediction_results_uri_pattern
    }]
  }
  Path(args.ui_metadata_output_path).parent.mkdir(parents=True, exist_ok=True)
  Path(args.ui_metadata_output_path).write_text(json.dumps(metadata))
  logging.info('Job completed.')


if __name__== "__main__":
  main()
