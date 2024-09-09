"""Util functions for Vertex Forecasting pipelines."""

import logging
import os
import pathlib
from typing import Any, Dict, FrozenSet, List, Optional, Tuple

_GCPC_FORECASTING_PATH = pathlib.Path(__file__).parent.resolve()

_RETAIL_MODEL_DISABLED_OPTIONS = frozenset([
    'quantiles',
    'enable_probabilistic_inference',
])


def _validate_start_max_parameters(
    starting_worker_count: int,
    max_worker_count: int,
    starting_count_name: str,
    max_count_name: str,
):
  if starting_worker_count > max_worker_count:
    raise ValueError(
        'Starting count must be less than or equal to max count.'
        f' {starting_count_name}: {starting_worker_count}, {max_count_name}:'
        f' {max_worker_count}'
    )


def _get_base_forecasting_parameters(
    *,
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    optimization_objective: str,
    transformations: Dict[str, List[str]],
    train_budget_milli_node_hours: float,
    time_column: str,
    time_series_identifier_columns: List[str],
    time_series_identifier_column: Optional[str] = None,
    time_series_attribute_columns: Optional[List[str]] = None,
    available_at_forecast_columns: Optional[List[str]] = None,
    unavailable_at_forecast_columns: Optional[List[str]] = None,
    forecast_horizon: Optional[int] = None,
    context_window: Optional[int] = None,
    evaluated_examples_bigquery_path: Optional[str] = None,
    window_predefined_column: Optional[str] = None,
    window_stride_length: Optional[int] = None,
    window_max_count: Optional[int] = None,
    holiday_regions: Optional[List[str]] = None,
    stage_1_num_parallel_trials: Optional[int] = None,
    stage_1_tuning_result_artifact_uri: Optional[str] = None,
    stage_2_num_parallel_trials: Optional[int] = None,
    num_selected_trials: Optional[int] = None,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    predefined_split_key: Optional[str] = None,
    timestamp_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    weight_column: Optional[str] = None,
    dataflow_service_account: Optional[str] = None,
    dataflow_subnetwork: Optional[str] = None,
    dataflow_use_public_ips: bool = True,
    feature_transform_engine_bigquery_staging_full_dataset_id: str = '',
    feature_transform_engine_dataflow_machine_type: str = 'n1-standard-16',
    feature_transform_engine_dataflow_max_num_workers: int = 10,
    feature_transform_engine_dataflow_disk_size_gb: int = 40,
    evaluation_batch_predict_machine_type: str = 'n1-standard-16',
    evaluation_batch_predict_starting_replica_count: int = 25,
    evaluation_batch_predict_max_replica_count: int = 25,
    evaluation_dataflow_machine_type: str = 'n1-standard-16',
    evaluation_dataflow_max_num_workers: int = 25,
    evaluation_dataflow_starting_num_workers: int = 22,
    evaluation_dataflow_disk_size_gb: int = 50,
    study_spec_parameters_override: Optional[List[Dict[str, Any]]] = None,
    stage_1_tuner_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    stage_2_trainer_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    enable_probabilistic_inference: bool = False,
    quantiles: Optional[List[float]] = None,
    encryption_spec_key_name: Optional[str] = None,
    model_display_name: Optional[str] = None,
    model_description: Optional[str] = None,
    run_evaluation: bool = True,
    group_columns: Optional[List[str]] = None,
    group_total_weight: float = 0.0,
    temporal_total_weight: float = 0.0,
    group_temporal_total_weight: float = 0.0,
    fields_to_exclude: FrozenSet[str] = frozenset(),
) -> Dict[str, Any]:
  """Formats a set of parameters common across Vertex forecasting pipelines."""
  if not study_spec_parameters_override:
    study_spec_parameters_override = []
  if not stage_1_tuner_worker_pool_specs_override:
    stage_1_tuner_worker_pool_specs_override = []
  if not stage_2_trainer_worker_pool_specs_override:
    stage_2_trainer_worker_pool_specs_override = []

  if time_series_identifier_column:
    logging.warning(
        'Deprecation warning: `time_series_identifier_column` will soon be'
        ' deprecated in favor of `time_series_identifier_columns`. Please'
        ' migrate workloads to use the new field.'
    )
    time_series_identifier_columns = [time_series_identifier_column]

  _validate_start_max_parameters(
      starting_worker_count=evaluation_batch_predict_starting_replica_count,
      max_worker_count=evaluation_batch_predict_max_replica_count,
      starting_count_name='evaluation_batch_predict_starting_replica_count',
      max_count_name='evaluation_batch_predict_max_replica_count',
  )

  _validate_start_max_parameters(
      starting_worker_count=evaluation_dataflow_starting_num_workers,
      max_worker_count=evaluation_dataflow_max_num_workers,
      starting_count_name='evaluation_dataflow_starting_num_workers',
      max_count_name='evaluation_dataflow_max_num_workers',
  )

  parameter_values = {}
  parameters = {
      'project': project,
      'location': location,
      'root_dir': root_dir,
      'dataflow_service_account': dataflow_service_account,
      'evaluated_examples_bigquery_path': evaluated_examples_bigquery_path,
      'target_column': target_column,
      'optimization_objective': optimization_objective,
      'transformations': transformations,
      'train_budget_milli_node_hours': train_budget_milli_node_hours,
      'time_column': time_column,
      'time_series_identifier_columns': time_series_identifier_columns,
      'time_series_attribute_columns': time_series_attribute_columns,
      'available_at_forecast_columns': available_at_forecast_columns,
      'unavailable_at_forecast_columns': unavailable_at_forecast_columns,
      'forecast_horizon': forecast_horizon,
      'context_window': context_window,
      'window_predefined_column': window_predefined_column,
      'window_stride_length': window_stride_length,
      'window_max_count': window_max_count,
      'holiday_regions': holiday_regions,
      'stage_1_num_parallel_trials': stage_1_num_parallel_trials,
      'stage_1_tuning_result_artifact_uri': stage_1_tuning_result_artifact_uri,
      'stage_2_num_parallel_trials': stage_2_num_parallel_trials,
      'num_selected_trials': num_selected_trials,
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'predefined_split_key': predefined_split_key,
      'timestamp_split_key': timestamp_split_key,
      'training_fraction': training_fraction,
      'validation_fraction': validation_fraction,
      'test_fraction': test_fraction,
      'weight_column': weight_column,
      'dataflow_subnetwork': dataflow_subnetwork,
      'feature_transform_engine_dataflow_machine_type': (
          feature_transform_engine_dataflow_machine_type
      ),
      'feature_transform_engine_dataflow_max_num_workers': (
          feature_transform_engine_dataflow_max_num_workers
      ),
      'feature_transform_engine_dataflow_disk_size_gb': (
          feature_transform_engine_dataflow_disk_size_gb
      ),
      'dataflow_use_public_ips': dataflow_use_public_ips,
      'feature_transform_engine_bigquery_staging_full_dataset_id': (
          feature_transform_engine_bigquery_staging_full_dataset_id
      ),
      'evaluation_batch_predict_machine_type': (
          evaluation_batch_predict_machine_type
      ),
      'evaluation_batch_predict_starting_replica_count': (
          evaluation_batch_predict_starting_replica_count
      ),
      'evaluation_batch_predict_max_replica_count': (
          evaluation_batch_predict_max_replica_count
      ),
      'evaluation_dataflow_machine_type': evaluation_dataflow_machine_type,
      'evaluation_dataflow_max_num_workers': (
          evaluation_dataflow_max_num_workers
      ),
      'evaluation_dataflow_starting_num_workers': (
          evaluation_dataflow_starting_num_workers
      ),
      'evaluation_dataflow_disk_size_gb': evaluation_dataflow_disk_size_gb,
      'study_spec_parameters_override': study_spec_parameters_override,
      'stage_1_tuner_worker_pool_specs_override': (
          stage_1_tuner_worker_pool_specs_override
      ),
      'stage_2_trainer_worker_pool_specs_override': (
          stage_2_trainer_worker_pool_specs_override
      ),
      'quantiles': quantiles,
      'encryption_spec_key_name': encryption_spec_key_name,
      'enable_probabilistic_inference': enable_probabilistic_inference,
      'model_display_name': model_display_name,
      'model_description': model_description,
      'run_evaluation': run_evaluation,
      'group_columns': group_columns,
      'group_total_weight': group_total_weight,
      'temporal_total_weight': temporal_total_weight,
      'group_temporal_total_weight': group_temporal_total_weight,
  }

  # Filter out empty values and those excluded from the particular pipeline.
  # (example: TFT and Seq2Seq don't support `quantiles`.)
  parameter_values.update({
      param: value
      for param, value in parameters.items()
      if value is not None and param not in fields_to_exclude
  })
  return parameter_values


