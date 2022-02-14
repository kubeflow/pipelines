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
"""Test Vertex AI Model Undeploy Remote Runner module."""

import json
from logging import raiseExceptions
import os
import time
import unittest
from unittest import mock
from google_cloud_pipeline_components.container.utils.execution_context import ExecutionContext
from google_cloud_pipeline_components.container.v1.gcp_launcher import undeploy_model_remote_runner
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
from google.protobuf import json_format
from google.cloud.aiplatform.compat.types import endpoint
import requests
import google.auth
import google.auth.transport.requests


class LroResult(object):
  pass


class ModelUndeployRemoteRunnerUtilsTests(unittest.TestCase):

  def setUp(self):
    super(ModelUndeployRemoteRunnerUtilsTests, self).setUp()
    self._project = 'test_project'
    self._location = 'test_region'
    self._type = 'UndeployModel'
    self._lro_name = f'projects/{self._project}/locations/{self._location}/operations/123'
    self._model_id = 'm12'
    self._model_name = 'projects/test_project/locations/test_region/models/m12'
    self._payload = (
        '{"endpoint": '
        '"projects/test_project/locations/test_region/endpoints/e12","model": '
        '"projects/test_project/locations/test_region/models/m12",'
        '"traffic_split": {}'
        '}')
    self._gcp_resources_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'gcp_resources')
    self._uri_prefix = f'https://{self._location}-aiplatform.googleapis.com/v1/'

  def tearDown(self):
    super().tearDown()
    if os.path.exists(self._gcp_resources_path):
      os.remove(self._gcp_resources_path)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  def test_undeploy_model_remote_runner_succeeded(self, mock_post_requests,
                                                  mock_get_requests, _,
                                                  mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    get_endpoint_request = mock.Mock()
    get_endpoint_request.json.return_value = {
        'deployedModels': [{
            'id': self._model_id,
            'model': 'projects/test_project/locations/test_region/models/m12',
        }]
    }
    mock_get_requests.return_value = get_endpoint_request

    undeploy_model_request = mock.Mock()
    undeploy_model_request.json.return_value = {
        'name': self._lro_name,
        'done': True,
    }
    mock_post_requests.return_value = undeploy_model_request

    undeploy_model_remote_runner.undeploy_model(self._type, '', '',
                                                self._payload,
                                                self._gcp_resources_path)
    mock_get_requests.assert_called_once_with(
        url=f'{self._uri_prefix}projects/test_project/locations/test_region/endpoints/e12',
        data='',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
            'User-Agent': 'google-cloud-pipeline-components'
        })

    mock_post_requests.assert_called_once_with(
        url=f'{self._uri_prefix}projects/test_project/locations/test_region/endpoints/e12:undeployModel',
        data=json.dumps({
            'deployed_model_id': self._model_id,
        }),
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
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  def test_undeploy_model_remote_runner_raises_exception_on_error(
      self, mock_post_requests, mock_get_requests, _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    get_endpoint_request = mock.Mock()
    get_endpoint_request.json.return_value = {
        'deployedModels': [{
            'id': self._model_id,
            'model': 'projects/test_project/locations/test_region/models/m12',
        }]
    }
    mock_get_requests.return_value = get_endpoint_request

    undeploy_model_request = mock.Mock()
    undeploy_model_request.json.return_value = {
        'name': self._lro_name,
        'done': True,
        'error': {
            'code': 1
        }
    }
    mock_post_requests.return_value = undeploy_model_request

    with self.assertRaises(RuntimeError):
      undeploy_model_remote_runner.undeploy_model(self._type, '', '',
                                                  self._payload,
                                                  self._gcp_resources_path)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_undeploy_model_remote_runner_poll_till_succeeded(
      self, mock_time_sleep, mock_get_requests, mock_post_requests, _,
      mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    undeploy_model_lro = mock.Mock()
    undeploy_model_lro.json.return_value = {
        'name': self._lro_name,
        'done': False
    }
    mock_post_requests.return_value = undeploy_model_lro

    # There are 3 get requests here:
    # 1. Request to retrieve the endpoint.
    # 2. The LRO is not yet done.
    # 3. The LRO is done.
    get_requests = mock.Mock()
    get_requests.json.side_effect = [{
        'deployedModels': [{
            'id': self._model_id,
            'model': 'projects/test_project/locations/test_region/models/m12',
        }]
    }, {
        'name': self._lro_name,
        'done': False
    }, {
        'name': self._lro_name,
        'done': True
    }]
    mock_get_requests.return_value = get_requests

    undeploy_model_remote_runner.undeploy_model(self._type, '', '',
                                                self._payload,
                                                self._gcp_resources_path)
    self.assertEqual(mock_post_requests.call_count, 1)
    self.assertEqual(mock_time_sleep.call_count, 2)
    self.assertEqual(mock_get_requests.call_count, 3)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(ExecutionContext, '__init__', autospec=True)
  def test_undeploy_model_remote_runner_cancel(self, mock_execution_context,
                                               mock_post_requests,
                                               mock_get_requests, _,
                                               mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, "project"]
    get_endpoint_request = mock.Mock()
    get_endpoint_request.json.return_value = {
        'deployedModels': [{
            'id': self._model_id,
            'model': 'projects/test_project/locations/test_region/models/m12',
        }]
    }
    mock_get_requests.return_value = get_endpoint_request

    undeploy_model_request = mock.Mock()
    undeploy_model_request.json.return_value = {
        'name': self._lro_name,
        'done': True,
    }
    mock_post_requests.return_value = undeploy_model_request
    mock_execution_context.return_value = None

    undeploy_model_remote_runner.undeploy_model(self._type, '', '',
                                                self._payload,
                                                self._gcp_resources_path)

    # Call cancellation handler
    mock_execution_context.call_args[1]['on_cancel']()
    self.assertEqual(mock_post_requests.call_count, 2)
    mock_post_requests.assert_called_with(
        url=f'{self._uri_prefix}{self._lro_name}:cancel',
        data='',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
        })
