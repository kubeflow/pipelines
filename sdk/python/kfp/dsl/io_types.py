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
"""Classes for input/output types in KFP SDK.

These are only compatible with v2 Pipelines.
"""

import os
from typing import Dict, List, Optional, Type, TypeVar


class Artifact(object):
  """Generic Artifact class.

  This class is meant to represent the metadata around an input or output
  machine-learning Artifact. Artifacts have URIs, which can either be a location
  on disk (or Cloud storage) or some other resource identifier such as
  an API resource name.

  Artifacts carry a `metadata` field, which is a dictionary for storing
  metadata related to this artifact.
  """
  TYPE_NAME = "system.Artifact"

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    """Initializes the Artifact with the given name, URI and metadata."""
    self.uri = uri or ''
    self.name = name or ''
    self.metadata = metadata or {}


class Model(Artifact):
  """An artifact representing an ML Model."""
  TYPE_NAME = "system.Model"

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)

  @property
  def framework(self) -> str:
    return self._get_framework()

  def _get_framework(self) -> str:
    return self.metadata.get('framework', '')

  @framework.setter
  def framework(self, framework: str):
    self._set_framework(framework)

  def _set_framework(self, framework: str):
    self.metadata['framework'] = framework


class Dataset(Artifact):
  """An artifact representing an ML Dataset."""
  TYPE_NAME = "system.Dataset"

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)


class Metrics(Artifact):
  """Represent a simple base Artifact type to store key-value scalar metrics.
  """
  TYPE_NAME = "system.Metrics"

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)

  def log_metric(self, metric: str, value: float):
    """Sets a custom scalar metric.

    Args:
      metric: Metric key
      value: Value of the metric.
    """
    self.metadata[metric] = value

