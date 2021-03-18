# Copyright 2020 Google LLC
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
"""MLMD artifact ontology in KFP SDK."""

from typing import Any, List, Optional

import os
import yaml

from kfp.dsl import artifact
from kfp.dsl import artifact_utils
from kfp.dsl import metrics_utils
from kfp.pipeline_spec import pipeline_spec_pb2

class Model(artifact.Artifact):

  TYPE_NAME = 'kfp.Model'

  def __init__(self):
    self._instance_schema = self.__class__.get_artifact_type()

    # Calling base class init to setup the instance.
    super().__init__()

  @classmethod
  def get_artifact_type(cls) -> str:
    return artifact_utils.read_schema_file('model.yaml')

  @classmethod
  def get_ir_type(cls) -> pipeline_spec_pb2.ArtifactTypeSchema:
    return pipeline_spec_pb2.ArtifactTypeSchema(
        instance_schema=cls.get_artifact_type())

class Dataset(artifact.Artifact):

  TYPE_NAME = 'kfp.Dataset'

  def __init__(self):
    self._instance_schema = self.__class__.get_artifact_type()

    # Calling base class init to setup the instance.
    super().__init__()

  @classmethod
  def get_artifact_type(cls) -> str:
    return artifact_utils.read_schema_file('dataset.yaml')

  @classmethod
  def get_ir_type(cls) -> pipeline_spec_pb2.ArtifactTypeSchema:
    return pipeline_spec_pb2.ArtifactTypeSchema(
        instance_schema=cls.get_artifact_type())

class Metrics(artifact.Artifact):
  """Represent Artifact type to store scalar metrics.

  Artifact used with this type will have metadata adhering to the
  schema type_schemas/metrics.yaml

  For metrics that are not part of the schema user can store them
  by calling log_metric method.
  """
  TYPE_NAME = 'kfp.Metrics'

  def __init__(self):
    self._instance_schema = self.__class__.get_artifact_type()

    # Calling base class init to setup the instance.
    super().__init__()

  @classmethod
  def get_artifact_type(cls) -> str:
    return artifact_utils.read_schema_file('metrics.yaml')

  @classmethod
  def get_ir_type(cls) -> pipeline_spec_pb2.ArtifactTypeSchema:
    return pipeline_spec_pb2.ArtifactTypeSchema(
      instance_schema=cls.get_artifact_type())

  def log_metric(self, metric: str, value: float):
    """Sets a custom scalar metric.

    Args:
      metric: Metric key
      value: Value of the metric.
    """
    self.metadata[metric] = value

class ClassificationMetrics(artifact.Artifact):
  """Metrics class representing a Classification Metrics.

  Clients using this class are expected to use log methods of the
  class to log metrics.
  """

  TYPE_NAME = 'kfp.ClassificationMetrics'

  def __init__(self):
    self._instance_schema = self.__class__.get_artifact_type()
    self._confusion_matrix = metrics_utils.ConfusionMatrix()
    super().__init__()

  @classmethod
  def get_artifact_type(cls) -> str:
    return artifact_utils.read_schema_file('classification_metrics.yaml')

  @classmethod
  def get_ir_type(cls) -> pipeline_spec_pb2.ArtifactTypeSchema:
    return pipeline_spec_pb2.ArtifactTypeSchema(
        instance_schema=cls.get_artifact_type())

  def log_roc_reading(self, threshold: float, tpr: float, fpr: float):
    """Logs a single data point in the ROC Curve.

    Args:
      threshold: Thresold value for the data point.
      tpr: True positive rate value of the data point.
      fpr: False positive rate value of the data point.
    """
    reading = metrics_utils.ConfidenceMetrics()
    reading.confidenceThreshold = threshold
    reading.recall = tpr
    reading.falsePositiveRate = fpr
    if 'confidenceMetrics' not in self.metadata.keys():
      self.metadata['confidenceMetrics'] = []

    self.metadata['confidenceMetrics'].append(reading.get_metrics())

  def load_roc_readings(self, readings: List[List[float]]):
    """Supports bulk loading ROC Curve readings.

    Args:
      readings: A 2-D list providing ROC Curve data points.
                The expected order of the data points is:
                  threshold, true_positive_rate, false_positive_rate.
    """
    self.metadata['confidenceMetrics'] = []
    for reading in readings:
      if len(reading) != 3:
        raise ValueError('Invalid ROC curve reading provided: {}. \
          Expected 3 values.'.format(reading))

      self.log_roc_reading(reading[0], reading[1], reading[2])

  def set_confusion_matrix_categories(self, categories: List[str]):
    """Stores confusion matrix categories.

    Categories are stored in the internal metrics_utils.ConfusionMatrix
    instance.

    Args:
      categories: List of strings specifying the categories.
    """

    self._confusion_matrix.set_categories(categories)
    self._update_confusion_matrix()

  def log_confusion_matrix_row(self, row_category: str, row: List[int]):
    """Logs a confusion matrix row.

    Row is updated on the internal metrics_utils.ConfusionMatrix
    instance.

    Args:
      row_category: Category to which the row belongs.
      row: List of integers specifying the values for the row.
    """
    self._confusion_matrix.log_row(row_category, row)
    self._update_confusion_matrix()

  def log_confusion_matrix_cell(self, row_category: str, col_category: str,
     value: int):
    """Logs a confusion matrix cell.

    Cell is updated on the internal metrics_utils.ConfusionMatrix
    instance.

    Args:
      row_category: String representing the name of the row category.
      col_category: String representing the name of the column category.
      value: Int value of the cell.
    """
    self._confusion_matrix.log_cell(row_category, col_category, value)
    self._update_confusion_matrix()

  def load_confusion_matrix(self, categories: List[str],
   matrix: List[List[int]]):
    """Supports bulk loading the whole confusion matrix.

    Args:
      categories: List of the category names.
      matrix: Complete confusion matrix.
    """

    self._confusion_matrix.load_matrix(categories, matrix)
    self._update_confusion_matrix()

  def _update_confusion_matrix(self):
    """Updates internal Confusion matrix instance."""
    self.metadata['confusionMatrix'] = self._confusion_matrix.get_metrics()


