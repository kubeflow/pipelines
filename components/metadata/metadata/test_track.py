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

import unittest
from track import track
from metadata import Metadata
from ml_metadata.proto import metadata_store_pb2

class TestTrack(unittest.TestCase):
    def test_track_succeed(self):
        connection_config = metadata_store_pb2.ConnectionConfig()
        connection_config.fake_database.SetInParent()
        mlmd = Metadata(connection_config)
        execution_id = track({
            'workflow_id': 'w1',
            'image': 'python:3.6',
            'command': ['sh', '-c'],
            'args': ['echo', 'hello world']
        }, mlmd)

        [execution] = mlmd._store.get_executions_by_id([execution_id])
        self.assertEqual('w1', execution.properties['workflow_id'].string_value)
        self.assertEqual('python:3.6', execution.properties['image'].string_value)

    def test_track_digest_unchanged(self):
        connection_config = metadata_store_pb2.ConnectionConfig()
        connection_config.fake_database.SetInParent()
        mlmd = Metadata(connection_config)
        e1_id = track({
            'workflow_id': 'w1',
            'image': 'python:3.6',
            'command': ['sh', '-c'],
            'args': ['echo', 'hello world']
        }, mlmd)
        e2_id = track({
            'workflow_id': 'w2',
            'image': 'python:3.6',
            'command': ['sh', '-c'],
            'args': ['echo', 'hello world']
        }, mlmd)

        [e1, e2] = mlmd._store.get_executions_by_id([e1_id, e2_id])
        self.assertEqual(e1.properties['digest'], e2.properties['digest'])
