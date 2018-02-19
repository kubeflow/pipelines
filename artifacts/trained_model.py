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


class XGBoostModel(object):
  """Defines directory structure and file formats for materialized XGBoost Model.

  An example of the directory structure:
  /model
    /model.bin
  /analysis
    /stats.json
    /col1_vocab.csv
  /features.json

  Files:
    /model directory: model files created by xgb.save_model() or xgb.dump_model().
    /analysis directory: See AnalysisResults class.
    features.json: Contains how to transform columnar data into numeric features. Used in prediction
      when it takes raw data, performs transformation and then feed it to model.
    Example:

      {
         "column_name_1": {"transform": "scale"},
         "column_name_2": {"transform": "log"},
         "column_name_3": {"transform": "bucket", "thresholds": "2, 5, 8"},
         "column_name_4": {"transform": "target"},
         "column_name_5": {"transform": "one_hot"},
         "column_name_6": {"transform": "bag_of_words"},
      }

    The format of the dict is `name`: `transform-dict` where the
    `name` is the name of the transformed feature. The `source_column`
    value lists what column in the input data is the source for this
    transformation. If `source_column` is missing, it is assumed the
    `name` is a source column and the transformed feature will have
    the same name as the input column.
    A list of supported `transform-dict`s is below.

    For integer and float columns (in schema, type == "integer" or type == "float"):
      {"transform": "identity"}: does nothing.
      {"transform": "scale", "value": a}: scale a numerical column to
          [-a, a]. If value is missing, x defaults to 1. Default.
      {"transform": "log"}: performs logarithmic scale on the value.
      {"transform": "bucket"}: Turn values into buckets and then perform one-hot encoding.

    For category columns (type == "category"):
      {"transform": "one_hot"}: makes a one-hot encoding. Default.
    
    For text columns (type == "text"):
      {"transform": "bag_of_words", "separator": " "}: bag of words transform for text columns. Default.
      {"transform": "tfidf", "separator": " "}: TFIDF transform for string columns.

    Other types of columns (type == "key", type == "image_url") are not
      supported in xgboost model so they will be ignored in training and prediction.
  """

  def __init__(self, storage, source, xgb_version):
    """
    Args
      storage: one of ['local', 'gs'].
      source: the path to the directory holding files.
      xgb_version: the version of xgboost that was used to train and export the model.
    """

    if storage == 'gs' and not source.startswith('gs://'):
      raise ValueError('Source does not look like a gcs path but storage is gs.')
      
    self._storage = storage
    self._source = source
    self._xgb_version = xgb_version

  @property
  def storage(self):
    return self._storage

  @property
  def source(self):
    return self._source

  @property
  def xgb_version(self):
    return self._xgb_version


class TFModel(object):
  """Defines directory structure and file formats for materialized TensorFlow Model.

  An example of the directory structure:
  /model
    /mymodel-1000.meta
    /checkpoint
  Files:
    /model directory: tensorflow's SavedModel (saved with tf.saved_model.builder)
  """

  _ALLOWED_INPUT_TYPE = ['csv_line', 'list_of_tensors']

  def __init__(self, storage, source, tf_version, input_type):
    """
    Args
      storage: one of ['local', 'gs'].
      source: the path to the directory holding files.
      tf_version: the version of TensorFlow that was used to train and export the model.
      input_type: the input data type, one of ['csv_line', 'list_of_tensors']
        csv_line: the input batch is a list of csv lines. Inside the TF graph
                  it decodes CSV and match each value to tensors.
        list_of_tensors: the input batch is a dictionary of tensor name to list of values.
    """

    if storage == 'gs' and not source.startswith('gs://'):
      raise ValueError('Source does not look like a gcs path but storage is gs.')

    if input_type not in TFModel._ALLOWED_INPUT_TYPE:
      raise ValueError('input_format "%s" not in %s' % (storage, TFModel._ALLOWED_INPUT_TYPE))

    self._storage = storage
    self._source = source
    self._tf_version = tf_version
    self._input_type = input_type

  @property
  def storage(self):
    return self._storage

  @property
  def source(self):
    return self._source

  @property
  def tf_version(self):
    return self._tf_version

  @property
  def input_type(self):
    return self._input_type
