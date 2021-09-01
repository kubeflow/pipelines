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
"""Google Cloud Pipeline Experimental Forecasting Components."""

import os
from typing import Optional

from kfp.components import load_component_from_file

__all__ = [
    'ForecastingPreprocessingOp',
    'ForecastingValidationOp',
]


def ForecastingPreprocessingOp(
    project: str,
    input_tables: str,
    preprocessing_bigquery_dataset: str = ''):
  """Preprocesses BigQuery tables for training or prediction.

  Creates a BigQuery table for training or prediction based on the input tables.
  For training, a primary table is required. Optionally, you can include some
  attribute tables. For prediction, you need to include all the tables that were
  used in the training, plus a plan table.

  Args:
    project (str): The GCP project id that runs the pipeline.
    input_tables (str): Serialized Json array that specifies input BigQuery
    tables and specs.
    preprocessing_bigquery_dataset (str): Optional BigQuery dataset to save the
    preprocessing result BigQuery table. If not present, a new dataset will be
    created by the component.

  Returns:
    None
  """
  # TODO(yzhaozh): update the documentation with Json object reference and
  # example.
  return load_component_from_file(
      os.path.join(
          os.path.dirname(__file__), 'preprocess/component.yaml'))(
              project=project,
              input_tables=input_tables,
              preprocessing_bigquery_dataset=preprocessing_bigquery_dataset)


def ForecastingValidationOp(input_tables: str, validation_theme: str):
  """Validates BigQuery tables for training or prediction.

  Validates BigQuery tables for training or prediction based on predefined
  requirements. For training, a primary table is required. Optionally, you
  can include some attribute tables. For prediction, you need to include all
  the tables that were used in the training, plus a plan table.

  Args:
    input_tables (str): Serialized Json array that specifies input BigQuery
    tables and specs.
    validation_theme (str): Theme to use for validating the BigQuery tables.
    Acceptable values are FORECASTING_TRAINING and FORECASTING_PREDICTION.

  Returns:
    None
  """
  # TODO(yzhaozh): update the documentation with Json object reference, example
  # and predefined validation requirements.
  return load_component_from_file(
      os.path.join(
          os.path.dirname(__file__), 'validate/component.yaml'))(
              input_tables=input_tables, validation_theme=validation_theme)
