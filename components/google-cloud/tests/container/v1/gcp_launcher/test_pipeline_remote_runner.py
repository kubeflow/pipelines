# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the 'License');
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an 'AS IS' BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Tests for Vertex AI Pipeline Remote Runner."""

import os
import time
from absl.testing import parameterized

import google.auth
import google.auth.transport.requests
from google.cloud import aiplatform
from google.cloud.aiplatform_v1.types import pipeline_state
from google.cloud.aiplatform_v1.types import training_pipeline
from google_cloud_pipeline_components.container.v1.gcp_launcher import pipeline_remote_runner
from google_cloud_pipeline_components.proto import gcp_resources_pb2
import requests

from google.rpc import code_pb2
from google.rpc import status_pb2
from google.protobuf import json_format
import unittest
from unittest import mock


class PipelineRemoteRunnerTest(parameterized.TestCase):
  """Tests for Vertex AI Pipeline Remote Runner."""

  def setUp(self):
    super().setUp()
    self._payload = '{"spec": "test_spec"}'
    self._project = 'test_project'
    self._location = 'test_region'
    self._pipeline_name = f'projects/{self._project}/locations/{self._location}/trainingPipelines/123456'
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'gcp_resources'
    )
    self._type = 'TrainingPipeline'
    self._pipeline_uri_prefix = (
        f'https://{self._location}-aiplatform.googleapis.com/v1/'
    )
    self._mock_pipeline_client = self.enter_context(
        mock.patch.object(
            aiplatform.gapic, 'PipelineServiceClient', autospec=True
        )
    )
    self._mock_pipeline_client.return_value = mock.MagicMock()
    self._remote_runner = pipeline_remote_runner.PipelineRemoteRunner(
        self._type, self._project, self._location, self._gcp_resources
    )

  def tearDown(self):
    super().tearDown()
    if os.path.exists(self._gcp_resources):
      os.remove(self._gcp_resources)

  def test_check_if_pipeline_exists_not_exist(self):
    self.assertIsNone(self._remote_runner.check_if_pipeline_exists())

  def test_check_if_pipeline_exists_already_exists(self):
    gcp_resources = gcp_resources_pb2.GcpResources()
    gcp_resource = gcp_resources.resources.add()
    gcp_resource.resource_type = self._type
    gcp_resource.resource_uri = self._pipeline_uri_prefix + self._pipeline_name

    with open(self._gcp_resources, 'w') as f:
      f.write(json_format.MessageToJson(gcp_resources))

    self.assertEqual(
        self._remote_runner.check_if_pipeline_exists(), self._pipeline_name
    )

  def test_check_if_pipeline_exists_empty_gcs_resources(self):
    gcp_resources = gcp_resources_pb2.GcpResources()

    with open(self._gcp_resources, 'w') as f:
      f.write(json_format.MessageToJson(gcp_resources))

    with self.assertRaisesRegex(
        ValueError, 'gcp_resources should contain one resource'
    ):
      self._remote_runner.check_if_pipeline_exists()

  def test_check_if_pipeline_exists_invalid_gcs_resources(self):
    """Pipeline throws ValueError for invalid URIs in GCS resources."""
    gcp_resources = gcp_resources_pb2.GcpResources()
    gcp_resource = gcp_resources.resources.add()
    gcp_resource.resource_type = self._type
    gcp_resource.resource_uri = 'invalid_uri'

    with open(self._gcp_resources, 'w') as f:
      f.write(json_format.MessageToJson(gcp_resources))

    with self.assertRaisesRegex(
        ValueError,
        'Pipeline Name in gcp_resource is not formatted correctly or is empty.',
    ):
      self._remote_runner.check_if_pipeline_exists()

  def test_create_pipeline(self):
    """Tests pipeline is created and GCP resources are correctly written."""
    mock_fn = mock.MagicMock()
    mock_fn.return_value = training_pipeline.TrainingPipeline(
        name=self._pipeline_name
    )
    self.assertEqual(
        self._remote_runner.create_pipeline(mock_fn, self._payload),
        self._pipeline_name,
    )

    with open(self._gcp_resources, 'r') as f:
      gcp_resources = json_format.Parse(
          f.read(), gcp_resources_pb2.GcpResources()
      )

    self.assertLen(gcp_resources.resources, 1)
    gcp_resource = gcp_resources.resources[0]
    self.assertEqual(gcp_resource.resource_type, self._type)
    self.assertEqual(
        gcp_resource.resource_uri,
        self._pipeline_uri_prefix + self._pipeline_name,
    )

  def test_poll_pipeline_success(self):
    """Successfully polls pipeline data."""
    mock_fn = mock.MagicMock()
    mock_fn.return_value = training_pipeline.TrainingPipeline(
        name=self._pipeline_name,
        state=pipeline_state.PipelineState.PIPELINE_STATE_SUCCEEDED,
    )
    self.assertEqual(
        self._remote_runner.poll_pipeline(mock_fn, self._pipeline_name),
        mock_fn.return_value,
    )
    mock_fn.assert_called_once_with(
        self._mock_pipeline_client.return_value, self._pipeline_name
    )

  @mock.patch.object(time, 'sleep')
  def test_poll_pipeline_connection_error(self, mock_sleep):
    """Verifies that pipeline retries when receiving a ConnectionError."""
    mock_fn = mock.MagicMock()
    pipeline_response = training_pipeline.TrainingPipeline(
        name=self._pipeline_name,
        state=pipeline_state.PipelineState.PIPELINE_STATE_SUCCEEDED,
    )
    mock_fn.side_effect = [ConnectionError(), pipeline_response]
    self.assertEqual(
        self._remote_runner.poll_pipeline(mock_fn, self._pipeline_name),
        pipeline_response,
    )
    self.assertEqual(mock_fn.call_count, 2)
    mock_sleep.assert_called_once()

  @parameterized.named_parameters(
      {
          'testcase_name': 'user_error',
          'code': code_pb2.INVALID_ARGUMENT,
          'expected_error': ValueError,
          'expected_msg': 'Pipeline failed with value error in error state',
      },
      {
          'testcase_name': 'unknown_error',
          'code': code_pb2.UNKNOWN,
          'expected_error': RuntimeError,
          'expected_msg': 'Pipeline failed with error state',
      },
  )
  def test_poll_pipeline(self, code: int, expected_error, expected_msg: str):
    """Pipeline terminated with error."""
    mock_fn = mock.MagicMock()
    mock_fn.return_value = training_pipeline.TrainingPipeline(
        name=self._pipeline_name,
        state=pipeline_state.PipelineState.PIPELINE_STATE_FAILED,
        error=status_pb2.Status(code=code),  # INVALID_ARGUMENT
    )
    with self.assertRaisesRegex(expected_error, expected_msg):
      self._remote_runner.poll_pipeline(mock_fn, self._pipeline_name)

  @parameterized.named_parameters(
      {
          'testcase_name': 'no_refresh',
          'creds_valid': True,
          'expect_refresh': False,
      },
      {
          'testcase_name': 'with_refresh',
          'creds_valid': False,
          'expect_refresh': True,
      },
  )
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(google.auth, 'default', autospec=True)
  def test_send_cancel_request(
      self,
      mock_auth,
      mock_requests_post,
      creds_valid: bool,
      expect_refresh: bool,
  ):
    """Verifies the cancellation request to REST API."""
    mock_creds = mock.MagicMock()
    mock_creds.valid = creds_valid
    mock_creds.token = 'test_token'
    mock_auth.return_value = (mock_creds, 'dummy')

    self._remote_runner.send_cancel_request(self._pipeline_name)

    if expect_refresh:
      mock_creds.refresh.assert_called_once()
    else:
      mock_creds.refresh.assert_not_called()
    mock_auth.assert_called_once_with(
        scopes=['https://www.googleapis.com/auth/cloud-platform'],
    )
    mock_requests_post.assert_called_once_with(
        url=f'{self._pipeline_uri_prefix}{self._pipeline_name}:cancel',
        data='',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer test_token',
        },
    )


if __name__ == '__main__':
  unittest.main()
