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

from __future__ import absolute_import

from kfp_component import BaseOp

from kubernetes import client, config
from kubernetes.client.rest import ApiException
from google.cloud import storage
import google

import mock
import unittest

class BaseOpTest(unittest.TestCase):

    @mock.patch('google.cloud.storage.Client')
    @mock.patch('kubernetes.client.CoreV1Api')
    @mock.patch('kubernetes.config.load_incluster_config')
    @mock.patch.dict('os.environ', {
        'KFP_POD_NAME': 'mock-pod-id',
        'KFP_TMP_LOCATION': 'gs://mock-tmp-location/dir'
    })
    def test_init_success(self, mock_load_incluster_config, mock_core_v1_api, mock_storage_client):
        mock_pod = mock_core_v1_api().read_namespaced_pod()
        mock_pod.metadata.labels = {
            'workflows.argoproj.io/workflow': 'mock-workflow'
        }
        mock_pod.metadata.annotations = {
            'workflows.argoproj.io/node-name': 'mock-node (1)'
        }

        mock_blob = mock_storage_client().get_bucket.return_value.blob.return_value
        mock_blob.download_as_string.return_value = '{"mock_id": "mock_value"}'
        
        op = BaseOp('test-op')
        self.assertEqual(op.name, 'test-op')
        self.assertEqual(op._argo_workflow_name, 'mock-workflow')
        self.assertEqual(op._argo_node_name, 'mock-node')
        self.assertEqual(op._staging_bucket, 'mock-tmp-location')
        self.assertEqual(op._staging_path, 'dir/tmp/mock-workflow/mock-node/test-op')
        self.assertEqual(op.staging_states, {'mock_id': 'mock_value'})

    @mock.patch('google.cloud.storage.Client')
    @mock.patch('kubernetes.client.CoreV1Api')
    @mock.patch('kubernetes.config.load_incluster_config')
    @mock.patch.dict('os.environ', {})
    def test_init_ignore_no_pod_name(self, mock_load_incluster_config, mock_core_v1_api, mock_storage_client):
        op = BaseOp('test-op')
        self.assertEqual(op.name, 'test-op')

    @mock.patch('google.cloud.storage.Client')
    @mock.patch('kubernetes.client.CoreV1Api')
    @mock.patch('kubernetes.config.load_incluster_config')
    @mock.patch.dict('os.environ', {
        'KFP_POD_NAME': 'mock-pod-id',
        'KFP_TMP_LOCATION': 'gs://mock-tmp-location/dir'
    })
    def test_init_ignore_get_pod_error(self, mock_load_incluster_config, mock_core_v1_api, mock_storage_client):
        mock_core_v1_api().read_namespaced_pod.side_effect = ApiException()

        op = BaseOp('test-op')
        self.assertEqual(op.name, 'test-op')

    @mock.patch('google.cloud.storage.Client')
    @mock.patch('kubernetes.client.CoreV1Api')
    @mock.patch('kubernetes.config.load_incluster_config')
    @mock.patch.dict('os.environ', {
        'KFP_POD_NAME': 'mock-pod-id'
    })
    def test_init_ignore_no_tmp_location(self, mock_load_incluster_config, mock_core_v1_api, mock_storage_client):
        mock_pod = mock_core_v1_api().read_namespaced_pod()
        mock_pod.metadata.labels = {
            'workflows.argoproj.io/workflow': 'mock-workflow'
        }
        mock_pod.metadata.annotations = {
            'workflows.argoproj.io/node-name': 'mock-node (1)'
        }

        op = BaseOp('test-op')
        self.assertEqual(op.name, 'test-op')
        self.assertEqual(op._argo_workflow_name, 'mock-workflow')
        self.assertEqual(op._argo_node_name, 'mock-node')

    @mock.patch('google.cloud.storage.Client')
    @mock.patch('kubernetes.client.CoreV1Api')
    @mock.patch('kubernetes.config.load_incluster_config')
    @mock.patch.dict('os.environ', {
        'KFP_POD_NAME': 'mock-pod-id',
        'KFP_TMP_LOCATION': 'gs://mock-tmp-location/dir'
    })
    def test_execute_fail_staging_states(self, mock_load_incluster_config, mock_core_v1_api, mock_storage_client):
        mock_pod = mock_core_v1_api().read_namespaced_pod()
        mock_pod.metadata.labels = {
            'workflows.argoproj.io/workflow': 'mock-workflow'
        }
        mock_pod.metadata.annotations = {
            'workflows.argoproj.io/node-name': 'mock-node (1)'
        }

        mock_blob = mock_storage_client().get_bucket.return_value.blob.return_value
        mock_blob.download_as_string.side_effect = google.cloud.exceptions.NotFound('test')
        
        op = BaseOp('test-op')
        op.on_executing = mock.MagicMock(side_effect=Exception())
        op.staging_states = {
            'mock-key': 'mock-value'
        }

        try:
            op.execute()
        except:
            pass

        mock_blob.upload_from_string.assert_called_once_with('{"mock-key": "mock-value"}')

    @mock.patch('google.cloud.storage.Client')
    @mock.patch('kubernetes.client.CoreV1Api')
    @mock.patch('kubernetes.config.load_incluster_config')
    @mock.patch.dict('os.environ', {
        'KFP_POD_NAME': 'mock-pod-id',
        'KFP_TMP_LOCATION': 'gs://mock-tmp-location/dir'
    })
    def test__exit_gracefully_cancel(self, mock_load_incluster_config, mock_core_v1_api, mock_storage_client):
        mock_pod = mock_core_v1_api().read_namespaced_pod()
        mock_pod.metadata.labels = {
            'workflows.argoproj.io/workflow': 'mock-workflow'
        }
        mock_pod.metadata.annotations = {
            'workflows.argoproj.io/node-name': 'mock-node (1)',
            'workflows.argoproj.io/execution': '{"deadline": "1970-01-01T00:00:00Z"}'
        }

        mock_blob = mock_storage_client().get_bucket.return_value.blob.return_value
        mock_blob.download_as_string.side_effect = google.cloud.exceptions.NotFound('test')
        
        op = BaseOp('test-op')
        op.on_cancelling = mock.MagicMock()

        op._exit_gracefully(0, 0)

        op.on_cancelling.assert_called_once()

if __name__ == '__main__':
    unittest.main()