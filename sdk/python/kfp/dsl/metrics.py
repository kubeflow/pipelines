# Copyright 2021 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Metric classes in MLMD ontology for KFP SDK."""

from typing import Any, Dict, List

from kfp.dsl import artifact

class Metrics(artifact.Artifact):
  """Supports capturing summary metrics for a model from components such as ModelEvaluation."""

  TYPE_NAME="METRICS"

  def __init__(self):
    #TODO: call super init with metrics schema
    pass

  def log_accuracy(self, value: float) -> None:
    """Logs Model's accuracy metric."""
    pass

  def log_precision(self, value: float) -> None:
    """Logs Model's precision metric."""
    pass

  def log_metric(self, metric: str, value: float) -> None:
    """Log a single custom metric key-value pair"""
    pass


class ClassificationMetrics(artifact.Artifact):
  """Base class for Classification Metrics types such as ROC-Curve, ConfusionMatrix."""

  TYPE_NAME="CLASSIFICATION_METRICS"

  def __init__(self):
    #TODO: call super init with classification metrics schema

  def load_classification_metrics(self, metrics_data:str) -> None:
    # Expects the classification metrics data to be stored in a json string
    # that adheres to the classification metrics schema.

class ROCCurve(ClassificationMetrics):
  """Metric class to support logging ROC curve.

  class maintains an internal store to keep track of the logged data points.
  Each update to this internal store will update the metadata field of the MLMd
  Artifact.
  """

  def __init__(self):
    #TODO: call super init with setting metric type.

  def log_roc_reading(self, tpr: float, fpr: float, threshold: float) -> None:
    """Logs a single ROC Curve data point."""

  def load_roc_readings(self, readings: List[List[float]]) -> None:
    """Bulk log ROC Curve"""
    # Verifies passed data conforms ROC data point format and update the
    # internal store and metadata.


class ConfusionMatrix(ClassificationMetrics):
  """Metric class to support logging ConfusionMatrix

  Class maintains an internal store to keep track of the logged data points.
  Each update to this internal store will update the metadata field in the MLMD
  artifact.
  """

  def set_categories(self, categories: List[str]) -> None:
    """Defines the categories for the confusion matrix"""
    pass

  def log_cell(self, row_category: str, column_category: str, value: Any) -> None:
    """Logs a single cell value"""
    pass

  def log_row(self, row_category: str, cells: List[Any]) -> None:
    """Logs a single row value"""
    pass

  def log_column(self, column_category: str, cells: List[Any]) -> None:
    """Logs a single column value"""
    pass

  def load_matrix(self, categories: List[str], cells: List[List[Any]]) -> None:
    """Logs a complete confusion matrix"""
    pass
