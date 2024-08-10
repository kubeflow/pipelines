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

"""AutoML Feature Ranking and Selection component spec."""

from typing import Optional

from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Dataset
from kfp.dsl import Input
from kfp.dsl import Output


# pylint: disable=dangerous-default-value,g-bare-generic,g-doc-args,unused-argument
@dsl.container_component
def tabular_feature_ranking_and_selection(
    project: str,
    location: str,
    root_dir: str,
    data_source: Input[Dataset],
    target_column_name: str,
    feature_ranking: Output[Artifact],
    selected_features: Output[Artifact],
    gcp_resources: dsl.OutputPath(str),
    dataflow_machine_type: Optional[str] = 'n1-standard-16',
    dataflow_max_num_workers: Optional[int] = 25,
    dataflow_disk_size_gb: Optional[int] = 40,
    dataflow_subnetwork: Optional[str] = '',
    dataflow_use_public_ips: Optional[bool] = True,
    dataflow_service_account: Optional[str] = '',
    encryption_spec_key_name: Optional[str] = '',
    algorithm: Optional[str] = 'AMI',
    prediction_type: Optional[str] = 'unknown',
    binary_classification: Optional[str] = 'false',
    max_selected_features: Optional[int] = 1000,
):
  # fmt: off
  """Launches a feature selection task to pick top features.

  Args:
      project: Project to run feature selection.
      location: Location for running the feature selection. If not set, default to us-central1.
      root_dir: The Cloud Storage location to store the output.
      dataflow_machine_type: The machine type used for dataflow jobs. If not set, default to n1-standard-16.
      dataflow_max_num_workers: The number of workers to run the dataflow job. If not set, default to 25.
      dataflow_disk_size_gb: The disk size, in gigabytes, to use on each Dataflow worker instance. If not set, default to 40.
      dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty the default subnetwork will be used. More details: https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
      dataflow_use_public_ips: Specifies whether Dataflow workers use public IP addresses.
      dataflow_service_account: Custom service account to run dataflow jobs.
      encryption_spec_key_name: Customer-managed encryption key. If this is set, then all resources will be encrypted with the provided encryption key. data_source(Dataset): The input dataset artifact which references csv, BigQuery, or TF Records. target_column_name(str): Target column name of the input dataset.
      max_selected_features: number of features to select by the algorithm. If not set, default to 1000.

  Returns:
      feature_ranking: the dictionary of feature names and feature ranking values.
      selected_features: A json array of selected feature names.
      gcp_resources: GCP resources created by this component. For more details, see https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
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
                  (
                      '{"display_name":'
                      f' "tabular-feature-selection-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}",'
                      ' "encryption_spec": {"kms_key_name":"'
                  ),
                  encryption_spec_key_name,
                  (
                      '"}, "job_spec": {"worker_pool_specs": [{"replica_count":'
                      ' 1, "machine_spec": {"machine_type": "n1-standard-8"},'
                      ' "container_spec": {"image_uri":"'
                  ),
                  'us-docker.pkg.dev/vertex-ai-restricted/automl-tabular/training:20240808_0625',
                  '", "args": ["feature_selection", "--data_source=',
                  data_source.uri,
                  '", "--target_column=',
                  target_column_name,
                  '", "--prediction_type=',
                  prediction_type,
                  '", "--binary_classification=',
                  binary_classification,
                  '", "--algorithm=',
                  algorithm,
                  '", "--feature_selection_dir=',
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/feature_selection/",'
                      f' "--job_name=tabular-feature-selection-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}'
                  ),
                  '", "--dataflow_project=',
                  project,
                  '", "--error_file_path=',
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/error.pb",'
                      ' "--dataflow_staging_dir='
                  ),
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/dataflow_staging",'
                      ' "--dataflow_tmp_dir='
                  ),
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/dataflow_tmp",'
                      ' "--dataflow_max_num_workers='
                  ),
                  dataflow_max_num_workers,
                  '", "--dataflow_worker_container_image=',
                  'us-docker.pkg.dev/vertex-ai/automl-tabular/dataflow-worker:20240808_0625',
                  '", "--dataflow_machine_type=',
                  dataflow_machine_type,
                  '", "--dataflow_disk_size_gb=',
                  dataflow_disk_size_gb,
                  '", "--dataflow_subnetwork_fully_qualified=',
                  dataflow_subnetwork,
                  '", "--dataflow_use_public_ips=',
                  dataflow_use_public_ips,
                  '", "--dataflow_service_account=',
                  dataflow_service_account,
                  '", "--dataflow_kms_key=',
                  encryption_spec_key_name,
                  '", "--max_selected_features=',
                  max_selected_features,
                  '", "--feature_selection_result_path=',
                  feature_ranking.uri,
                  '", "--selected_features_path=',
                  selected_features.uri,
                  '", "--parse_json=true"]}}]}}',
              ]
          ),
      ],
  )
