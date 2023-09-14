# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
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
"""Module for preparing the training input."""

from typing import NamedTuple, Optional

from kfp import dsl


# define component without decorator so that we can test function implementation independently
def prepare_data_for_train(
    input_tables: list,
    preprocess_metadata: dict,
    model_feature_columns: Optional[list] = None,
) -> NamedTuple(
    'Outputs',
    [
        ('time_series_identifier_column', str),
        ('time_series_attribute_columns', list),
        ('available_at_forecast_columns', list),
        ('unavailable_at_forecast_columns', list),
        ('column_transformations', list),
        ('preprocess_bq_uri', str),
        ('target_column', str),
        ('time_column', str),
        ('predefined_split_column', str),
        ('weight_column', str),
        ('data_granularity_unit', str),
        ('data_granularity_count', int),
    ],
):
  # fmt: off
  """Prepares the parameters for the training step.

  Converts the input_tables and the output of ForecastingPreprocessingOp
  to the input parameters of TimeSeriesDatasetCreateOp and
  AutoMLForecastingTrainingJobRunOp.

  Args:
    input_tables: Serialized Json array that specifies input BigQuery tables and specs.
    preprocess_metadata: The output of ForecastingPreprocessingOp that is a serialized dictionary with 2 fields: processed_bigquery_table_uri and column_metadata.
    model_feature_columns: Serialized list of column names that will be used as input feature in the training step. If None, all columns will be used in training.

  Returns:
    time_series_identifier_column: Name of the column that identifies the time series.
    time_series_attribute_columns: Serialized column names that should be used as attribute columns.
    available_at_forecast_columns: Serialized column names of columns that are available at forecast.
    unavailable_at_forecast_columns: Serialized column names of columns that are unavailable at forecast.
    column_transformations: Serialized transformations to apply to the input columns.
    preprocess_bq_uri: The BigQuery table that saves the preprocessing result and will be used as training input.
    target_column: The name of the column values of which the Model is to predict.
    time_column: Name of the column that identifies time order in the time series.
    predefined_split_column: Name of the column that specifies an ML use of the row.
    weight_column: Name of the column that should be used as the weight column.
    data_granularity_unit: The data granularity unit.
    data_granularity_count: The number of data granularity units between data points in the training data.
  """
  # fmt: on
  # pylint: disable=g-import-not-at-top,import-outside-toplevel,redefined-outer-name,reimported
  from typing import NamedTuple

  # pylint: enable=g-import-not-at-top,import-outside-toplevel,redefined-outer-name,reimported

  column_metadata = preprocess_metadata['column_metadata']
  bigquery_table_uri = preprocess_metadata['processed_bigquery_table_uri']

  primary_table_specs = next(
      table
      for table in input_tables
      if table['table_type'] == 'FORECASTING_PRIMARY'
  )
  primary_metadata = primary_table_specs['forecasting_primary_table_metadata']

  feature_columns = None
  if model_feature_columns:
    feature_columns = set(model_feature_columns)

  time_series_identifier_column = ''
  time_series_attribute_columns = []
  available_at_forecast_columns = []
  unavailable_at_forecast_columns = []
  column_transformations = []
  predefined_split_column = (
      ''
      if 'predefined_splits_column' not in primary_metadata
      else primary_metadata['predefined_splits_column']
  )
  weight_column = (
      ''
      if 'weight_column' not in primary_metadata
      else primary_metadata['weight_column']
  )

  for name, details in column_metadata.items():
    if name == predefined_split_column or name == weight_column:
      continue
    # Determine temporal type of the column
    if details['tag'] == 'vertex_time_series_key':
      time_series_identifier_column = name
      continue
    elif details['tag'] == 'primary_table':
      if name in (primary_metadata['time_series_identifier_columns']):
        time_series_attribute_columns.append(name)
      elif name == primary_metadata['target_column']:
        unavailable_at_forecast_columns.append(name)
      elif name in primary_metadata['unavailable_at_forecast_columns']:
        unavailable_at_forecast_columns.append(name)
      else:
        available_at_forecast_columns.append(name)
    elif details['tag'] == 'attribute_table':
      time_series_attribute_columns.append(name)
    elif details['tag'] == 'fe_static':
      time_series_attribute_columns.append(name)
    elif details['tag'] == 'fe_available_past_future':
      available_at_forecast_columns.append(name)
    elif details['tag'] == 'fe_available_past_only':
      unavailable_at_forecast_columns.append(name)

    # Determine data type of the column. Note: columns without transformations
    # (except for target) will be ignored during training.
    if feature_columns and name not in feature_columns:
      continue
    if details['type'] == 'STRING':
      trans = {'categorical': {'column_name': name}}
      column_transformations.append(trans)
    elif details['type'] in ['INT64', 'INTEGER', 'NUMERIC', 'FLOAT64', 'FLOAT']:
      trans = {'numeric': {'column_name': name}}
      column_transformations.append(trans)
    elif details['type'] in ['DATETIME', 'DATE', 'TIMESTAMP']:
      trans = {'timestamp': {'column_name': name}}
      column_transformations.append(trans)

  return NamedTuple(
      'Outputs',
      [
          ('time_series_identifier_column', str),
          ('time_series_attribute_columns', list),
          ('available_at_forecast_columns', list),
          ('unavailable_at_forecast_columns', list),
          ('column_transformations', list),
          ('preprocess_bq_uri', str),
          ('target_column', str),
          ('time_column', str),
          ('predefined_split_column', str),
          ('weight_column', str),
          ('data_granularity_unit', str),
          ('data_granularity_count', int),
      ],
  )(
      time_series_identifier_column,
      time_series_attribute_columns,
      available_at_forecast_columns,
      unavailable_at_forecast_columns,
      column_transformations,
      bigquery_table_uri,
      primary_metadata['target_column'],
      primary_metadata['time_column'],
      predefined_split_column,
      weight_column,
      primary_metadata['time_granularity']['unit'].lower(),
      primary_metadata['time_granularity']['quantity'],
  )


prepare_data_for_train_implementation = prepare_data_for_train

prepare_data_for_train = dsl.component(base_image='python:3.8')(
    prepare_data_for_train
)
