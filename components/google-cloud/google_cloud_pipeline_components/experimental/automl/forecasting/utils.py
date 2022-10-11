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
