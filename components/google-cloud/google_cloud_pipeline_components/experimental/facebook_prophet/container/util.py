# Copyright 2021 Google LLC. All Rights Reserved.
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
import pandas as pd
from enum import Enum
from google.cloud import storage, bigquery, bigquery_storage
from google.cloud.exceptions import NotFound
import tempfile
import os

class DataSourceType(Enum):
  REMOTE_CSV = 1
  BIG_QUERY = 2
  LOCAL_CSV = 3


class DataSource():
  """Identifies a tabular data source"""

  def __init__(self, resource_locator: str):
    self.resource_locator = resource_locator
    if resource_locator.startswith('gs://'):
      self.resource_locator = resource_locator
      self.resource_type = DataSourceType.REMOTE_CSV
    elif os.path.isfile(resource_locator):
      self.resource_locator = resource_locator
      self.resource_type = DataSourceType.LOCAL_CSV
    else:
      try:
        bigquery.Client().get_table(resource_locator)
        self.resource_type = DataSourceType.BIG_QUERY
      except NotFound:
        raise ValueError('The data source is not a valid GCS URI, existing file, or existing BigQuery table.')

  def load(self) -> pd.DataFrame:
    if self.resource_type == DataSourceType.REMOTE_CSV:
      with tempfile.TemporaryFile() as temp_file:
        storage.Client().download_blob_to_file(self.resource_locator, temp_file)
        temp_file.seek(0)
        return pd.read_csv(temp_file)
    elif self.resource_type == DataSourceType.BIG_QUERY:
      table = bigquery.TableReference.from_string(self.resource_locator)
      rows = bigquery.Client().list_rows(table)
      return rows.to_dataframe(
          bqstorage_client=bigquery_storage.BigQueryReadClient())
    else:
      return pd.read_csv(self.resource_locator)