def get_learn_to_learn_forecasting_pipeline_and_parameters(
    *,
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    optimization_objective: str,
    transformations: Dict[str, List[str]],
    train_budget_milli_node_hours: float,
    time_column: str,
    time_series_identifier_columns: List[str],
    time_series_identifier_column: Optional[str] = None,
    time_series_attribute_columns: Optional[List[str]] = None,
    available_at_forecast_columns: Optional[List[str]] = None,
    unavailable_at_forecast_columns: Optional[List[str]] = None,
    forecast_horizon: Optional[int] = None,
    context_window: Optional[int] = None,
    evaluated_examples_bigquery_path: Optional[str] = None,
    window_predefined_column: Optional[str] = None,
    window_stride_length: Optional[int] = None,
    window_max_count: Optional[int] = None,
    holiday_regions: Optional[List[str]] = None,
    stage_1_num_parallel_trials: Optional[int] = None,
    stage_1_tuning_result_artifact_uri: Optional[str] = None,
    stage_2_num_parallel_trials: Optional[int] = None,
    num_selected_trials: Optional[int] = None,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    predefined_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    weight_column: Optional[str] = None,
    dataflow_service_account: Optional[str] = None,
    dataflow_subnetwork: Optional[str] = None,
    dataflow_use_public_ips: bool = True,
    feature_transform_engine_bigquery_staging_full_dataset_id: str = '',
    feature_transform_engine_dataflow_machine_type: str = 'n1-standard-16',
    feature_transform_engine_dataflow_max_num_workers: int = 10,
    feature_transform_engine_dataflow_disk_size_gb: int = 40,
    evaluation_batch_predict_machine_type: str = 'n1-standard-16',
    evaluation_batch_predict_starting_replica_count: int = 25,
    evaluation_batch_predict_max_replica_count: int = 25,
    evaluation_dataflow_machine_type: str = 'n1-standard-16',
    evaluation_dataflow_max_num_workers: int = 25,
    evaluation_dataflow_starting_num_workers: int = 22,
    evaluation_dataflow_disk_size_gb: int = 50,
    study_spec_parameters_override: Optional[List[Dict[str, Any]]] = None,
    stage_1_tuner_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    stage_2_trainer_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    enable_probabilistic_inference: bool = False,
    quantiles: Optional[List[float]] = None,
    encryption_spec_key_name: Optional[str] = None,
    model_display_name: Optional[str] = None,
    model_description: Optional[str] = None,
    run_evaluation: bool = True,
    group_columns: Optional[List[str]] = None,
    group_total_weight: float = 0.0,
    temporal_total_weight: float = 0.0,
    group_temporal_total_weight: float = 0.0,
) -> Tuple[str, Dict[str, Any]]:
  # fmt: off
  """Returns l2l_forecasting pipeline and formatted parameters.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column: The target column name.
    optimization_objective: "minimize-rmse", "minimize-mae", "minimize-rmsle", "minimize-rmspe", "minimize-wape-mae", "minimize-mape", or "minimize-quantile-loss".
    transformations: Dict mapping auto and/or type-resolutions to feature columns. The supported types are: auto, categorical, numeric, text, and timestamp.
    train_budget_milli_node_hours: The train budget of creating this model, expressed in milli node hours i.e. 1,000 value in this field means 1 node hour.
    time_column: The column that indicates the time.
    time_series_identifier_columns: The columns which distinguish different time series.
    time_series_identifier_column: [Deprecated] The column which distinguishes different time series.
    time_series_attribute_columns: The columns that are invariant across the same time series.
    available_at_forecast_columns: The columns that are available at the forecast time.
    unavailable_at_forecast_columns: The columns that are unavailable at the forecast time.
    forecast_horizon: The length of the horizon.
    context_window: The length of the context window.
    evaluated_examples_bigquery_path: The bigquery dataset to write the predicted examples into for evaluation, in the format `bq://project.dataset`.
    window_predefined_column: The column that indicate the start of each window.
    window_stride_length: The stride length to generate the window.
    window_max_count: The maximum number of windows that will be generated.
    holiday_regions: The geographical regions where the holiday effect is applied in modeling.
    stage_1_num_parallel_trials: Number of parallel trails for stage 1.
    stage_1_tuning_result_artifact_uri: The stage 1 tuning result artifact GCS URI.
    stage_2_num_parallel_trials: Number of parallel trails for stage 2.
    num_selected_trials: Number of selected trails.
    data_source_csv_filenames: A string that represents a list of comma separated CSV filenames.
    data_source_bigquery_table_path: The BigQuery table path of format bq://bq_project.bq_dataset.bq_table
    predefined_split_key: The predefined_split column name.
    training_fraction: The training fraction.
    validation_fraction: The validation fraction.
    test_fraction: The test fraction.
    weight_column: The weight column name.
    dataflow_service_account: The full service account name.
    dataflow_subnetwork: The dataflow subnetwork.
    dataflow_use_public_ips: `True` to enable dataflow public IPs.
    feature_transform_engine_bigquery_staging_full_dataset_id: The full id of the feature transform engine staging dataset.
    feature_transform_engine_dataflow_machine_type: The dataflow machine type of the feature transform engine.
    feature_transform_engine_dataflow_max_num_workers: The max number of dataflow workers of the feature transform engine.
    feature_transform_engine_dataflow_disk_size_gb: The disk size of the dataflow workers of the feature transform engine.
    evaluation_batch_predict_machine_type: Machine type for the batch prediction job in evaluation, such as 'n1-standard-16'.
    evaluation_batch_predict_starting_replica_count: Number of replicas to use in the batch prediction cluster at startup time.
    evaluation_batch_predict_max_replica_count: The maximum count of replicas the batch prediction job can scale to.
    evaluation_dataflow_machine_type: Machine type for the dataflow job in evaluation, such as 'n1-standard-16'.
    evaluation_dataflow_max_num_workers: Maximum number of dataflow workers.
    evaluation_dataflow_starting_num_workers: Starting number of dataflow workers.
    evaluation_dataflow_disk_size_gb: The disk space in GB for dataflow.
    study_spec_parameters_override: The list for overriding study spec.
    stage_1_tuner_worker_pool_specs_override: The dictionary for overriding stage 1 tuner worker pool spec.
    stage_2_trainer_worker_pool_specs_override: The dictionary for overriding stage 2 trainer worker pool spec.
    enable_probabilistic_inference: If probabilistic inference is enabled, the model will fit a distribution that captures the uncertainty of a prediction. If quantiles are specified, then the quantiles of the distribution are also returned.
    quantiles: Quantiles to use for probabilistic inference. Up to 5 quantiles are allowed of values between 0 and 1, exclusive. Represents the quantiles to use for that objective. Quantiles must be unique.
    encryption_spec_key_name: The KMS key name.
    model_display_name: Optional display name for model.
    model_description: Optional description.
    run_evaluation: `True` to evaluate the ensembled model on the test split.
    group_columns: A list of time series attribute column names that define the time series hierarchy.
    group_total_weight: The weight of the loss for predictions aggregated over time series in the same group.
    temporal_total_weight: The weight of the loss for predictions aggregated over the horizon for a single time series.
    group_temporal_total_weight: The weight of the loss for predictions aggregated over both the horizon and time series in the same hierarchy group.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  # fmt: on
  parameter_values = _get_base_forecasting_parameters(
      project=project,
      location=location,
      root_dir=root_dir,
      target_column=target_column,
      evaluated_examples_bigquery_path=evaluated_examples_bigquery_path,
      optimization_objective=optimization_objective,
      transformations=transformations,
      train_budget_milli_node_hours=train_budget_milli_node_hours,
      time_column=time_column,
      dataflow_service_account=dataflow_service_account,
      time_series_identifier_columns=time_series_identifier_columns,
      time_series_identifier_column=time_series_identifier_column,
      time_series_attribute_columns=time_series_attribute_columns,
      available_at_forecast_columns=available_at_forecast_columns,
      unavailable_at_forecast_columns=unavailable_at_forecast_columns,
      forecast_horizon=forecast_horizon,
      context_window=context_window,
      window_predefined_column=window_predefined_column,
      window_stride_length=window_stride_length,
      window_max_count=window_max_count,
      holiday_regions=holiday_regions,
      stage_1_num_parallel_trials=stage_1_num_parallel_trials,
      stage_1_tuning_result_artifact_uri=stage_1_tuning_result_artifact_uri,
      stage_2_num_parallel_trials=stage_2_num_parallel_trials,
      num_selected_trials=num_selected_trials,
      data_source_csv_filenames=data_source_csv_filenames,
      data_source_bigquery_table_path=data_source_bigquery_table_path,
      predefined_split_key=predefined_split_key,
      training_fraction=training_fraction,
      validation_fraction=validation_fraction,
      test_fraction=test_fraction,
      weight_column=weight_column,
      dataflow_use_public_ips=dataflow_use_public_ips,
      dataflow_subnetwork=dataflow_subnetwork,
      feature_transform_engine_bigquery_staging_full_dataset_id=feature_transform_engine_bigquery_staging_full_dataset_id,
      feature_transform_engine_dataflow_machine_type=feature_transform_engine_dataflow_machine_type,
      feature_transform_engine_dataflow_max_num_workers=feature_transform_engine_dataflow_max_num_workers,
      feature_transform_engine_dataflow_disk_size_gb=feature_transform_engine_dataflow_disk_size_gb,
      evaluation_batch_predict_machine_type=evaluation_batch_predict_machine_type,
      evaluation_batch_predict_starting_replica_count=evaluation_batch_predict_starting_replica_count,
      evaluation_batch_predict_max_replica_count=evaluation_batch_predict_max_replica_count,
      evaluation_dataflow_machine_type=evaluation_dataflow_machine_type,
      evaluation_dataflow_max_num_workers=evaluation_dataflow_max_num_workers,
      evaluation_dataflow_starting_num_workers=evaluation_dataflow_starting_num_workers,
      evaluation_dataflow_disk_size_gb=evaluation_dataflow_disk_size_gb,
      study_spec_parameters_override=study_spec_parameters_override,
      stage_1_tuner_worker_pool_specs_override=stage_1_tuner_worker_pool_specs_override,
      stage_2_trainer_worker_pool_specs_override=stage_2_trainer_worker_pool_specs_override,
      quantiles=quantiles,
      encryption_spec_key_name=encryption_spec_key_name,
      enable_probabilistic_inference=enable_probabilistic_inference,
      model_display_name=model_display_name,
      model_description=model_description,
      run_evaluation=run_evaluation,
      group_columns=group_columns,
      group_total_weight=group_total_weight,
      temporal_total_weight=temporal_total_weight,
      group_temporal_total_weight=group_temporal_total_weight,
  )

  pipeline_definition_path = os.path.join(
      _GCPC_FORECASTING_PATH,
      'learn_to_learn_forecasting_pipeline.yaml',
  )

  return pipeline_definition_path, parameter_values


def get_time_series_dense_encoder_forecasting_pipeline_and_parameters(
    *,
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    optimization_objective: str,
    transformations: Dict[str, List[str]],
    train_budget_milli_node_hours: float,
    time_column: str,
    time_series_identifier_columns: List[str],
    time_series_identifier_column: Optional[str] = None,
    time_series_attribute_columns: Optional[List[str]] = None,
    available_at_forecast_columns: Optional[List[str]] = None,
    unavailable_at_forecast_columns: Optional[List[str]] = None,
    forecast_horizon: Optional[int] = None,
    context_window: Optional[int] = None,
    evaluated_examples_bigquery_path: Optional[str] = None,
    window_predefined_column: Optional[str] = None,
    window_stride_length: Optional[int] = None,
    window_max_count: Optional[int] = None,
    holiday_regions: Optional[List[str]] = None,
    stage_1_num_parallel_trials: Optional[int] = None,
    stage_1_tuning_result_artifact_uri: Optional[str] = None,
    stage_2_num_parallel_trials: Optional[int] = None,
    num_selected_trials: Optional[int] = None,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    predefined_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    weight_column: Optional[str] = None,
    dataflow_service_account: Optional[str] = None,
    dataflow_subnetwork: Optional[str] = None,
    dataflow_use_public_ips: bool = True,
    feature_transform_engine_bigquery_staging_full_dataset_id: str = '',
    feature_transform_engine_dataflow_machine_type: str = 'n1-standard-16',
    feature_transform_engine_dataflow_max_num_workers: int = 10,
    feature_transform_engine_dataflow_disk_size_gb: int = 40,
    evaluation_batch_predict_machine_type: str = 'n1-standard-16',
    evaluation_batch_predict_starting_replica_count: int = 25,
    evaluation_batch_predict_max_replica_count: int = 25,
    evaluation_dataflow_machine_type: str = 'n1-standard-16',
    evaluation_dataflow_max_num_workers: int = 25,
    evaluation_dataflow_starting_num_workers: int = 22,
    evaluation_dataflow_disk_size_gb: int = 50,
    study_spec_parameters_override: Optional[List[Dict[str, Any]]] = None,
    stage_1_tuner_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    stage_2_trainer_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    enable_probabilistic_inference: bool = False,
    quantiles: Optional[List[float]] = None,
    encryption_spec_key_name: Optional[str] = None,
    model_display_name: Optional[str] = None,
    model_description: Optional[str] = None,
    run_evaluation: bool = True,
    group_columns: Optional[List[str]] = None,
    group_total_weight: float = 0.0,
    temporal_total_weight: float = 0.0,
    group_temporal_total_weight: float = 0.0,
) -> Tuple[str, Dict[str, Any]]:
  # fmt: off
  """Returns timeseries_dense_encoder_forecasting pipeline and parameters.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column: The target column name.
    optimization_objective: "minimize-rmse", "minimize-mae", "minimize-rmsle", "minimize-rmspe", "minimize-wape-mae", "minimize-mape", or "minimize-quantile-loss".
    transformations: Dict mapping auto and/or type-resolutions to feature columns. The supported types are: auto, categorical, numeric, text, and timestamp.
    train_budget_milli_node_hours: The train budget of creating this model, expressed in milli node hours i.e. 1,000 value in this field means 1 node hour.
    time_column: The column that indicates the time.
    time_series_identifier_columns: The columns which distinguish different time series.
    time_series_identifier_column: [Deprecated] The column which distinguishes different time series.
    time_series_attribute_columns: The columns that are invariant across the same time series.
    available_at_forecast_columns: The columns that are available at the forecast time.
    unavailable_at_forecast_columns: The columns that are unavailable at the forecast time.
    forecast_horizon: The length of the horizon.
    context_window: The length of the context window.
    evaluated_examples_bigquery_path: The bigquery dataset to write the predicted examples into for evaluation, in the format `bq://project.dataset`.
    window_predefined_column: The column that indicate the start of each window.
    window_stride_length: The stride length to generate the window.
    window_max_count: The maximum number of windows that will be generated.
    holiday_regions: The geographical regions where the holiday effect is applied in modeling.
    stage_1_num_parallel_trials: Number of parallel trails for stage 1.
    stage_1_tuning_result_artifact_uri: The stage 1 tuning result artifact GCS URI.
    stage_2_num_parallel_trials: Number of parallel trails for stage 2.
    num_selected_trials: Number of selected trails.
    data_source_csv_filenames: A string that represents a list of comma separated CSV filenames.
    data_source_bigquery_table_path: The BigQuery table path of format bq://bq_project.bq_dataset.bq_table
    predefined_split_key: The predefined_split column name.
    training_fraction: The training fraction.
    validation_fraction: The validation fraction.
    test_fraction: The test fraction.
    weight_column: The weight column name.
    dataflow_service_account: The full service account name.
    dataflow_subnetwork: The dataflow subnetwork.
    dataflow_use_public_ips: `True` to enable dataflow public IPs.
    feature_transform_engine_bigquery_staging_full_dataset_id: The full id of the feature transform engine staging dataset.
    feature_transform_engine_dataflow_machine_type: The dataflow machine type of the feature transform engine.
    feature_transform_engine_dataflow_max_num_workers: The max number of dataflow workers of the feature transform engine.
    feature_transform_engine_dataflow_disk_size_gb: The disk size of the dataflow workers of the feature transform engine.
    evaluation_batch_predict_machine_type: Machine type for the batch prediction job in evaluation, such as 'n1-standard-16'.
    evaluation_batch_predict_starting_replica_count: Number of replicas to use in the batch prediction cluster at startup time.
    evaluation_batch_predict_max_replica_count: The maximum count of replicas the batch prediction job can scale to.
    evaluation_dataflow_machine_type: Machine type for the dataflow job in evaluation, such as 'n1-standard-16'.
    evaluation_dataflow_max_num_workers: Maximum number of dataflow workers.
    evaluation_dataflow_starting_num_workers: Starting number of dataflow workers.
    evaluation_dataflow_disk_size_gb: The disk space in GB for dataflow.
    study_spec_parameters_override: The list for overriding study spec.
    stage_1_tuner_worker_pool_specs_override: The dictionary for overriding stage 1 tuner worker pool spec.
    stage_2_trainer_worker_pool_specs_override: The dictionary for overriding stage 2 trainer worker pool spec.
    enable_probabilistic_inference: If probabilistic inference is enabled, the model will fit a distribution that captures the uncertainty of a prediction. If quantiles are specified, then the quantiles of the distribution are also returned.
    quantiles: Quantiles to use for probabilistic inference. Up to 5 quantiles are allowed of values between 0 and 1, exclusive. Represents the quantiles to use for that objective. Quantiles must be unique.
    encryption_spec_key_name: The KMS key name.
    model_display_name: Optional display name for model.
    model_description: Optional description.
    run_evaluation: `True` to evaluate the ensembled model on the test split.
    group_columns: A list of time series attribute column names that define the time series hierarchy.
    group_total_weight: The weight of the loss for predictions aggregated over time series in the same group.
    temporal_total_weight: The weight of the loss for predictions aggregated over the horizon for a single time series.
    group_temporal_total_weight: The weight of the loss for predictions aggregated over both the horizon and time series in the same hierarchy group.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  # fmt: on
  parameter_values = _get_base_forecasting_parameters(
      project=project,
      location=location,
      root_dir=root_dir,
      target_column=target_column,
      evaluated_examples_bigquery_path=evaluated_examples_bigquery_path,
      optimization_objective=optimization_objective,
      transformations=transformations,
      train_budget_milli_node_hours=train_budget_milli_node_hours,
      time_column=time_column,
      dataflow_service_account=dataflow_service_account,
      time_series_identifier_columns=time_series_identifier_columns,
      time_series_identifier_column=time_series_identifier_column,
      time_series_attribute_columns=time_series_attribute_columns,
      available_at_forecast_columns=available_at_forecast_columns,
      unavailable_at_forecast_columns=unavailable_at_forecast_columns,
      forecast_horizon=forecast_horizon,
      context_window=context_window,
      window_predefined_column=window_predefined_column,
      window_stride_length=window_stride_length,
      window_max_count=window_max_count,
      holiday_regions=holiday_regions,
      stage_1_num_parallel_trials=stage_1_num_parallel_trials,
      stage_1_tuning_result_artifact_uri=stage_1_tuning_result_artifact_uri,
      stage_2_num_parallel_trials=stage_2_num_parallel_trials,
      num_selected_trials=num_selected_trials,
      data_source_csv_filenames=data_source_csv_filenames,
      data_source_bigquery_table_path=data_source_bigquery_table_path,
      predefined_split_key=predefined_split_key,
      training_fraction=training_fraction,
      validation_fraction=validation_fraction,
      test_fraction=test_fraction,
      weight_column=weight_column,
      dataflow_use_public_ips=dataflow_use_public_ips,
      dataflow_subnetwork=dataflow_subnetwork,
      feature_transform_engine_bigquery_staging_full_dataset_id=feature_transform_engine_bigquery_staging_full_dataset_id,
      feature_transform_engine_dataflow_machine_type=feature_transform_engine_dataflow_machine_type,
      feature_transform_engine_dataflow_max_num_workers=feature_transform_engine_dataflow_max_num_workers,
      feature_transform_engine_dataflow_disk_size_gb=feature_transform_engine_dataflow_disk_size_gb,
      evaluation_batch_predict_machine_type=evaluation_batch_predict_machine_type,
      evaluation_batch_predict_starting_replica_count=evaluation_batch_predict_starting_replica_count,
      evaluation_batch_predict_max_replica_count=evaluation_batch_predict_max_replica_count,
      evaluation_dataflow_machine_type=evaluation_dataflow_machine_type,
      evaluation_dataflow_max_num_workers=evaluation_dataflow_max_num_workers,
      evaluation_dataflow_starting_num_workers=evaluation_dataflow_starting_num_workers,
      evaluation_dataflow_disk_size_gb=evaluation_dataflow_disk_size_gb,
      study_spec_parameters_override=study_spec_parameters_override,
      stage_1_tuner_worker_pool_specs_override=stage_1_tuner_worker_pool_specs_override,
      stage_2_trainer_worker_pool_specs_override=stage_2_trainer_worker_pool_specs_override,
      quantiles=quantiles,
      encryption_spec_key_name=encryption_spec_key_name,
      enable_probabilistic_inference=enable_probabilistic_inference,
      model_display_name=model_display_name,
      model_description=model_description,
      run_evaluation=run_evaluation,
      group_columns=group_columns,
      group_total_weight=group_total_weight,
      temporal_total_weight=temporal_total_weight,
      group_temporal_total_weight=group_temporal_total_weight,
  )

  pipeline_definition_path = os.path.join(
      _GCPC_FORECASTING_PATH,
      'time_series_dense_encoder_forecasting_pipeline.yaml',
  )

  return pipeline_definition_path, parameter_values


