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

"""Prophet trainer component spec."""

from typing import Optional
from google_cloud_pipeline_components.types.artifact_types import UnmanagedContainerModel

from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Output


# pylint: disable=g-doc-args,unused-argument
@dsl.container_component
def prophet_trainer(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    time_column: str,
    time_series_identifier_column: str,
    forecast_horizon: int,
    window_column: str,
    data_granularity_unit: str,
    predefined_split_column: str,
    source_bigquery_uri: str,
    gcp_resources: dsl.OutputPath(str),
    unmanaged_container_model: Output[UnmanagedContainerModel],
    evaluated_examples_directory: Output[Artifact],
    optimization_objective: Optional[str] = 'rmse',
    max_num_trials: Optional[int] = 6,
    encryption_spec_key_name: Optional[str] = '',
    dataflow_max_num_workers: Optional[int] = 10,
    dataflow_machine_type: Optional[str] = 'n1-standard-1',
    dataflow_disk_size_gb: Optional[int] = 40,
    dataflow_service_account: Optional[str] = '',
    dataflow_subnetwork: Optional[str] = '',
    dataflow_use_public_ips: Optional[bool] = True,
):
  # fmt: off
  """Trains and tunes one Prophet model per time series using Dataflow.

  Args:
      project: The GCP project that runs the pipeline components.
      location: The GCP region for Vertex AI.
      root_dir: The Cloud Storage location to store the output.
      time_column: Name of the column that identifies time order in the
        time series.
      time_series_identifier_column: Name of the column that identifies
        the time series.
      target_column: Name of the column that the model is to predict
        values for.
      forecast_horizon: The number of time periods into the future for
        which forecasts will be created. Future periods start after the latest
        timestamp for each time series.
      optimization_objective: Optimization objective for tuning. Supported
        metrics come from Prophet's performance_metrics function. These are mse,
        rmse, mae, mape, mdape, smape, and coverage.
      data_granularity_unit: String representing the units of time for the
        time column.
      predefined_split_column: The predefined_split column name. A string
        that represents a list of comma separated CSV filenames.
      source_bigquery_uri: The BigQuery table path of format
          bq (str)://bq_project.bq_dataset.bq_table
      window_column: Name of the column that should be used to filter
        input rows.  The column should contain either booleans or string
        booleans; if the value of the row is True, generate a sliding window
        from that row.
      max_num_trials: Maximum number of tuning trials to perform
        per time series. There are up to 100 possible combinations to explore
        for each time series. Recommended values to try are 3, 6, and 24.
      encryption_spec_key_name: Customer-managed encryption key.
      dataflow_machine_type: The dataflow machine type used for
        training.
      dataflow_max_num_workers: The max number of Dataflow
        workers used for training.
      dataflow_disk_size_gb: Dataflow worker's disk size in GB
        during training.
      dataflow_service_account: Custom service account to run
        dataflow jobs.
      dataflow_subnetwork: Dataflow's fully qualified subnetwork
        name, when empty the default subnetwork will be used.
      dataflow_use_public_ips: Specifies whether Dataflow
        workers use public IP addresses.

  Returns:
      gcp_resources: Serialized gcp_resources proto tracking the custom training
        job.
      unmanaged_container_model: The UnmanagedContainerModel artifact.
  """
  # fmt: on

  return dsl.ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:1.0.44',
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.custom_job.launcher',
      ],
      args=[
          '--type',
          'CustomJob',
          '--project',
          project,
          '--location',
          location,
          '--gcp_resources',
          gcp_resources,
          '--payload',
          dsl.ConcatPlaceholder(
              items=[
                  '{"display_name": '
                  + f'"prophet-trainer-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}", ',
                  '"encryption_spec": {"kms_key_name":"',
                  encryption_spec_key_name,
                  '"}, ',
                  '"job_spec": {"worker_pool_specs": [{"replica_count":"1", ',
                  '"machine_spec": {"machine_type": "n1-standard-4"}, ',
                  (
                      '"container_spec":'
                      ' {"image_uri":"us-docker.pkg.dev/vertex-ai-restricted/automl-tabular/training:20230619_1325", '
                  ),
                  '"args": ["prophet_trainer", "',
                  f'--job_name=dataflow-{dsl.PIPELINE_JOB_NAME_PLACEHOLDER}", "',
                  (
                      '--dataflow_worker_container_image=us-docker.pkg.dev/vertex-ai/automl-tabular/dataflow-worker:20230619_1325", "'
                  ),
                  (
                      '--prediction_container_image=us-docker.pkg.dev/vertex-ai/automl-tabular/fte-prediction-server:20230619_1325", "'
                  ),
                  '--artifacts_dir=',
                  root_dir,
                  f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/model/", "',
                  '--evaluated_examples_dir=',
                  root_dir,
                  f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/eval/", "',
                  '--region=',
                  location,
                  '", "',
                  '--source_bigquery_uri=',
                  source_bigquery_uri,
                  '", "',
                  '--target_column=',
                  target_column,
                  '", "',
                  '--time_column=',
                  time_column,
                  '", "',
                  '--time_series_identifier_column=',
                  time_series_identifier_column,
                  '", "',
                  '--forecast_horizon=',
                  forecast_horizon,
                  '", "',
                  '--window_column=',
                  window_column,
                  '", "',
                  '--optimization_objective=',
                  optimization_objective,
                  '", "',
                  '--data_granularity_unit=',
                  data_granularity_unit,
                  '", "',
                  '--predefined_split_column=',
                  predefined_split_column,
                  '", "',
                  '--max_num_trials=',
                  max_num_trials,
                  '", "',
                  '--dataflow_project=',
                  project,
                  '", "',
                  '--dataflow_max_num_workers=',
                  dataflow_max_num_workers,
                  '", "',
                  '--dataflow_machine_type=',
                  dataflow_machine_type,
                  '", "',
                  '--dataflow_disk_size_gb=',
                  dataflow_disk_size_gb,
                  '", "',
                  '--dataflow_service_account=',
                  dataflow_service_account,
                  '", "',
                  '--dataflow_subnetwork=',
                  dataflow_subnetwork,
                  '", "',
                  '--dataflow_use_public_ips=',
                  dataflow_use_public_ips,
                  '", "',
                  '--gcp_resources_path=',
                  gcp_resources,
                  '", "',
                  '--executor_input={{$.json_escape[1]}}"]}}]}}',
              ]
          ),
      ],
  )
