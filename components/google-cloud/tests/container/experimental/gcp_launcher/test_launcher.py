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
"""Test Vertex AI Launcher Client module."""

import unittest
from unittest import mock

import google_cloud_pipeline_components
from google_cloud_pipeline_components.container.experimental.gcp_launcher import launcher


class LauncherJobUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherJobUtilsTests, self).setUp()
    self._project = 'test_project'
    self._location = 'test_region'
    self._gcp_resources = 'test_file_path/test_file.txt'

  @mock.patch.object(
      google_cloud_pipeline_components.container.experimental.gcp_launcher
      .custom_job_remote_runner,
      'create_custom_job',
      autospec=True
  )
  def test_launcher_on_custom_job_type_calls_custom_job_remote_runner(
      self, mock_custom_job_remote_runner):
    job_type = 'CustomJob'
    payload = ('{"display_name": "ContainerComponent", "job_spec": '
               '{"worker_pool_specs": [{"machine_spec": {"machine_type": '
               '"n1-standard-4"}, "replica_count": 1, "container_spec": '
               '{"image_uri": "google/cloud-sdk:latest", "command": ["sh", '
               '"-c", "set -e -x\\necho \\"$0, this is an output '
               'parameter\\"\\n", "{{$.inputs.parameters[\'input_text\']}}", '
               '"{{$.outputs.parameters[\'output_value\'].output_file}}"]}}]}}')
    input_args = [
        '--type', job_type, '--project', self._project, '--location',
        self._location, '--payload', payload, '--gcp_resources',
        'test_file_path/test_file.txt', '--extra_arg', 'extra_arg_value'
    ]
    launcher.main(input_args)
    mock_custom_job_remote_runner.assert_called_once_with(
        type=job_type,
        project=self._project,
        location=self._location,
        payload=payload,
        gcp_resources=self._gcp_resources)

  @mock.patch.object(
      google_cloud_pipeline_components.container.experimental.gcp_launcher
      .batch_prediction_job_remote_runner,
      'create_batch_prediction_job',
      autospec=True
  )
  def test_launcher_on_batch_prediction_job_type_calls_batch_prediction_job_remote_runner(
      self, mock_batch_prediction_job_remote_runner):
    job_type = 'BatchPredictionJob'
    payload = ('{"batchPredictionJob": {"displayName": '
               '"BatchPredictionComponentName", "model": '
               '"projects/test/locations/test/models/test-model","inputConfig":'
               ' {"instancesFormat": "CSV","gcsSource": {"uris": '
               '["test_gcs_source"]}}, "outputConfig": {"predictionsFormat": '
               '"CSV", "gcsDestination": {"outputUriPrefix": '
               '"test_gcs_destination"}}}}')
    input_args = [
        '--type', job_type, '--project', self._project, '--location',
        self._location, '--payload', payload, '--gcp_resources',
        'test_file_path/test_file.txt', '--extra_arg', 'extra_arg_value'
    ]
    launcher.main(input_args)
    mock_batch_prediction_job_remote_runner.assert_called_once_with(
        type=job_type,
        project=self._project,
        location=self._location,
        payload=payload,
        gcp_resources=self._gcp_resources)


class LauncherUploadModelUtilsTests(unittest.TestCase):

    def setUp(self):
        super(LauncherUploadModelUtilsTests, self).setUp()
        self._input_args = [
            "--type", "UploadModel", "--project", "test_project", "--location",
            "us_central1", "--payload", "test_payload", "--gcp_resources",
            "test_file_path/test_file.txt", "--executor_input", "executor_input"
        ]

    @mock.patch.object(
        google_cloud_pipeline_components.container.experimental.gcp_launcher
        .upload_model_remote_runner,
        "upload_model",
        autospec=True)
    def test_launcher_on_upload_model_type_calls_upload_model_remote_runner(
            self, mock_upload_model_remote_runner):
        launcher.main(self._input_args)
        mock_upload_model_remote_runner.assert_called_once_with(
            type='UploadModel',
            project='test_project',
            location='us_central1',
            payload='test_payload',
            gcp_resources='test_file_path/test_file.txt',
            executor_input='executor_input')

class LauncherCreateEndpointUtilsTests(unittest.TestCase):

    def setUp(self):
        super(LauncherCreateEndpointUtilsTests, self).setUp()
        self._input_args = [
            "--type", "CreateEndpoint", "--project", "test_project", "--location",
            "us_central1", "--payload", "test_payload", "--gcp_resources",
            "test_file_path/test_file.txt", "--executor_input", "executor_input"
        ]

    @mock.patch.object(
        google_cloud_pipeline_components.container.experimental.gcp_launcher
        .create_endpoint_remote_runner,
        "create_endpoint",
        autospec=True)
    def test_launcher_on_create_endpoint_type_calls_create_endpoint_remote_runner(
            self, create_endpoint_remote_runner):
        launcher.main(self._input_args)
        create_endpoint_remote_runner.assert_called_once_with(
            type='CreateEndpoint',
            project='test_project',
            location='us_central1',
            payload='test_payload',
            gcp_resources='test_file_path/test_file.txt',
            executor_input='executor_input')
