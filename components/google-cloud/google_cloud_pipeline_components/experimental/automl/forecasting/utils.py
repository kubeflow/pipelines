"""Util functions for Vertex Forecasting pipelines."""

import os
import pathlib
from typing import Any, Dict, List, Optional, Tuple


def get_bqml_arima_train_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    time_column: str,
    time_series_identifier_column: str,
    target_column: str,
    forecast_horizon: int,
    data_granularity_unit: str,
    predefined_split_key: str = '-',
    timestamp_split_key: str = '-',
    training_fraction: float = -1.0,
    validation_fraction: float = -1.0,
    test_fraction: float = -1.0,
    data_source_csv_filenames: str = '-',
    data_source_bigquery_table_path: str = '-',
    window_column: str = '-',
    window_stride_length: int = -1,
    window_max_count: int = -1,
    bigquery_destination_uri: str = '-',
    override_destination: bool = False,
    max_order: int = 5,
) -> Tuple[str, Dict[str, Any]]:
  """Get the BQML ARIMA_PLUS training pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region for Vertex AI.
    root_dir: The Cloud Storage location to store the output.
    time_column: Name of the column that identifies time order in the time
      series.
    time_series_identifier_column: Name of the column that identifies the time
      series.
    target_column: Name of the column that the model is to predict values for.
    forecast_horizon: The number of time periods into the future for which
      forecasts will be created. Future periods start after the latest timestamp
      for each time series.
    data_granularity_unit: The data granularity unit. Accepted values are:
      minute, hour, day, week, month, year.
    predefined_split_key: The predefined_split column name.
    timestamp_split_key: The timestamp_split column name.
    training_fraction: The training fraction.
    validation_fraction: The validation fraction.
    test_fraction: float = The test fraction.
    data_source_csv_filenames: A string that represents a list of comma
      separated CSV filenames.
    data_source_bigquery_table_path: The BigQuery table path of format
      bq://bq_project.bq_dataset.bq_table
    window_column: Name of the column that should be used to filter input rows.
      The column should contain either booleans or string booleans; if the value
      of the row is True, generate a sliding window from that row.
    window_stride_length: Step length used to generate input examples. Every
      window_stride_length rows will be used to generate a sliding window.
    window_max_count: Number of rows that should be used to generate input
      examples. If the total row count is larger than this number, the input
      data will be randomly sampled to hit the count.
    bigquery_destination_uri: URI of the desired destination dataset. If not
      specified, resources will be created under a new dataset in the project.
      Unlike in Vertex Forecasting, all resources will be given hardcoded names
      under this dataset, and the model artifact will also be exported here.
    override_destination: Whether to overwrite the metrics and evaluated
      examples tables if they already exist. If this is False and the tables
      exist, this pipeline will fail.
    max_order: Integer between 1 and 5 representing the size of the parameter
      search space for ARIMA_PLUS. 5 would result in the highest accuracy model,
      but also the longest training runtime.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  parameter_values = {
      'project': project,
      'location': location,
      'root_dir': root_dir,
      'time_column': time_column,
      'time_series_identifier_column': time_series_identifier_column,
      'target_column': target_column,
      'forecast_horizon': forecast_horizon,
      'data_granularity_unit': data_granularity_unit,
      'predefined_split_key': predefined_split_key,
      'timestamp_split_key': timestamp_split_key,
      'training_fraction': training_fraction,
      'validation_fraction': validation_fraction,
      'test_fraction': test_fraction,
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'window_column': window_column,
      'window_stride_length': window_stride_length,
      'window_max_count': window_max_count,
      'bigquery_destination_uri': bigquery_destination_uri,
      'override_destination': override_destination,
      'max_order': max_order,
  }
  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(), 'bqml_arima_train_pipeline.json'
  )
  return pipeline_definition_path, parameter_values


def get_bqml_arima_predict_pipeline_and_parameters(
    project: str,
    location: str,
    model_name: str,
    data_source_csv_filenames: str = '-',
    data_source_bigquery_table_path: str = '-',
    bigquery_destination_uri: str = '-',
    generate_explanation: bool = False,
) -> Tuple[str, Dict[str, Any]]:
  """Get the BQML ARIMA_PLUS prediction pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region for Vertex AI.
    model_name: ARIMA_PLUS BQML model URI.
    data_source_csv_filenames: A string that represents a list of comma
      separated CSV filenames.
    data_source_bigquery_table_path: The BigQuery table path of format
      bq://bq_project.bq_dataset.bq_table
    bigquery_destination_uri: URI of the desired destination dataset. If not
      specified, a resource will be created under a new dataset in the project.
    generate_explanation: Generate explanation along with the batch prediction
      results. This will cause the batch prediction output to include
      explanations.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  parameter_values = {
      'project': project,
      'location': location,
      'model_name': model_name,
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'bigquery_destination_uri': bigquery_destination_uri,
      'generate_explanation': generate_explanation,
  }
  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'bqml_arima_predict_pipeline.json',
  )
  return pipeline_definition_path, parameter_values


