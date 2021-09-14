from typing import NamedTuple, Optional

from kfp.components import create_component_from_func


def prepare_data_for_train(
  input_tables: str,
  preprocess_metadata: str,
  model_feature_columns: Optional[str] = None,
  ) -> NamedTuple(
    'Outputs',
    [
      ('time_series_identifier_column', str),
      ('time_series_attribute_columns', str),
      ('available_at_forecast_columns', str),
      ('unavailable_at_forecast_columns', str),
      ('column_transformations', str),
      ('preprocess_bq_uri', str),
      ('target_column', str),
      ('time_column', str),
      ('data_granularity_unit', str),
      ('data_granularity_count', str),
    ]
):
  """Prepares the parameters for the training step.
  
  Converts the input_tables and the output of ForecastingPreprocessingOp
  to the input parameters of TimeSeriesDatasetCreateOp and
  AutoMLForecastingTrainingJobRunOp.

  Args:
    input_tables (str): Serialized Json array that specifies input BigQuery
    tables and specs.
    preprocess_metadata (str): The output of ForecastingPreprocessingOp that is
    a serialized dictionary with 2 fields: processed_bigquery_table_uri and
    column_metadata.
    model_feature_columns (str): Serialized list of column names that will be
    used as input feature in the training step. If None, all columns will be
    used in training.
  """

  import json

  column_metadata = json.loads(preprocess_metadata)['column_metadata']

  tables = json.loads(input_tables)
  bigquery_table_uri = (
    json.loads(preprocess_metadata)['processed_bigquery_table_uri'])

  primary_table_specs = next(
    table for table in tables
    if table['table_type'] == 'FORECASTING_PRIMARY'
  )
  primary_metadata =  primary_table_specs['forecasting_primary_table_metadata']

  feature_columns = None
  if model_feature_columns:
    feature_columns = set(json.loads(model_feature_columns))

  time_series_identifier_column = ''
  time_series_attribute_columns = []
  available_at_forecast_columns = []
  unavailable_at_forecast_columns = []
  column_transformations = []

  for name, details in column_metadata.items():
    # Determine temporal type of the column
    if details['tag'] == 'vertex_time_series_key':
      time_series_identifier_column = name
      continue
    elif details['tag'] == 'primary_table':
      if name in (
        primary_metadata['time_series_identifier_columns']
      ):
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

  return (
    time_series_identifier_column,
    json.dumps(time_series_attribute_columns),
    json.dumps(available_at_forecast_columns),
    json.dumps(unavailable_at_forecast_columns),
    json.dumps(column_transformations),
    bigquery_table_uri,
    primary_metadata['target_column'],
    primary_metadata['time_column'],
    primary_metadata['time_granularity']['unit'].lower(),
    str(primary_metadata['time_granularity']['quantity']),
  )


if __name__ == '__main__':
  prepare_data_for_train_op = create_component_from_func(
      prepare_data_for_train,
      base_image='python:3.8',
      output_component_file='component.yaml',
  )