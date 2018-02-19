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


class ColumnarInstances(object):
  """Defines Columnar Instances artifact.

  These are any columnar data where each instance must share the same schema.
  """

  _ALLOWED_STORAGE = ['local', 'gs', 'bq']
  # None for bq storage
  _ALLOWED_FORMAT = ['csv', 'json', None]
  _ALLOWED_SCHEMA_TYPE = ['key', 'category', 'text', 'integer', 'float', 'image_url']

  def __init__(self, storage, format, source, schema):
    """
    Args
      storage: one of ['local', 'gs', 'bq'].
      format: one of ['csv', 'json']. required of storage is 'local' or 'gs'. 
      source: value depend on format and storage. Can be a file pattern path or a bq table.
      schema: a list of {name: col_name, type: col_type}. col_type can be one of:
              'category', 'text', 'integer', 'float', 'image_url'.
    """

    if storage not in ColumnarInstances._ALLOWED_STORAGE:
      raise ValueError('Storage "%s" not in %s' % (storage, ColumnarInstances._ALLOWED_STORAGE))

    if format not in ColumnarInstances._ALLOWED_FORMAT:
      raise ValueError('Format "%s" not in %s' % (storage, ColumnarInstances._ALLOWED_FORMAT))
    
    if storage in ['local', 'gs'] and format is None:
      raise ValueError('Storage indicates it is a file system storage so format is required.')

    if storage == 'gs' and not source.startswith('gs://'):
      raise ValueError('Source does not look like a gcs path but storage is gs.')
      
    if storage == 'bq' and not (source.count('.') < 3 and source.count('.') > 0):
      raise ValueError('Source must be project.dataset.table or dataset.table when format is bq.')
      
    if any(x['type'] not in ColumnarInstances._ALLOWED_SCHEMA_TYPE for x in schema):
      raise ValueError('Only %s data types are supported' % ColumnarInstances._ALLOWED_SCHEMA_TYPE)

    self._storage = storage
    self._format = format
    self._source = source
    self._schema = schema
    self._schema_dict = {x['name']: x['type'] for x in schema}

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
  def schema(self):
    return self._schema

  @property
  def schema_dict(self):
    return self._schema_dict

