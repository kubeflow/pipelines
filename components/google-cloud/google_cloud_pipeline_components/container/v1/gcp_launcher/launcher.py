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
"""Launcher client to launch jobs for various job types."""

import argparse
import logging
import os
import sys

from . import batch_prediction_job_remote_runner
from . import bigquery_job_remote_runner
from . import create_endpoint_remote_runner
from . import custom_job_remote_runner
from . import dataproc_batch_remote_runner
from . import delete_endpoint_remote_runner
from . import delete_model_remote_runner
from . import deploy_model_remote_runner
from . import export_model_remote_runner
from . import hyperparameter_tuning_job_remote_runner
from . import undeploy_model_remote_runner
from . import upload_model_remote_runner
from . import wait_gcp_resources

_JOB_TYPE_TO_ACTION_MAP = {
    'CustomJob':
        custom_job_remote_runner.create_custom_job,
    'BatchPredictionJob':
        batch_prediction_job_remote_runner.create_batch_prediction_job,
    'HyperparameterTuningJob':
        hyperparameter_tuning_job_remote_runner
        .create_hyperparameter_tuning_job,
    'UploadModel':
        upload_model_remote_runner.upload_model,
    'CreateEndpoint':
        create_endpoint_remote_runner.create_endpoint,
    'DeleteEndpoint':
        delete_endpoint_remote_runner.delete_endpoint,
    'ExportModel':
        export_model_remote_runner.export_model,
    'DeployModel':
        deploy_model_remote_runner.deploy_model,
    'DeleteModel':
        delete_model_remote_runner.delete_model,
    'UndeployModel':
        undeploy_model_remote_runner.undeploy_model,
    'BigqueryQueryJob':
        bigquery_job_remote_runner.bigquery_query_job,
    'BigqueryCreateModelJob':
        bigquery_job_remote_runner.bigquery_create_model_job,
    'BigqueryDropModelJob':
        bigquery_job_remote_runner.bigquery_drop_model_job,
    'BigqueryPredictModelJob':
        bigquery_job_remote_runner.bigquery_predict_model_job,
    'BigqueryMLForecastJob':
        bigquery_job_remote_runner.bigquery_forecast_model_job,
    'BigqueryExplainPredictModelJob':
        bigquery_job_remote_runner.bigquery_explain_predict_model_job,
    'BigqueryExplainForecastModelJob':
        bigquery_job_remote_runner.bigquery_explain_forecast_model_job,
    'BigqueryExportModelJob':
        bigquery_job_remote_runner.bigquery_export_model_job,
    'BigqueryEvaluateModelJob':
        bigquery_job_remote_runner.bigquery_evaluate_model_job,
    'BigqueryMLArimaCoefficientsJob':
        bigquery_job_remote_runner.bigquery_ml_arima_coefficients,
    'BigqueryMLArimaEvaluateJob':
        bigquery_job_remote_runner.bigquery_ml_arima_evaluate_job,
    'BigqueryMLCentroidsJob':
        bigquery_job_remote_runner.bigquery_ml_centroids_job,
    'BigqueryMLWeightsJob':
        bigquery_job_remote_runner.bigquery_ml_weights_job,
    'BigqueryMLReconstructionLossJob':
        bigquery_job_remote_runner.bigquery_ml_reconstruction_loss_job,
    'BigqueryMLTrialInfoJob':
        bigquery_job_remote_runner.bigquery_ml_trial_info_job,
    'BigqueryMLTrainingInfoJob':
        bigquery_job_remote_runner.bigquery_ml_training_info_job,
    'BigqueryMLAdvancedWeightsJob':
        bigquery_job_remote_runner.bigquery_ml_advanced_weights_job,
    'BigqueryMLConfusionMatrixJob':
        bigquery_job_remote_runner.bigquery_ml_confusion_matrix_job,
    'BigqueryMLFeatureInfoJob':
        bigquery_job_remote_runner.bigquery_ml_feature_info_job,
    'BigqueryMLRocCurveJob':
        bigquery_job_remote_runner.bigquery_ml_roc_curve_job,
    'BigqueryMLPrincipalComponentsJob':
        bigquery_job_remote_runner.bigquery_ml_principal_components_job,
    'BigqueryMLPrincipalComponentInfoJob':
        bigquery_job_remote_runner.bigquery_ml_principal_component_info_job,
    'BigqueryMLFeatureImportanceJob':
        bigquery_job_remote_runner.bigquery_ml_feature_importance_job,
    'BigqueryMLRecommendJob':
        bigquery_job_remote_runner.bigquery_ml_recommend_job,
    'BigqueryMLGlobalExplainJob':
        bigquery_job_remote_runner.bigquery_ml_global_explain_job,
    'BigqueryDetectAnomaliesModelJob':
        bigquery_job_remote_runner.bigquery_detect_anomalies_model_job,
    'DataprocPySparkBatch':
        dataproc_batch_remote_runner.create_pyspark_batch,
    'DataprocSparkBatch':
        dataproc_batch_remote_runner.create_spark_batch,
    'DataprocSparkRBatch':
        dataproc_batch_remote_runner.create_spark_r_batch,
    'DataprocSparkSqlBatch':
        dataproc_batch_remote_runner.create_spark_sql_batch,
    'Wait':
        wait_gcp_resources.wait_gcp_resources
}


