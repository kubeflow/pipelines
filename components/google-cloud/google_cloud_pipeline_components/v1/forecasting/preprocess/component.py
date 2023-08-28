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

from kfp import dsl


@dsl.container_component
def forecasting_preprocessing(
    project: str,
    input_tables: list,
    preprocess_metadata: dsl.OutputPath(dict),
    preprocessing_bigquery_dataset: Optional[str] = '',
    location: Optional[str] = 'US',
):
  # fmt: off
  """Preprocesses BigQuery tables for training or prediction.

  Creates a BigQuery table for training or prediction based on the input tables.
  For training, a primary table is required. Optionally, you can include some
  attribute tables. For prediction, you need to include all the tables that were
  used in the training, plus a plan table.

  Args:
    project: The GCP project id that runs the pipeline.
    input_tables: Serialized Json array that specifies input BigQuery tables and specs.
    preprocessing_bigquery_dataset: Optional BigQuery dataset to save the preprocessing result BigQuery table.
        If not present, a new dataset will be created by the component.
    location: Optional location for the BigQuery data, default is US.

  Returns:
    preprocess_metadata
  """
  # fmt: on

  return dsl.ContainerSpec(
      image=(
          'us-docker.pkg.dev/vertex-ai/time-series-forecasting/forecasting:prod'
      ),
      command=['python', '/launcher.py'],
      args=[
          'forecasting_preprocess',
          '--project_id',
          project,
          '--input_table_specs',
          input_tables,
          '--bigquery_dataset_id',
          preprocessing_bigquery_dataset,
          '--preprocess_metadata_path',
          preprocess_metadata,
          '--location',
          location,
      ],
  )