def get_prophet_train_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    time_column: str,
    time_series_identifier_column: str,
    target_column: str,
    forecast_horizon: int,
    optimization_objective: str,
    data_granularity_unit: str,
    predefined_split_key: str = '-',
    timestamp_split_key: str = '-',
    training_fraction: float = -1.0,
    validation_fraction: float = -1.0,
    test_fraction: float = -1.0,
    data_source_csv_filenames: str = '-',
    data_source_bigquery_table_path: str = '-',
    window_column: str = '-',
    window_stride_length: int = -1,
    window_max_count: int = -1,
    bigquery_destination_uri: str = '-',  # Empty string not supported.
    max_num_trials: int = 35,
    trainer_dataflow_machine_type: str = 'n1-standard-1',
    trainer_dataflow_max_num_workers: int = 200,
    trainer_dataflow_disk_size_gb: int = 40,
    evaluation_dataflow_machine_type: str = 'n1-standard-1',
    evaluation_dataflow_max_num_workers: int = 200,
    evaluation_dataflow_disk_size_gb: int = 40,
    dataflow_service_account: str = '-',
    dataflow_subnetwork: str = '-',
    dataflow_use_public_ips: bool = True,
) -> Tuple[str, Dict[str, Any]]:
  """Returns Prophet train pipeline and formatted parameters.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region for Vertex AI.
    root_dir: The Cloud Storage location to store the output.
    time_column: Name of the column that identifies time order in the time
      series.
    time_series_identifier_column: Name of the column that identifies the time
      series.
    target_column: Name of the column that the model is to predict values for.
    forecast_horizon: The number of time periods into the future for which
      forecasts will be created. Future periods start after the latest timestamp
      for each time series.
    optimization_objective: Optimization objective for the model.
    data_granularity_unit: String representing the units of time for the time
      column.
    predefined_split_key: The predefined_split column name.
    timestamp_split_key: The timestamp_split column name.
    training_fraction: The training fraction.
    validation_fraction: The validation fraction.
    test_fraction: float = The test fraction.
    data_source_csv_filenames: A string that represents a list of comma
      separated CSV filenames.
    data_source_bigquery_table_path: The BigQuery table path of format
      bq://bq_project.bq_dataset.bq_table
    window_column: Name of the column that should be used to filter input rows.
      The column should contain either booleans or string booleans; if the value
      of the row is True, generate a sliding window from that row.
    window_stride_length: Step length used to generate input examples. Every
      window_stride_length rows will be used to generate a sliding window.
    window_max_count: Number of rows that should be used to generate input
      examples. If the total row count is larger than this number, the input
      data will be randomly sampled to hit the count.
    bigquery_destination_uri: URI of the desired destination dataset. If not
      specified, resources will be created under a new dataset in the project.
    max_num_trials: Maximum number of tuning trials to perform per time series.
    trainer_dataflow_machine_type: The dataflow machine type used for training.
    trainer_dataflow_max_num_workers: The max number of Dataflow workers used
      for training.
    trainer_dataflow_disk_size_gb: Dataflow worker's disk size in GB during
      training.
    evaluation_dataflow_machine_type: The dataflow machine type used for
      evaluation.
    evaluation_dataflow_max_num_workers: The max number of Dataflow workers used
      for evaluation.
    evaluation_dataflow_disk_size_gb: Dataflow worker's disk size in GB during
      evaluation.
    dataflow_service_account: Custom service account to run dataflow jobs.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used.
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  parameter_values = {
      'project': project,
      'location': location,
      'root_dir': root_dir,
      'time_column': time_column,
      'time_series_identifier_column': time_series_identifier_column,
      'target_column': target_column,
      'forecast_horizon': forecast_horizon,
      'predefined_split_key': predefined_split_key,
      'timestamp_split_key': timestamp_split_key,
      'training_fraction': training_fraction,
      'validation_fraction': validation_fraction,
      'test_fraction': test_fraction,
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'window_column': window_column,
      'window_stride_length': window_stride_length,
      'window_max_count': window_max_count,
      'bigquery_destination_uri': bigquery_destination_uri,
      'max_num_trials': max_num_trials,
      'optimization_objective': optimization_objective,
      'data_granularity_unit': data_granularity_unit,
      'trainer_dataflow_machine_type': trainer_dataflow_machine_type,
      'trainer_dataflow_max_num_workers': trainer_dataflow_max_num_workers,
      'trainer_dataflow_disk_size_gb': trainer_dataflow_disk_size_gb,
      'evaluation_dataflow_machine_type': evaluation_dataflow_machine_type,
      'evaluation_dataflow_max_num_workers': (
          evaluation_dataflow_max_num_workers
      ),
      'evaluation_dataflow_disk_size_gb': evaluation_dataflow_disk_size_gb,
      'dataflow_service_account': dataflow_service_account,
      'dataflow_subnetwork': dataflow_subnetwork,
      'dataflow_use_public_ips': dataflow_use_public_ips,
  }
  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(), 'prophet_trainer_pipeline.yaml'
  )
  return pipeline_definition_path, parameter_values


def get_prophet_prediction_pipeline_and_parameters(
    project: str,
    location: str,
    model_name: str,
    time_column: str,
    time_series_identifier_column: str,
    target_column: str,
    data_source_csv_filenames: str = '-',
    data_source_bigquery_table_path: str = '-',
    bigquery_destination_uri: str = '-',  # Empty string not supported.
    machine_type: str = 'n1-standard-2',
    max_num_workers: int = 200,
) -> Tuple[str, Dict[str, Any]]:
  """Returns Prophet prediction pipeline and formatted parameters.

  Unlike the prediction server for Vertex Forecasting, the Prophet prediction
  server returns predictions batched by time series id. This pipeline shows how
  these predictions can be disaggregated to get results similar to what Vertex
  Forecasting provides.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region for Vertex AI.
    model_name: The name of the Model resource, in a form of
      projects/{project}/locations/{location}/models/{model}.
    time_column: Name of the column that identifies time order in the time
      series.
    time_series_identifier_column: Name of the column that identifies the time
      series.
    target_column: Name of the column that the model is to predict values for.
    data_source_csv_filenames: A string that represents a list of comma
      separated CSV filenames.
    data_source_bigquery_table_path: The BigQuery table path of format
      bq://bq_project.bq_dataset.bq_table
    bigquery_destination_uri: URI of the desired destination dataset. If not
      specified, resources will be created under a new dataset in the project.
    machine_type: The machine type used for batch prediction.
    max_num_workers: The max number of workers used for batch prediction.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  parameter_values = {
      'project': project,
      'location': location,
      'model_name': model_name,
      'time_column': time_column,
      'time_series_identifier_column': time_series_identifier_column,
      'target_column': target_column,
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'bigquery_destination_uri': bigquery_destination_uri,
      'machine_type': machine_type,
      'max_num_workers': max_num_workers,
  }
  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(), 'prophet_predict_pipeline.yaml'
  )
  return pipeline_definition_path, parameter_values


