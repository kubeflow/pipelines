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
from kfp.dsl import Artifact
from kfp.dsl import Output


@dsl.container_component
def train_test_split_jsonl_with_sklearn(
    input_data_path: str,
    training_data_path: Output[Artifact],
    validation_data_path: Output[Artifact],
    validation_split: Optional[float] = 0.2,
    random_seed: Optional[int] = 0,
):
  # fmt: off
  """
  Split JSONL Data into training and validation data using scikit-learn.
  Args:
    training_data_path (str):
        Output path for the training data (JSONL format).
    validation_data_path (str):
        Output path for the validation data (JSONL format).
    input_data_path (str):
        Input data in JSON lines format.
    validation_split (Optional[float]):
        Fraction of data that will make up validation dataset. Default is 0.2 (20% as validation
        data, the rest as training data).
    random_seed (Optional[int]):
      Global random seed to ensure the output is deterministic.
    """
  # fmt: on
  return dsl.ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:2.0.0b1',
      command=[
          'python3',
          '-m',
          'google_cloud_pipeline_components.container.experimental.sklearn.train_test_split_jsonl',
      ],
      args=[
          '--input-data-path',
          input_data_path,
          '--validataion-split',
          validation_split,
          '--random-seed',
          random_seed,
          '--training-data-path',
          training_data_path.path,
          '--validation-data-path',
          validation_data_path.path,
      ],
  )
