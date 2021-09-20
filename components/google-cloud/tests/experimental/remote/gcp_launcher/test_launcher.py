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
from google_cloud_pipeline_components.experimental.remote.gcp_launcher import launcher


class LauncherUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherUtilsTests, self).setUp()
    self._project = 'test_project'
    self._location = 'test_region'
    self._gcp_resources_path = 'test_file_path/test_file.txt'

  @mock.patch.object(
      google_cloud_pipeline_components.experimental.remote.gcp_launcher
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
        '--job_type', job_type, '--project', self._project, '--location',
        self._location, '--payload', payload, '--gcp_resources',
        'test_file_path/test_file.txt', '--extra_arg', 'extra_arg_value'
    ]
    launcher.main(input_args)
    mock_custom_job_remote_runner.assert_called_once_with(
        job_type=job_type,
        project=self._project,
        location=self._location,
        payload=payload,
        gcp_resources=self._gcp_resources_path)

  @mock.patch.object(
      google_cloud_pipeline_components.experimental.remote.gcp_launcher
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
        '--job_type', job_type, '--project', self._project, '--location',
        self._location, '--payload', payload, '--gcp_resources',
        'test_file_path/test_file.txt', '--extra_arg', 'extra_arg_value'
    ]
    launcher.main(input_args)
    mock_batch_prediction_job_remote_runner.assert_called_once_with(
        job_type=job_type,
        project=self._project,
        location=self._location,
        payload=payload,
        gcp_resources=self._gcp_resources_path)