def get_learn_to_learn_forecasting_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    optimization_objective: str,
    transformations: str,
    train_budget_milli_node_hours: float,
    time_column: str,
    time_series_identifier_column: str,
    time_series_attribute_columns: Optional[str] = None,
    available_at_forecast_columns: Optional[str] = None,
    unavailable_at_forecast_columns: Optional[str] = None,
    forecast_horizon: Optional[int] = None,
    context_window: Optional[int] = None,
    window_predefined_column: Optional[str] = None,
    window_stride_length: Optional[int] = None,
    window_max_count: Optional[int] = None,
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
    feature_transform_engine_dataflow_machine_type: Optional[str] = None,
    feature_transform_engine_dataflow_max_num_workers: Optional[int] = None,
    feature_transform_engine_dataflow_disk_size_gb: Optional[int] = None,
    feature_transform_engine_dataflow_subnetwork: Optional[str] = None,
    feature_transform_engine_dataflow_use_public_ips: Optional[bool] = None,
    study_spec_parameters_override: Optional[List[Dict[str, Any]]] = None,
    stage_1_tuner_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    stage_2_trainer_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    quantiles: Optional[str] = None,
    encryption_spec_key_name: Optional[str] = None,
    enable_probabilistic_inference: bool = False,
    model_display_name: Optional[str] = None,
    model_description: Optional[str] = None,
):
  """Returns l2l_forecasting pipeline and formatted parameters."""

  if not study_spec_parameters_override:
    study_spec_parameters_override = []
  if not stage_1_tuner_worker_pool_specs_override:
    stage_1_tuner_worker_pool_specs_override = []
  if not stage_2_trainer_worker_pool_specs_override:
    stage_2_trainer_worker_pool_specs_override = []

  parameter_values = {}
  parameters = {
      'project': project,
      'location': location,
      'root_dir': root_dir,
      'target_column': target_column,
      'optimization_objective': optimization_objective,
      'transformations': transformations,
      'train_budget_milli_node_hours': train_budget_milli_node_hours,
      'time_column': time_column,
      'time_series_identifier_column': time_series_identifier_column,
      'time_series_attribute_columns': time_series_attribute_columns,
      'available_at_forecast_columns': available_at_forecast_columns,
      'unavailable_at_forecast_columns': unavailable_at_forecast_columns,
      'forecast_horizon': forecast_horizon,
      'context_window': context_window,
      'window_predefined_column': window_predefined_column,
      'window_stride_length': window_stride_length,
      'window_max_count': window_max_count,
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
      'feature_transform_engine_dataflow_machine_type': (
          feature_transform_engine_dataflow_machine_type
      ),
      'feature_transform_engine_dataflow_max_num_workers': (
          feature_transform_engine_dataflow_max_num_workers
      ),
      'feature_transform_engine_dataflow_disk_size_gb': (
          feature_transform_engine_dataflow_disk_size_gb
      ),
      'feature_transform_engine_dataflow_subnetwork': (
          feature_transform_engine_dataflow_subnetwork
      ),
      'feature_transform_engine_dataflow_use_public_ips': (
          feature_transform_engine_dataflow_use_public_ips
      ),
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
  }
  parameter_values.update(
      {param: value for param, value in parameters.items() if value is not None}
  )

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'learn_to_learn_forecasting_pipeline.yaml',
  )

  return pipeline_definition_path, parameter_values


