# Copyright 2018 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
# in compliance with the License. You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software distributed under the License
# is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
# or implied. See the License for the specific language governing permissions and limitations under
# the License.


class ConfusionMatrixInstances(ColumnarInstances):
  """Defines Confusion Matrix Instances artifact.
  
  These are aggregated results from prediction results instance in classification
  scenario. Each instance should contain target (truth), predicted, and count values.
  """

  def __init__(self, storage, format, source, schema, target_column,
               predicted_column, count_column):
    """
    Args
      storage: one of ['local', 'gs', 'bq'].
      format: one of ['csv', 'json']. required of storage is 'local' or 'gs'. 
      source: value depend on format and storage. Can be a file pattern path or a bq table.
      schema: a list of {name: col_name, type: col_type}. col_type can be one of:
              'category', 'text', 'integer', 'float', 'image_url'.      
      target_column: name of target column. Must be of type category.
      predicted_column: name of predicted column. Must be of type category.
      count_column: name of count column. Must be of type integer.
    """

    super(ConfusionMatrixInstances, self).__init__(storage, format, source, schema)

    if not {target_column, predicted_column, count_column} <= {x['name'] for x in self._schema}:
      raise ValueError('target, predicted, or count column not in schema.')

    if self.schema_dict[target_column] != 'category' or self.schema_dict[predicted_column] != 'category':
      raise ValueError('target_column and predicted_column type has to be category.')

    if self.schema_dict[count_column] != 'integer':
      raise ValueError('count_column type has to be integer.')

    self._target_column = target_column
    self._predicted_column = predicted_column
    self._count_column = count_column    
  
  @property
  def target_column(self):
    return self._target_column

  @property
  def predicted_column(self):
    return self._predicted_column

  @property
  def count_column(self):
    return self._count_column


class ROCInstances(ColumnarInstances):
  """Defines ROC Instances artifact.
  
  These are aggregated results from prediction results instance above in binary classification
  scenario. See https://en.wikipedia.org/wiki/Receiver_operating_characteristic.
  """

  def __init__(self, storage, format, source, schema, fpr_column, tpr_column, threshold_column):
    """
    Args
      storage: one of ['local', 'gs', 'bq'].
      format: one of ['csv', 'json']. required of storage is 'local' or 'gs'. 
      source: value depend on format and storage. Can be a file pattern path or a bq table.
      schema: a list of {name: col_name, type: col_type}. col_type can be one of:
              'category', 'text', 'integer', 'float', 'image_url'.      
      fpr_column: false positive rate column. Must be of type float.
      tpr_column: true positive rate column. Must be of type float.
      threshold_column: threshold column. Must be of type float.
    """

    super(ROCInstances, self).__init__(storage, format, source, schema)

    if not {fpr_column, tpr_column, threshold_column} <= {x['name'] for x in self._schema}:
      raise ValueError('fpr, tpr, or threshold column not in schema.')

    if (self.schema_dict[fpr_column] != 'float' or
        self.schema_dict[tpr_column] != 'float' or
        self.schema_dict[threshold_column] != 'float'):
      raise ValueError('fpr, tpr, or threshold column must be of type float.')

    self._fpr_column = fpr_column
    self._tpr_column = tpr_column
    self._threshold_column = threshold_column    
  
  @property
  def fpr_column(self):
    return self._fpr_column

  @property
  def tpr_column(self):
    return self._tpr_column

  @property
  def threshold_column(self):
    return self._threshold_column


class FeatureSliceInstances(ColumnarInstances):
  """Defines Feature Slice Instances artifact.
  
  These are aggregated results from prediction results instance above. It basically slice
  features into buckets and compute metrics for each bucket.
  """

  def __init__(self, storage, format, source, schema, feature_bucket_column,
               count_column, metrics_columns):
    """
    Args
      storage: one of ['local', 'gs', 'bq'].
      format: one of ['csv', 'json']. required of storage is 'local' or 'gs'. 
      source: value depend on format and storage. Can be a file pattern path or a bq table.
      schema: a list of {name: col_name, type: col_type}. col_type can be one of:
              'category', 'text', 'integer', 'float', 'image_url'.      
      feature_bucket_column: name of column that holds feature bucket. Example of values:
                             'a > 1.5', 'weekday==Monday'.
      count_column: name of column that holds the count of instance in feature buckets.
      metrics_columns: A list of columns that hold metrics values for each feature bucket.
    """

    super(FeatureSliceInstances, self).__init__(storage, format, source, schema)

    if (not set([feature_bucket_column, count_column] + metrics_columns) <=
        {x['name'] for x in self._schema}):
      raise ValueError('one or more columns not in schema.')

    if (self.schema_dict[feature_bucket_column] != 'category'):
      raise ValueError('feature_bucket_column must be of type category.')
      
    if (self.schema_dict[count_column] != 'integer'):
      raise ValueError('count_column must be of type integer.')
      
    if any(self.schema_dict[x] != 'float' for x in metrics_columns):
      raise ValueError('metrics_columns must all be of type float.')         

    self._feature_bucket_column = feature_bucket_column
    self._count_column = count_column
    self._metrics_columns = metrics_columns    

  @property
  def feature_bucket_column(self):
    return self._feature_bucket_column

  @property
  def count_column(self):
    return self._count_column

  @property
  def metrics_columns(self):
    return self._metrics_columns
