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


class PreprocessedInstances(object):
  """Defines Preprocessed Instances artifact.
  
  These are preprocessed instances that can be used directly for training.
  Most if not all data should be numeric.
  """

  _ALLOWED_STORAGE = ['local', 'gs']
  _ALLOWED_FORMAT = ['libsvm', 'tfrecord', 'mat']
  
  def __init__(self, storage, format, source, num_features=None):
    """
    Args
      storage: one of ['local', 'gs'].
      format: one of ['libsvm', 'tfrecord', 'mat'].
      source: file pattern path.
      num_features: libsvm instances need to also provide feature size to its readers.
    """

    if storage not in PreprocessedInstances._ALLOWED_STORAGE:
      raise ValueError('Storage "%s" not in %s' % (storage, PreprocessedInstances._ALLOWED_STORAGE))

    if format not in PreprocessedInstances._ALLOWED_FORMAT:
      raise ValueError('Format "%s" not in %s' % (storage, PreprocessedInstances._ALLOWED_FORMAT))

    if storage == 'gs' and not source.startswith('gs://'):
      raise ValueError('Source does not look like a gcs path while storage is gs.')
      
    if format == 'libsvm' and num_features is None:
      raise ValueError('libsvm requires num_features.')      

    self._storage = storage
    self._format = format
    self._source = source
    self._num_features = num_features

  @property
  def storage(self):
    return self._storage

  @property
  def format(self):
    return self._format

  @property
  def source(self):
    return self._source

  @property
  def num_features(self):
    return self._num_features

