# Copyright 2019 Google LLC
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

from ml_metadata.metadata_store import metadata_store
from ml_metadata.proto import metadata_store_pb2
from typing import List, Text, Dict, Any
import tensorflow as tf

class Metadata:
    def __init__(self, connection_config: metadata_store_pb2.ConnectionConfig):
        self._store = metadata_store.MetadataStore(connection_config)

    def get_all_artifacts(self) -> List[metadata_store_pb2.Artifact]:
        try:
            return self._store.get_artifacts()
        except tf.errors.NotFoundError:
            return []

    def _prepare_execution_type(self, type_name: Text,
                                exec_properties: Dict[Text, Any]) -> int:
      """Get a execution type. Use existing type if available."""
      try:
        execution_type = self._store.get_execution_type(type_name)
        if execution_type is None:
          raise RuntimeError('Execution type is None for {}.'.format(type_name))
        return execution_type.id
      except tf.errors.NotFoundError:
        execution_type = metadata_store_pb2.ExecutionType(name=type_name)
        execution_type.properties['state'] = metadata_store_pb2.STRING
        for k in exec_properties.keys():
          execution_type.properties[k] = metadata_store_pb2.STRING

        return self._store.put_execution_type(execution_type)

    def _prepare_execution(
        self, type_name: Text, state: Text,
        exec_properties: Dict[Text, Any]) -> metadata_store_pb2.Execution:
      """Create a new execution with given type and state."""
      execution = metadata_store_pb2.Execution()
      execution.type_id = self._prepare_execution_type(type_name, exec_properties)
      execution.properties['state'].string_value = tf.compat.as_text(state)
      for k, v in exec_properties.items():
        # We always convert execution properties to unicode.
        execution.properties[k].string_value = tf.compat.as_text(
            tf.compat.as_str_any(v))
      return execution

    def prepare_execution(self, type_name: Text, exec_properties: Any) -> int:
      execution = self._prepare_execution(type_name, 'new', exec_properties)
      [execution_id] = self._store.put_executions([execution])
      return execution_id

