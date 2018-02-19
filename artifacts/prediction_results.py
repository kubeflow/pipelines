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


class TopNPredictionResults(ColumnarInstances):
  """Defines TopN prediction results artifact.

  TopN prediction results are like (each line below is a column, 4 instances)
    predicted: sunflower, roses, daisy, daisy
    predicted1: roses, tulips, dendalion, dendalion
    predicted2: daisy, daisy, tulips, tulips
    prob: 0.5, 0.4, 0.97, 0.8
    prob1: 0.3, 0.3, 0.01, 0.1
    prob2: 0.1, 0.05, 0.01, 0.05
    target: sunflower, roses, tulips, daisy
  """

  def __init__(self, storage, format, source, schema, predicted_columns, prob_columns, target_column=None):
    """
    Args
      storage: one of ['local', 'gs', 'bq'].
      format: one of ['csv', 'json']. required of storage is 'local' or 'gs'. 
      source: value depend on format and storage. Can be a file pattern path or a bq table.
      schema: a list of {name: col_name, type: col_type}. col_type can be one of:
              'category', 'text', 'integer', 'float', 'image_url'.
      predicted_columns: list of column names in the desc order of probanilities.
      prob_columns: list of column names in the desc order or probabilities.
      target_column: optionally, name of truth column.
    """

    super(TopNPredictionResults, self).__init__(storage, format, source, schema)
    if len(predicted_columns) != len(prob_columns):
      raise ValueError('Length of predicted_columns does not match length of prob_columns.')
      
    if not set(predicted_columns) <= {x['name'] for x in self._schema}:
      raise ValueError('predicted_columns contain columns not in schema.')

    if not set(prob_columns) <= {x['name'] for x in self._schema}:
      raise ValueError('prob_columns contain columns not in schema.')
      
    for predicted_column in predicted_columns:
      if self.schema_dict[predicted_column] != 'category':
        raise ValueError('All predicted_columns need to be of type category.')
        
    for prob_column in prob_columns:
      if self.schema_dict[prob_column] != 'float':
        raise ValueError('All prob_columns need to be of type float.')
        
    if target_column is not None:
      if target_column not in {x['name'] for x in self._schema}:
        raise ValueError('target_column not in schema.')

      if self.schema_dict[target_column] != 'category':
        raise ValueError('target_column type has to be category.')
      
    self._predicted_columns = predicted_columns
    self._prob_columns = prob_columns
    self._target_column = target_column

  @property
  def predicted_columns(self):
    return self._predicted_columns

  @property
  def prob_columns(self):
    return self._prob_columns
  
  @property
  def target_column(self):
    return self._target_column


class AllClassesPredictionResults(ColumnarInstances):
  """Defines All-classes Prediction Results artifacts.

  All classes prediction results are like (each line below is a column, 4 instances)
    sunflower: 0.1, 0.2, 0.98, 0.4
    roses: 0.8, 0.1, 0.005, 0.3
    daisy: 0.05, 0.6, 0.005, 0.2
    dandelion: 0.01, 0.05, 0.002, 0.05
    tulips: 0.04, 0.05, 0.008, 0.05
    target: roses, daisy, sunflower, roses
  """

  def __init__(self, storage, format, source, schema, prob_columns, target_column=None):
    """
    Args
      storage: one of ['local', 'gs', 'bq'].
      format: one of ['csv', 'json']. required of storage is 'local' or 'gs'. 
      source: value depend on format and storage. Can be a file pattern path or a bq table.
      schema: a list of {name: col_name, type: col_type}. col_type can be one of:
              'category', 'text', 'integer', 'float', 'image_url'.
      prob_columns: columns one for each class, containing prob values.
          for example: ['sunflower', 'daisy', 'roses']
      target_column: optionally,name of truth column.
    """

    super(AllClassesPredictionResults, self).__init__(storage, format, source, schema)

    if not set(prob_columns) <= {x['name'] for x in self._schema}:
      raise ValueError('prob_columns contain columns not in schema.')
      
    for prob_column in prob_columns:
      if self.schema_dict[prob_column] != 'float':
        raise ValueError('All prob_columns need to be of type float.')

    if target_column is not None:
      if target_column not in {x['name'] for x in self._schema}:
        raise ValueError('target_column not in schema.')

      if self.schema_dict[target_column] != 'category':
        raise ValueError('target_column type has to be category.')

    self._prob_columns = prob_columns
    self._target_column = target_column

  @property
  def prob_columns(self):
    return self._prob_columns
  
  @property
  def target_column(self):
    return self._target_column


class RegressionPredictionResults(ColumnarInstances):
  """Defines Regression Prediction Results artifacts.

  Regression Prediction Results are like (each line below is a column, 3 instances)
    predicted: 1.5, -0.2, 0.1
    target: 1.8, 0.1, -0.05
  """

  def __init__(self, storage, format, source, schema, predicted_column, target_column=None):
    """
    Args
      storage: one of ['local', 'gs', 'bq'].
      format: one of ['csv', 'json']. required of storage is 'local' or 'gs'. 
      source: value depend on format and storage. Can be a file pattern path or a bq table.
      schema: a list of {name: col_name, type: col_type}. col_type can be one of:
              'category', 'text', 'integer', 'float', 'image_url'.
      predicted_column: name of column containing predicted values.
      target_column: optionally, name of truth column.
    """

    super(RegressionPredictionResults, self).__init__(storage, format, source, schema)

    if predicted_column not in {x['name'] for x in self._schema}:
      raise ValueError('predicted_column not in schema.')

    if self.schema_dict[predicted_column] != 'float':
      raise ValueError('predicted_column type has to be float.')

    if target_column is not None:
      if target_column not in {x['name'] for x in self._schema}:
        raise ValueError('target_column not in schema.')

      if self.schema_dict[target_column] != 'float':
        raise ValueError('target_column type has to be float.')

    self._predicted_column = predicted_column
    self._target_column = target_column

  @property
  def predicted_column(self):
    return self._predicted_column
  
  @property
  def target_column(self):
    return self._target_column
