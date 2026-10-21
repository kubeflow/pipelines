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
def forecasting_validation(
    input_tables: list,
    validation_theme: str,
    location: Optional[str] = 'US',
):
  # fmt: off
  """Validates BigQuery tables for training or prediction.

  Validates BigQuery tables for training or prediction based on predefined
  requirements. For training, a primary table is required. Optionally, you
  can include some attribute tables. For prediction, you need to include all
  the tables that were used in the training, plus a plan table.

  Args:
    input_tables: Serialized Json array that specifies input BigQuery tables and specs.
    validation_theme: Theme to use for validating the BigQuery tables. Acceptable values are FORECASTING_TRAINING and FORECASTING_PREDICTION.
    location: Optional location for the BigQuery data, default is US.
  """
  # fmt: on

  return dsl.ContainerSpec(
      image=(
          'us-docker.pkg.dev/vertex-ai/time-series-forecasting/forecasting:prod'
      ),
      command=['python', '/launcher.py'],
      args=[
          'forecasting_validation',
          '--input_table_specs',
          input_tables,
          '--validation_theme',
          validation_theme,
          '--location',
          location,
      ],
  )