class SlicedClassificationMetrics(artifact.Artifact):
  """Metrics class representing Sliced Classification Metrics.

  Similar to ClassificationMetrics clients using this class are expected to use
  log methods of the class to log metrics with the difference being each log
  method takes a slice to associate the ClassificationMetrics.

  """

  TYPE_NAME = 'kfp.SlicedClassificationMetrics'

  def __init__(self):
    self._instance_schema = self.__class__.get_artifact_type()
    # Stores Sliced Classification Metrics instances.
    self._sliced_metrics = {}

    super().__init__()

  @classmethod
  def get_artifact_type(cls) -> str:
    return artifact_utils.read_schema_file(
      'sliced_classification_metrics.yaml')

  @classmethod
  def get_ir_type(cls) -> pipeline_spec_pb2.ArtifactTypeSchema:
    return pipeline_spec_pb2.ArtifactTypeSchema(
        instance_schema=cls.get_artifact_type())

  def _upsert_classification_metrics_for_slice(self, slice: str):
    """Upserts the classification metrics instance for a slice."""
    if slice not in self._sliced_metrics:
      self._sliced_metrics[slice] = ClassificationMetrics()

  def _update_metadata(self, slice: str):
    """Updates metadata to adhere to the metrics schema."""
    self.metadata = {}
    self.metadata['evaluationSlices'] = []
    for slice in self._sliced_metrics.keys():
      slice_metrics = {
        'slice': slice,
        'sliceClassificationMetrics': self._sliced_metrics[slice].metadata
      }
      self.metadata['evaluationSlices'].append(slice_metrics)

  def log_roc_reading(self, slice: str, threshold: float, tpr: float,
    fpr: float):
    """Logs a single data point in the ROC Curve of a slice.

    Args:
      slice: String representing slice label.
      threshold: Thresold value for the data point.
      tpr: True positive rate value of the data point.
      fpr: False positive rate value of the data point.
    """

    self._upsert_classification_metrics_for_slice(slice)
    self._sliced_metrics[slice].log_roc_reading(threshold, tpr, fpr)
    self._update_metadata(slice)

  def load_roc_readings(self, slice: str, readings: List[List[float]]):
    """Supports bulk loading ROC Curve readings for a slice.

    Args:
      slice: String representing slice label.
      readings: A 2-D list providing ROC Curve data points.
                The expected order of the data points is:
                  threshold, true_positive_rate, false_positive_rate.
    """
    self._upsert_classification_metrics_for_slice(slice)
    self._sliced_metrics[slice].load_roc_readings(readings)
    self._update_metadata(slice)

  def set_confusion_matrix_categories(self, slice: str, categories: List[str]):
    """Stores confusion matrix categories for a slice..

    Categories are stored in the internal metrics_utils.ConfusionMatrix
    instance of the slice.

    Args:
      slice: String representing slice label.
      categories: List of strings specifying the categories.
    """
    self._upsert_classification_metrics_for_slice(slice)
    self._sliced_metrics[slice].set_confusion_matrix_categories(
      categories)
    self._update_metadata(slice)

  def log_confusion_matrix_row(self, slice: str, row_category: str,
    row: List[int]):
    """Logs a confusion matrix row for a slice.

    Row is updated on the internal metrics_utils.ConfusionMatrix
    instance of the slice.

    Args:
      slice: String representing slice label.
      row_category: Category to which the row belongs.
      row: List of integers specifying the values for the row.
    """
    self._upsert_classification_metrics_for_slice(slice)
    self._sliced_metrics[slice].log_confusion_matrix_row(
      row_category, row)
    self._update_metadata(slice)

  def log_confusion_matrix_cell(self, slice: str, row_category: str,
    col_category: str, value: int):
    """Logs a confusion matrix cell for a slice..

    Cell is updated on the internal metrics_utils.ConfusionMatrix
    instance of the slice.

    Args:
      slice: String representing slice label.
      row_category: String representing the name of the row category.
      col_category: String representing the name of the column category.
      value: Int value of the cell.
    """
    self._upsert_classification_metrics_for_slice(slice)
    self._sliced_metrics[slice].log_confusion_matrix_cell(
      row_category, col_category, value)
    self._update_metadata(slice)

  def load_confusion_matrix(self, slice: str, categories: List[str],
   matrix: List[List[int]]):
    """Supports bulk loading the whole confusion matrix for a slice.

    Args:
      slice: String representing slice label.
      categories: List of the category names.
      matrix: Complete confusion matrix.
    """
    self._upsert_classification_metrics_for_slice(slice)
    self._sliced_metrics[slice].log_confusion_matrix_cell(categories,
      matrix)
    self._update_metadata(slice)
