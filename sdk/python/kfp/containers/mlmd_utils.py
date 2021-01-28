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
import ml_metadata as mlmd
import ml_metadata as mlmd
from ml_metadata.proto import metadata_store_pb2
from ml_metadata.proto import metadata_store_service_pb2
import random
import time

# Number of times to retry initialization of connection.
_MAX_INIT_RETRY = 10


class Metadata(object):
  """Helper class to handle interaction with metadata store."""
  def __init__(
      self,
      connection_config: metadata_store_pb2.MetadataStoreClientConfig):
    for _ in range(_MAX_INIT_RETRY):
      try:
        self._store = mlmd.MetadataStore(connection_config)
      except RuntimeError as err:
        time.sleep(random.random())
        continue
      else:
        return

    raise RuntimeError('Failed to establish MLMD connection with error.')

  @property
  def store(self) -> mlmd.MetadataStore:
    """Returns underlying MetadataStore."""
    if self._store is None:
      raise RuntimeError('Metadata object is not instantiated.')
    return self._store



