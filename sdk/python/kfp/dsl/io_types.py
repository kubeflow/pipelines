# Copyright 2021 The Kubeflow Authors
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
from typing import Dict, Generic, List, Optional, Type, TypeVar, Union


_GCS_LOCAL_MOUNT_PREFIX = '/gcs/'


class Artifact(object):
  """Generic Artifact class.

  This class is meant to represent the metadata around an input or output
  machine-learning Artifact. Artifacts have URIs, which can either be a location
  on disk (or Cloud storage) or some other resource identifier such as
  an API resource name.

  Artifacts carry a `metadata` field, which is a dictionary for storing
  metadata related to this artifact.
  """
  TYPE_NAME = 'system.Artifact'

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    """Initializes the Artifact with the given name, URI and metadata."""
    self.uri = uri or ''
    self.name = name or ''
    self.metadata = metadata or {}

  @property
  def path(self):
    return self._get_path()

  @path.setter
  def path(self, path):
    self._set_path(path)

  def _get_path(self) -> str:
    if self.uri.startswith('gs://'):
      return _GCS_LOCAL_MOUNT_PREFIX + self.uri[len('gs://'):]

  def _set_path(self, path):
    if path.startswith(_GCS_LOCAL_MOUNT_PREFIX):
      path = 'gs://' + path[len(_GCS_LOCAL_MOUNT_PREFIX):]
    self.uri = path


class Model(Artifact):
  """An artifact representing an ML Model."""
  TYPE_NAME = 'system.Model'

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
  TYPE_NAME = 'system.Dataset'

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)


