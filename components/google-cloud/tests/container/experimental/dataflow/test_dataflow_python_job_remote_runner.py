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
"""Test Dataflow python Job Remote Runner client module."""

import json
import os
import subprocess
from unittest import mock
from google.cloud import storage
from google_cloud_pipeline_components.container.experimental.dataflow import dataflow_python_job_remote_runner
import unittest


class DataflowPythonJobRemoteRunnerUtilsTests(unittest.TestCase):

  def setUp(self):
    super(DataflowPythonJobRemoteRunnerUtilsTests, self).setUp()
    self._job_type = 'DataflowJob'
    self._project = 'test_project'
    self._location = 'test_region'
    self._test_bucket_name = 'test_bucket_name'
    self._test_blob_path = 'test_blob_path'
    self._gcs_temp_path = f'gs://{self._test_bucket_name}/{self._test_blob_path}'
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'gcp_resources')
    self._local_file_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'local_file')
    self._requirement_file_path = f'gs://{self._test_bucket_name}/requirements.txt'
    self._job_id = 'test_job_id'
    self._args = ['test_arg']
    self._setup_file_path = 'gs://{self._test_bucket_name}/setup.py'

  def tearDown(self):
    super(DataflowPythonJobRemoteRunnerUtilsTests, self).tearDown()
    if os.path.exists(self._gcp_resources):
      os.remove(self._gcp_resources)

  def test_parse_blob_path_returns_bucket_name_and_blob_name(self):
    result = dataflow_python_job_remote_runner.parse_blob_path(
        self._gcs_temp_path)
    self.assertEqual(result, (self._test_bucket_name, self._test_blob_path))

  def test_parse_blob_path_raises_value_error_for_invalid_path(self):
    with self.assertRaises(ValueError):
      dataflow_python_job_remote_runner.parse_blob_path(
          'not_gs://test_invalid_path')

  def test_process_returns_stdout_as_lines(self):
    process = dataflow_python_job_remote_runner.Process(['echo', 'test'])
    for line in process.read_lines():
      self.assertEqual(line, b'test\n')

  def test_process_raises_error_on_failed_command(self):
    with self.assertRaises(FileNotFoundError):
      dataflow_python_job_remote_runner.Process(['failed_command'])

  @mock.patch.object(storage, 'Client', autospec=True)
  def test_download_blob_writes_output_correctly(self, mock_client):
    mock_storage_client = mock.Mock()
    mock_client.return_value = mock_storage_client
    mock_bucket = mock.Mock()
    mock_storage_client.bucket.return_value = mock_bucket
    mock_blob = mock.Mock()
    mock_bucket.blob.return_value = mock_blob
    dataflow_python_job_remote_runner.download_blob(self._gcs_temp_path,
                                                    self._local_file_path)
    mock_storage_client.bucket.assert_called_once_with(self._test_bucket_name)
    mock_bucket.blob.assert_called_once_with(self._test_blob_path)
    mock_blob.download_to_file.assert_called_once()

  @mock.patch.object(
      dataflow_python_job_remote_runner, 'download_blob', autospec=True)
  def test_stage_file_writes_pulls_correct_file(self, mock_download_blob):
    dataflow_python_job_remote_runner.stage_file(self._gcs_temp_path)
    mock_download_blob.assert_called_once_with(self._gcs_temp_path, mock.ANY)

  @mock.patch.object(
      dataflow_python_job_remote_runner, 'stage_file', autospec=True)
  @mock.patch.object(subprocess, 'check_call', autospec=True)
  def test_install_requirements_calls_correct_cmd(self, mock_sub_process,
                                                  mock_stage_file):
    mock_stage_file.return_value = self._local_file_path
    dataflow_python_job_remote_runner.install_requirements(self._gcs_temp_path)
    mock_stage_file.assert_called_with(self._gcs_temp_path)
    mock_sub_process.assert_called_with(
        ['pip', 'install', '-r', self._local_file_path])

  def test_extract_job_id_and_location_returns_values_correctly_when_found(
      self):
    job_id, location = dataflow_python_job_remote_runner.extract_job_id_and_location(
        f'console.cloud.google.com/dataflow/jobs/{self._location}/{self._job_id}'
        .encode('utf-8'))
    self.assertEqual(job_id, self._job_id)
    self.assertEqual(location, self._location)

  def test_extract_job_id_and_location_returns_none_when_not_found(self):
    job_id, location = dataflow_python_job_remote_runner.extract_job_id_and_location(
        b'test_invalid_string')
    self.assertIsNone(job_id)
    self.assertIsNone(location)

  def test_prepare_cmd_returns_correct_command_values(self):
    command = dataflow_python_job_remote_runner.prepare_cmd(
        project_id=self._project,
        region=self._location,
        python_file_path=self._local_file_path,
        args=self._args,
        temp_location=self._gcs_temp_path)
    expected_results = [
        'python3', '-u', self._local_file_path, '--runner', 'DataflowRunner',
        '--project', self._project, '--region', self._location,
        '--temp_location', self._gcs_temp_path, 'test_arg'
    ]
    self.assertListEqual(command, expected_results)

  @mock.patch.object(
      dataflow_python_job_remote_runner, 'stage_file', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner, 'prepare_cmd', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner, 'Process', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner,
      'extract_job_id_and_location',
      autospec=True)
  def test_create_python_job_raises_error_on_no_job_id(
      self, mock_extract_job_id_and_location, mock_process,
      unused_mock_prepare_cmd, unused_mock_stage_file):
    mock_process_client = mock.Mock()
    mock_process.return_value = mock_process_client
    mock_process_client.read_lines.return_value = ['test_line']
    mock_extract_job_id_and_location.return_value = (None, None)

    with self.assertRaises(RuntimeError):
      dataflow_python_job_remote_runner.create_python_job(
          python_module_path=self._local_file_path,
          project=self._project,
          gcp_resources=self._gcp_resources,
          location=self._location,
          temp_location=self._gcs_temp_path)

  @mock.patch.object(
      dataflow_python_job_remote_runner, 'stage_file', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner, 'prepare_cmd', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner, 'Process', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner,
      'extract_job_id_and_location',
      autospec=True)
  def test_create_python_job_parses_with_emtpy_args_list_parses_correctly(
      self, mock_extract_job_id_and_location, mock_process, mock_prepare_cmd,
      unused_mock_stage_file):
    mock_process_client = mock.Mock()
    mock_process.return_value = mock_process_client
    mock_process_client.read_lines.return_value = ['test_line']
    mock_extract_job_id_and_location.return_value = (None, None)

    with self.assertRaises(RuntimeError):
      dataflow_python_job_remote_runner.create_python_job(
          python_module_path=self._local_file_path,
          project=self._project,
          gcp_resources=self._gcp_resources,
          location=self._location,
          temp_location=self._gcs_temp_path)
    mock_prepare_cmd.assert_called_once_with(self._project, self._location,
                                             mock.ANY, [], self._gcs_temp_path)

  @mock.patch.object(
      dataflow_python_job_remote_runner, 'stage_file', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner, 'prepare_cmd', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner, 'Process', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner,
      'extract_job_id_and_location',
      autospec=True)
  def test_create_python_job_parses_with_json_array_args_list_parses_correctly(
      self, mock_extract_job_id_and_location, mock_process, mock_prepare_cmd,
      unused_mock_stage_file):
    mock_process_client = mock.Mock()
    mock_process.return_value = mock_process_client
    mock_process_client.read_lines.return_value = ['test_line']
    mock_extract_job_id_and_location.return_value = (None, None)

    with self.assertRaises(RuntimeError):
      dataflow_python_job_remote_runner.create_python_job(
          python_module_path=self._local_file_path,
          project=self._project,
          gcp_resources=self._gcp_resources,
          location=self._location,
          temp_location=self._gcs_temp_path,
          args=json.dumps(self._args))
    mock_prepare_cmd.assert_called_once_with(self._project, self._location,
                                             mock.ANY, self._args,
                                             self._gcs_temp_path)

  @mock.patch.object(
      dataflow_python_job_remote_runner, 'stage_file', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner, 'prepare_cmd', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner, 'Process', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner,
      'extract_job_id_and_location',
      autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner, 'install_requirements', autospec=True)
  def test_create_python_job_installs_requirements_if_file_is_provided(
      self, mock_install_requirements, unused_mock_extract_job_id_and_location,
      unused_mock_process, unused_mock_prepare_cmd, unused_mock_stage_file):
    with self.assertRaises(RuntimeError):
      dataflow_python_job_remote_runner.create_python_job(
          python_module_path=self._local_file_path,
          project=self._project,
          gcp_resources=self._gcp_resources,
          location=self._location,
          temp_location=self._gcs_temp_path,
          requirements_file_path=self._requirement_file_path)
    mock_install_requirements.assert_called_with(self._requirement_file_path)

  @mock.patch.object(
      dataflow_python_job_remote_runner, 'stage_file', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner, 'prepare_cmd', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner, 'Process', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner,
      'extract_job_id_and_location',
      autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner, 'install_requirements', autospec=True)
  def test_create_python_job_installs_requirements_creates_gcp_resouce_output_correctly(
      self, unused_mock_install_requirements, mock_extract_job_id_and_location,
      unused_mock_process, unused_mock_prepare_cmd, unused_mock_stage_file):
    mock_sub_process = mock.Mock()
    unused_mock_process.return_value = mock_sub_process
    mock_sub_process.read_lines.return_value = ['test_line1']
    mock_extract_job_id_and_location.return_value = (self._job_id,
                                                     self._location)

    dataflow_python_job_remote_runner.create_python_job(
        python_module_path=self._local_file_path,
        project=self._project,
        gcp_resources=self._gcp_resources,
        location=self._location,
        temp_location=self._gcs_temp_path)

    expected_result = {
        'resources': [{
            'resourceType':
                self._job_type,
            'resourceUri':
                f'https://dataflow.googleapis.com/v1b3/projects/{self._project}/locations/{self._location}/jobs/{self._job_id}'
        }]
    }

    with open(self._gcp_resources, 'r') as f:
      output = f.read()
      self.assertDictEqual(json.loads(output), expected_result)

  @mock.patch.object(
      dataflow_python_job_remote_runner, 'stage_file', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner, 'prepare_cmd', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner, 'Process', autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner,
      'extract_job_id_and_location',
      autospec=True)
  @mock.patch.object(
      dataflow_python_job_remote_runner, 'install_requirements', autospec=True)
  def test_create_python_job_stages_requirements_and_setup_file_locally(
      self, unused_mock_install_requirements, mock_extract_job_id_and_location,
      mock_process, unused_mock_prepare_cmd, mock_stage_file):
    mock_sub_process = mock.Mock()
    mock_process.return_value = mock_sub_process
    mock_sub_process.read_lines.return_value = ['test_line1']
    mock_extract_job_id_and_location.return_value = (self._job_id,
                                                     self._location)

    dataflow_python_job_remote_runner.create_python_job(
        python_module_path=self._local_file_path,
        project=self._project,
        gcp_resources=self._gcp_resources,
        location=self._location,
        temp_location=self._gcs_temp_path,
        args=json.dumps([
            '--requirements_file', self._requirement_file_path, '--setup_file',
            self._setup_file_path
        ]))

    stage_file_calls = [
        mock.call(self._local_file_path),
        mock.call(self._requirement_file_path),
        mock.call(self._setup_file_path)
    ]
    mock_stage_file.assert_has_calls(stage_file_calls, any_order=True)
