"""Util functions for Vertex Forecasting pipelines."""

import os
import pathlib
from typing import Any, Dict, List, Optional, Tuple, Union


def get_bqml_arima_train_pipeline_and_parameters(
    project: str,
    location: str,
    time_column: str,
    time_series_identifier_column: str,
    target_column_name: str,
    forecast_horizon: int,
    data_granularity_unit: str,
    data_source: Dict[str, Dict[str, Union[List[str], str]]],
    split_spec: Optional[Dict[str, Dict[str, Union[str, float]]]] = None,
    bigquery_destination_uri: str = '',
    override_destination: bool = False,
    max_order: int = 5,
) -> Tuple[str, Dict[str, Any]]:
  """Get the BQML ARIMA_PLUS training pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    time_column: Name of the column that identifies time order in the time
      series.
    time_series_identifier_column: Name of the column that identifies the time
      series.
    target_column_name: Name of the column that the model is to predict values
      for.
    forecast_horizon: The number of time periods into the future for which
      forecasts will be created. Future periods start after the latest timestamp
      for each time series.
    data_granularity_unit: The data granularity unit. Accepted values are:
      minute, hour, day, week, month, year.
    data_source: Serialized JSON with URI of BigQuery table containing training
      data. This table should be provided in a JSON object that looks like:
      {
        "big_query_data_source": {
          "big_query_table_path": "bq://[PROJECT].[DATASET].[TABLE]"
        }
      }
      or
      {
        "csv_data_source": {
          "csv_filenames": [ [GCS_PATHS] ],
      }
    split_spec: Serialized JSON with name of the column containing the dataset
      each row belongs to. Valid values in this column are: TRAIN, VALIDATE, and
      TEST. This column should be provided in a JSON object that looks like:
      {"predefined_split": {"key": "[SPLIT_COLUMN]"}}
      or
      {
        'fraction_split': {
            'training_fraction': 0.8,
            'validation_fraction': 0.1,
            'test_fraction': 0.1,
        },
      }
    bigquery_destination_uri: URI of the desired destination dataset. If not
      specified, resources will be created under a new dataset in the project.
      Unlike in Vertex Forecasting, all resources will be given hardcoded names
      under this dataset, and the model artifact will also be exported here.
    override_destination: Whether to override a
      model or table if it already exists. If False and the resource exists, the
      training job will fail.
    max_order: Integer between 1 and 5 representing the size of the parameter
      search space for ARIMA_PLUS. 5 would result in the highest accuracy model,
      but also the longest training runtime.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  if split_spec is None:
    split_spec = {
        'fraction_split': {
            'training_fraction': 0.8,
            'validation_fraction': 0.1,
            'test_fraction': 0.1,
        },
    }
  parameter_values = {
      'project': project,
      'location': location,
      'time_column': time_column,
      'time_series_identifier_column': time_series_identifier_column,
      'target_column_name': target_column_name,
      'forecast_horizon': forecast_horizon,
      'data_granularity_unit': data_granularity_unit,
      'data_source': data_source,
      'split_spec': split_spec,
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
    data_source: Dict[str, Dict[str, Union[List[str], str]]],
    bigquery_destination_uri: str = '',
    generate_explanation: bool = False,
) -> Tuple[str, Dict[str, Any]]:
  """Get the BQML ARIMA_PLUS prediction pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    model_name: ARIMA_PLUS BQML model URI.
    data_source: Serialized JSON with URI of BigQuery table containing input
      data. This table should be provided in a JSON object that looks like:
      {
        "big_query_data_source": {
          "big_query_table_path": "bq://[PROJECT].[DATASET].[TABLE]"
        }
      }
      or
      {
        "csv_data_source": {
          "csv_filenames": [ [GCS_PATHS] ],
      }
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
      'data_source': data_source,
      'bigquery_destination_uri': bigquery_destination_uri,
      'generate_explanation': generate_explanation,
  }
  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'bqml_arima_predict_pipeline.json')
  return pipeline_definition_path, parameter_values
