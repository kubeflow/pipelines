# Copyright 2022 The Kubeflow Authors. All Rights Reserved.
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
"""Test Vertex AI Delete Endpoint Remote Runner module."""

import json
from logging import raiseExceptions
import os
import time
import unittest
from unittest import mock
from google_cloud_pipeline_components.container.utils.execution_context import ExecutionContext
from google_cloud_pipeline_components.container.v1.gcp_launcher import delete_endpoint_remote_runner
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
from google.protobuf import json_format
import requests
import google.auth
import google.auth.transport.requests


class LroResult(object):
    pass


class DeleteEndpointRemoteRunnerUtilsTests(unittest.TestCase):

    def setUp(self):
        super(DeleteEndpointRemoteRunnerUtilsTests, self).setUp()
        self._project = 'test_project'
        self._location = 'test_region'
        self._payload = '{"endpoint": "projects/test_project/locations/test_region/endpoints/e12"}'
        self._type = 'DeleteEndpoint'
        self._lro_name = f'projects/{self._project}/locations/{self._location}/operations/123'
        self._gcp_resources_path = os.path.join(os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), "gcp_resources")
        self._uri_prefix = f"https://{self._location}-aiplatform.googleapis.com/v1/"

    def tearDown(self):
        if os.path.exists(self._gcp_resources_path):
            os.remove(self._gcp_resources_path)

    @mock.patch.object(google.auth, 'default', autospec=True)
    @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
    @mock.patch.object(requests, 'delete', autospec=True)
    def test_delete_endpoint_remote_runner_succeeded(self, mock_delete_requests, _,
                                                  mock_auth):
        creds = mock.Mock()
        creds.token = 'fake_token'
        mock_auth.return_value = [creds, "project"]
        delete_endpoint_lro = mock.Mock()
        delete_endpoint_lro.json.return_value = {
            'name': self._lro_name,
            'done': True,
        }
        mock_delete_requests.return_value = delete_endpoint_lro

        delete_endpoint_remote_runner.delete_endpoint(self._type, '', '', self._payload,
                                                self._gcp_resources_path)
        mock_delete_requests.assert_called_once_with(
            url=f'{self._uri_prefix}projects/test_project/locations/test_region/endpoints/e12',
            data="",
            headers={
                'Content-type': 'application/json',
                'Authorization': 'Bearer fake_token',
                'User-Agent': 'google-cloud-pipeline-components'
            })

        with open(self._gcp_resources_path) as f:
            serialized_gcp_resources = f.read()
            # Instantiate GCPResources Proto
            lro_resources = json_format.Parse(serialized_gcp_resources,
                                              GcpResources())

            self.assertEqual(len(lro_resources.resources), 1)
            self.assertEqual(lro_resources.resources[0].resource_uri,
                             self._uri_prefix + self._lro_name)

    @mock.patch.object(google.auth, 'default', autospec=True)
    @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
    @mock.patch.object(requests, 'delete', autospec=True)
    def test_delete_endpoint_remote_runner_raises_exception_on_error(
            self, mock_delete_requests, _, mock_auth):
        creds = mock.Mock()
        creds.token = 'fake_token'
        mock_auth.return_value = [creds, "project"]
        delete_endpoint_lro = mock.Mock()
        delete_endpoint_lro.json.return_value = {
            'name': self._lro_name,
            'done': True,
            'error': {
                'code': 1
            }
        }
        mock_delete_requests.return_value = delete_endpoint_lro

        with self.assertRaises(RuntimeError):
            delete_endpoint_remote_runner.delete_endpoint(self._type, '', '',
                                                    self._payload,
                                                    self._gcp_resources_path)

    @mock.patch.object(google.auth, 'default', autospec=True)
    @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
    @mock.patch.object(requests, 'delete', autospec=True)
    @mock.patch.object(requests, 'get', autospec=True)
    @mock.patch.object(time, "sleep", autospec=True)
    def test_delete_endpoint_remote_runner_poll_till_succeeded(
            self, mock_time_sleep, mock_get_requests, mock_delete_requests, _,
            mock_auth):
        creds = mock.Mock()
        creds.token = 'fake_token'
        mock_auth.return_value = [creds, "project"]
        delete_endpoint_lro = mock.Mock()
        delete_endpoint_lro.json.return_value = {
            'name': self._lro_name,
            'done': False
        }
        mock_delete_requests.return_value = delete_endpoint_lro

        poll_lro = mock.Mock()
        poll_lro.json.side_effect = [{
            'name': self._lro_name,
            'done': False
        }, {
            'name': self._lro_name,
            'done': True
        }]
        mock_get_requests.return_value = poll_lro

        delete_endpoint_remote_runner.delete_endpoint(self._type, '', '',
                                                self._payload,
                                                self._gcp_resources_path)
        self.assertEqual(mock_delete_requests.call_count, 1)
        self.assertEqual(mock_time_sleep.call_count, 2)
        self.assertEqual(mock_get_requests.call_count, 2)


    @mock.patch.object(google.auth, 'default', autospec=True)
    @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
    @mock.patch.object(requests, 'delete', autospec=True)
    @mock.patch.object(requests, 'post', autospec=True)
    @mock.patch.object(ExecutionContext, '__init__', autospec=True)
    def test_delete_endpoint_remote_runner_cancel(self, mock_execution_context,
                                                  mock_post_requests,
                                                  mock_delete_requests, _,
                                                  mock_auth):
        creds = mock.Mock()
        creds.token = 'fake_token'
        mock_auth.return_value = [creds, "project"]
        delete_endpoint_lro = mock.Mock()
        delete_endpoint_lro.json.return_value = {
            'name': self._lro_name,
            'done': True,
        }
        mock_delete_requests.return_value = delete_endpoint_lro
        mock_execution_context.return_value = None

        delete_endpoint_remote_runner.delete_endpoint(
            self._type, '', '', self._payload, self._gcp_resources_path)

        # Call cancellation handler
        mock_execution_context.call_args[1]['on_cancel']()
        mock_post_requests.assert_called_once_with(
            url=f'{self._uri_prefix}{self._lro_name}:cancel',
            data='',
            headers={
                'Content-type': 'application/json',
                'Authorization': 'Bearer fake_token',
            })
