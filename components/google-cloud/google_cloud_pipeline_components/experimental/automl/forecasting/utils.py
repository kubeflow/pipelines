"""Util functions for Vertex Forecasting pipelines."""

import os
import pathlib
from typing import Any, Dict, Tuple


def get_bqml_arima_train_pipeline_and_parameters(
    project: str,
    location: str,
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
    bigquery_destination_uri: str = '',
    override_destination: bool = False,
    max_order: int = 5,
) -> Tuple[str, Dict[str, Any]]:
  """Get the BQML ARIMA_PLUS training pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region for Vertex AI.
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
      pathlib.Path(__file__).parent.resolve(),
      'bqml_arima_train_pipeline.json')
  return pipeline_definition_path, parameter_values


def get_bqml_arima_predict_pipeline_and_parameters(
    project: str,
    location: str,
    model_name: str,
    data_source_csv_filenames: str = '-',
    data_source_bigquery_table_path: str = '-',
    bigquery_destination_uri: str = '',
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
      'bqml_arima_predict_pipeline.json')
  return pipeline_definition_path, parameter_values


def get_prophet_pipeline_and_parameters(
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
    trainer_service_account: str = '-',
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
      Unlike in Vertex Forecasting, all resources will be given hardcoded names
      under this dataset, and the model artifact will also be exported here.
    max_num_trials: Maximum number of tuning trials to perform per time series.
    trainer_service_account: Service account to use when running the CustomJob
      to train the models.
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
      'trainer_service_account': trainer_service_account,
      'optimization_objective': optimization_objective,
      'data_granularity_unit': data_granularity_unit,
      'trainer_dataflow_machine_type': trainer_dataflow_machine_type,
      'trainer_dataflow_max_num_workers': trainer_dataflow_max_num_workers,
      'trainer_dataflow_disk_size_gb': trainer_dataflow_disk_size_gb,
      'evaluation_dataflow_machine_type': evaluation_dataflow_machine_type,
      'evaluation_dataflow_max_num_workers': (
          evaluation_dataflow_max_num_workers),
      'evaluation_dataflow_disk_size_gb': evaluation_dataflow_disk_size_gb,
      'dataflow_service_account': dataflow_service_account,
      'dataflow_subnetwork': dataflow_subnetwork,
      'dataflow_use_public_ips': dataflow_use_public_ips,
  }
  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'prophet_trainer_pipeline.json')
  return pipeline_definition_path, parameter_values
