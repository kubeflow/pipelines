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

"""AutoML Transform component spec."""

from typing import Optional

from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Dataset
from kfp.dsl import Input
from kfp.dsl import Output


@dsl.container_component
def automl_tabular_transform(
    project: str,
    location: str,
    root_dir: str,
    metadata: Input[Artifact],
    dataset_schema: Input[Artifact],
    train_split: Input[Dataset],
    eval_split: Input[Dataset],
    test_split: Input[Dataset],
    materialized_train_split: Output[Artifact],
    materialized_eval_split: Output[Artifact],
    materialized_test_split: Output[Artifact],
    training_schema_uri: Output[Artifact],
    transform_output: Output[Artifact],
    gcp_resources: dsl.OutputPath(str),
    dataflow_machine_type: Optional[str] = 'n1-standard-16',
    dataflow_max_num_workers: Optional[int] = 25,
    dataflow_disk_size_gb: Optional[int] = 40,
    dataflow_subnetwork: Optional[str] = '',
    dataflow_use_public_ips: Optional[bool] = True,
    dataflow_service_account: Optional[str] = '',
    encryption_spec_key_name: Optional[str] = '',
):
  # fmt: off
  """Transforms raw features to engineered features.

  Args:
      project: Project to run Cross-validation trainer.
      location: Location for running the Cross-validation trainer.
      root_dir: The Cloud Storage location to store the output.
      metadata: The tabular example gen metadata.
      dataset_schema: The schema of the dataset.
      train_split: The train split.
      eval_split: The eval split.
      test_split: The test split.
      dataflow_machine_type: The machine type used for dataflow jobs. If not set, default to n1-standard-16.
      dataflow_max_num_workers: The number of workers to run the dataflow job. If not set, default to 25.
      dataflow_disk_size_gb: The disk size, in gigabytes, to use on each Dataflow worker instance. If not set, default to 40.
      dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty the default subnetwork will be used. More details: https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
      dataflow_use_public_ips: Specifies whether Dataflow workers use public IP addresses.
      dataflow_service_account: Custom service account to run dataflow jobs.
      encryption_spec_key_name: Customer-managed encryption key.

  Returns:
      materialized_train_split: The materialized train split.
      materialized_eval_split: The materialized eval split.
      materialized_eval_split: The materialized test split.
      training_schema_uri: The training schema.
      transform_output: The transform output artifact.
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
                      f' "automl-tabular-transform-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}",'
                      ' "encryption_spec": {"kms_key_name":"'
                  ),
                  encryption_spec_key_name,
                  (
                      '"}, "job_spec": {"worker_pool_specs": [{"replica_count":'
                      ' 1, "machine_spec": {"machine_type": "n1-standard-8"},'
                      ' "container_spec": {"image_uri":"'
                  ),
                  'us-docker.pkg.dev/vertex-ai-restricted/automl-tabular/training:20240419_0625',
                  (
                      '", "args": ["transform", "--is_mp=true",'
                      ' "--transform_output_artifact_path='
                  ),
                  transform_output.uri,
                  '", "--transform_output_path=',
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/transform",'
                      ' "--materialized_splits_output_path='
                  ),
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/transform_materialized",'
                      ' "--metadata_path='
                  ),
                  metadata.uri,
                  '", "--dataset_schema_path=',
                  dataset_schema.uri,
                  '", "--train_split=',
                  train_split.uri,
                  '", "--eval_split=',
                  eval_split.uri,
                  '", "--test_split=',
                  test_split.uri,
                  '", "--materialized_train_split=',
                  materialized_train_split.uri,
                  '", "--materialized_eval_split=',
                  materialized_eval_split.uri,
                  '", "--materialized_test_split=',
                  materialized_test_split.uri,
                  '", "--training_schema_path=',
                  training_schema_uri.uri,
                  (
                      f'", "--job_name=automl-tabular-transform-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}'
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
                  '", "--dataflow_machine_type=',
                  dataflow_machine_type,
                  '", "--dataflow_worker_container_image=',
                  'us-docker.pkg.dev/vertex-ai/automl-tabular/dataflow-worker:20240419_0625',
                  '", "--dataflow_disk_size_gb=',
                  dataflow_disk_size_gb,
                  '", "--dataflow_subnetwork_fully_qualified=',
                  dataflow_subnetwork,
                  '", "--dataflow_use_public_ips=',
                  dataflow_use_public_ips,
                  '", "--dataflow_kms_key=',
                  encryption_spec_key_name,
                  '", "--dataflow_service_account=',
                  dataflow_service_account,
                  '", "--lro_job_info=',
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/lro",'
                      ' "--gcp_resources_path='
                  ),
                  gcp_resources,
                  '"]}}]}}',
              ]
          ),
      ],
  )