class ClassificationMetrics(Artifact):
  """Represents Artifact class to store Classification Metrics."""
  TYPE_NAME = "system.ClassificationMetrics"

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)

  def log_roc_reading(self, threshold: float, tpr: float, fpr: float):
    """Logs a single data point in the ROC Curve.

    Args:
      threshold: Thresold value for the data point.
      tpr: True positive rate value of the data point.
      fpr: False positive rate value of the data point.
    """

    roc_reading = {'confidenceThreshold': threshold, 'recall': tpr, 'falsePositiveRate': fpr}
    if 'confidenceMetrics' not in self.metadata.keys():
      self.metadata['confidenceMetrics'] = []

    self.metadata['confidenceMetrics'].append(roc_reading)

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

    Args:
      categories: List of strings specifying the categories.
    """

    self._categories =[]
    annotation_specs = []
    for category in categories:
      annotation_spec = {
        'displayName' : category
      }
      self._categories.append(category)
      annotation_specs.append(annotation_spec)

    self._matrix = [
        [ 0 for i in range(len(self._categories)) ]
          for j in range(len(self._categories)) ]

    self._confusion_matrix = {}
    self._confusion_matrix['annotationSpecs'] = annotation_specs
    self._confusion_matrix['rows'] = self._matrix
    self.metadata['confusionMatrix'] = self._confusion_matrix

  def log_confusion_matrix_row(self, row_category: str, row: List[float]):
    """Logs a confusion matrix row.

    Args:
      row_category: Category to which the row belongs.
      row: List of integers specifying the values for the row.

     Raises:
      ValueError: If row_category is not in the list of categories
      set in set_categories call.
    """
    if row_category not in self._categories:
      raise ValueError('Invalid category: {} passed. Expected one of: {}'.\
        format(row_category, self._categories))

    if len(row) != len(self._categories):
      raise ValueError('Invalid row. Expected size: {} got: {}'.\
        format(len(self._categories), len(row)))

    self._matrix[self._categories.index(row_category)] = row
    self.metadata['confusionMatrix'] = self._confusion_matrix

  def log_confusion_matrix_cell(self, row_category: str, col_category: str, value: int):
    """Logs a cell in the confusion matrix.

    Args:
      row_category: String representing the name of the row category.
      col_category: String representing the name of the column category.
      value: Int value of the cell.

    Raises:
      ValueError: If row_category or col_category is not in the list of
       categories set in set_categories.
    """
    if row_category not in self._categories:
      raise ValueError('Invalid category: {} passed. Expected one of: {}'.\
        format(row_category, self._categories))

    if col_category not in self._categories:
      raise ValueError('Invalid category: {} passed. Expected one of: {}'.\
        format(row_category, self._categories))

    self._matrix[self._categories.index(row_category)][
      self._categories.index(col_category)] = value
    self.metadata['confusionMatrix'] = self._confusion_matrix

  def load_confusion_matrix(self, categories: List[str], matrix: List[List[int]]):
    """Supports bulk loading the whole confusion matrix.

    Args:
      categories: List of the category names.
      matrix: Complete confusion matrix.

    Raises:
      ValueError: Length of categories does not match number of rows or columns.
    """
    self.set_confusion_matrix_categories(categories)

    if len(matrix) != len(categories):
      raise ValueError('Invalid matrix: {} passed for categories: {}'.\
        format(matrix, categories))

    for index in range(len(categories)):
      if len(matrix[index]) != len(categories):
        raise ValueError('Invalid matrix: {} passed for categories: {}'.\
          format(matrix, categories))

      self.log_confusion_matrix_row(categories[index], matrix[index])

    self.metadata['confusionMatrix'] = self._confusion_matrix

T = TypeVar('T', bound=Artifact)

_GCS_LOCAL_MOUNT_PREFIX = '/gcs/'


class _IOArtifact():
  """Internal wrapper class for representing Input/Output Artifacts."""

  def __init__(self, artifact_type: Type[T], artifact: Optional[T] = None):
    self.type = artifact_type
    self._artifact = artifact

  def get(self) -> T:
    return self._artifact

  @property
  def uri(self):
    return self._artifact.uri

  @uri.setter
  def uri(self, uri):
    self._artifact.uri = uri

  @property
  def path(self):
    return self._get_path()

  @path.setter
  def path(self, path):
    self._set_path(path)

  def _get_path(self) -> str:
    if self._artifact.uri.startswith('gs://'):
      return _GCS_LOCAL_MOUNT_PREFIX + self._artifact.uri[len('gs://'):]

  def _set_path(self, path):
    if path.startswith(_GCS_LOCAL_MOUNT_PREFIX):
      path = 'gs://' + path[len(_GCS_LOCAL_MOUNT_PREFIX):]
    self._artifact.uri = path


class InputArtifact(_IOArtifact):

  def __init__(self, artifact_type: Type[T], artifact: Optional[T] = None):
    super().__init__(artifact_type=artifact_type, artifact=artifact)


class OutputArtifact(_IOArtifact):

  def __init__(self, artifact_type: Type[T], artifact: Optional[T] = None):
    super().__init__(artifact_type=artifact_type, artifact=artifact)
    if artifact is not None:
      os.makedirs(self.path, exist_ok=True)
      self.path = os.path.join(self.path, 'data')


_SCHEMA_TITLE_TO_TYPE: Dict[str, Artifact] = {
    x.TYPE_NAME: x for x in [Artifact, Model, Dataset, Metrics, ClassificationMetrics]
}


def create_runtime_artifact(runtime_artifact: Dict) -> Artifact:
  """Creates an Artifact instance from the specified RuntimeArtifact.

  Args:
    runtime_artifact: Dictionary representing JSON-encoded RuntimeArtifact.
  """
  schema_title = runtime_artifact.get('type', {}).get('schemaTitle', '')

  artifact_type = _SCHEMA_TITLE_TO_TYPE.get(schema_title)
  if not artifact_type:
    artifact_type = Artifact
  return artifact_type(
      uri=runtime_artifact.get('uri', ''),
      name=runtime_artifact.get('name', ''),
      metadata=runtime_artifact.get('metadata', {}),
  )
