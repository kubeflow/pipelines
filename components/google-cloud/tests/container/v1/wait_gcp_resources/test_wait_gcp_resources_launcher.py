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
"""Test Vertex AI Wait GCP Resources Launcher Client module."""

import os

from google_cloud_pipeline_components.container.v1.wait_gcp_resources import launcher
from google_cloud_pipeline_components.container.v1.wait_gcp_resources import remote_runner

import unittest
from unittest import mock


class LauncherWaitGCPResourcesUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherWaitGCPResourcesUtilsTests, self).setUp()
    self._project = 'test_project'
    self._location = 'test_region'
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')

  @mock.patch.object(
      remote_runner, 'wait_gcp_resources', autospec=True)
  def test_launcher_on_wait_type(self, mock_wait_gcp_resources):
    job_type = 'Wait'
    payload = 'test_payload'
    input_args = [
        '--type', job_type, '--project', self._project, '--location',
        self._location, '--payload', payload, '--gcp_resources',
        self._gcp_resources
    ]
    launcher.main(input_args)
    mock_wait_gcp_resources.assert_called_once_with(
        type=job_type,
        project=self._project,
        location=self._location,
        payload=payload,
        gcp_resources=self._gcp_resources)