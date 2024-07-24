# Copyright 2024 The Kubeflow Authors. All Rights Reserved.
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
"""Defines the pipeline for Starry Net."""

from typing import List

# pylint: disable=g-importing-member
from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components._implementation.starry_net import DataprepOp
from google_cloud_pipeline_components._implementation.starry_net import EvaluationOp
from google_cloud_pipeline_components._implementation.starry_net import GetTrainingArtifactsOp
from google_cloud_pipeline_components._implementation.starry_net import MaybeSetTfrecordArgsOp
from google_cloud_pipeline_components._implementation.starry_net import SetDataprepArgsOp
from google_cloud_pipeline_components._implementation.starry_net import SetEvalArgsOp
from google_cloud_pipeline_components._implementation.starry_net import SetTestSetOp
from google_cloud_pipeline_components._implementation.starry_net import SetTfrecordArgsOp
from google_cloud_pipeline_components._implementation.starry_net import SetTrainArgsOp
from google_cloud_pipeline_components._implementation.starry_net import TrainOp
from google_cloud_pipeline_components._implementation.starry_net import UploadDecompositionPlotsOp
from google_cloud_pipeline_components._implementation.starry_net import UploadModelOp
from google_cloud_pipeline_components.preview.model_evaluation import model_evaluation_import_component
from google_cloud_pipeline_components.types import artifact_types
from google_cloud_pipeline_components.v1 import batch_predict_job
from kfp import dsl


