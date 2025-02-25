# Copyright 2024 The Kubeflow Authors. All Rights Reserved.
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
"""Starry Net component for data preparation."""

from google_cloud_pipeline_components import utils
from google_cloud_pipeline_components._implementation.starry_net import version
from kfp import dsl


@dsl.container_component
def dataprep(
    gcp_resources: dsl.OutputPath(str),
    dataprep_dir: dsl.Output[dsl.Artifact],  # pytype: disable=unsupported-operands
    backcast_length: int,
    forecast_length: int,
    train_end_date: str,
    n_val_windows: int,
    n_test_windows: int,
    test_set_stride: int,
    model_blocks: str,
    bigquery_source: str,
    ts_identifier_columns: str,
    time_column: str,
    static_covariate_columns: str,
    static_covariates_vocab_path: str,  # pytype: disable=unused-argument
    target_column: str,
    machine_type: str,
    docker_region: str,
    location: str,
    project: str,
    job_id: str,
    job_name_prefix: str,
    num_workers: int,
    max_num_workers: int,
    disk_size_gb: int,
    test_set_only: bool,
    bigquery_output: str,
    nan_threshold: float,
    zero_threshold: float,
    gcs_source: str,
    gcs_static_covariate_source: str,
    encryption_spec_key_name: str,
):
  # fmt: off
  """Runs Dataprep for training and evaluating a STARRY-Net model.

  Args:
    gcp_resources: Serialized JSON of ``gcp_resources`` which tracks the
      CustomJob.
    dataprep_dir: The gcp bucket path where all dataprep artifacts
      are saved.
    backcast_length: The length of the input window to feed into the model.
    forecast_length: The length of the forecast horizon.
    train_end_date: The last date of data to use in the training set. All
      subsequent dates are part of the test set.
    n_val_windows: The number of windows to use for the val set. If 0, no
      validation set is used.
    n_test_windows: The number of windows to use for the test set. Must be >= 1.
    test_set_stride: The number of timestamps to roll forward when
      constructing the val and test sets.
    model_blocks: The stringified tuple of blocks to use in the order
      that they appear in the model. Possible values are `cleaning`,
      `change_point`, `trend`, `hour_of_week-hybrid`, `day_of_week-hybrid`,
      `day_of_year-hybrid`, `week_of_year-hybrid`, `month_of_year-hybrid`,
      `residual`, `quantile`.
    bigquery_source: The BigQuery source of the data.
    ts_identifier_columns: The columns that identify unique time series in the BigQuery
      data source.
    time_column: The column with timestamps in the BigQuery source.
    static_covariate_columns: The names of the staic covariates.
    static_covariates_vocab_path: The path to the master static covariates vocab
      json.
    target_column: The target column in the Big Query data source.
    machine_type: The machine type of the dataflow workers.
    docker_region: The docker region, used to determine which image to use.
    location: The location where the job is run.
    project: The name of the project.
    job_id: The pipeline job id.
    job_name_prefix: The name of the dataflow job name prefix.
    num_workers: The initial number of workers in the dataflow job.
    max_num_workers: The maximum number of workers in the dataflow job.
    disk_size_gb: The disk size of each dataflow worker.
    test_set_only: Whether to only create the test set BigQuery table or also
      to create TFRecords for traiing and validation.
    bigquery_output: The BigQuery dataset where the test set is written in the
      form bq://project.dataset.
    nan_threshold: Series having more nan / missing values than
      nan_threshold (inclusive) in percentage for either backtest or forecast
      will not be sampled in the training set (including missing due to
      train_start and train_end). All existing nans are replaced by zeros.
    zero_threshold: Series having more 0.0 values than zero_threshold
      (inclusive) in percentage for either backtest or forecast will not be
      sampled in the training set.
    gcs_source: The path the csv file of the data source.
    gcs_static_covariate_source: The path to the csv file of static covariates.
    encryption_spec_key_name: Customer-managed encryption key options for the
      CustomJob. If this is set, then all resources created by the CustomJob
      will be encrypted with the provided encryption key.

  Returns:
    gcp_resources: Serialized JSON of ``gcp_resources`` which tracks the
      CustomJob.
    dataprep_dir: The gcp bucket path where all dataprep artifacts
      are saved.
  """
  job_name = f'{job_name_prefix}-{job_id}'
  payload = {
      'display_name': job_name,
      'encryption_spec': {
          'kms_key_name': str(encryption_spec_key_name),
      },
      'job_spec': {
          'worker_pool_specs': [{
              'replica_count': '1',
              'machine_spec': {
                  'machine_type': str(machine_type),
              },
              'disk_spec': {
                  'boot_disk_type': 'pd-ssd',
                  'boot_disk_size_gb': 100,
              },
              'container_spec': {
                  'image_uri': f'{docker_region}-docker.pkg.dev/vertex-ai-restricted/starryn/dataprep:captain_{version.DATAPREP_VERSION}',
                  'args': [
                      '--config=starryn/experiments/configs/vertex.py',
                      f'--config.datasets.backcast_length={backcast_length}',
                      f'--config.datasets.forecast_length={forecast_length}',
                      f'--config.datasets.train_end_date={train_end_date}',
                      f'--config.datasets.n_val_windows={n_val_windows}',
                      f'--config.datasets.val_rolling_window_size={test_set_stride}',
                      f'--config.datasets.n_test_windows={n_test_windows}',
                      f'--config.datasets.test_rolling_window_size={test_set_stride}',
                      f'--config.datasets.nan_threshold={nan_threshold}',
                      f'--config.datasets.zero_threshold={zero_threshold}',
                      f'--config.model.static_cov_names={static_covariate_columns}',
                      f'--config.model.blocks_list={model_blocks}',
                      f'--bigquery_source={bigquery_source}',
                      f'--bigquery_output={bigquery_output}',
                      f'--gcs_source={gcs_source}',
                      f'--gcs_static_covariate_source={gcs_static_covariate_source}',
                      f'--ts_identifier_columns={ts_identifier_columns}',
                      f'--time_column={time_column}',
                      f'--target_column={target_column}',
                      f'--job_id={job_name}',
                      f'--num_workers={num_workers}',
                      f'--max_num_workers={max_num_workers}',
                      f'--root_bucket={dataprep_dir.uri}',
                      f'--disk_size={disk_size_gb}',
                      f'--machine_type={machine_type}',
                      f'--test_set_only={test_set_only}',
                      f'--image_uri={docker_region}-docker.pkg.dev/vertex-ai-restricted/starryn/dataprep:replica_{version.DATAPREP_VERSION}',
                  ],
              },
          }]
      }
    }
  return utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=payload,
      gcp_resources=gcp_resources,
    )