class Metrics(Artifact):
  """Represent a simple base Artifact type to store key-value scalar metrics."""
  TYPE_NAME = 'system.Metrics'

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
  TYPE_NAME = 'system.ClassificationMetrics'

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)

  def log_roc_data_point(self, fpr: float, tpr: float, threshold: float):
    """Logs a single data point in the ROC Curve.

    Args:
      fpr: False positive rate value of the data point.
      tpr: True positive rate value of the data point.
      threshold: Threshold value for the data point.
    """

    roc_reading = {
        'confidenceThreshold': threshold,
        'recall': tpr,
        'falsePositiveRate': fpr
    }
    if 'confidenceMetrics' not in self.metadata.keys():
      self.metadata['confidenceMetrics'] = []

    self.metadata['confidenceMetrics'].append(roc_reading)

  def log_roc_curve(self, fpr: List[float], tpr: List[float],
                    threshold: List[float]):
    """Logs an ROC curve.

    The list length of fpr, tpr and threshold must be the same.

    Args:
      fpr: List of false positive rate values.
      tpr: List of true positive rate values.
      threshold: List of threshold values.
    """
    if len(fpr) != len(tpr) or len(fpr) != len(threshold) or len(tpr) != len(
        threshold):
      raise ValueError('Length of fpr, tpr and threshold must be the same. '
                       'Got lengths {}, {} and {} respectively.'.format(
                           len(fpr), len(tpr), len(threshold)))

    for i in range(len(fpr)):
      self.log_roc_data_point(fpr=fpr[i], tpr=tpr[i], threshold=threshold[i])

  def set_confusion_matrix_categories(self, categories: List[str]):
    """Stores confusion matrix categories.

    Args:
      categories: List of strings specifying the categories.
    """

    self._categories = []
    annotation_specs = []
    for category in categories:
      annotation_spec = {'displayName': category}
      self._categories.append(category)
      annotation_specs.append(annotation_spec)

    self._matrix = []
    for row in range(len(self._categories)):
      self._matrix.append({'row': [0] * len(self._categories)})

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

    self._matrix[self._categories.index(row_category)] = {'row': row}
    self.metadata['confusionMatrix'] = self._confusion_matrix

  def log_confusion_matrix_cell(self, row_category: str, col_category: str,
                                value: int):
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

    self._matrix[self._categories.index(row_category)]['row'][
        self._categories.index(col_category)] = value
    self.metadata['confusionMatrix'] = self._confusion_matrix

  def log_confusion_matrix(self, categories: List[str],
                           matrix: List[List[int]]):
    """Logs a confusion matrix.

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


class SlicedClassificationMetrics(Artifact):
  """Metrics class representing Sliced Classification Metrics.

  Similar to ClassificationMetrics clients using this class are expected to use
  log methods of the class to log metrics with the difference being each log
  method takes a slice to associate the ClassificationMetrics.

  """

  TYPE_NAME = 'system.SlicedClassificationMetrics'

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)

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
                The expected order of the data points is: threshold,
                  true_positive_rate, false_positive_rate.
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
    self._sliced_metrics[slice].set_confusion_matrix_categories(categories)
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
    self._sliced_metrics[slice].log_confusion_matrix_row(row_category, row)
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
    self._sliced_metrics[slice].log_confusion_matrix_cell(categories, matrix)
    self._update_metadata(slice)


T = TypeVar('T')


class InputAnnotation():
  """Marker type for input artifacts."""
  pass



class OutputAnnotation():
  """Marker type for output artifacts."""
  pass


# TODO: Use typing.Annotated instead of this hack.
# With typing.Annotated (Python 3.9+ or typing_extensions package), the
# following would look like:
# Input = typing.Annotated[T, InputAnnotation]
# Output = typing.Annotated[T, OutputAnnotation]


# Input represents an Input artifact of type T.
Input = Union[T, InputAnnotation]

# Output represents an Output artifact of type T.
Output = Union[T, OutputAnnotation]


def is_artifact_annotation(typ) -> bool:
  if hasattr(typ, '_subs_tree'):  # Python 3.6
    subs_tree = typ._subs_tree()
    return len(subs_tree) == 3 and subs_tree[0] == Union and subs_tree[2] in [InputAnnotation, OutputAnnotation]

  if not hasattr(typ, '__origin__'):
    return False


  if typ.__origin__ != Union and type(typ.__origin__) != type(Union):
    return False


  if not hasattr(typ, '__args__') or len(typ.__args__) != 2:
    return False

  if typ.__args__[1] not in [InputAnnotation, OutputAnnotation]:
    return False

  return True

def is_input_artifact(typ) -> bool:
  """Returns True if typ is of type Input[T]."""
  if not is_artifact_annotation(typ):
    return False

  if hasattr(typ, '_subs_tree'):  # Python 3.6
    subs_tree = typ._subs_tree()
    return len(subs_tree) == 3 and subs_tree[2]  == InputAnnotation

  return typ.__args__[1] == InputAnnotation

def is_output_artifact(typ) -> bool:
  """Returns True if typ is of type Output[T]."""
  if not is_artifact_annotation(typ):
    return False

  if hasattr(typ, '_subs_tree'):  # Python 3.6
    subs_tree = typ._subs_tree()
    return len(subs_tree) == 3 and subs_tree[2]  == OutputAnnotation

  return typ.__args__[1] == OutputAnnotation

def get_io_artifact_class(typ):
  if not is_artifact_annotation(typ):
    return None
  if typ == Input or typ == Output:
    return None

  if hasattr(typ, '_subs_tree'):  # Python 3.6
    subs_tree = typ._subs_tree()
    if len(subs_tree) != 3:
      return None
    return subs_tree[1]

  return typ.__args__[0]

def get_io_artifact_annotation(typ):
  if not is_artifact_annotation(typ):
    return None

  if hasattr(typ, '_subs_tree'):  # Python 3.6
    subs_tree = typ._subs_tree()
    if len(subs_tree) != 3:
      return None
    return subs_tree[2]

  return typ.__args__[1]



_SCHEMA_TITLE_TO_TYPE: Dict[str, Artifact] = {
    x.TYPE_NAME: x
    for x in [Artifact, Model, Dataset, Metrics, ClassificationMetrics]
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