@dsl.pipeline
def starry_net(  # pylint: disable=dangerous-default-value
    tensorboard_instance_id: str,
    dataprep_backcast_length: int,
    dataprep_forecast_length: int,
    dataprep_train_end_date: str,
    dataprep_n_val_windows: int,
    dataprep_n_test_windows: int,
    dataprep_test_set_stride: int,
    dataprep_test_set_bigquery_dataset: str,
    dataflow_machine_type: str = 'n1-standard-16',
    dataflow_max_replica_count: int = 50,
    dataflow_starting_replica_count: int = 1,
    dataflow_disk_size_gb: int = 50,
    dataprep_csv_data_path: str = '',
    dataprep_csv_static_covariates_path: str = '',
    dataprep_bigquery_data_path: str = '',
    dataprep_ts_identifier_columns: List[str] = [],
    dataprep_time_column: str = '',
    dataprep_target_column: str = '',
    dataprep_static_covariate_columns: List[str] = [],
    dataprep_previous_run_dir: str = '',
    dataprep_nan_threshold: float = 0.2,
    dataprep_zero_threshold: float = 0.2,
    trainer_machine_type: str = 'n1-standard-4',
    trainer_accelerator_type: str = 'NVIDIA_TESLA_V100',
    trainer_num_epochs: int = 50,
    trainer_cleaning_activation_regularizer_coeff: float = 1e3,
    trainer_change_point_activation_regularizer_coeff: float = 1e3,
    trainer_change_point_output_regularizer_coeff: float = 1e3,
    trainer_trend_alpha_upper_bound: float = 0.5,
    trainer_trend_beta_upper_bound: float = 0.2,
    trainer_trend_phi_lower_bound: float = 0.99,
    trainer_trend_b_fixed_val: int = -1,
    trainer_trend_b0_fixed_val: int = -1,
    trainer_trend_phi_fixed_val: int = -1,
    trainer_quantiles: List[float] = [],
    trainer_model_blocks: List[str] = [
        'cleaning',
        'change_point',
        'trend',
        'day_of_week',
        'week_of_year',
        'residual',
    ],
    tensorboard_n_decomposition_plots: int = 25,
    encryption_spec_key_name: str = '',
    location: str = _placeholders.LOCATION_PLACEHOLDER,
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
):
  # fmt: off
  """Starry Net is a state-of-the-art forecaster used internally by Google.

  Starry Net is a glass-box neural network inspired by statistical time series
  models, capable of cleaning step changes and spikes, modeling seasonality and
  events, forecasting trend, and providing both point and prediction interval
  forecasts in a single, lightweight model. Starry Net stands out among neural
  network based forecasting models by providing the explainability,
  interpretability and tunability of traditional statistical forecasters.
  For example, it features time series feature decomposition and damped local
  linear exponential smoothing model as the trend structure.

  Args:
    tensorboard_instance_id: The tensorboard instance ID. This must be in same
      location as the pipeline job.
    dataprep_backcast_length: The length of the context window to feed into the
      model.
    dataprep_forecast_length: The length of the forecast horizon used in the
      loss function during training and during evaluation, so that the model is
      optimized to produce forecasts from 0 to H.
    dataprep_train_end_date: The last date of data to use in the training and
      validation set. All dates after a train_end_date are part of the test set.
      If last_forecasted_date is equal to the final day forecasted in the test
      set, then last_forecasted_date =
          train_end_date + forecast_length + (n_test_windows * test_set_stride).
      last_forecasted_date must be included in the dataset.
    dataprep_n_val_windows: The number of windows to use for the val set. If 0,
      no validation set is used.
    dataprep_n_test_windows: The number of windows to use for the test set. Must
      be >= 1. See note in dataprep_train_end_date.
    dataprep_test_set_stride: The number of timestamps to roll forward
      when constructing each window of the val and test sets. See note in
      dataprep_train_end_date.
    dataprep_test_set_bigquery_dataset: The bigquery dataset where the test set
      is saved in the format bq://project.dataset. This must be in the same
      region or multi-region as the output or staging bucket of the pipeline and
      the dataprep_bigquery_data_path, if using a Big Query data source.
    dataflow_machine_type: The type of machine to use for dataprep,
      batch prediction, and evaluation jobs..
    dataflow_max_replica_count: The maximum number of replicas to scale the
      dataprep, batch prediction, and evaluation jobs.
    dataflow_starting_replica_count: The number of replicas to start the
      dataprep, batch prediction, and evaluation jobs.
    dataflow_disk_size_gb: The disk size of dataflow workers in GB for the
      dataprep, batch prediction, and evaluation jobs.
    dataprep_csv_data_path: The path to the training data csv in the format
      gs://bucket_name/sub_dir/blob_name.csv. Each row of the csv represents
      a time series, where the column names are the dates, and the index is the
      unique time series names.
    dataprep_csv_static_covariates_path: The path to the static covariates csv.
      Each row of the csv represents the static covariate values for the series,
      where the column names are the static covariate names, and the
      index is the unique time series names. The index values must match the
      index values of dataprep_csv_data_path. The column values must match
      dataprep_static_covariate_columns.
    dataprep_bigquery_data_path: The path to the training data on BigQuery in
      the format bq://project.dataset.table_id. You should only set this or
       csv_data_path. This must be in the same region or multi-region as the
       output or staging bucket of the pipeline and the
       dataprep_test_set_bigquery_dataset.
    dataprep_ts_identifier_columns: The list of ts_identifier columns from the
      BigQuery data source. These columns are used to distinguish the different
      time series, so that if multiple rows have identical ts_identifier
      columns, the series is generated by summing the target columns for each
      timestamp. This is only used if dataprep_bigquery_data_path is set.
    dataprep_time_column: The time column from the BigQuery data source. This is
      only used if dataprep_bigquery_data_path is set.
    dataprep_target_column: The column to be forecasted from the BigQuery data
      source. This is only used if dataprep_bigquery_data_path is set.
    dataprep_static_covariate_columns: The list of strings of static covariate
      names. This needs to be set if training with static covariates regardless
      of whether you're using bigquery_data_path or csv_static_covariates_path.
    dataprep_previous_run_dir: The dataprep dir from a previous run. Use this
      to save time if you've already created TFRecords from your BigQuery
      dataset with the same dataprep parameters as this run.
    dataprep_nan_threshold: Series having more nan / missing values than
      nan_threshold (inclusive) in percentage for either backtest or forecast
      will not be sampled in the training set (including missing due to
      train_start and train_end). All existing nans are replaced by zeros.
    dataprep_zero_threshold: Series having more 0.0 values than zero_threshold
      (inclusive) in percentage for either backtest or forecast will not be
      sampled in the training set.
    trainer_machine_type: The machine type for training. Must be compatible with
      trainer_accelerator_type.
    trainer_accelerator_type: The accelerator type for training.
    trainer_num_epochs: The number of epochs to train for.
    trainer_cleaning_activation_regularizer_coeff: The L1 regularization
      coefficient for the anomaly detection activation in the cleaning block.
      The larger the value, the less aggressive the cleaning, so fewer and only
      the most extreme anomalies are detected. A rule of thumb is that this
      value should be about the same scale of your series.
    trainer_change_point_activation_regularizer_coeff: The L1 regularization
      coefficient for the change point detection activation in the change point
      block. The larger the value, the less aggressive the cleaning, so fewer
      and only the most extreme change points are detected. A rule of thumb is
      that this value should be a ratio of the
      trainer_change_point_output_regularizer_coeff to determine the sparsity
      of the changes. If you want the model to detect many small step changes
      this number should be smaller than the
      trainer_change_point_output_regularizer_coeff. To detect fewer large step
      changes, this number should be about equal to or larger than the
      trainer_change_point_output_regularizer_coeff.
    trainer_change_point_output_regularizer_coeff: The L2 regularization
      penalty applied to the mean lag-one difference of the cleaned output of
      the change point block. Intutively,
      trainer_change_point_activation_regularizer_coeff determines how many
      steps to detect in the series, while this parameter determines how
      aggressively to clean the detected steps. The higher this value, the more
      aggressive the cleaning. A rule of thumb is that this value should be
      about the same scale of your series.
    trainer_trend_alpha_upper_bound: The upper bound for data smooth parameter
      alpha in the trend block.
    trainer_trend_beta_upper_bound: The upper bound for trend smooth parameter
      beta in the trend block.
    trainer_trend_phi_lower_bound: The lower bound for damping param phi in the
      trend block.
    trainer_trend_b_fixed_val: The fixed value for long term trend parameter b
      in the trend block. If set to anything other than -1, the trend block will
      not learn to provide estimates but use the fixed value directly.
    trainer_trend_b0_fixed_val: The fixed value for starting short-term trend
      parameter b0 in the trend block. If set to anything other than -1, the
      trend block will not learn to provide estimates but use the fixed value
      directly.
    trainer_trend_phi_fixed_val: The fixed value for the damping parameter phi
      in the trend block. If set to anything other than -1, the trend block will
      not learn to provide estimates but use the fixed value directly.
    trainer_quantiles: The list of floats representing quantiles. Leave blank if
      only training to produce point forecasts.
    trainer_model_blocks: The list of model blocks to use in the order they will
      appear in the model. Possible values are `cleaning`, `change_point`,
      `trend`, `hour_of_week`, `day_of_week`, `day_of_year`, `week_of_year`,
      `month_of_year`, `residual`.
    tensorboard_n_decomposition_plots: How many decomposition plots from the
      test set to save to tensorboard.
    encryption_spec_key_name: Customer-managed encryption key options for the
      CustomJob. If this is set, then all resources created by the CustomJob
      will be encrypted with the provided encryption key.
    location: The location where the pipeline components are run.
    project: The project where the pipeline is run. Defaults to current project.
  """
  job_id = dsl.PIPELINE_JOB_NAME_PLACEHOLDER
  create_dataprep_args_task = SetDataprepArgsOp(
      model_blocks=trainer_model_blocks,
      ts_identifier_columns=dataprep_ts_identifier_columns,
      static_covariate_columns=dataprep_static_covariate_columns,
      csv_data_path=dataprep_csv_data_path,
      previous_run_dir=dataprep_previous_run_dir,
      location=location,
  )
  create_trainer_args_task = SetTrainArgsOp(
      quantiles=trainer_quantiles,
      model_blocks=trainer_model_blocks,
      static_covariates=dataprep_static_covariate_columns,
  )
  with dsl.If(create_dataprep_args_task.outputs['create_tf_records'] == True,  # pylint: disable=singleton-comparison
              'create-tf-records'):
    create_tf_records_task = DataprepOp(
        backcast_length=dataprep_backcast_length,
        forecast_length=dataprep_forecast_length,
        train_end_date=dataprep_train_end_date,
        n_val_windows=dataprep_n_val_windows,
        n_test_windows=dataprep_n_test_windows,
        test_set_stride=dataprep_test_set_stride,
        model_blocks=create_dataprep_args_task.outputs['model_blocks'],
        bigquery_source=dataprep_bigquery_data_path,
        ts_identifier_columns=create_dataprep_args_task.outputs[
            'ts_identifier_columns'],
        time_column=dataprep_time_column,
        static_covariate_columns=create_dataprep_args_task.outputs[
            'static_covariate_columns'],
        static_covariates_vocab_path='',
        target_column=dataprep_target_column,
        machine_type=dataflow_machine_type,
        docker_region=create_dataprep_args_task.outputs['docker_region'],
        location=location,
        project=project,
        job_id=job_id,
        job_name_prefix='tf-records',
        num_workers=dataflow_starting_replica_count,
        max_num_workers=dataflow_max_replica_count,
        disk_size_gb=dataflow_disk_size_gb,
        test_set_only=False,
        bigquery_output=dataprep_test_set_bigquery_dataset,
        nan_threshold=dataprep_nan_threshold,
        zero_threshold=dataprep_zero_threshold,
        gcs_source=dataprep_csv_data_path,
        gcs_static_covariate_source=dataprep_csv_static_covariates_path,
        encryption_spec_key_name=encryption_spec_key_name
    )
    create_tf_records_task.set_display_name('create-tf-records')
    set_tfrecord_args_this_run_task = (
        SetTfrecordArgsOp(
            dataprep_dir=create_tf_records_task.outputs['dataprep_dir'],
            static_covariates=dataprep_static_covariate_columns))
  with dsl.Else('skip-tf-record-generation'):
    set_tfrecord_args_previous_run_task = (
        MaybeSetTfrecordArgsOp(
            dataprep_previous_run_dir=dataprep_previous_run_dir,
            static_covariates=dataprep_static_covariate_columns))
    set_tfrecord_args_previous_run_task.set_display_name(
        'set_tfrecord_args_previous_run')
  static_covariates_vocab_path = dsl.OneOf(
      set_tfrecord_args_previous_run_task.outputs[
          'static_covariates_vocab_path'],
      set_tfrecord_args_this_run_task.outputs['static_covariates_vocab_path']
  )
  test_set_task = DataprepOp(
      backcast_length=dataprep_backcast_length,
      forecast_length=dataprep_forecast_length,
      train_end_date=dataprep_train_end_date,
      n_val_windows=dataprep_n_val_windows,
      n_test_windows=dataprep_n_test_windows,
      test_set_stride=dataprep_test_set_stride,
      model_blocks=create_dataprep_args_task.outputs['model_blocks'],
      bigquery_source=dataprep_bigquery_data_path,
      ts_identifier_columns=create_dataprep_args_task.outputs[
          'ts_identifier_columns'],
      time_column=dataprep_time_column,
      static_covariate_columns=create_dataprep_args_task.outputs[
          'static_covariate_columns'],
      static_covariates_vocab_path=static_covariates_vocab_path,
      target_column=dataprep_target_column,
      machine_type=dataflow_machine_type,
      docker_region=create_dataprep_args_task.outputs['docker_region'],
      location=location,
      project=project,
      job_id=job_id,
      job_name_prefix='test-set',
      num_workers=dataflow_starting_replica_count,
      max_num_workers=dataflow_max_replica_count,
      disk_size_gb=dataflow_disk_size_gb,
      test_set_only=True,
      bigquery_output=dataprep_test_set_bigquery_dataset,
      nan_threshold=dataprep_nan_threshold,
      zero_threshold=dataprep_zero_threshold,
      gcs_source=dataprep_csv_data_path,
      gcs_static_covariate_source=dataprep_csv_static_covariates_path,
      encryption_spec_key_name=encryption_spec_key_name
  )
  test_set_task.set_display_name('create-test-set')
  set_test_set_task = SetTestSetOp(
      dataprep_dir=test_set_task.outputs['dataprep_dir'])
  train_tf_record_patterns = dsl.OneOf(
      set_tfrecord_args_previous_run_task.outputs['train_tf_record_patterns'],
      set_tfrecord_args_this_run_task.outputs['train_tf_record_patterns']
  )
  val_tf_record_patterns = dsl.OneOf(
      set_tfrecord_args_previous_run_task.outputs['val_tf_record_patterns'],
      set_tfrecord_args_this_run_task.outputs['val_tf_record_patterns']
  )
  test_tf_record_patterns = dsl.OneOf(
      set_tfrecord_args_previous_run_task.outputs['test_tf_record_patterns'],
      set_tfrecord_args_this_run_task.outputs['test_tf_record_patterns']
  )
  trainer_task = TrainOp(
      num_epochs=trainer_num_epochs,
      backcast_length=dataprep_backcast_length,
      forecast_length=dataprep_forecast_length,
      train_end_date=dataprep_train_end_date,
      csv_data_path=dataprep_csv_data_path,
      csv_static_covariates_path=dataprep_csv_static_covariates_path,
      static_covariates_vocab_path=static_covariates_vocab_path,
      train_tf_record_patterns=train_tf_record_patterns,
      val_tf_record_patterns=val_tf_record_patterns,
      test_tf_record_patterns=test_tf_record_patterns,
      n_decomposition_plots=tensorboard_n_decomposition_plots,
      n_val_windows=dataprep_n_val_windows,
      n_test_windows=dataprep_n_test_windows,
      test_set_stride=dataprep_test_set_stride,
      nan_threshold=dataprep_nan_threshold,
      zero_threshold=dataprep_zero_threshold,
      cleaning_activation_regularizer_coeff=trainer_cleaning_activation_regularizer_coeff,
      change_point_activation_regularizer_coeff=trainer_change_point_activation_regularizer_coeff,
      change_point_output_regularizer_coeff=trainer_change_point_output_regularizer_coeff,
      alpha_upper_bound=trainer_trend_alpha_upper_bound,
      beta_upper_bound=trainer_trend_beta_upper_bound,
      phi_lower_bound=trainer_trend_phi_lower_bound,
      b_fixed_val=trainer_trend_b_fixed_val,
      b0_fixed_val=trainer_trend_b0_fixed_val,
      phi_fixed_val=trainer_trend_phi_fixed_val,
      quantiles=create_trainer_args_task.outputs['quantiles'],
      use_static_covariates=create_trainer_args_task.outputs[
          'use_static_covariates'],
      static_covariate_names=create_trainer_args_task.outputs[
          'static_covariate_names'],
      model_blocks=create_trainer_args_task.outputs['model_blocks'],
      freeze_point_forecasts=create_trainer_args_task.outputs[
          'freeze_point_forecasts'],
      machine_type=trainer_machine_type,
      accelerator_type=trainer_accelerator_type,
      docker_region=create_dataprep_args_task.outputs['docker_region'],
      location=location,
      job_id=job_id,
      project=project,
      encryption_spec_key_name=encryption_spec_key_name
  )
  _ = UploadDecompositionPlotsOp(
      project=project,
      location=location,
      tensorboard_id=tensorboard_instance_id,
      display_name=job_id,
      trainer_dir=trainer_task.outputs['trainer_dir'])
  training_artifacts_task = GetTrainingArtifactsOp(
      docker_region=create_dataprep_args_task.outputs['docker_region'],
      trainer_dir=trainer_task.outputs['trainer_dir'])
  model = dsl.importer(
      artifact_uri=training_artifacts_task.outputs['artifact_uri'],
      artifact_class=artifact_types.UnmanagedContainerModel,
      metadata={
          'predictSchemata': {
              'instanceSchemaUri': training_artifacts_task.outputs[
                  'instance_schema_uri'],
              'predictionSchemaUri': training_artifacts_task.outputs[
                  'prediction_schema_uri'],
          },
          'containerSpec': {
              'imageUri': training_artifacts_task.outputs['image_uri'],
              'healthRoute': '/health',
              'predictRoute': '/predict',
          }
      },
  )
  model.set_display_name('set-model')
  upload_model_task = UploadModelOp(
      project=project,
      location=location,
      display_name=job_id,
      unmanaged_container_model=model.output,
      encryption_spec_key_name=encryption_spec_key_name,
  )
  upload_model_task.set_display_name('upload-model')
  batch_predict_task = batch_predict_job.ModelBatchPredictOp(
      project=project,
      location=location,
      unmanaged_container_model=model.output,
      job_display_name=f'batch-predict-{job_id}',
      instances_format='bigquery',
      predictions_format='bigquery',
      bigquery_source_input_uri=set_test_set_task.outputs['uri'],
      bigquery_destination_output_uri=dataprep_test_set_bigquery_dataset,
      machine_type=dataflow_machine_type,
      starting_replica_count=dataflow_starting_replica_count,
      max_replica_count=dataflow_max_replica_count,
      encryption_spec_key_name=encryption_spec_key_name,
      generate_explanation=False,
  )
  batch_predict_task.set_display_name('run-batch-prediction')
  set_eval_args_task = SetEvalArgsOp(
      big_query_source=batch_predict_task.outputs['bigquery_output_table'],
      quantiles=trainer_quantiles)
  eval_task = EvaluationOp(
      project=project,
      location=location,
      root_dir=test_set_task.outputs['dataprep_dir'],
      target_field_name='HORIZON__x',
      predictions_format='bigquery',
      ground_truth_format='bigquery',
      predictions_bigquery_source=batch_predict_task.outputs[
          'bigquery_output_table'],
      ground_truth_bigquery_source=set_eval_args_task.outputs[
          'big_query_source'],
      ground_truth_gcs_source=[],
      forecasting_type=set_eval_args_task.outputs['forecasting_type'],
      forecasting_quantiles=set_eval_args_task.outputs['quantiles'],
      prediction_score_column=set_eval_args_task.outputs[
          'prediction_score_column'],
      dataflow_service_account=_placeholders.SERVICE_ACCOUNT_PLACEHOLDER,
      dataflow_machine_type=dataflow_machine_type,
      dataflow_max_workers_num=dataflow_max_replica_count,
      dataflow_workers_num=dataflow_starting_replica_count,
      dataflow_disk_size=dataflow_disk_size_gb,
      dataflow_use_public_ips=True,
      encryption_spec_key_name=encryption_spec_key_name,
  )
  model_evaluation_import_component.model_evaluation_import(
      forecasting_metrics=eval_task.outputs['evaluation_metrics'],
      model=upload_model_task.outputs['model'],
      dataset_type='bigquery',
      dataset_path=set_test_set_task.outputs['uri'],
      display_name=job_id,
      problem_type='forecasting',
    )
