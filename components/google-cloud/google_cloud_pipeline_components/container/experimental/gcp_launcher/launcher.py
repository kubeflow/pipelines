# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import argparse
import os
import sys

from . import batch_prediction_job_remote_runner
from . import bigquery_job_remote_runner
from . import create_endpoint_remote_runner
from . import custom_job_remote_runner
from . import deploy_model_remote_runner
from . import export_model_remote_runner
from . import hyperparameter_tuning_job_remote_runner
from . import upload_model_remote_runner
from . import wait_gcp_resources


def _make_parent_dirs_and_return_path(file_path: str):
  os.makedirs(os.path.dirname(file_path), exist_ok=True)
  return file_path


def _parse_args(args):
  """Parse command line arguments.

    Args:
        args: A list of arguments.

    Returns:
        An argparse.Namespace class instance holding parsed args.
    """
  parser = argparse.ArgumentParser(
      prog='Vertex Pipelines service launcher', description='')
  parser.add_argument(
      '--type', dest='type', type=str, required=True, default=argparse.SUPPRESS)
  parser.add_argument(
      '--project',
      dest='project',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--location',
      dest='location',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--payload',
      dest='payload',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--gcp_resources',
      dest='gcp_resources',
      type=_make_parent_dirs_and_return_path,
      required=True,
      default=argparse.SUPPRESS)
  parsed_args, _ = parser.parse_known_args(args)
  # Parse the conditionally required arguments
  parser.add_argument(
      '--executor_input',
      dest='executor_input',
      type=str,
      # executor_input is only needed for components that emit output artifacts.
      required=(parsed_args.type in {
          'UploadModel', 'CreateEndpoint', 'BatchPredictionJob',
          'BigqueryQueryJob', 'BigqueryCreateModelJob',
          'BigqueryPredictModelJob', 'BigqueryExportModelJob', 'BigQueryEvaluateModelJob'
      }),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--output_info',
      dest='output_info',
      type=str,
      # output_info is only needed for ExportModel component.
      required=(parsed_args.type == 'ExportModel'),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--job_configuration_query_override',
      dest='job_configuration_query_override',
      type=str,
      # payload_override is only needed for BigQuery job component.
      required=(parsed_args.type in {
          'BigqueryQueryJob', 'BigqueryCreateModelJob',
          'BigqueryPredictModelJob', 'BigQueryEvaluateModelJob'
      }),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--model_name',
      dest='model_name',
      type=str,
      required=(parsed_args.type
                in {'BigqueryPredictModelJob', 'BigqueryExportModelJob', 'BigQueryEvaluateModelJob'}),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--model_destination_path',
      dest='model_destination_path',
      type=str,
      required=(parsed_args.type == 'BigqueryExportModelJob'),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--table_name',
      dest='table_name',
      type=str,
      # table_name is only needed for BigQuery model tvf job component.
      required=(parsed_args.type
                in {'BigqueryPredictModelJob', 'BigQueryEvaluateModelJob'}),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--query_statement',
      dest='query_statement',
      type=str,
      # query_statement is only needed for BigQuery model tvf job component.
      required=(parsed_args.type
                in {'BigqueryPredictModelJob', 'BigQueryEvaluateModelJob'}),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--threshold',
      dest='threshold',
      type=float,
      # threshold is only needed for BigQuery model tvf job component.
      required=(parsed_args.type
                in {'BigqueryPredictModelJob', 'BigQueryEvaluateModelJob'}),
      default=argparse.SUPPRESS)
  parsed_args, _ = parser.parse_known_args(args)
  return vars(parsed_args)


def main(argv):
  """Main entry.

    expected input args are as follows:
    Project - Required. The project of which the resource will be launched.
    Region - Required. The region of which the resource will be launched.
    Type - Required. GCP launcher is a single container. This Enum will
        specify which resource to be launched.
    Request payload - Required. The full serialized json of the resource spec.
        Note this can contain the Pipeline Placeholders.
    gcp_resources placeholder output for returning job_id.

    Args:
        argv: A list of system arguments.
  """

  parsed_args = _parse_args(argv)

  if parsed_args['type'] == 'CustomJob':
    custom_job_remote_runner.create_custom_job(**parsed_args)
  if parsed_args['type'] == 'BatchPredictionJob':
    batch_prediction_job_remote_runner.create_batch_prediction_job(
        **parsed_args)
  if parsed_args['type'] == 'HyperparameterTuningJob':
    hyperparameter_tuning_job_remote_runner.create_hyperparameter_tuning_job(
        **parsed_args)
  if parsed_args['type'] == 'UploadModel':
    upload_model_remote_runner.upload_model(**parsed_args)
  if parsed_args['type'] == 'CreateEndpoint':
    create_endpoint_remote_runner.create_endpoint(**parsed_args)
  if parsed_args['type'] == 'ExportModel':
    export_model_remote_runner.export_model(**parsed_args)
  if parsed_args['type'] == 'DeployModel':
    deploy_model_remote_runner.deploy_model(**parsed_args)
  if parsed_args['type'] == 'BigqueryQueryJob':
    bigquery_job_remote_runner.bigquery_query_job(**parsed_args)
  if parsed_args['type'] == 'BigqueryCreateModelJob':
    bigquery_job_remote_runner.bigquery_create_model_job(**parsed_args)
  if parsed_args['type'] == 'BigqueryPredictModelJob':
    bigquery_job_remote_runner.bigquery_predict_model_job(**parsed_args)
  if parsed_args['type'] == 'BigqueryExportModelJob':
    bigquery_job_remote_runner.bigquery_export_model_job(**parsed_args)
  if parsed_args['type'] == 'BigqueryEvaluateModelJob':
    bigquery_job_remote_runner.bigquery_evaluate_model_job(**parsed_args)
  if parsed_args['type'] == 'Wait':
    wait_gcp_resources.wait_gcp_resources(**parsed_args)


if __name__ == '__main__':
  main(sys.argv[1:])
