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
"""Test Vertex AI Batch Prediction Job Launcher Client module."""

import os

from google_cloud_pipeline_components.container.v1.endpoint.create_endpoint import launcher
from google_cloud_pipeline_components.container.v1.endpoint.create_endpoint import remote_runner

import unittest
from unittest import mock


class LauncherCreateEndpointUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherCreateEndpointUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'CreateEndpoint', '--project', 'test_project', '--location',
        'us_central1', '--payload', 'test_payload', '--gcp_resources',
        self._gcp_resources, '--executor_input', 'executor_input'
    ]

  @mock.patch.object(
      remote_runner, 'create_endpoint', autospec=True)
  def test_launcher_on_create_endpoint_type(self, mock_create_endpoint):
    launcher.main(self._input_args)
    mock_create_endpoint.assert_called_once_with(
        type='CreateEndpoint',
        project='test_project',
        location='us_central1',
        payload='test_payload',
        gcp_resources=self._gcp_resources,
        executor_input='executor_input')
