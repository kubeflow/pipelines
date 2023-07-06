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

"""AutoML Feature Transform Engine component spec."""

from typing import Optional

from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Dataset
from kfp.dsl import Output


@dsl.container_component
def feature_transform_engine(
    root_dir: str,
    project: str,
    location: str,
    dataset_stats: Output[Artifact],
    materialized_data: Output[Dataset],
    transform_output: Output[Artifact],
    split_example_counts: dsl.OutputPath(str),
    instance_schema: Output[Artifact],
    training_schema: Output[Artifact],
    bigquery_train_split_uri: dsl.OutputPath(str),
    bigquery_validation_split_uri: dsl.OutputPath(str),
    bigquery_test_split_uri: dsl.OutputPath(str),
    bigquery_downsampled_test_split_uri: dsl.OutputPath(str),
    feature_ranking: Output[Artifact],
    gcp_resources: dsl.OutputPath(str),
    dataset_level_custom_transformation_definitions: Optional[list] = [],
    dataset_level_transformations: Optional[list] = [],
    forecasting_time_column: Optional[str] = '',
    forecasting_time_series_identifier_column: Optional[str] = '',
    forecasting_time_series_attribute_columns: Optional[list] = [],
    forecasting_unavailable_at_forecast_columns: Optional[list] = [],
    forecasting_available_at_forecast_columns: Optional[list] = [],
    forecasting_forecast_horizon: Optional[int] = -1,
    forecasting_context_window: Optional[int] = -1,
    forecasting_predefined_window_column: Optional[str] = '',
    forecasting_window_stride_length: Optional[int] = -1,
    forecasting_window_max_count: Optional[int] = -1,
    forecasting_holiday_regions: Optional[list] = [],
    forecasting_apply_windowing: Optional[bool] = True,
    predefined_split_key: Optional[str] = '',
    stratified_split_key: Optional[str] = '',
    timestamp_split_key: Optional[str] = '',
    training_fraction: Optional[float] = -1,
    validation_fraction: Optional[float] = -1,
    test_fraction: Optional[float] = -1,
    tf_transform_execution_engine: Optional[str] = 'dataflow',
    tf_auto_transform_features: Optional[dict] = {},
    tf_custom_transformation_definitions: Optional[list] = [],
    tf_transformations_path: Optional[str] = '',
    legacy_transformations_path: Optional[str] = '',
    target_column: Optional[str] = '',
    weight_column: Optional[str] = '',
    prediction_type: Optional[str] = '',
    model_type: Optional[str] = None,
    multimodal_image_columns: Optional[list] = [],
    multimodal_text_columns: Optional[list] = [],
    run_distill: Optional[bool] = False,
    run_feature_selection: Optional[bool] = False,
    feature_selection_algorithm: Optional[str] = 'AMI',
    materialized_examples_format: Optional[str] = 'tfrecords_gzip',
    max_selected_features: Optional[int] = 1000,
    data_source_csv_filenames: Optional[str] = '',
    data_source_bigquery_table_path: Optional[str] = '',
    bigquery_staging_full_dataset_id: Optional[str] = '',
    dataflow_machine_type: Optional[str] = 'n1-standard-16',
    dataflow_max_num_workers: Optional[int] = 25,
    dataflow_disk_size_gb: Optional[int] = 40,
    dataflow_subnetwork: Optional[str] = '',
    dataflow_use_public_ips: Optional[bool] = True,
    dataflow_service_account: Optional[str] = '',
    encryption_spec_key_name: Optional[str] = '',
    autodetect_csv_schema: Optional[bool] = False,
    group_columns: Optional[list] = None,
    group_total_weight: float = 0.0,
    temporal_total_weight: float = 0.0,
    group_temporal_total_weight: float = 0.0,
):
  # fmt: off
  """Transforms raw data to engineered features.

  FTE performs dataset level transformations, data splitting, data statistic
  generation, and TensorFlow-based row level transformations on the input
  dataset based on the provided transformation configuration.

  Args:
      root_dir: The Cloud Storage location to store the output.
      project: Project to run feature transform engine.
      location: Location for the created GCP services.
      dataset_level_custom_transformation_definitions: List of dataset-level custom transformation definitions.  Custom,
        bring-your-own dataset-level transform functions, where users can define
        and import their own transform function and use it with FTE's built-in
        transformations. Using custom transformations is an experimental feature
        and it is currently not supported during batch prediction.
          Example:  .. code-block:: python  [ { "transformation": "ConcatCols",
            "module_path": "/path/to/custom_transform_fn_dlt.py",
            "function_name": "concat_cols" } ]  Using custom transform function
            together with FTE's built-in transformations:  .. code-block::
            python  [ { "transformation": "Join", "right_table_uri":
            "bq://test-project.dataset_test.table", "join_keys":
            [["join_key_col", "join_key_col"]] },{ "transformation":
            "ConcatCols", "cols": ["feature_1", "feature_2"], "output_col":
            "feature_1_2" } ]
      dataset_level_transformations: List of dataset-level
        transformations.
          Example:  .. code-block:: python  [ { "transformation": "Join",
            "right_table_uri": "bq://test-project.dataset_test.table",
            "join_keys": [["join_key_col", "join_key_col"]] }, ... ]  Additional
            information about FTE's currently supported built-in
            transformations:
              Join: Joins features from right_table_uri. For each join key, the
                left table keys will be included and the right table keys will
                be dropped.
                  Example:  .. code-block:: python  { "transformation": "Join",
                    "right_table_uri": "bq://test-project.dataset_test.table",
                    "join_keys": [["join_key_col", "join_key_col"]] }
                  Arguments:
                      right_table_uri: Right table BigQuery uri to join
                        with input_full_table_id.
                      join_keys: Features to join on. For each
                        nested list, the first element is a left table column
                        and the second is its corresponding right table column.
              TimeAggregate: Creates a new feature composed of values of an
                existing feature from a fixed time period ago or in the future.
                Ex: A feature for sales by store 1 year ago.
                  Example:  .. code-block:: python  { "transformation":
                    "TimeAggregate", "time_difference": 40,
                    "time_difference_units": "DAY",
                    "time_series_identifier_columns": ["store_id"],
                    "time_column": "time_col", "time_difference_target_column":
                    "target_col", "output_column": "output_col" }
                  Arguments:
                      time_difference: Number of time_difference_units to
                        look back or into the future on our
                        time_difference_target_column.
                      time_difference_units: Units of time_difference to
                        look back or into the future on our
                        time_difference_target_column. Must be one of * 'DAY' *
                        'WEEK' (Equivalent to 7 DAYs) * 'MONTH' * 'QUARTER' *
                        'YEAR'
                      time_series_identifier_columns: Names of the
                        time series identifier columns.
                      time_column: Name of the time column.
                      time_difference_target_column: Column we wish to get
                        the value of time_difference time_difference_units in
                        the past or future.
                      output_column: Name of our new time aggregate
                        feature.
                      is_future: Whether we wish to look
                        forward in time. Defaults to False.
                        PartitionByMax/PartitionByMin/PartitionByAvg/PartitionBySum:
                        Performs a partition by reduce operation (one of max,
                        min, avg, or sum) with a fixed historic time period. Ex:
                        Getting avg sales (the reduce column) for each store
                        (partition_by_column) over the previous 5 days
                        (time_column, time_ago_units, and time_ago).
                  Example:  .. code-block:: python  { "transformation":
                    "PartitionByMax", "reduce_column": "sell_price",
                    "partition_by_columns": ["store_id", "state_id"],
                    "time_column": "date", "time_ago": 1, "time_ago_units":
                    "WEEK", "output_column": "partition_by_reduce_max_output" }
                  Arguments:
                      reduce_column: Column to apply the reduce operation
                        on. Reduce operations include the
                          following: Max, Min, Avg, Sum.
                      partition_by_columns: List of columns to
                        partition by.
                      time_column: Time column for the partition by
                        operation's window function.
                      time_ago: Number of time_ago_units to look back on
                        our target_column, starting from time_column
                        (inclusive).
                      time_ago_units: Units of time_ago to look back on
                        our target_column. Must be one of * 'DAY' * 'WEEK'
                      output_column: Name of our output feature.
      forecasting_time_column: Forecasting time column.
      forecasting_time_series_identifier_column: Forecasting
        time series identifier column.
      forecasting_time_series_attribute_columns: Forecasting
        time series attribute columns.
      forecasting_unavailable_at_forecast_columns: Forecasting
        unavailable at forecast columns.
      forecasting_available_at_forecast_columns: Forecasting
        available at forecast columns.
      forecasting_forecast_horizon: Forecasting horizon.
      forecasting_context_window: Forecasting context window.
      forecasting_predefined_window_column: Forecasting predefined window column.
      forecasting_window_stride_length: Forecasting window stride length.
      forecasting_window_max_count: Forecasting window max count.
      forecasting_holiday_regions: The geographical region based on which the
        holiday effect is applied in modeling by adding holiday categorical
        array feature that include all holidays matching the date. This option
        only allowed when data granularity is day. By default, holiday effect
        modeling is disabled. To turn it on, specify the holiday region using
        this option.
        Top level: * 'GLOBAL'
        Second level: continental regions: * 'NA': North America
        * 'JAPAC': Japan and Asia Pacific
        * 'EMEA': Europe, the Middle East and Africa
        * 'LAC': Latin America and the Caribbean
        Third level: countries from ISO 3166-1 Country codes.
        Valid regions: * 'GLOBAL' * 'NA' * 'JAPAC' * 'EMEA' * 'LAC' * 'AE'
        * 'AR' * 'AT' * 'AU' * 'BE' * 'BR' * 'CA' * 'CH' * 'CL' * 'CN' * 'CO'
        * 'CZ' * 'DE' * 'DK' * 'DZ' * 'EC' * 'EE' * 'EG' * 'ES' * 'FI' * 'FR'
        * 'GB' * 'GR' * 'HK' * 'HU' * 'ID' * 'IE' * 'IL' * 'IN' * 'IR' * 'IT'
        * 'JP' * 'KR' * 'LV' * 'MA' * 'MX' * 'MY' * 'NG' * 'NL' * 'NO' * 'NZ'
        * 'PE' * 'PH' * 'PK' * 'PL' * 'PT' * 'RO' * 'RS' * 'RU' * 'SA' * 'SE'
        * 'SG' * 'SI' * 'SK' * 'TH' * 'TR' * 'TW' * 'UA' * 'US' * 'VE' * 'VN'
        * 'ZA'
      forecasting_apply_windowing: Whether to apply window strategy.
      predefined_split_key: Predefined split key.
      stratified_split_key: Stratified split key.
      timestamp_split_key: Timestamp split key.
      training_fraction: Fraction of input data for training.
      validation_fraction: Fraction of input data for validation.
      test_fraction: Fraction of input data for testing.
      tf_transform_execution_engine: Execution engine to perform
        row-level TF transformations. Can be one of: "dataflow" (by default) or
        "bigquery". Using "bigquery" as the execution engine is experimental and
        is for allowlisted customers only. In addition, executing on "bigquery"
        only supports auto transformations (i.e., specified by
        tf_auto_transform_features) and will raise an error when
        tf_custom_transformation_definitions or tf_transformations_path is set.
      tf_auto_transform_features: Dict mapping auto and/or type-resolutions to
        TF transform features. FTE will automatically configure a set of
        built-in transformations for each feature based on its data statistics.
        If users do not want auto type resolution, but want the set of
        transformations for a given type to be automatically generated, they
        may specify pre-resolved transformations types. The following type hint
        dict keys are supported: * 'auto' * 'categorical' * 'numeric' * 'text'
        * 'timestamp'
          Example:  .. code-block:: python { "auto": ["feature1"],
            "categorical": ["feature2", "feature3"], }  Note that the target and
            weight column may not be included as an auto transformation unless
            users are running forecasting.
      tf_custom_transformation_definitions: List of
        TensorFlow-based custom transformation definitions.  Custom,
        bring-your-own transform functions, where users can define and import
        their own transform function and use it with FTE's built-in
        transformations.
          Example:  .. code-block:: python  [ { "transformation": "PlusOne",
            "module_path": "gs://bucket/custom_transform_fn.py",
            "function_name": "plus_one_transform" }, { "transformation":
            "MultiplyTwo", "module_path": "gs://bucket/custom_transform_fn.py",
            "function_name": "multiply_two_transform" } ]  Using custom
            transform function together with FTE's built-in transformations:  ..
            code-block:: python  [ { "transformation": "CastToFloat",
            "input_columns": ["feature_1"], "output_columns": ["feature_1"] },{
            "transformation": "PlusOne", "input_columns": ["feature_1"]
            "output_columns": ["feature_1_plused_one"] },{ "transformation":
            "MultiplyTwo", "input_columns": ["feature_1"] "output_columns":
            ["feature_1_multiplied_two"] } ]
      tf_transformations_path: Path to TensorFlow-based
        transformation configuration.  Path to a JSON file used to specified
        FTE's TF transformation configurations.  In the following, we provide
        some sample transform configurations to demonstrate FTE's capabilities.
        All transformations on input columns are explicitly specified with FTE's
        built-in transformations. Chaining of multiple transformations on a
        single column is also supported. For example:  .. code-block:: python  [
        { "transformation": "ZScale", "input_columns": ["feature_1"] }, {
        "transformation": "ZScale", "input_columns": ["feature_2"] } ]
        Additional information about FTE's currently supported built-in
        transformations:
              Datetime: Extracts datetime featues from a column containing
                timestamp strings.
                  Example:  .. code-block:: python  { "transformation":
                    "Datetime", "input_columns": ["feature_1"], "time_format":
                    "%Y-%m-%d" }
                  Arguments:
                      input_columns: A list with a single column to
                        perform the datetime transformation on.
                      output_columns: Names of output
                        columns, one for each datetime_features element.
                      time_format: Datetime format string. Time format is
                        a combination of Date + Time Delimiter (optional) + Time
                        (optional) directives. Valid date directives are as
                        follows * '%Y-%m-%d'  # 2018-11-30 * '%Y/%m/%d'  #
                        2018/11/30 * '%y-%m-%d'  # 18-11-30 * '%y/%m/%d'  #
                        18/11/30 * '%m-%d-%Y'  # 11-30-2018 * '%m/%d/%Y'  #
                        11/30/2018 * '%m-%d-%y'  # 11-30-18 * '%m/%d/%y'  #
                        11/30/18 * '%d-%m-%Y'  # 30-11-2018 * '%d/%m/%Y'  #
                        30/11/2018 * '%d-%B-%Y'  # 30-November-2018 * '%d-%m-%y'
                        # 30-11-18 * '%d/%m/%y'  # 30/11/18 * '%d-%B-%y'  #
                        30-November-18 * '%d%m%Y'    # 30112018 * '%m%d%Y'    #
                        11302018 * '%Y%m%d'    # 20181130 Valid time delimiters
                        are as follows * 'T' * ' ' Valid time directives are as
                        follows * '%H:%M'          # 23:59 * '%H:%M:%S'       #
                        23:59:58 * '%H:%M:%S.%f'    # 23:59:58[.123456] *
                          '%H:%M:%S.%f%z'  # 23:59:58[.123456]+0000 *
                          '%H:%M:%S%z',    # 23:59:58+0000
                      datetime_features: List of datetime
                        features to be extract. Each entry must be one of *
                        'YEAR' * 'MONTH' * 'DAY' * 'DAY_OF_WEEK' * 'DAY_OF_YEAR'
                        * 'WEEK_OF_YEAR' * 'QUARTER' * 'HOUR' * 'MINUTE' *
                        'SECOND' Defaults to ['YEAR', 'MONTH', 'DAY',
                        'DAY_OF_WEEK', 'DAY_OF_YEAR', 'WEEK_OF_YEAR']
              Log: Performs the natural log on a numeric column.
                  Example:  .. code-block:: python  { "transformation": "Log",
                    "input_columns": ["feature_1"] }
                  Arguments:
                      input_columns: A list with a single column to
                        perform the log transformation on.
                      output_columns: A list with a single
                        output column name, corresponding to the output of our
                        transformation.
              ZScale: Performs Z-scale normalization on a numeric column.
                  Example:  .. code-block:: python  { "transformation":
                    "ZScale", "input_columns": ["feature_1"] }
                  Arguments:
                      input_columns: A list with a single column to
                        perform the z-scale transformation on.
                      output_columns: A list with a single
                        output column name, corresponding to the output of our
                        transformation.
              Vocabulary: Converts strings to integers, where each unique string
                gets a unique integer representation.
                  Example:  .. code-block:: python  { "transformation":
                    "Vocabulary", "input_columns": ["feature_1"] }
                  Arguments:
                      input_columns: A list with a single column to
                        perform the vocabulary transformation on.
                      output_columns: A list with a single
                        output column name, corresponding to the output of our
                        transformation.
                      top_k: Number of the most frequent words
                        in the vocabulary to use for generating dictionary
                        lookup indices. If not specified, all words in the
                        vocabulary will be used. Defaults to None.
                      frequency_threshold: Limit the vocabulary
                        only to words whose number of occurrences in the input
                        exceeds frequency_threshold. If not specified, all words
                        in the vocabulary will be included. If both top_k and
                        frequency_threshold are specified, a word must satisfy
                        both conditions to be included. Defaults to None.
              Categorical: Transforms categorical columns to integer columns.
                  Example:  .. code-block:: python  { "transformation":
                    "Categorical", "input_columns": ["feature_1"], "top_k": 10 }
                  Arguments:
                      input_columns: A list with a single column to
                        perform the categorical transformation on.
                      output_columns: A list with a single
                        output column name, corresponding to the output of our
                        transformation.
                      top_k: Number of the most frequent words
                        in the vocabulary to use for generating dictionary
                        lookup indices. If not specified, all words in the
                        vocabulary will be used.
                      frequency_threshold: Limit the vocabulary
                        only to words whose number of occurrences in the input
                        exceeds frequency_threshold. If not specified, all words
                        in the vocabulary will be included. If both top_k and
                        frequency_threshold are specified, a word must satisfy
                        both conditions to be included.
              Reduce: Given a column where each entry is a numeric array,
                reduces arrays according to our reduce_mode.
                  Example:  .. code-block:: python  { "transformation":
                    "Reduce", "input_columns": ["feature_1"], "reduce_mode":
                    "MEAN", "output_columns": ["feature_1_mean"] }
                  Arguments:
                      input_columns: A list with a single column to
                        perform the reduce transformation on.
                      output_columns: A list with a single
                        output column name, corresponding to the output of our
                        transformation.
                      reduce_mode: One of * 'MAX' * 'MIN' *
                        'MEAN' * 'LAST_K' Defaults to 'MEAN'.
                      last_k: The number of last k elements when
                        'LAST_K' reduce mode is used. Defaults to 1.
              SplitString: Given a column of strings, splits strings into token
                arrays.
                  Example:  .. code-block:: python  { "transformation":
                    "SplitString", "input_columns": ["feature_1"], "separator":
                    "$" }
                  Arguments:
                      input_columns: A list with a single column to
                        perform the split string transformation on.
                      output_columns: A list with a single
                        output column name, corresponding to the output of our
                        transformation.
                      separator: Separator to split input string
                        into tokens. Defaults to ' '.
                      missing_token: Missing token to use when
                        no string is included. Defaults to ' _MISSING_ '.
              NGram: Given a column of strings, splits strings into token arrays
                where each token is an integer.
                  Example:  .. code-block:: python  { "transformation": "NGram",
                    "input_columns": ["feature_1"], "min_ngram_size": 1,
                    "max_ngram_size": 2, "separator": " " }
                  Arguments:
                      input_columns: A list with a single column to
                        perform the n-gram transformation on.
                      output_columns: A list with a single
                        output column name, corresponding to the output of our
                        transformation.
                      min_ngram_size: Minimum n-gram size. Must
                        be a positive number and <= max_ngram_size. Defaults to
                        1.
                      max_ngram_size: Maximum n-gram size. Must
                        be a positive number and >= min_ngram_size. Defaults to
                        2.
                      top_k: Number of the most frequent words
                        in the vocabulary to use for generating dictionary
                        lookup indices. If not specified, all words in the
                        vocabulary will be used. Defaults to None.
                      frequency_threshold: Limit the
                        dictionary's vocabulary only to words whose number of
                        occurrences in the input exceeds frequency_threshold. If
                        not specified, all words in the vocabulary will be
                        included. If both top_k and frequency_threshold are
                        specified, a word must satisfy both conditions to be
                        included. Defaults to None.
                      separator: Separator to split input string
                        into tokens. Defaults to ' '.
                      missing_token: Missing token to use when
                        no string is included. Defaults to ' _MISSING_ '.
              Clip: Given a numeric column, clips elements such that elements <
                min_value are assigned min_value, and elements > max_value are
                assigned max_value.
                  Example:  .. code-block:: python  { "transformation": "Clip",
                    "input_columns": ["col1"], "output_columns":
                    ["col1_clipped"], "min_value": 1., "max_value": 10., }
                  Arguments:
                      input_columns: A list with a single column to
                        perform the n-gram transformation on.
                      output_columns: A list with a single
                        output column name, corresponding to the output of our
                        transformation.
                      min_value: Number where all values below
                        min_value are set to min_value. If no min_value is
                        provided, min clipping will not occur. Defaults to None.
                      max_value: Number where all values above
                        max_value are set to max_value If no max_value is
                        provided, max clipping will not occur. Defaults to None.
              MultiHotEncoding: Performs multi-hot encoding on a categorical
                array column.
                  Example:  .. code-block:: python  { "transformation":
                    "MultiHotEncoding", "input_columns": ["col1"], }  The number
                    of classes is determened by the largest number included in
                    the input if it is numeric or the total number of unique
                    values of the input if it is type str.  If the input is has
                    type str and an element contians separator tokens, the input
                    will be split at separator indices, and the each element of
                    the split list will be considered a seperate class. For
                    example,
                  Input:  .. code-block:: python  [ ["foo bar"],      # Example
                    0 ["foo", "bar"],   # Example 1 ["foo"],          # Example
                    2 ["bar"],          # Example 3 ]
                  Output (with default separator=" "):  .. code-block:: python [
                    [1, 1],          # Example 0 [1, 1],          # Example 1
                    [1, 0],          # Example 2 [0, 1],          # Example 3 ]
                  Arguments:
                      input_columns: A list with a single column to
                        perform the multi-hot-encoding on.
                      output_columns: A list with a single
                        output column name, corresponding to the output of our
                        transformation.
                      top_k: Number of the most frequent words
                        in the vocabulary to use for generating dictionary
                        lookup indices. If not specified, all words in the
                        vocabulary will be used. Defaults to None.
                      frequency_threshold: Limit the
                        dictionary's vocabulary only to words whose number of
                        occurrences in the input exceeds frequency_threshold. If
                        not specified, all words in the vocabulary will be
                        included. If both top_k and frequency_threshold are
                        specified, a word must satisfy both conditions to be
                        included. Defaults to None.
                      separator: Separator to split input string
                        into tokens. Defaults to ' '.
              MaxAbsScale: Performs maximum absolute scaling on a numeric
                column.
                  Example:  .. code-block:: python  { "transformation":
                    "MaxAbsScale", "input_columns": ["col1"], "output_columns":
                    ["col1_max_abs_scaled"] }
                  Arguments:
                      input_columns: A list with a single column to
                        perform max-abs-scale on.
                      output_columns: A list with a single
                        output column name, corresponding to the output of our
                        transformation.
              Custom: Transformations defined in
                tf_custom_transformation_definitions are included here in the
                TensorFlow-based transformation configuration.  For example,
                given the following tf_custom_transformation_definitions:  ..
                code-block:: python  [ { "transformation": "PlusX",
                "module_path": "gs://bucket/custom_transform_fn.py",
                "function_name": "plus_one_transform" } ]  We can include the
                following transformation:  .. code-block:: python  {
                "transformation": "PlusX", "input_columns": ["col1"],
                "output_columns": ["col1_max_abs_scaled"] "x": 5 }  Note that
                input_columns must still be included in our arguments and
                output_columns is optional. All other arguments are those
                defined in custom_transform_fn.py, which includes `"x"` in this
                case. See tf_custom_transformation_definitions above.
                legacy_transformations_path (Optional[str]) Deprecated. Prefer
                tf_auto_transform_features.  Path to a GCS file containing JSON
                string for legacy style transformations. Note that
                legacy_transformations_path and tf_auto_transform_features
                cannot both be specified.
      target_column: Target column of input data.
      weight_column: Weight column of input data.
      prediction_type: Model prediction type. One of
        "classification", "regression", "time_series".
      run_distill: Whether the distillation should be applied
        to the training.
      run_feature_selection: Whether the feature selection
        should be applied to the dataset.
      feature_selection_algorithm: The algorithm of feature
        selection. One of "AMI", "CMIM", "JMIM", "MRMR", default to be "AMI".
        The algorithms available are: AMI(Adjusted Mutual Information):
           Reference:
             https://scikit-learn.org/stable/modules/generated/sklearn.metrics.adjusted_mutual_info_score.html
               Arrays are not yet supported in this algorithm.  CMIM(Conditional
               Mutual Information Maximization): Reference paper: Mohamed
               Bennasar, Yulia Hicks, Rossitza Setchi, “Feature selection using
               Joint Mutual Information Maximisation,” Expert Systems with
               Applications, vol. 42, issue 22, 1 December 2015, Pages
               8520-8532. JMIM(Joint Mutual Information Maximization): Reference
               paper: Mohamed Bennasar, Yulia Hicks, Rossitza Setchi, “Feature
                 selection using Joint Mutual Information Maximisation,” Expert
                 Systems with Applications, vol. 42, issue 22, 1 December 2015,
                 Pages 8520-8532. MRMR(MIQ Minimum-redundancy
                 Maximum-relevance): Reference paper: Hanchuan Peng, Fuhui Long,
                 and Chris Ding. "Feature selection based on mutual information
                 criteria of max-dependency, max-relevance, and min-redundancy."
                 IEEE Transactions on pattern analysis and machine intelligence
                 27, no.
               8: 1226-1238.
      materialized_examples_format: The format to use for the
        materialized examples. Should be either 'tfrecords_gzip' (default) or
        'parquet'.
      max_selected_features: Maximum number of features to
        select.  If specified, the transform config will be purged by only using
        the selected features that ranked top in the feature ranking, which has
        the ranking value for all supported features. If the number of input
        features is smaller than max_selected_features specified, we will still
        run the feature selection process and generate the feature ranking, no
        features will be excluded.  The value will be set to 1000 by default if
        run_feature_selection is enabled.
      data_source_csv_filenames: CSV input data source to run
        feature transform on.
      data_source_bigquery_table_path: BigQuery input data
        source to run feature transform on.
      bigquery_staging_full_dataset_id: Dataset in
        "projectId.datasetId" format for storing intermediate-FTE BigQuery
        tables.  If the specified dataset does not exist in BigQuery, FTE will
        create the dataset. If no bigquery_staging_full_dataset_id is specified,
        all intermediate tables will be stored in a dataset created under the
        provided project in the input data source's location during FTE
        execution called
        "vertex_feature_transform_engine_staging_{location.replace('-', '_')}".
        All tables generated by FTE will have a 30 day TTL.
      model_type: Model type, which we wish to engineer features
        for. Can be one of: neural_network, boosted_trees, l2l, seq2seq, tft, or
        tide. Defaults to the empty value, `None`.
      multimodal_image_columns: List of multimodal image
        columns. Defaults to an empty list.
      multimodal_text_columns: List of multimodal text
        columns. Defaults to an empty list.
      dataflow_machine_type: The machine type used for dataflow
        jobs. If not set, default to n1-standard-16.
      dataflow_max_num_workers: The number of workers to run the
        dataflow job. If not set, default to 25.
      dataflow_disk_size_gb: The disk size, in gigabytes, to use
        on each Dataflow worker instance. If not set, default to 40.
      dataflow_subnetwork: Dataflow's fully qualified subnetwork
        name, when empty the default subnetwork will be used. More details:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
      dataflow_use_public_ips: Specifies whether Dataflow
        workers use public IP addresses.
      dataflow_service_account: Custom service account to run
        Dataflow jobs.
      encryption_spec_key_name: Customer-managed encryption key.
      autodetect_csv_schema: If True, infers the column types
        when importing CSVs into BigQuery.

  Returns:
      dataset_stats: The stats of the dataset.
      materialized_data: The materialized dataset.
      transform_output: The transform output artifact.
      split_example_counts: JSON string of data split example counts for train,
        validate, and test splits.
      bigquery_train_split_uri: BigQuery URI for the train split to pass to the
        batch prediction component during distillation.
      bigquery_validation_split_uri: BigQuery URI for the validation split to
        pass to the batch prediction component during distillation.
      bigquery_test_split_uri: BigQuery URI for the test split to pass to the
        batch prediction component during evaluation.
      bigquery_downsampled_test_split_uri: BigQuery URI for the downsampled test
        split to pass to the batch prediction component during batch explain.
      instance_schema_path: Schema of input data to the tf_model at serving
        time.
      training_schema_path: Schema of input data to the tf_model at training
        time.
      feature_ranking: The ranking of features, all features supported in the
        dataset will be included. For "AMI" algorithm, array features won't be
        available in the ranking as arrays are not supported yet.
      gcp_resources: GCP resources created by this component. For more details,
          see
          https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
      group_columns: A list of time series attribute column names that define
          the time series hierarchy.
      group_total_weight: The weight of the loss for predictions aggregated over
        time series in the same group.
      temporal_total_weight: The weight of the loss for predictions aggregated
        over the horizon for a single time series.
      group_temporal_total_weight: The weight of the loss for predictions
        aggregated over both the horizon and time series in the same hierarchy
        group.
  """
  # fmt: on

  return dsl.ContainerSpec(
      image='us-docker.pkg.dev/vertex-ai/automl-tabular/feature-transform-engine:20230619_1325',
      command=[],
      args=[
          'feature_transform_engine',
          dsl.ConcatPlaceholder(items=['--project=', project]),
          dsl.ConcatPlaceholder(items=['--location=', location]),
          dsl.ConcatPlaceholder(
              items=[
                  '--dataset_level_custom_transformation_definitions=',
                  dataset_level_custom_transformation_definitions,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--dataset_level_transformations=',
                  dataset_level_transformations,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=['--forecasting_time_column=', forecasting_time_column]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--forecasting_time_series_identifier_column=',
                  forecasting_time_series_identifier_column,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--forecasting_time_series_attribute_columns=',
                  forecasting_time_series_attribute_columns,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--forecasting_unavailable_at_forecast_columns=',
                  forecasting_unavailable_at_forecast_columns,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--forecasting_available_at_forecast_columns=',
                  forecasting_available_at_forecast_columns,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--forecasting_forecast_horizon=',
                  forecasting_forecast_horizon,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--forecasting_context_window=',
                  forecasting_context_window,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--forecasting_predefined_window_column=',
                  forecasting_predefined_window_column,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--forecasting_window_stride_length=',
                  forecasting_window_stride_length,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--forecasting_window_max_count=',
                  forecasting_window_max_count,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--forecasting_holiday_regions=',
                  forecasting_holiday_regions,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--forecasting_apply_windowing=',
                  forecasting_apply_windowing,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=['--predefined_split_key=', predefined_split_key]
          ),
          dsl.ConcatPlaceholder(
              items=['--stratified_split_key=', stratified_split_key]
          ),
          dsl.ConcatPlaceholder(
              items=['--timestamp_split_key=', timestamp_split_key]
          ),
          dsl.ConcatPlaceholder(
              items=['--training_fraction=', training_fraction]
          ),
          dsl.ConcatPlaceholder(
              items=['--validation_fraction=', validation_fraction]
          ),
          dsl.ConcatPlaceholder(items=['--test_fraction=', test_fraction]),
          dsl.ConcatPlaceholder(
              items=[
                  '--tf_transform_execution_engine=',
                  tf_transform_execution_engine,
              ]
          ),
          dsl.IfPresentPlaceholder(
              input_name='tf_auto_transform_features',
              then=dsl.ConcatPlaceholder(
                  items=[
                      '--tf_auto_transform_features=',
                      tf_auto_transform_features,
                  ]
              ),
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--tf_custom_transformation_definitions=',
                  tf_custom_transformation_definitions,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=['--tf_transformations_path=', tf_transformations_path]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--legacy_transformations_path=',
                  legacy_transformations_path,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=['--data_source_csv_filenames=', data_source_csv_filenames]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--data_source_bigquery_table_path=',
                  data_source_bigquery_table_path,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--bigquery_staging_full_dataset_id=',
                  bigquery_staging_full_dataset_id,
              ]
          ),
          dsl.ConcatPlaceholder(items=['--target_column=', target_column]),
          dsl.ConcatPlaceholder(items=['--weight_column=', weight_column]),
          dsl.ConcatPlaceholder(items=['--prediction_type=', prediction_type]),
          dsl.IfPresentPlaceholder(
              input_name='model_type',
              then=dsl.ConcatPlaceholder(items=['--model_type=', model_type]),
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--multimodal_image_columns=',
                  multimodal_image_columns,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--multimodal_text_columns=',
                  multimodal_text_columns,
              ]
          ),
          dsl.ConcatPlaceholder(items=['--run_distill=', run_distill]),
          dsl.ConcatPlaceholder(
              items=['--run_feature_selection=', run_feature_selection]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--materialized_examples_format=',
                  materialized_examples_format,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=['--max_selected_features=', max_selected_features]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--feature_selection_staging_dir=',
                  root_dir,
                  f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/feature_selection_staging_dir',
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--feature_selection_algorithm=',
                  feature_selection_algorithm,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=['--feature_ranking_path=', feature_ranking.uri]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--error_file_path=',
                  root_dir,
                  f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/error.txt',
              ]
          ),
          dsl.ConcatPlaceholder(
              items=['--stats_result_path=', dataset_stats.uri]
          ),
          dsl.ConcatPlaceholder(
              items=['--transform_output_artifact_path=', transform_output.uri]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--transform_output_path=',
                  root_dir,
                  f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/transform',
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--materialized_examples_path=',
                  root_dir,
                  f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/materialized',
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--export_data_path=',
                  root_dir,
                  f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/export',
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--materialized_data_path=',
                  root_dir,
                  f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/materialized_data',
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--materialized_data_artifact_path=',
                  materialized_data.uri,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--bigquery_train_split_uri_path=',
                  bigquery_train_split_uri,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--bigquery_validation_split_uri_path=',
                  bigquery_validation_split_uri,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=['--bigquery_test_split_uri_path=', bigquery_test_split_uri]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--bigquery_downsampled_test_split_uri_path=',
                  bigquery_downsampled_test_split_uri,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=['--split_example_counts_path=', split_example_counts]
          ),
          dsl.ConcatPlaceholder(
              items=['--instance_schema_path=', instance_schema.path]
          ),
          dsl.ConcatPlaceholder(
              items=['--training_schema_path=', training_schema.path]
          ),
          f'--job_name=feature-transform-engine-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}',
          dsl.ConcatPlaceholder(items=['--dataflow_project=', project]),
          dsl.ConcatPlaceholder(
              items=[
                  '--dataflow_staging_dir=',
                  root_dir,
                  f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/dataflow_staging',
              ]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--dataflow_tmp_dir=',
                  root_dir,
                  f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/dataflow_tmp',
              ]
          ),
          dsl.ConcatPlaceholder(
              items=['--dataflow_max_num_workers=', dataflow_max_num_workers]
          ),
          dsl.ConcatPlaceholder(
              items=['--dataflow_machine_type=', dataflow_machine_type]
          ),
          '--dataflow_worker_container_image=us-docker.pkg.dev/vertex-ai/automl-tabular/dataflow-worker:20230619_1325',
          '--feature_transform_engine_docker_uri=us-docker.pkg.dev/vertex-ai/automl-tabular/feature-transform-engine:20230619_1325',
          dsl.ConcatPlaceholder(
              items=['--dataflow_disk_size_gb=', dataflow_disk_size_gb]
          ),
          dsl.ConcatPlaceholder(
              items=[
                  '--dataflow_subnetwork_fully_qualified=',
                  dataflow_subnetwork,
              ]
          ),
          dsl.ConcatPlaceholder(
              items=['--dataflow_use_public_ips=', dataflow_use_public_ips]
          ),
          dsl.ConcatPlaceholder(
              items=['--dataflow_service_account=', dataflow_service_account]
          ),
          dsl.ConcatPlaceholder(
              items=['--dataflow_kms_key=', encryption_spec_key_name]
          ),
          dsl.ConcatPlaceholder(
              items=['--autodetect_csv_schema=', autodetect_csv_schema]
          ),
          dsl.ConcatPlaceholder(items=['--gcp_resources_path=', gcp_resources]),
          dsl.IfPresentPlaceholder(
              input_name='group_columns',
              then=dsl.ConcatPlaceholder(
                  items=['--group_columns=', group_columns]
              ),
          ),
          dsl.IfPresentPlaceholder(
              input_name='group_total_weight',
              then=dsl.ConcatPlaceholder(
                  items=['--group_total_weight=', group_total_weight]
              ),
          ),
          dsl.IfPresentPlaceholder(
              input_name='temporal_total_weight',
              then=dsl.ConcatPlaceholder(
                  items=['--temporal_total_weight=', temporal_total_weight]
              ),
          ),
          dsl.IfPresentPlaceholder(
              input_name='group_temporal_total_weight',
              then=dsl.ConcatPlaceholder(
                  items=[
                      '--group_temporal_total_weight=',
                      group_temporal_total_weight,
                  ]
              ),
          ),
      ],
  )