def get_temporal_fusion_transformer_forecasting_pipeline_and_parameters(
    *,
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    optimization_objective: str,
    transformations: Dict[str, List[str]],
    train_budget_milli_node_hours: float,
    time_column: str,
    time_series_identifier_columns: List[str],
    time_series_identifier_column: Optional[str] = None,
    time_series_attribute_columns: Optional[List[str]] = None,
    available_at_forecast_columns: Optional[List[str]] = None,
    unavailable_at_forecast_columns: Optional[List[str]] = None,
    forecast_horizon: Optional[int] = None,
    context_window: Optional[int] = None,
    evaluated_examples_bigquery_path: Optional[str] = None,
    window_predefined_column: Optional[str] = None,
    window_stride_length: Optional[int] = None,
    window_max_count: Optional[int] = None,
    holiday_regions: Optional[List[str]] = None,
    stage_1_num_parallel_trials: Optional[int] = None,
    stage_1_tuning_result_artifact_uri: Optional[str] = None,
    stage_2_num_parallel_trials: Optional[int] = None,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    predefined_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    weight_column: Optional[str] = None,
    dataflow_service_account: Optional[str] = None,
    dataflow_subnetwork: Optional[str] = None,
    dataflow_use_public_ips: bool = True,
    feature_transform_engine_bigquery_staging_full_dataset_id: str = '',
    feature_transform_engine_dataflow_machine_type: str = 'n1-standard-16',
    feature_transform_engine_dataflow_max_num_workers: int = 10,
    feature_transform_engine_dataflow_disk_size_gb: int = 40,
    evaluation_batch_predict_machine_type: str = 'n1-standard-16',
    evaluation_batch_predict_starting_replica_count: int = 25,
    evaluation_batch_predict_max_replica_count: int = 25,
    evaluation_dataflow_machine_type: str = 'n1-standard-16',
    evaluation_dataflow_max_num_workers: int = 25,
    evaluation_dataflow_starting_num_workers: int = 22,
    evaluation_dataflow_disk_size_gb: int = 50,
    study_spec_parameters_override: Optional[List[Dict[str, Any]]] = None,
    stage_1_tuner_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    stage_2_trainer_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    encryption_spec_key_name: Optional[str] = None,
    model_display_name: Optional[str] = None,
    model_description: Optional[str] = None,
    run_evaluation: bool = True,
):
  # fmt: off
  """Returns tft_forecasting pipeline and formatted parameters.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column: The target column name.
    optimization_objective: "minimize-rmse", "minimize-mae", "minimize-rmsle", "minimize-rmspe", "minimize-wape-mae", "minimize-mape", or "minimize-quantile-loss".
    transformations: Dict mapping auto and/or type-resolutions to feature columns. The supported types are: auto, categorical, numeric, text, and timestamp.
    train_budget_milli_node_hours: The train budget of creating this model, expressed in milli node hours i.e. 1,000 value in this field means 1 node hour.
    time_column: The column that indicates the time.
    time_series_identifier_columns: The columns which distinguish different time series.
    time_series_identifier_column: [Deprecated] The column which distinguishes different time series.
    time_series_attribute_columns: The columns that are invariant across the same time series.
    available_at_forecast_columns: The columns that are available at the forecast time.
    unavailable_at_forecast_columns: The columns that are unavailable at the forecast time.
    forecast_horizon: The length of the horizon.
    context_window: The length of the context window.
    evaluated_examples_bigquery_path: The bigquery dataset to write the predicted examples into for evaluation, in the format `bq://project.dataset`.
    window_predefined_column: The column that indicate the start of each window.
    window_stride_length: The stride length to generate the window.
    window_max_count: The maximum number of windows that will be generated.
    holiday_regions: The geographical regions where the holiday effect is applied in modeling.
    stage_1_num_parallel_trials: Number of parallel trails for stage 1.
    stage_1_tuning_result_artifact_uri: The stage 1 tuning result artifact GCS URI.
    stage_2_num_parallel_trials: Number of parallel trails for stage 2.
    data_source_csv_filenames: A string that represents a list of comma separated CSV filenames.
    data_source_bigquery_table_path: The BigQuery table path of format bq://bq_project.bq_dataset.bq_table
    predefined_split_key: The predefined_split column name.
    training_fraction: The training fraction.
    validation_fraction: The validation fraction.
    test_fraction: The test fraction.
    weight_column: The weight column name.
    dataflow_service_account: The full service account name.
    dataflow_subnetwork: The dataflow subnetwork.
    dataflow_use_public_ips: `True` to enable dataflow public IPs.
    feature_transform_engine_bigquery_staging_full_dataset_id: The full id of the feature transform engine staging dataset.
    feature_transform_engine_dataflow_machine_type: The dataflow machine type of the feature transform engine.
    feature_transform_engine_dataflow_max_num_workers: The max number of dataflow workers of the feature transform engine.
    feature_transform_engine_dataflow_disk_size_gb: The disk size of the dataflow workers of the feature transform engine.
    evaluation_batch_predict_machine_type: Machine type for the batch prediction job in evaluation, such as 'n1-standard-16'.
    evaluation_batch_predict_starting_replica_count: Number of replicas to use in the batch prediction cluster at startup time.
    evaluation_batch_predict_max_replica_count: The maximum count of replicas the batch prediction job can scale to.
    evaluation_dataflow_machine_type: Machine type for the dataflow job in evaluation, such as 'n1-standard-16'.
    evaluation_dataflow_max_num_workers: Maximum number of dataflow workers.
    evaluation_dataflow_starting_num_workers: Starting number of dataflow workers.
    evaluation_dataflow_disk_size_gb: The disk space in GB for dataflow.
    study_spec_parameters_override: The list for overriding study spec.
    stage_1_tuner_worker_pool_specs_override: The dictionary for overriding stage 1 tuner worker pool spec.
    stage_2_trainer_worker_pool_specs_override: The dictionary for overriding stage 2 trainer worker pool spec.
    encryption_spec_key_name: The KMS key name.
    model_display_name: Optional display name for model.
    model_description: Optional description.
    run_evaluation: `True` to evaluate the ensembled model on the test split.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  # fmt: on
  # TFT should only have 1 selected trial to freeze the ensemble size at 1.
  excluded_parameters = _RETAIL_MODEL_DISABLED_OPTIONS.union({
      'num_selected_trials',
  })
  parameter_values = _get_base_forecasting_parameters(
      project=project,
      location=location,
      root_dir=root_dir,
      target_column=target_column,
      evaluated_examples_bigquery_path=evaluated_examples_bigquery_path,
      optimization_objective=optimization_objective,
      transformations=transformations,
      train_budget_milli_node_hours=train_budget_milli_node_hours,
      time_column=time_column,
      dataflow_service_account=dataflow_service_account,
      time_series_identifier_columns=time_series_identifier_columns,
      time_series_identifier_column=time_series_identifier_column,
      time_series_attribute_columns=time_series_attribute_columns,
      available_at_forecast_columns=available_at_forecast_columns,
      unavailable_at_forecast_columns=unavailable_at_forecast_columns,
      forecast_horizon=forecast_horizon,
      context_window=context_window,
      window_predefined_column=window_predefined_column,
      window_stride_length=window_stride_length,
      window_max_count=window_max_count,
      holiday_regions=holiday_regions,
      stage_1_num_parallel_trials=stage_1_num_parallel_trials,
      stage_1_tuning_result_artifact_uri=stage_1_tuning_result_artifact_uri,
      stage_2_num_parallel_trials=stage_2_num_parallel_trials,
      data_source_csv_filenames=data_source_csv_filenames,
      data_source_bigquery_table_path=data_source_bigquery_table_path,
      predefined_split_key=predefined_split_key,
      training_fraction=training_fraction,
      validation_fraction=validation_fraction,
      test_fraction=test_fraction,
      weight_column=weight_column,
      dataflow_use_public_ips=dataflow_use_public_ips,
      dataflow_subnetwork=dataflow_subnetwork,
      feature_transform_engine_bigquery_staging_full_dataset_id=feature_transform_engine_bigquery_staging_full_dataset_id,
      feature_transform_engine_dataflow_machine_type=feature_transform_engine_dataflow_machine_type,
      feature_transform_engine_dataflow_max_num_workers=feature_transform_engine_dataflow_max_num_workers,
      feature_transform_engine_dataflow_disk_size_gb=feature_transform_engine_dataflow_disk_size_gb,
      evaluation_batch_predict_machine_type=evaluation_batch_predict_machine_type,
      evaluation_batch_predict_starting_replica_count=evaluation_batch_predict_starting_replica_count,
      evaluation_batch_predict_max_replica_count=evaluation_batch_predict_max_replica_count,
      evaluation_dataflow_machine_type=evaluation_dataflow_machine_type,
      evaluation_dataflow_max_num_workers=evaluation_dataflow_max_num_workers,
      evaluation_dataflow_starting_num_workers=evaluation_dataflow_starting_num_workers,
      evaluation_dataflow_disk_size_gb=evaluation_dataflow_disk_size_gb,
      study_spec_parameters_override=study_spec_parameters_override,
      stage_1_tuner_worker_pool_specs_override=stage_1_tuner_worker_pool_specs_override,
      stage_2_trainer_worker_pool_specs_override=stage_2_trainer_worker_pool_specs_override,
      encryption_spec_key_name=encryption_spec_key_name,
      model_display_name=model_display_name,
      model_description=model_description,
      run_evaluation=run_evaluation,
      fields_to_exclude=excluded_parameters,
  )

  pipeline_definition_path = os.path.join(
      _GCPC_FORECASTING_PATH,
      'temporal_fusion_transformer_forecasting_pipeline.yaml',
  )

  return pipeline_definition_path, parameter_values


def get_sequence_to_sequence_forecasting_pipeline_and_parameters(
    *,
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    optimization_objective: str,
    transformations: Dict[str, List[str]],
    train_budget_milli_node_hours: float,
    time_column: str,
    time_series_identifier_columns: List[str],
    time_series_identifier_column: Optional[str] = None,
    time_series_attribute_columns: Optional[List[str]] = None,
    available_at_forecast_columns: Optional[List[str]] = None,
    unavailable_at_forecast_columns: Optional[List[str]] = None,
    forecast_horizon: Optional[int] = None,
    context_window: Optional[int] = None,
    evaluated_examples_bigquery_path: Optional[str] = None,
    window_predefined_column: Optional[str] = None,
    window_stride_length: Optional[int] = None,
    window_max_count: Optional[int] = None,
    holiday_regions: Optional[List[str]] = None,
    stage_1_num_parallel_trials: Optional[int] = None,
    stage_1_tuning_result_artifact_uri: Optional[str] = None,
    stage_2_num_parallel_trials: Optional[int] = None,
    num_selected_trials: Optional[int] = None,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    predefined_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    weight_column: Optional[str] = None,
    dataflow_service_account: Optional[str] = None,
    dataflow_subnetwork: Optional[str] = None,
    dataflow_use_public_ips: bool = True,
    feature_transform_engine_bigquery_staging_full_dataset_id: str = '',
    feature_transform_engine_dataflow_machine_type: str = 'n1-standard-16',
    feature_transform_engine_dataflow_max_num_workers: int = 10,
    feature_transform_engine_dataflow_disk_size_gb: int = 40,
    evaluation_batch_predict_machine_type: str = 'n1-standard-16',
    evaluation_batch_predict_starting_replica_count: int = 25,
    evaluation_batch_predict_max_replica_count: int = 25,
    evaluation_dataflow_machine_type: str = 'n1-standard-16',
    evaluation_dataflow_max_num_workers: int = 25,
    evaluation_dataflow_starting_num_workers: int = 22,
    evaluation_dataflow_disk_size_gb: int = 50,
    study_spec_parameters_override: Optional[List[Dict[str, Any]]] = None,
    stage_1_tuner_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    stage_2_trainer_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    encryption_spec_key_name: Optional[str] = None,
    model_display_name: Optional[str] = None,
    model_description: Optional[str] = None,
    run_evaluation: bool = True,
):
  # fmt: off
  """Returns seq2seq forecasting pipeline and formatted parameters.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column: The target column name.
    optimization_objective: "minimize-rmse", "minimize-mae", "minimize-rmsle", "minimize-rmspe", "minimize-wape-mae", "minimize-mape", or "minimize-quantile-loss".
    transformations: Dict mapping auto and/or type-resolutions to feature columns. The supported types are: auto, categorical, numeric, text, and timestamp.
    train_budget_milli_node_hours: The train budget of creating this model, expressed in milli node hours i.e. 1,000 value in this field means 1 node hour.
    time_column: The column that indicates the time.
    time_series_identifier_columns: The columns which distinguish different time series.
    time_series_identifier_column: [Deprecated] The column which distinguishes different time series.
    time_series_attribute_columns: The columns that are invariant across the same time series.
    available_at_forecast_columns: The columns that are available at the forecast time.
    unavailable_at_forecast_columns: The columns that are unavailable at the forecast time.
    forecast_horizon: The length of the horizon.
    context_window: The length of the context window.
    evaluated_examples_bigquery_path: The bigquery dataset to write the predicted examples into for evaluation, in the format `bq://project.dataset`.
    window_predefined_column: The column that indicate the start of each window.
    window_stride_length: The stride length to generate the window.
    window_max_count: The maximum number of windows that will be generated.
    holiday_regions: The geographical regions where the holiday effect is applied in modeling.
    stage_1_num_parallel_trials: Number of parallel trails for stage 1.
    stage_1_tuning_result_artifact_uri: The stage 1 tuning result artifact GCS URI.
    stage_2_num_parallel_trials: Number of parallel trails for stage 2.
    num_selected_trials: Number of selected trails.
    data_source_csv_filenames: A string that represents a list of comma separated CSV filenames.
    data_source_bigquery_table_path: The BigQuery table path of format bq://bq_project.bq_dataset.bq_table
    predefined_split_key: The predefined_split column name.
    training_fraction: The training fraction.
    validation_fraction: The validation fraction.
    test_fraction: The test fraction.
    weight_column: The weight column name.
    dataflow_service_account: The full service account name.
    dataflow_subnetwork: The dataflow subnetwork.
    dataflow_use_public_ips: `True` to enable dataflow public IPs.
    feature_transform_engine_bigquery_staging_full_dataset_id: The full id of the feature transform engine staging dataset.
    feature_transform_engine_dataflow_machine_type: The dataflow machine type of the feature transform engine.
    feature_transform_engine_dataflow_max_num_workers: The max number of dataflow workers of the feature transform engine.
    feature_transform_engine_dataflow_disk_size_gb: The disk size of the dataflow workers of the feature transform engine.
    evaluation_batch_predict_machine_type: Machine type for the batch prediction job in evaluation, such as 'n1-standard-16'.
    evaluation_batch_predict_starting_replica_count: Number of replicas to use in the batch prediction cluster at startup time.
    evaluation_batch_predict_max_replica_count: The maximum count of replicas the batch prediction job can scale to.
    evaluation_dataflow_machine_type: Machine type for the dataflow job in evaluation, such as 'n1-standard-16'.
    evaluation_dataflow_max_num_workers: Maximum number of dataflow workers.
    evaluation_dataflow_starting_num_workers: Starting number of dataflow workers.
    evaluation_dataflow_disk_size_gb: The disk space in GB for dataflow.
    study_spec_parameters_override: The list for overriding study spec.
    stage_1_tuner_worker_pool_specs_override: The dictionary for overriding stage 1 tuner worker pool spec.
    stage_2_trainer_worker_pool_specs_override: The dictionary for overriding stage 2 trainer worker pool spec.
    encryption_spec_key_name: The KMS key name.
    model_display_name: Optional display name for model.
    model_description: Optional description.
    run_evaluation: `True` to evaluate the ensembled model on the test split.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  # fmt: on
  parameter_values = _get_base_forecasting_parameters(
      project=project,
      location=location,
      root_dir=root_dir,
      target_column=target_column,
      evaluated_examples_bigquery_path=evaluated_examples_bigquery_path,
      optimization_objective=optimization_objective,
      transformations=transformations,
      train_budget_milli_node_hours=train_budget_milli_node_hours,
      time_column=time_column,
      dataflow_service_account=dataflow_service_account,
      time_series_identifier_columns=time_series_identifier_columns,
      time_series_identifier_column=time_series_identifier_column,
      time_series_attribute_columns=time_series_attribute_columns,
      available_at_forecast_columns=available_at_forecast_columns,
      unavailable_at_forecast_columns=unavailable_at_forecast_columns,
      forecast_horizon=forecast_horizon,
      context_window=context_window,
      window_predefined_column=window_predefined_column,
      window_stride_length=window_stride_length,
      window_max_count=window_max_count,
      holiday_regions=holiday_regions,
      stage_1_num_parallel_trials=stage_1_num_parallel_trials,
      stage_1_tuning_result_artifact_uri=stage_1_tuning_result_artifact_uri,
      stage_2_num_parallel_trials=stage_2_num_parallel_trials,
      num_selected_trials=num_selected_trials,
      data_source_csv_filenames=data_source_csv_filenames,
      data_source_bigquery_table_path=data_source_bigquery_table_path,
      predefined_split_key=predefined_split_key,
      training_fraction=training_fraction,
      validation_fraction=validation_fraction,
      test_fraction=test_fraction,
      weight_column=weight_column,
      dataflow_use_public_ips=dataflow_use_public_ips,
      dataflow_subnetwork=dataflow_subnetwork,
      feature_transform_engine_bigquery_staging_full_dataset_id=feature_transform_engine_bigquery_staging_full_dataset_id,
      feature_transform_engine_dataflow_machine_type=feature_transform_engine_dataflow_machine_type,
      feature_transform_engine_dataflow_max_num_workers=feature_transform_engine_dataflow_max_num_workers,
      feature_transform_engine_dataflow_disk_size_gb=feature_transform_engine_dataflow_disk_size_gb,
      evaluation_batch_predict_machine_type=evaluation_batch_predict_machine_type,
      evaluation_batch_predict_starting_replica_count=evaluation_batch_predict_starting_replica_count,
      evaluation_batch_predict_max_replica_count=evaluation_batch_predict_max_replica_count,
      evaluation_dataflow_machine_type=evaluation_dataflow_machine_type,
      evaluation_dataflow_max_num_workers=evaluation_dataflow_max_num_workers,
      evaluation_dataflow_starting_num_workers=evaluation_dataflow_starting_num_workers,
      evaluation_dataflow_disk_size_gb=evaluation_dataflow_disk_size_gb,
      study_spec_parameters_override=study_spec_parameters_override,
      stage_1_tuner_worker_pool_specs_override=stage_1_tuner_worker_pool_specs_override,
      stage_2_trainer_worker_pool_specs_override=stage_2_trainer_worker_pool_specs_override,
      encryption_spec_key_name=encryption_spec_key_name,
      model_display_name=model_display_name,
      model_description=model_description,
      run_evaluation=run_evaluation,
      fields_to_exclude=_RETAIL_MODEL_DISABLED_OPTIONS,
  )

  pipeline_definition_path = os.path.join(
      _GCPC_FORECASTING_PATH,
      'sequence_to_sequence_forecasting_pipeline.yaml',
  )

  return pipeline_definition_path, parameter_values
