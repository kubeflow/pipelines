# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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
"""Test Vertex AI Model Upload Remote Runner module."""

import json
from logging import raiseExceptions
import os
import time
import unittest
from unittest import mock
from google_cloud_pipeline_components.container.utils.execution_context import ExecutionContext
from google_cloud_pipeline_components.container.v1.gcp_launcher import upload_model_remote_runner
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
from google.protobuf import json_format
import requests
import google.auth
import google.auth.transport.requests


class LroResult(object):
  pass


class ModelUploadRemoteRunnerUtilsTests(unittest.TestCase):

  def setUp(self):
    super(ModelUploadRemoteRunnerUtilsTests, self).setUp()
    self._project = 'test_project'
    self._location = 'test_region'
    self._payload = '{"display_name": "model1"}'
    self._type = 'UploadModel'
    self._lro_name = f'projects/{self._project}/locations/{self._location}/operations/123'
    self._model_name = f'projects/{self._project}/locations/{self._location}/models/123'
    self._output_file_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'localpath/foo')
    self._executor_input = '{"outputs":{"artifacts":{"model":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.VertexModel"},"uri":"gs://abc"}]}},"outputFile":"' + self._output_file_path + '"}}'
    self._gcp_resources_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'gcp_resources')
    self._uri_prefix = f'https://{self._location}-aiplatform.googleapis.com/v1/'

  def tearDown(self):
    if os.path.exists(self._gcp_resources_path):
      os.remove(self._gcp_resources_path)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  def test_model_upload_remote_runner_succeeded(self, mock_post_requests, _,
                                                mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    upload_model_lro = mock.Mock()
    upload_model_lro.json.return_value = {
        'name': self._lro_name,
        'done': True,
        'response': {
            'model': self._model_name
        }
    }
    mock_post_requests.return_value = upload_model_lro

    upload_model_remote_runner.upload_model(self._type, self._project,
                                            self._location, self._payload,
                                            self._gcp_resources_path,
                                            self._executor_input)
    mock_post_requests.assert_called_once_with(
        url=f'{self._uri_prefix}projects/test_project/locations/test_region/models:upload',
        data='{"model": {"display_name": "model1"}}',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
            'User-Agent': 'google-cloud-pipeline-components'
        })

    with open(self._output_file_path) as f:
      executor_output = json.load(f, strict=False)
      self.assertEqual(
          executor_output,
          json.loads(
              '{"artifacts": {"model": {"artifacts": [{"metadata": {"resourceName": "projects/test_project/locations/test_region/models/123"}, "name": "foobar", "type": {"schemaTitle": "google.VertexModel"}, "uri": "https://test_region-aiplatform.googleapis.com/v1/projects/test_project/locations/test_region/models/123"}]}}}'
          ))

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
  @mock.patch.object(requests, 'post', autospec=True)
  def test_model_upload_remote_runner_append_unmanaged_model_succeeded(
      self, mock_post_requests, _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    upload_model_lro = mock.Mock()
    upload_model_lro.json.return_value = {
        'name': self._lro_name,
        'done': True,
        'response': {
            'model': self._model_name
        }
    }
    mock_post_requests.return_value = upload_model_lro

    self._executor_input = (
        '{"inputs":{"artifacts":{"unmanaged_container_model":{"artifacts":[{"metadata":{"predictSchemata":{"instanceSchemaUri":"instance_a"},'
        ' '
        '"containerSpec":{"imageUri":"image_foo"}},"name":"unmanaged_container_model","type":{"schemaTitle":"google.UnmanagedContainerModel"},"uri":"gs://abc"}]}}},"outputs":{"artifacts":{"model":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.VertexModel"},"uri":"gs://abc"}]}},"outputFile":"'
    ) + self._output_file_path + '"}}'
    upload_model_remote_runner.upload_model(self._type, self._project,
                                            self._location, self._payload,
                                            self._gcp_resources_path,
                                            self._executor_input)
    mock_post_requests.assert_called_once_with(
        url=f'{self._uri_prefix}projects/test_project/locations/test_region/models:upload',
        data='{"model": {"display_name": "model1", "predict_schemata": {"instance_schema_uri": "instance_a"}, "container_spec": {"image_uri": "image_foo"}, "artifact_uri": "gs://abc"}}',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
            'User-Agent': 'google-cloud-pipeline-components'
        })

    with open(self._output_file_path) as f:
      executor_output = json.load(f, strict=False)
      self.assertEqual(
          executor_output,
          json.loads(
              '{"artifacts": {"model": {"artifacts": [{"metadata": {"resourceName": "projects/test_project/locations/test_region/models/123"}, "name": "foobar", "type": {"schemaTitle": "google.VertexModel"}, "uri": "https://test_region-aiplatform.googleapis.com/v1/projects/test_project/locations/test_region/models/123"}]}}}'
          ))

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
  @mock.patch.object(requests, 'post', autospec=True)
  def test_upload_model_remote_runner_raises_exception_on_error(
      self, mock_post_requests, _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    upload_model_lro = mock.Mock()
    upload_model_lro.json.return_value = {
        'name': self._lro_name,
        'done': True,
        'error': {
            'code': 1
        }
    }
    mock_post_requests.return_value = upload_model_lro

    with self.assertRaises(RuntimeError):
      upload_model_remote_runner.upload_model(self._type, self._project,
                                              self._location, self._payload,
                                              self._gcp_resources_path,
                                              self._executor_input)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_upload_model_remote_runner_poll_till_succeeded(
      self, mock_time_sleep, mock_get_requests, mock_post_requests, _,
      mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    upload_model_lro = mock.Mock()
    upload_model_lro.json.return_value = {'name': self._lro_name, 'done': False}
    mock_post_requests.return_value = upload_model_lro

    poll_lro = mock.Mock()
    poll_lro.json.side_effect = [{
        'name': self._lro_name,
        'done': False
    }, {
        'name': self._lro_name,
        'done': True,
        'response': {
            'model': self._model_name
        }
    }]
    mock_get_requests.return_value = poll_lro

    upload_model_remote_runner.upload_model(self._type, self._project,
                                            self._location, self._payload,
                                            self._gcp_resources_path,
                                            self._executor_input)
    self.assertEqual(mock_post_requests.call_count, 1)
    self.assertEqual(mock_time_sleep.call_count, 2)
    self.assertEqual(mock_get_requests.call_count, 2)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(ExecutionContext, '__init__', autospec=True)
  def test_model_upload_remote_runner_cancel(self, mock_execution_context,
                                             mock_post_requests,
                                             _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    upload_model_lro = mock.Mock()
    upload_model_lro.json.return_value = {
        'name': self._lro_name,
        'done': True,
        'response': {
            'model': self._model_name
        }
    }
    mock_post_requests.return_value = upload_model_lro
    mock_execution_context.return_value = None

    upload_model_remote_runner.upload_model(self._type, self._project,
                                            self._location, self._payload,
                                            self._gcp_resources_path,
                                            self._executor_input)
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
