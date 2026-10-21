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


from typing import Optional

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components.types.artifact_types import VertexDataset
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp import dsl
from kfp.dsl import Input
from kfp.dsl import Output


@dsl.container_component
def automl_forecasting_training_job(
    project: str,
    display_name: str,
    target_column: str,
    time_column: str,
    time_series_identifier_column: str,
    unavailable_at_forecast_columns: list,
    available_at_forecast_columns: list,
    forecast_horizon: int,
    data_granularity_unit: str,
    data_granularity_count: int,
    dataset: Input[VertexDataset],
    model: Output[VertexModel],
    location: Optional[str] = 'us-central1',
    optimization_objective: Optional[str] = None,
    time_series_attribute_columns: Optional[list] = None,
    context_window: Optional[int] = None,
    quantiles: Optional[list] = None,
    validation_options: Optional[str] = None,
    labels: Optional[dict] = {},
    training_encryption_spec_key_name: Optional[str] = None,
    model_encryption_spec_key_name: Optional[str] = None,
    budget_milli_node_hours: Optional[int] = None,
    model_display_name: Optional[str] = None,
    model_labels: Optional[dict] = None,
    model_id: Optional[str] = None,
    parent_model: Optional[str] = None,
    is_default_version: Optional[bool] = None,
    model_version_aliases: Optional[list] = None,
    model_version_description: Optional[str] = None,
    hierarchy_group_columns: Optional[list] = None,
    hierarchy_group_total_weight: Optional[float] = None,
    hierarchy_temporal_total_weight: Optional[float] = None,
    hierarchy_group_temporal_total_weight: Optional[float] = None,
    window_column: Optional[str] = None,
    window_stride_length: Optional[int] = None,
    window_max_count: Optional[int] = None,
    holiday_regions: Optional[list] = None,
    column_specs: Optional[dict] = None,
    column_transformations: Optional[list] = None,
    training_fraction_split: Optional[float] = None,
    validation_fraction_split: Optional[float] = None,
    test_fraction_split: Optional[float] = None,
    predefined_split_column_name: Optional[str] = None,
    timestamp_split_column_name: Optional[str] = None,
    weight_column: Optional[str] = None,
    export_evaluated_data_items: Optional[bool] = False,
    export_evaluated_data_items_bigquery_destination_uri: Optional[str] = None,
    export_evaluated_data_items_override_destination: Optional[bool] = None,
    additional_experiments: Optional[list] = None,
):
  # fmt: off
  """Runs the training job and returns a model.

  If training on a Vertex AI dataset, you can use one of the following split configurations: Data fraction splits: Any of `training_fraction_split`, `validation_fraction_split` and `test_fraction_split` may optionally be provided, they must sum to up to 1. If the provided ones sum to less than 1, the remainder is assigned to sets as decided by Vertex AI. If none of the fractions are set, by default roughly 80% of data will be used for training, 10% for validation, and 10% for test. Predefined splits: Assigns input data to training, validation, and test sets based on the value of a provided key. If using predefined splits, `predefined_split_column_name` must be provided. Supported only for tabular Datasets. Timestamp splits: Assigns input data to training, validation, and test sets based on a provided timestamps. The youngest data pieces are assigned to training set, next to validation set, and the oldest to the test set. Supported only for tabular Datasets.

  Args:
      dataset: The dataset within the same Project from which data will be used to train the Model. The Dataset must use schema compatible with Model being trained, and what is compatible should be described in the used TrainingPipeline's [training_task_definition] [google.cloud.aiplatform.v1beta1.TrainingPipeline.training_task_definition]. For time series Datasets, all their data is exported to training, to pick and choose from.
      target_column: Name of the column that the Model is to predict values for. This column must be unavailable at forecast.
      time_column: Name of the column that identifies time order in the time series. This column must be available at forecast.
      time_series_identifier_column: Name of the column that identifies the time series.
      unavailable_at_forecast_columns: Column names of columns that are unavailable at forecast. Each column contains information for the given entity (identified by the [time_series_identifier_column]) that is unknown before the forecast (e.g. population of a city in a given year, or weather on a given day).
      available_at_forecast_columns: Column names of columns that are available at forecast. Each column contains information for the given entity (identified by the [time_series_identifier_column]) that is known at forecast.
      forecast_horizon: The amount of time into the future for which forecasted values for the target are returned. Expressed in number of units defined by the [data_granularity_unit] and [data_granularity_count] field. Inclusive.
      data_granularity_unit: The data granularity unit. Accepted values are `minute`, `hour`, `day`, `week`, `month`, `year`.
      data_granularity_count: The number of data granularity units between data points in the training data. If [data_granularity_unit] is `minute`, can be 1, 5, 10, 15, or 30. For all other values of [data_granularity_unit], must be 1.
      training_fraction_split: The fraction of the input data that is to be used to train the Model. This is ignored if Dataset is not provided.
      validation_fraction_split: The fraction of the input data that is to be used to validate the Model. This is ignored if Dataset is not provided.
      test_fraction_split: The fraction of the input data that is to be used to evaluate the Model. This is ignored if Dataset is not provided.
      predefined_split_column_name: The key is a name of one of the Dataset's data columns. The value of the key (either the label's value or value in the column) must be one of {`TRAIN`, `VALIDATE`, `TEST`}, and it defines to which set the given piece of data is assigned. If for a piece of data the key is not present or has an invalid value, that piece is ignored by the pipeline. Supported only for tabular and time series Datasets.
      timestamp_split_column_name: The key is a name of one of the Dataset's data columns. The value of the key values of the key (the values in the column) must be in RFC 3339 `date-time` format, where `time-offset` = `"Z"` (e.g. 1985-04-12T23:20:50.52Z). If for a piece of data the key is not present or has an invalid value, that piece is ignored by the pipeline. Supported only for tabular and time series Datasets. This parameter must be used with training_fraction_split, validation_fraction_split, and test_fraction_split.
      weight_column: Name of the column that should be used as the weight column. Higher values in this column give more importance to the row during Model training. The column must have numeric values between 0 and 10000 inclusively, and 0 value means that the row is ignored. If the weight column field is not set, then all rows are assumed to have equal weight of 1.
      time_series_attribute_columns: Column names that should be used as attribute columns. Each column is constant within a time series.
      context_window: The amount of time into the past training and prediction data is used for model training and prediction respectively. Expressed in number of units defined by the [data_granularity_unit] and [data_granularity_count] fields. When not provided uses the default value of 0 which means the model sets each series context window to be 0 (also known as "cold start"). Inclusive.
      export_evaluated_data_items: Whether to export the test set predictions to a BigQuery table. If False, then the export is not performed.
      export_evaluated_data_items_bigquery_destination_uri: URI of desired destination BigQuery table for exported test set predictions. Expected format: `bq://<project_id>:<dataset_id>:<table>` If not specified, then results are exported to the following auto-created BigQuery table: `<project_id>:export_evaluated_examples_<model_name>_<yyyy_MM_dd'T'HH_mm_ss_SSS'Z'>.evaluated_examples` Applies only if [export_evaluated_data_items] is True.
      export_evaluated_data_items_override_destination: Whether to override the contents of [export_evaluated_data_items_bigquery_destination_uri], if the table exists, for exported test set predictions. If False, and the table exists, then the training job will fail. Applies only if [export_evaluated_data_items] is True and [export_evaluated_data_items_bigquery_destination_uri] is specified.
      quantiles: Quantiles to use for the `minimize-quantile-loss` [AutoMLForecastingTrainingJob.optimization_objective]. This argument is required in this case. Accepts up to 5 quantiles in the form of a double from 0 to 1, exclusive. Each quantile must be unique.
      validation_options: Validation options for the data validation component. The available options are: "fail-pipeline" - (default), will validate against the validation and fail the pipeline if it fails. "ignore-validation" - ignore the results of the validation and continue the pipeline
      budget_milli_node_hours: The train budget of creating this Model, expressed in milli node hours i.e. 1,000 value in this field means 1 node hour. The training cost of the model will not exceed this budget. The final cost will be attempted to be close to the budget, though may end up being (even) noticeably smaller - at the backend's discretion. This especially may happen when further model training ceases to provide any improvements. If the budget is set to a value known to be insufficient to train a Model for the given training set, the training won't be attempted and will error. The minimum value is 1000 and the maximum is 72000.
      model_display_name: If the script produces a managed Vertex AI Model. The display name of the Model. The name can be up to 128 characters long and can be consist of any UTF-8 characters. If not provided upon creation, the job's display_name is used.
      model_labels: The labels with user-defined metadata to organize your Models. Label keys and values can be no longer than 64 characters (Unicode codepoints), can only contain lowercase letters, numeric characters, underscores and dashes. International characters are allowed. See https://goo.gl/xmQnxf for more information and examples of labels.
      model_id: The ID to use for the Model produced by this job, which will become the final component of the model resource name. This value may be up to 63 characters, and valid characters are `[a-z0-9_-]`. The first character cannot be a number or hyphen.
      parent_model: The resource name or model ID of an existing model. The new model uploaded by this job will be a version of `parent_model`. Only set this field when training a new version of an existing model.
      is_default_version: When set to True, the newly uploaded model version will automatically have alias "default" included. Subsequent uses of the model produced by this job without a version specified will use this "default" version. When set to False, the "default" alias will not be moved. Actions targeting the model version produced by this job will need to specifically reference this version by ID or alias. New model uploads, i.e. version 1, will always be "default" aliased.
      model_version_aliases: User provided version aliases so that the model version uploaded by this job can be referenced via alias instead of auto-generated version ID. A default version alias will be created for the first version of the model. The format is [a-z][a-zA-Z0-9-]{0,126}[a-z0-9]
      model_version_description: The description of the model version being uploaded by this job.
      hierarchy_group_columns: A list of time series attribute column names that define the time series hierarchy. Only one level of hierarchy is supported, ex. `region` for a hierarchy of stores or `department` for a hierarchy of products. If multiple columns are specified, time series will be grouped by their combined values, ex. (`blue`, `large`) for `color` and `size`, up to 5 columns are accepted. If no group columns are specified, all time series are considered to be part of the same group.
      hierarchy_group_total_weight: The weight of the loss for predictions aggregated over time series in the same hierarchy group.
      hierarchy_temporal_total_weight: The weight of the loss for predictions aggregated over the horizon for a single time series.
      hierarchy_group_temporal_total_weight: The weight of the loss for predictions aggregated over both the horizon and time series in the same hierarchy group.
      window_column: Name of the column that should be used to filter input rows. The column should contain either booleans or string booleans; if the value of the row is True, generate a sliding window from that row.
      window_stride_length: Step length used to generate input examples. Every `window_stride_length` rows will be used to generate a sliding window.
      window_max_count: Number of rows that should be used to generate input examples. If the total row count is larger than this number, the input data will be randomly sampled to hit the count.
      holiday_regions: The geographical regions to use when creating holiday features. This option is only allowed when data_granularity_unit is `day`. Acceptable values can come from any of the following levels:
        Top level: GLOBAL Second level: continental regions
            NA: North America
            JAPAC: Japan and Asia Pacific
            EMEA: Europe, the Middle East and Africa
            LAC: Latin America and the Caribbean Third level: countries from ISO 3166-1 Country codes.
      display_name: The user-defined name of this TrainingPipeline.
      optimization_objective: Objective function the model is to be optimized towards. The training process creates a Model that optimizes the value of the objective function over the validation set. The supported optimization objectives: "minimize-rmse" (default) - Minimize root-mean-squared error (RMSE). "minimize-mae" - Minimize mean-absolute error (MAE). "minimize-rmsle" - Minimize root-mean-squared log error (RMSLE). "minimize-rmspe" - Minimize root-mean-squared percentage error (RMSPE). "minimize-wape-mae" - Minimize the combination of weighted absolute percentage error (WAPE) and mean-absolute-error (MAE). "minimize-quantile-loss" - Minimize the quantile loss at the defined quantiles. (Set this objective to build quantile forecasts.)
      column_specs: Alternative to column_transformations where the keys of the dict are column names and their respective values are one of AutoMLTabularTrainingJob.column_data_types. When creating transformation for BigQuery Struct column, the column should be flattened using "." as the delimiter. Only columns with no child should have a transformation. If an input column has no transformations on it, such a column is ignored by the training, except for the targetColumn, which should have no transformations defined on. Only one of column_transformations or column_specs should be passed.
      column_transformations: Transformations to apply to the input columns (i.e. columns other than the targetColumn). Each transformation may produce multiple result values from the column's value, and all are used for training. When creating transformation for BigQuery Struct column, the column should be flattened using "." as the delimiter. Only columns with no child should have a transformation. If an input column has no transformations on it, such a column is ignored by the training, except for the targetColumn, which should have no transformations defined on. Only one of column_transformations or column_specs should be passed. Consider using column_specs as column_transformations will be deprecated eventually.
      project: Project to retrieve dataset from.
      location: Optional location to retrieve dataset from.
      labels: The labels with user-defined metadata to organize TrainingPipelines. Label keys and values can be no longer than 64 characters (Unicode codepoints), can only contain lowercase letters, numeric characters, underscores and dashes. International characters are allowed. See https://goo.gl/xmQnxf for more information and examples of labels.
      training_encryption_spec_key_name: The Cloud KMS resource identifier of the customer managed encryption key used to protect the training pipeline. Has the form: `projects/my-project/locations/my-region/keyRings/my-kr/cryptoKeys/my-key`. The key needs to be in the same region as where the compute resource is created. If set, this TrainingPipeline will be secured by this key. Note: Model trained by this TrainingPipeline is also secured by this key if `model_to_upload` is not set separately. Overrides encryption_spec_key_name set in aiplatform.init.
      model_encryption_spec_key_name: The Cloud KMS resource identifier of the customer managed encryption key used to protect the model. Has the form: `projects/my-project/locations/my-region/keyRings/my-kr/cryptoKeys/my-key`. The key needs to be in the same region as where the compute resource is created. If set, the trained Model will be secured by this key. Overrides encryption_spec_key_name set in aiplatform.init.
      additional_experiments: Additional experiment flags for the time series forcasting training.

  Returns:
      model: The trained Vertex AI Model resource or None if training did not produce a Vertex AI Model.
  """
  # fmt: on

  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-m',
          'google_cloud_pipeline_components.container.v1.aiplatform.remote_runner',
          '--cls_name',
          'AutoMLForecastingTrainingJob',
          '--method_name',
          'run',
      ],
      args=[
          '--init.project',
          project,
          '--init.location',
          location,
          '--init.display_name',
          display_name,
          '--method.target_column',
          target_column,
          '--method.time_column',
          time_column,
          '--method.time_series_identifier_column',
          time_series_identifier_column,
          '--method.unavailable_at_forecast_columns',
          unavailable_at_forecast_columns,
          '--method.available_at_forecast_columns',
          available_at_forecast_columns,
          '--method.forecast_horizon',
          forecast_horizon,
          '--method.data_granularity_unit',
          data_granularity_unit,
          '--method.data_granularity_count',
          data_granularity_count,
          '--method.dataset',
          dataset.metadata['resourceName'],
          dsl.IfPresentPlaceholder(
              input_name='optimization_objective',
              then=['--init.optimization_objective', optimization_objective],
          ),
          dsl.IfPresentPlaceholder(
              input_name='training_encryption_spec_key_name',
              then=[
                  '--init.training_encryption_spec_key_name',
                  training_encryption_spec_key_name,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='model_encryption_spec_key_name',
              then=[
                  '--init.model_encryption_spec_key_name',
                  model_encryption_spec_key_name,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='context_window',
              then=['--method.context_window', context_window],
          ),
          dsl.IfPresentPlaceholder(
              input_name='validation_options',
              then=['--method.validation_options', validation_options],
          ),
          dsl.IfPresentPlaceholder(
              input_name='budget_milli_node_hours',
              then=[
                  '--method.budget_milli_node_hours',
                  budget_milli_node_hours,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='model_display_name',
              then=['--method.model_display_name', model_display_name],
          ),
          dsl.IfPresentPlaceholder(
              input_name='training_fraction_split',
              then=[
                  '--method.training_fraction_split',
                  training_fraction_split,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='validation_fraction_split',
              then=[
                  '--method.validation_fraction_split',
                  validation_fraction_split,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='test_fraction_split',
              then=['--method.test_fraction_split', test_fraction_split],
          ),
          dsl.IfPresentPlaceholder(
              input_name='predefined_split_column_name',
              then=[
                  '--method.predefined_split_column_name',
                  predefined_split_column_name,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='timestamp_split_column_name',
              then=[
                  '--method.timestamp_split_column_name',
                  timestamp_split_column_name,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='weight_column',
              then=['--method.weight_column', weight_column],
          ),
          dsl.IfPresentPlaceholder(
              input_name='export_evaluated_data_items',
              then=[
                  '--method.export_evaluated_data_items',
                  export_evaluated_data_items,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='export_evaluated_data_items_bigquery_destination_uri',
              then=[
                  '--method.export_evaluated_data_items_bigquery_destination_uri',
                  export_evaluated_data_items_bigquery_destination_uri,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='export_evaluated_data_items_override_destination',
              then=[
                  '--method.export_evaluated_data_items_override_destination',
                  export_evaluated_data_items_override_destination,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='time_series_attribute_columns',
              then=[
                  '--method.time_series_attribute_columns',
                  time_series_attribute_columns,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='quantiles', then=['--method.quantiles', quantiles]
          ),
          dsl.IfPresentPlaceholder(
              input_name='labels', then=['--init.labels', labels]
          ),
          dsl.IfPresentPlaceholder(
              input_name='model_labels',
              then=['--method.model_labels', model_labels],
          ),
          dsl.IfPresentPlaceholder(
              input_name='model_id', then=['--method.model_id', model_id]
          ),
          dsl.IfPresentPlaceholder(
              input_name='parent_model',
              then=['--method.parent_model', parent_model],
          ),
          dsl.IfPresentPlaceholder(
              input_name='is_default_version',
              then=['--method.is_default_version', is_default_version],
          ),
          dsl.IfPresentPlaceholder(
              input_name='model_version_aliases',
              then=['--method.model_version_aliases', model_version_aliases],
          ),
          dsl.IfPresentPlaceholder(
              input_name='model_version_description',
              then=[
                  '--method.model_version_description',
                  model_version_description,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='hierarchy_group_columns',
              then=[
                  '--method.hierarchy_group_columns',
                  hierarchy_group_columns,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='hierarchy_group_total_weight',
              then=[
                  '--method.hierarchy_group_total_weight',
                  hierarchy_group_total_weight,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='hierarchy_temporal_total_weight',
              then=[
                  '--method.hierarchy_temporal_total_weight',
                  hierarchy_temporal_total_weight,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='hierarchy_group_temporal_total_weight',
              then=[
                  '--method.hierarchy_group_temporal_total_weight',
                  hierarchy_group_temporal_total_weight,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='window_column',
              then=['--method.window_column', window_column],
          ),
          dsl.IfPresentPlaceholder(
              input_name='window_stride_length',
              then=['--method.window_stride_length', window_stride_length],
          ),
          dsl.IfPresentPlaceholder(
              input_name='window_max_count',
              then=['--method.window_max_count', window_max_count],
          ),
          dsl.IfPresentPlaceholder(
              input_name='holiday_regions',
              then=['--method.holiday_regions', holiday_regions],
          ),
          dsl.IfPresentPlaceholder(
              input_name='column_specs',
              then=['--init.column_specs', column_specs],
          ),
          dsl.IfPresentPlaceholder(
              input_name='column_transformations',
              then=['--init.column_transformations', column_transformations],
          ),
          dsl.IfPresentPlaceholder(
              input_name='additional_experiments',
              then=['--method.additional_experiments', additional_experiments],
          ),
          '--executor_input',
          '{{$}}',
          '--resource_name_output_artifact_uri',
          model.uri,
      ],
  )
