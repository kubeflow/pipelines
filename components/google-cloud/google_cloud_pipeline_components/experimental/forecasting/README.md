# Forecasting Components Inputs

## input_tables

**input_tables** is a seriazlied JSON array required by both ForecastingPreprocessingOp and ForecastingValidationOp.

Proto definition of TableSpecs:
```protobuf
// Desribes a BigQuery user input table for Vertex AI validation, preprocessing
// and training.
message TableSpecs {
  // [Required] BigQuery table path of the table. e.g.:
  // bq://projectId.datasetId.tableId
  string bigquery_uri = 1;

  // [Required] The table type from the eligible types: FORECASTING_PRIMARY,
  // FORECASTING_ATTRIBUTE and FORECASTING_PLAN
  string table_type = 2;

  // Some table types require additional information about the table. If
  // table_type is FORECASTING_PRIMARY, forecasting_primary_table_metadata is
  // required. If table_type is FORECASTING_ATTRIBUTE,
  // forecasting_attribute_table_metadata is required.
  oneof metadata {
    ForecastingPrimaryTableMetadata forecasting_primary_table_metadata = 3;
    ForecastingAttributeTableMetadata forecasting_attribute_table_metadata = 4;
  }
}

// The metadata that desribes the primary table in Vertex forecasting.
//
// The primary table must contain historical data at the granularity it will
// predict at. For example, if the task is to predict daily sales, this table
// should have a target column with historical sales data at daily granularity.
//
// One or more time series identifier columns are needed in this table. If this
// table has 2 time series identifier columns - "channel_id" and "product_id",
// a time series will be identified by the combination of these 2 columns.
//
// A time column must be present in this table with DATE, DATETIME or TIMESTAMP
// type that reflects the specified granularity.
//
// Two rows cannot have the same value in both the time column and the time
// series identifier column(s).
//
// Except for the time series identifier column(s), every column in the
// primary table will be considered time variant. For example, a holiday
// column or promotion column could have different values at different time
// given a specific time series identifier. If a column has fixed value given a
// time series identifier, i.e. the color of a product given the product ID as
// time series identifier, the column should be moved to the attribute table.
message ForecastingPrimaryTableMetadata {
  // [Required] The name of the column that identifies time order in the time
  // series.
  string time_column = 1;
  // [Required] The name of the column that the model is to predict.
  string target_column = 2;
  // [Required] Names of columns that jointly identify the time series.
  repeated string time_series_identifier_columns = 3;
  // [Optional] Names of columns that are unavailable when a forecast is
  // requested. This column contains information for the given entity
  // (identified by the time_series_identifier_columns) that is unknown before
  // the forecast For example, actual weather on a given day.
  repeated string unavailable_at_forecast_columns = 4;

  // [Required] The granularity of time units presented in the time_column.
  TimeGranularity time_granularity = 5;
  // [Optional] The name of the column that splits the table. Eligible values
  // are: TRAIN, VALIDATE, TEST
  string predefined_splits_column = 6;
  // [Optional] The name of the column that measures the importance of the row.
  // Higher weight values give more importance to the corresponding row during
  // model training. For example, to let the model pay more attention to
  // holidays, the holiday rows can have weight value 1.0 and the rest rows have
  // a weight value 0.5. Weight value must be >= 0. 0 means ignored in training,
  // validation or test. If not specified, all rows will have an equal weight of
  // 1.
  string weight_column = 7;
}

// A duration of time expressed in time granularity units.
message TimeGranularity {
  // [Required] The unit of this time period. Eligible values are: MINUTE, HOUR,
  // DAY, WEEK, MONTH, YEAR
  string unit = 1;
  // [Required] The number of units per period, e.g. 3 weeks or 2 months.
  int64 quantity = 2;
}

// The metadata that desribes the attribute table in Vertex forecasting.
//
// Attribute table contains features that desribe time series that are not
// changed with time. For example, if the primary table has 2
// time_series_identifier_columns columns - product_id and channel_id, an
// optional attribute table can provide product attributes such as category,
// color etc.
//
// The attribute table should have a single identifier column that is the same
// as one of the time_series_identifier_columns in
// ForecastingPrimaryTableMetadata.
message ForecastingAttributeTableMetadata {
  // [Required] The name of the primary key column.
  string primary_key_column = 1;
}
```

# Example pipeline using ForecastingPreprocessingOp and ForecastingValidationOp
```python
import json
from google_cloud_pipeline_components.experimental import forecasting


primary_table_specs = {
    "bigquery_uri": "bq://endless-forms-most-beautiful.iowa_liquor_sales_forecast.sales_table",
    "table_type": "FORECASTING_PRIMARY",
    "forecasting_primary_table_metadata": {
        "time_column": "datetime",
        "target_column": "gross_quantity",
        "time_series_identifier_columns": ["product_id", "location_id"],
        "unavailable_at_forecast_columns": ['sale_dollars', 'state_bottle_cost', 'state_bottle_retail'],
        "time_granularity": {"unit": "DAY", "quantity": 1 },
        "predefined_splits_column": "ml_use"
    }
}

attribute_table_specs1 = {
    "bigquery_uri": "bq://endless-forms-most-beautiful.iowa_liquor_sales_forecast.product_table",
    "table_type": "FORECASTING_ATTRIBUTE",
    "forecasting_attribute_table_metadata": {
        "primary_key_column": "product_id"
    }
}

attribute_table_specs2 = {
    "bigquery_uri": "bq://endless-forms-most-beautiful.iowa_liquor_sales_forecast.location_table",
    "table_type": "FORECASTING_ATTRIBUTE",
    "forecasting_attribute_table_metadata": {
        "primary_key_column": "location_id"
    }
}

input_table_specs = [primary_table_specs, attribute_table_specs1, attribute_table_specs2]
input_tables = json.dumps(input_table_specs)


@dsl.pipeline(name='forecasting-pipeline-training')
def pipeline(input_tables: str):
  # A workflow consists of training validation and preprocessing:
  validation = forecasting.ForecastingValidationOp(input_tables=input_tables, validation_theme='FORECASTING_TRAINING')
  preprocess = forecasting.ForecastingPreprocessingOp(project_id='endless-forms-most-beautiful', input_tables=input_tables)
  preprocess.after(validation)
```
