# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
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
"""Test Vertex AI Get Model Launcher Client module."""

import os

import google.auth
from google.cloud import aiplatform
from google_cloud_pipeline_components.container.experimental.model.get_model import get_model
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import artifact_util
from google_cloud_pipeline_components.proto import gcp_resources_pb2

from google.protobuf import json_format
import unittest
from unittest import mock


MODEL_NAME = (
    'projects/test-project/locations/test-location/models/test-model-name'
)


def mock_api_call(test_func):
  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(
      aiplatform.gapic.ModelServiceClient,
      'get_model',
      autospec=True,
  )
  @mock.patch.object(artifact_util, 'update_output_artifacts', autospec=True)
  def mocked_test(self, mock_update_artifacts, mock_get_model, mock_auth):
    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = 'token'
    mock_auth.return_value = [mock_creds, 'project']

    mock_update_artifacts.return_value = None

    test_func(self, mock_get_model)

  return mocked_test


class GetModelTests(unittest.TestCase):

  def setUp(self):
    super(GetModelTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'test_file_path/test_file.txt'
    )
    self._input_args = [
        '--gcp_resources',
        self._gcp_resources,
        '--executor_input',
        'executor_input',
    ]

  @mock_api_call
  def test_main_model_name(self, mock_api):
    self._input_args.extend((
        '--model_name',
        MODEL_NAME,
    ))
    get_model_response = mock.Mock()
    get_model_response.name = MODEL_NAME
    mock_api.return_value = get_model_response

    get_model.main(self._input_args)

    mock_api.assert_called_once_with(
        mock.ANY,
        name=MODEL_NAME,
    )

  @mock_api_call
  def test_main_model_name_with_version(self, mock_api):
    self._input_args.extend((
        '--model_name',
        f'{MODEL_NAME}@2',
    ))
    get_model_response = mock.Mock()
    get_model_response.name = MODEL_NAME
    get_model_response.version_id = '2'
    mock_api.return_value = get_model_response

    get_model.main(self._input_args)

    mock_api.assert_called_once_with(
        mock.ANY,
        name=f'{MODEL_NAME}@2',
    )

  @mock_api_call
  def test_main_model_name_and_version(self, mock_api):
    self._input_args.extend((
        '--model_name',
        MODEL_NAME,
        '--model_version',
        '3',
    ))
    get_model_response = mock.Mock()
    get_model_response.name = MODEL_NAME
    get_model_response.version_id = '3'
    mock_api.return_value = get_model_response

    get_model.main(self._input_args)

    mock_api.assert_called_once_with(
        mock.ANY,
        name=f'{MODEL_NAME}@3',
    )

  @mock_api_call
  def test_main_model_name_and_version_override(self, mock_api):
    self._input_args.extend((
        '--model_name',
        f'{MODEL_NAME}@2',
        '--model_version',
        '3',
    ))
    get_model_response = mock.Mock()
    get_model_response.name = MODEL_NAME
    get_model_response.version_id = '3'
    mock_api.return_value = get_model_response

    get_model.main(self._input_args)

    mock_api.assert_called_once_with(
        mock.ANY,
        name=f'{MODEL_NAME}@3',
    )

  def test_main_argparse_raises(self):
    with self.assertRaises(SystemExit):
      get_model.main(self._input_args)

  def test_launcher_argparse_raises_model_version(self):
    self._input_args.extend((
        '--model_version',
        '3',
    ))
    with self.assertRaises(SystemExit):
      get_model.main(self._input_args)

  @mock_api_call
  def test_main_model_name_in_model_garden(self, _):
    self._input_args.extend((
        '--model_name',
        f'projects/test-project/locations/test-location/publishers/test-publisher/models/test-model-name',
    ))

    get_model.main(self._input_args)

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()

      # Instantiate GCPResources Proto
      model_resources = json_format.Parse(
          serialized_gcp_resources, gcp_resources_pb2.GcpResources()
      )

      self.assertLen(model_resources.resources, 1)
      model_name = model_resources.resources[0].resource_uri[
          len('https://test-location-aiplatform.googleapis.com/v1/') :
      ]
      self.assertEqual(
          model_name,
          f'projects/test-project/locations/test-location/publishers/test-publisher/models/test-model-name',
      )

  def test_main_argparse_raises(self):
    with self.assertRaises(SystemExit):
      get_model.main(self._input_args)

  def test_launcher_argparse_raises_model_version(self):
    self._input_args.extend((
        '--model_version',
        '3',
    ))
    with self.assertRaises(SystemExit):
      get_model.main(self._input_args)

  @mock_api_call
  def test_get_model_gcp_resources(self, mock_api):
    get_model_response = mock.Mock()
    get_model_response.name = MODEL_NAME
    get_model_response.version_id = '1'
    mock_api.return_value = get_model_response

    self._input_args.extend((
        '--model_name',
        f'{MODEL_NAME}@1',
    ))

    get_model.main(self._input_args)

    mock_api.assert_called_once_with(
        mock.ANY,
        name=f'{MODEL_NAME}@1',
    )

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()

      # Instantiate GCPResources Proto
      model_resources = json_format.Parse(
          serialized_gcp_resources, gcp_resources_pb2.GcpResources()
      )

      self.assertLen(model_resources.resources, 1)
      model_name = model_resources.resources[0].resource_uri[
          len('https://test-location-aiplatform.googleapis.com/v1/') :
      ]
      self.assertEqual(
          model_name,
          f'{MODEL_NAME}@1',
      )