def get_time_series_dense_encoder_forecasting_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    optimization_objective: str,
    transformations: str,
    train_budget_milli_node_hours: float,
    time_column: str,
    time_series_identifier_column: str,
    time_series_attribute_columns: Optional[str] = None,
    available_at_forecast_columns: Optional[str] = None,
    unavailable_at_forecast_columns: Optional[str] = None,
    forecast_horizon: Optional[int] = None,
    context_window: Optional[int] = None,
    window_predefined_column: Optional[str] = None,
    window_stride_length: Optional[int] = None,
    window_max_count: Optional[int] = None,
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
    feature_transform_engine_dataflow_machine_type: Optional[str] = None,
    feature_transform_engine_dataflow_max_num_workers: Optional[int] = None,
    feature_transform_engine_dataflow_disk_size_gb: Optional[int] = None,
    feature_transform_engine_dataflow_subnetwork: Optional[str] = None,
    feature_transform_engine_dataflow_use_public_ips: Optional[bool] = None,
    study_spec_parameters_override: Optional[List[Dict[str, Any]]] = None,
    stage_1_tuner_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    stage_2_trainer_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    quantiles: Optional[str] = None,
    encryption_spec_key_name: Optional[str] = None,
    enable_probabilistic_inference: bool = False,
    model_display_name: Optional[str] = None,
    model_description: Optional[str] = None,
):
  """Returns timeseries_dense_encoder_forecasting pipeline and formatted parameters."""

  if not study_spec_parameters_override:
    study_spec_parameters_override = []
  if not stage_1_tuner_worker_pool_specs_override:
    stage_1_tuner_worker_pool_specs_override = []
  if not stage_2_trainer_worker_pool_specs_override:
    stage_2_trainer_worker_pool_specs_override = []

  parameter_values = {}
  parameters = {
      'project': project,
      'location': location,
      'root_dir': root_dir,
      'target_column': target_column,
      'optimization_objective': optimization_objective,
      'transformations': transformations,
      'train_budget_milli_node_hours': train_budget_milli_node_hours,
      'time_column': time_column,
      'time_series_identifier_column': time_series_identifier_column,
      'time_series_attribute_columns': time_series_attribute_columns,
      'available_at_forecast_columns': available_at_forecast_columns,
      'unavailable_at_forecast_columns': unavailable_at_forecast_columns,
      'forecast_horizon': forecast_horizon,
      'context_window': context_window,
      'window_predefined_column': window_predefined_column,
      'window_stride_length': window_stride_length,
      'window_max_count': window_max_count,
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
      'feature_transform_engine_dataflow_machine_type': (
          feature_transform_engine_dataflow_machine_type
      ),
      'feature_transform_engine_dataflow_max_num_workers': (
          feature_transform_engine_dataflow_max_num_workers
      ),
      'feature_transform_engine_dataflow_disk_size_gb': (
          feature_transform_engine_dataflow_disk_size_gb
      ),
      'feature_transform_engine_dataflow_subnetwork': (
          feature_transform_engine_dataflow_subnetwork
      ),
      'feature_transform_engine_dataflow_use_public_ips': (
          feature_transform_engine_dataflow_use_public_ips
      ),
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
  }
  parameter_values.update(
      {param: value for param, value in parameters.items() if value is not None}
  )

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'time_series_dense_encoder_forecasting_pipeline.yaml',
  )

  return pipeline_definition_path, parameter_values
