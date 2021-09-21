# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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
# ============================================================================
"""Postprocess component for time series data."""
import kfp
from kfp.v2.dsl import Dataset
from kfp.v2.dsl import Input
from kfp.v2.dsl import Output


def postprocess(input_dataset: Input[Dataset],
                predictions_dataset: Input[Dataset],
                postprocessed_dataset: Output[Dataset]):
  """Fills missing timestamps and missing values in predictions.

  Args:
    input_dataset: Input with GCS path to input time series csv.
    predictions_dataset: Input with GCS path to predictions csv.
    postprocessed_dataset: Output with GCS path to postprocessed csv.

  Returns:
    A postprocessed time series dataframe with the same number of rows as the
    input time series.
  """
  import pandas as pd

  fill_values = {'anomaly_score': 0, 'tail_probability': 1, 'label': 0}
  # Set index_col=[0] to prevent unnamed column after merge.
  data = pd.read_csv(input_dataset.path, index_col=[0])
  predictions = pd.read_csv(predictions_dataset.path, index_col=[0])
  merged = data.merge(
      predictions, how='outer', on=['timestamp'], suffixes=('', '_predictions'))
  merged = merged.fillna(value=fill_values)
  merged.to_csv(postprocessed_dataset.path)


def generate_component_file():
  packages = ['pandas']
  kfp.components.create_component_from_func_v2(
      postprocess,
      packages_to_install=packages,
      output_component_file='postprocess.yaml')
