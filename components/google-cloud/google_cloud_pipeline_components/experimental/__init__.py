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
"""Google Cloud Pipeline Experimental Components."""

import os
from .custom_job.custom_job import run_as_vertex_ai_custom_job
from kfp.components import load_component_from_file
from .tensorflow_probability.anomaly_detection import tfp_anomaly_detection

__all__ = [
    'ForecastingPreprocessingOp',
    'ForecastingValidationOp',
]


def ForecastingPreprocessingOp(**kwargs):
  """Preprocesses BigQuery tables for training or prediction.

  Creates a BigQuery table for training or prediction based on the input tables.
  For training, a primary table is required. Optionally, you can include some
  attribute tables. For prediction, you need to include all the tables that were
  used in the training, plus a plan table.

  Accepted kwargs:
      input_tables (str): Serialized Json array that specifies input BigQuery
      tables and specs.
      preprocess_metadata (str): Path to a file used by the component to save
      the output BigQuery table uri and column metadata. Do not set value to
      this arg. The value will be automatically overriden by Managed Pipeline.

  Args:
    **kwargs: See Accepted kwargs.

  Returns:
    None
  """
  # TODO(yzhaozh): update the documentation with Json object reference and
  # example.
  return load_component_from_file(
      os.path.join(
          os.path.dirname(__file__),
          'forecasting/preprocess/component.yaml'))(**kwargs)


def ForecastingValidationOp(**kwargs):
  """Validates BigQuery tables for training or prediction.

  Validates BigQuery tables for training or prediction based on predefined
  requirements. For training, a primary table is required. Optionally, you
  can include some attribute tables. For prediction, you need to include all
  the tables that were used in the training, plus a plan table.

  Accepted kwargs:
      input_tables (str): Serialized Json array that specifies input BigQuery
      tables and specs.

  Args:
    **kwargs: See Accepted kwargs.

  Returns:
    None
  """
  # TODO(yzhaozh): update the documentation with Json object reference, example
  # and predefined validation requirements.
  return load_component_from_file(
      os.path.join(
          os.path.dirname(__file__),
          'forecasting/validate/component.yaml'))(**kwargs)