def _make_parent_dirs_and_return_path(file_path: str):
  os.makedirs(os.path.dirname(file_path), exist_ok=True)
  return file_path


def _parse_args(args):
  """Parse command line arguments."""
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
          'UploadModel',
          'CreateEndpoint',
          'BatchPredictionJob',
          'BigqueryQueryJob',
          'BigqueryCreateModelJob',
          'BigqueryPredictModelJob',
          'BigQueryEvaluateModelJob',
          'BigQueryMLArimaCoefficientsJob',
          'BigQueryMLArimaEvaluateJob',
          'BigQueryMLWeightsJob',
          'BigqueryMLTrialInfoJob',
          'BigqueryMLReconstructionLossJob',
          'BigqueryMLTrainingInfoJob',
          'BigqueryMLAdvancedWeightsJob',
          'BigqueryDropModelJob',
          'BigqueryMLCentroidsJob',
          'BigqueryMLConfusionMatrixJob',
          'BigqueryMLFeatureInfoJob',
          'BigqueryMLRocCurveJob',
          'BigQueryMLPrincipalComponentsJob',
          'BigQueryMLPrincipalComponentInfoJob',
          'BigqueryMLRecommendJob',
          'BigqueryExplainForecastModelJob'
          'BigqueryMLForecastJob',
          'BigqueryDetectAnomaliesModelJob'
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
      required=(parsed_args.type in {
          'BigqueryQueryJob',
          'BigqueryCreateModelJob',
          'BigqueryPredictModelJob',
          'BigQueryEvaluateModelJob',
          'BigQueryMLWeightsJob',
          'BigqueryMLTrialInfoJob',
          'BigqueryMLReconstructionLossJob',
          'BigqueryMLTrainingInfoJob',
          'BigqueryMLAdvancedWeightsJob',
          'BigqueryDropModelJob',
          'BigqueryMLCentroidsJob',
          'BigqueryMLConfusionMatrixJob',
          'BigqueryMLFeatureInfoJob',
          'BigqueryMLRocCurveJob',
          'BigQueryMLPrincipalComponentsJob',
          'BigQueryMLPrincipalComponentInfoJob',
          'BigQueryMLPrincipalComponentInfoJob',
          'BigqueryMLRecommendJob',
          'BigqueryExplainForecastModelJob',
          'BigqueryMLForecastJob',
          'BigqueryDetectAnomaliesModelJob'
      }),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--model_name',
      dest='model_name',
      type=str,
      required=(parsed_args.type in {
          'BigqueryPredictModelJob',
          'BigqueryExportModelJob',
          'BigQueryEvaluateModelJob',
          'BigQueryMLArimaCoefficientsJob',
          'BigQueryMLArimaEvaluateJob',
          'BigQueryMLWeightsJob',
          'BigqueryMLTrialInfoJob',
          'BigqueryMLReconstructionLossJob',
          'BigqueryMLTrainingInfoJob',
          'BigqueryMLAdvancedWeightsJob',
          'BigqueryDropModelJob',
          'BigqueryMLCentroidsJob',
          'BigqueryMLConfusionMatrixJob',
          'BigqueryMLFeatureInfoJob',
          'BigqueryMLRocCurveJob',
          'BigQueryMLPrincipalComponentsJob',
          'BigQueryMLPrincipalComponentInfoJob',
          'BigqueryMLFeatureImportanceJob',
          'BigqueryMLRecommendJob',
          'BigqueryExplainForecastModelJob',
          'BigqueryMLForecastJob',
          'BigqueryDetectAnomaliesModelJob'
      }),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--model_destination_path',
      dest='model_destination_path',
      type=str,
      required=(parsed_args.type == 'BigqueryExportModelJob'),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--exported_model_path',
      dest='exported_model_path',
      type=str,
      required=(parsed_args.type == 'BigqueryExportModelJob'),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--table_name',
      dest='table_name',
      type=str,
      # table_name is only needed for BigQuery tvf model job component.
      required=(parsed_args.type in {
          'BigqueryPredictModelJob', 'BigQueryEvaluateModelJob',
          'BigqueryMLReconstructionLossJob', 'BigqueryMLConfusionMatrixJob',
          'BigqueryMLRocCurveJob'
      }),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--query_statement',
      dest='query_statement',
      type=str,
      # query_statement is only needed for BigQuery predict model job component.
      required=(parsed_args.type in {
          'BigqueryPredictModelJob', 'BigQueryEvaluateModelJob',
          'BigqueryMLReconstructionLossJob', 'BigqueryMLConfusionMatrixJob',
          'BigqueryMLRocCurveJob'
      }),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--threshold',
      dest='threshold',
      type=float,
      # threshold is only needed for BigQuery tvf model job component.
      required=(parsed_args.type in {
          'BigqueryPredictModelJob', 'BigQueryEvaluateModelJob',
          'BigqueryMLConfusionMatrixJob'
      }),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--thresholds',
      dest='thresholds',
      type=str,
      # thresholds is only needed for BigQuery tvf model job component.
      required=(parsed_args.type in {'BigqueryMLRocCurveJob'}),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--batch_id',
      dest='batch_id',
      type=str,
      required=(parsed_args.type in {
          'DataprocPySparkBatch', 'DataprocSparkBatch', 'DataprocSparkRBatch',
          'DataprocSparkSqlBatch'
      }),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--standardize',
      dest='standardize',
      type=bool,
      required=(parsed_args.type in {'BigqueryMLCentroidsJob'}),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--class_level_explain',
      dest='class_level_explain',
      type=bool,
      required=False,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--show_all_candidate_models',
      dest='show_all_candidate_models',
      type=bool,
      required=(parsed_args.type in {'BigqueryMLArimaEvaluateJob'}),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--horizon',
      dest='horizon',
      type=int,
      required=(parsed_args.type
                in {'BigqueryMLForecastJob',
                    'BigqueryExplainForecastModelJob'}),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--confidence_level',
      dest='confidence_level',
      type=float,
      required=(parsed_args.type
                in {'BigqueryMLForecastJob',
                    'BigqueryExplainForecastModelJob'}),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--contamination',
      dest='contamination',
      type=float,
      required=(parsed_args.type in {'BigqueryDetectAnomaliesModelJob'}),
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--anomaly_prob_threshold',
      dest='anomaly_prob_threshold',
      type=float,
      required=(parsed_args.type in {'BigqueryDetectAnomaliesModelJob'}),
      default=argparse.SUPPRESS)
  parsed_args, _ = parser.parse_known_args(args)
  return vars(parsed_args)


def main(argv):
  """Main entry.

  Expected input args are as follows:
    Project - Required. The project of which the resource will be launched.
    Region - Required. The region of which the resource will be launched.
    Type - Required. GCP launcher is a single container. This Enum will
        specify which resource to be launched.
    Request payload - Required. The full serialized json of the resource spec.
        Note this can contain the Pipeline Placeholders.
    gcp_resources - placeholder output for returning job_id.

  Args:
    argv: A list of system arguments.
  """
  parsed_args = _parse_args(argv)
  job_type = parsed_args['type']

  if job_type not in _JOB_TYPE_TO_ACTION_MAP:
    raise ValueError('Unsupported job type: ' + job_type)

  logging.info('Job started for type: ' + job_type)

  _JOB_TYPE_TO_ACTION_MAP[job_type](**parsed_args)


if __name__ == '__main__':
  main(sys.argv[1:])
