# Copyright 2018 Google LLC
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
import mock
import unittest

from googleapiclient import errors
from kfp_component.google.dataproc import delete_cluster

MODULE = 'kfp_component.google.dataproc._delete_cluster'

@mock.patch(MODULE + '.KfpExecutionContext')
@mock.patch(MODULE + '.DataprocClient')
class TestDeleteCluster(unittest.TestCase):

    def test_delete_cluster_succeed(self, mock_client, mock_context):
        mock_context().__enter__().context_id.return_value = 'ctx1'

        delete_cluster('mock-project', 'mock-region', 'mock-cluster')

        mock_client().delete_cluster.assert_called_with('mock-project', 
            'mock-region', 'mock-cluster', request_id='ctx1')

    def test_delete_cluster_ignore_not_found(self, mock_client, mock_context):
        mock_context().__enter__().context_id.return_value = 'ctx1'
        mock_client().delete_cluster.side_effect = errors.HttpError(
            resp = mock.Mock(status=404),
            content = b'not found'
        )

        delete_cluster('mock-project', 'mock-region', 'mock-cluster')

