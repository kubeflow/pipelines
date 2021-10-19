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
"""Test Vertex AI Dataflow Launcher Client module."""

import os

from google_cloud_pipeline_components.container.experimental.dataflow import dataflow_launcher
from google_cloud_pipeline_components.container.experimental.dataflow import dataflow_python_job_remote_runner

import unittest
from google3.testing.pybase.googletest import mock


class DataflowLauncherUtilsTests(unittest.TestCase):

  def setUp(self):
    super(DataflowLauncherUtilsTests, self).setUp()
    self._project = 'test_project'
    self._location = 'test_region'
    self._python_module_path = 'test_python_module_path'
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._test_bucket_name = 'test_bucket_name'
    self._test_blob_path = 'test_blob_path'
    self._gcs_temp_path = f'gs://{self._test_bucket_name}/{self._test_blob_path}'
    self._requirement_file_path = f'gs://{self._test_bucket_name}/requirements.txt'
    self._args = 'test_arg'
    self._test_file_path = '/tmp/test_filepath/test_file.txt'

  def tearDown(self):
    super(DataflowLauncherUtilsTests, self).tearDown()
    if os.path.exists(self._gcp_resources):
      os.remove(self._gcp_resources)
    if os.path.exists(os.path.dirname(self._test_file_path)):
      os.rmdir(os.path.dirname(self._test_file_path))

  @mock.patch.object(
      dataflow_python_job_remote_runner, 'create_python_job', autospec=True)
  def test_launcher_on_python_job_calls_custom_job_remote_runner(
      self, mock_dataflow_remote_runner):
    input_args = [
        '--python_module_path', self._python_module_path, '--project',
        self._project, '--location', self._location, '--gcp_resources',
        self._gcp_resources, '--args', self._args, '--requirements_file_path',
        self._requirement_file_path, '--temp_location', self._gcs_temp_path
    ]
    dataflow_launcher.main(input_args)

    mock_dataflow_remote_runner.assert_called_once_with(
        python_module_path=self._python_module_path,
        project=self._project,
        gcp_resources=self._gcp_resources,
        location=self._location,
        temp_location=self._gcs_temp_path,
        requirements_file_path=self._requirement_file_path,
        args=self._args)

  def test_make_parent_dirs_and_return_path_creates_the_path(self):
    if os.path.exists(os.path.dirname(self._test_file_path)):
      os.rmdir(os.path.dirname(self._test_file_path))
    self.assertFalse(os.path.exists(os.path.dirname(self._test_file_path)))
    dataflow_launcher._make_parent_dirs_and_return_path(self._test_file_path)
    self.assertTrue(os.path.exists(os.path.dirname(self._test_file_path)))
