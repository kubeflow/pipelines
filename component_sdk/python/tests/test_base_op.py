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

import mock
import unittest

@mock.patch('kubernetes.config.load_incluster_config')
@mock.patch('kubernetes.client.CoreV1Api')
class BaseOpTest(unittest.TestCase):

    def test_execute_success(self, mock_k8s_client, mock_load_config):
        BaseOp().execute()

    @mock.patch.dict('os.environ', {
        'KFP_POD_NAME': 'mock-pod-id'
    })
    def test__exit_gracefully_cancel(self, mock_k8s_client, mock_load_config):
        mock_pod = mock_k8s_client().read_namespaced_pod.return_value
        mock_pod.metadata.annotations = {
            'workflows.argoproj.io/execution': '{"deadline": "1970-01-01T00:00:00Z"}'
        }

        op = BaseOp()
        op.on_cancelling = mock.MagicMock()

        op._exit_gracefully(0, 0)

        op.on_cancelling.assert_called_once()

    @mock.patch.dict('os.environ', {
        'KFP_POD_NAME': 'mock-pod-id'
    })
    def test__exit_gracefully_no_cancel(self, mock_k8s_client, mock_load_config):
        mock_pod = mock_k8s_client().read_namespaced_pod.return_value
        mock_pod.metadata.annotations = {}

        op = BaseOp()
        op.on_cancelling = mock.MagicMock()

        op._exit_gracefully(0, 0)

        op.on_cancelling.assert_not_called()