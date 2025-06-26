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

"""Auto Feature Engineering component spec."""

from typing import Optional

from kfp import dsl


@dsl.container_component
def automated_feature_engineering(
    root_dir: str,
    project: str,
    location: str,
    gcp_resources: dsl.OutputPath(str),
    materialized_data: dsl.Output[dsl.Dataset],
    feature_ranking: dsl.Output[dsl.Artifact],
    target_column: Optional[str] = '',
    weight_column: Optional[str] = '',
    data_source_csv_filenames: Optional[str] = '',
    data_source_bigquery_table_path: Optional[str] = '',
    bigquery_staging_full_dataset_id: Optional[str] = '',
    materialized_examples_format: Optional[str] = 'tfrecords_gzip',
):
  """Find the top features from the dataset."""
  # fmt: off
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
                      f' "auto-feature-engineering-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}",'
                  ),
                  (
                      '"job_spec": {"worker_pool_specs": [{"replica_count":'
                      ' 1, "machine_spec": {"machine_type": "n1-standard-16"},'
                      ' "container_spec": {"image_uri":"'
                  ),
                  'us-docker.pkg.dev/vertex-ai-restricted/automl-tabular/training:20250620_0525',
                  '", "args": ["feature_engineering", "--project=', project,
                  '", "--location=', location, '", "--data_source_bigquery_table_path=',
                  data_source_bigquery_table_path,
                  '", "--target_column=',
                  target_column,
                  '", "--weight_column=',
                  weight_column,
                  '", "--bigquery_staging_full_dataset_id=',
                  bigquery_staging_full_dataset_id,
                  '", "--materialized_data_path=',
                  root_dir, f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/materialized_data", ',
                  ' "--materialized_examples_path=',
                  root_dir, f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/materialized", '
                  ' "--error_file_path=',
                  root_dir, f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/error.pb", '
                  ' "--materialized_data_artifact_path=',
                  materialized_data.uri,
                  '", "--feature_ranking_path=',
                  feature_ranking.uri, '"]}}]}}'
              ]
          ),
      ],
  )
