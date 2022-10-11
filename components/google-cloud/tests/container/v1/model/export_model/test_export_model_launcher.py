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

from google_cloud_pipeline_components.container.v1.model.export_model import launcher
from google_cloud_pipeline_components.container.v1.model.export_model import remote_runner

import unittest
from unittest import mock


class LauncherExportModelUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherExportModelUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'ExportModel', '--project', '', '--location',
        '', '--payload', 'test_payload', '--gcp_resources',
        self._gcp_resources, '--output_info', 'test_output_info'
    ]

  @mock.patch.object(
      remote_runner, 'export_model', autospec=True)
  def test_launcher_on_export_model_type(self, mock_export_model):
    launcher.main(self._input_args)
    mock_export_model.assert_called_once_with(
        type='ExportModel',
        project='',
        location='',
        payload='test_payload',
        gcp_resources=self._gcp_resources,
        output_info='test_output_info')
