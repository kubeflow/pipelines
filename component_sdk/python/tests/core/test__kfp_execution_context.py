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

from kfp_component.core import KfpExecutionContext

from kubernetes import client, config
from kubernetes.client.rest import ApiException

import mock
import unittest

@mock.patch('kubernetes.config.load_incluster_config')
@mock.patch('kubernetes.client.CoreV1Api')
class KfpExecutionContextTest(unittest.TestCase):

    def test_init_succeed_without_pod_name(self, 
        mock_k8s_client, mock_load_config):
        with KfpExecutionContext() as ctx:
            self.assertFalse(ctx.under_kfp_environment())
            pass

    @mock.patch.dict('os.environ', {
        'KFP_POD_NAME': 'mock-pod-id'
    })
    def test_init_succeed_when_load_k8s_config_fail(self, 
        mock_k8s_client, mock_load_config):
        mock_load_config.side_effect = Exception()

        with KfpExecutionContext() as ctx:
            self.assertFalse(ctx.under_kfp_environment())
            pass

    @mock.patch.dict('os.environ', {
        'KFP_POD_NAME': 'mock-pod-id'
    })
    def test_init_succeed_when_load_k8s_client_fail(self, 
        mock_k8s_client, mock_load_config):
        mock_k8s_client.side_effect = Exception()

        with KfpExecutionContext() as ctx:
            self.assertFalse(ctx.under_kfp_environment())
            pass

    @mock.patch.dict('os.environ', {
        'KFP_POD_NAME': 'mock-pod-id'
    })
    def test_init_succeed_when_load_pod_fail(self, 
        mock_k8s_client, mock_load_config):
        mock_k8s_client().read_namespaced_pod.side_effect = Exception()

        with KfpExecutionContext() as ctx:
            self.assertFalse(ctx.under_kfp_environment())
            pass

    @mock.patch.dict('os.environ', {
        'KFP_POD_NAME': 'mock-pod-id'
    })
    def test_init_succeed_no_argo_node_name(self, 
        mock_k8s_client, mock_load_config):
        mock_pod = mock_k8s_client().read_namespaced_pod.return_value
        mock_pod.metadata.annotations = {}
        with KfpExecutionContext() as ctx:
            self.assertFalse(ctx.under_kfp_environment())
            pass

    @mock.patch.dict('os.environ', {
        'KFP_POD_NAME': 'mock-pod-id',
        'KFP_NAMESPACE': 'mock-namespace'
    })
    def test_init_succeed(self, 
        mock_k8s_client, mock_load_config):
        mock_pod = mock_k8s_client().read_namespaced_pod.return_value
        mock_pod.metadata.annotations = {
            'workflows.argoproj.io/node-name': 'node-1'
        }
        with KfpExecutionContext() as ctx:
            self.assertTrue(ctx.under_kfp_environment())
            pass
        mock_k8s_client().read_namespaced_pod.assert_called_with('mock-pod-id', 'mock-namespace')

    @mock.patch.dict('os.environ', {
        'KFP_POD_NAME': 'mock-pod-id'
    })
    def test__exit_gracefully_cancel(self, 
        mock_k8s_client, mock_load_config):
        mock_pod = mock_k8s_client().read_namespaced_pod.return_value
        mock_pod.metadata.annotations = {
            'workflows.argoproj.io/node-name': 'node-1',
            'workflows.argoproj.io/execution': '{"deadline": "1970-01-01T00:00:00Z"}'
        }
        cancel_handler = mock.Mock()
        context = KfpExecutionContext(on_cancel=cancel_handler)

        context._exit_gracefully(0, 0)

        cancel_handler.assert_called_once()

    @mock.patch.dict('os.environ', {
        'KFP_POD_NAME': 'mock-pod-id'
    })
    def test__exit_gracefully_no_cancel(self, 
        mock_k8s_client, mock_load_config):
        mock_pod = mock_k8s_client().read_namespaced_pod.return_value
        mock_pod.metadata.annotations = {
            'workflows.argoproj.io/node-name': 'node-1'
        }
        cancel_handler = mock.Mock()
        context = KfpExecutionContext(on_cancel=cancel_handler)

        context._exit_gracefully(0, 0)

        cancel_handler.assert_not_called()

    @mock.patch.dict('os.environ', {
        'KFP_POD_NAME': 'mock-pod-id'
    })
    def test_context_id_stable_across_retries(self, 
        mock_k8s_client, mock_load_config):
        mock_pod = mock_k8s_client().read_namespaced_pod.return_value
        mock_pod.metadata.annotations = {
            'workflows.argoproj.io/node-name': 'node-1'
        }
        ctx1 = KfpExecutionContext()
        ctx2 = KfpExecutionContext()

        self.assertEqual(ctx1.context_id(), ctx2.context_id())