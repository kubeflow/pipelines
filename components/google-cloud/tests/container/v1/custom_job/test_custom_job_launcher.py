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
"""Test Vertex AI Custom Job Launcher Client module."""

import os

from google_cloud_pipeline_components.container.v1.custom_job import launcher
from google_cloud_pipeline_components.container.v1.custom_job import remote_runner

import unittest
from unittest import mock


class LauncherCustomJobUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherCustomJobUtilsTests, self).setUp()
    self._project = 'test_project'
    self._location = 'test_region'
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')

  @mock.patch.object(
      remote_runner, 'create_custom_job', autospec=True)
  def test_launcher_on_custom_job_type(self, mock_create_custom_job):
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
        self._gcp_resources, '--extra_arg', 'extra_arg_value'
    ]
    launcher.main(input_args)
    mock_create_custom_job.assert_called_once_with(
        type=job_type,
        project=self._project,
        location=self._location,
        payload=payload,
        gcp_resources=self._gcp_resources)