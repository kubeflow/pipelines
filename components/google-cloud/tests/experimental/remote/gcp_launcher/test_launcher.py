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
from google_cloud_pipeline_components.experimental.remote.gcp_launcher import launcher
import google_cloud_pipeline_components


class LauncherUtilsTests(unittest.TestCase):

    def setUp(self):
        super(LauncherUtilsTests, self).setUp()
        self._input_args = [
            "--type", "CustomJob", "--gcp_project", "test_project",
            "--gcp_region", "us_central1", "--payload", "test_payload",
            "--gcp_resources", "test_file_path/test_file.txt"
        ]
        self._payload = '{"display_name": "ContainerComponent", "job_spec": {"worker_pool_specs": [{"machine_spec": {"machine_type": "n1-standard-4"}, "replica_count": 1, "container_spec": {"image_uri": "google/cloud-sdk:latest", "command": ["sh", "-c", "set -e -x\\necho \\"$0, this is an output parameter\\"\\n", "{{$.inputs.parameters[\'input_text\']}}", "{{$.outputs.parameters[\'output_value\'].output_file}}"]}}]}}'
        self._gcp_project = 'test_gcp_project'
        self._gcp_region = 'test_region'
        self._gcp_resouces_path = 'gcp_resouces'
        self._type = 'CustomJob'

    @mock.patch.object(
        google_cloud_pipeline_components.experimental.remote.gcp_launcher.
        custom_job_remote_runner,
        "create_custom_job",
        autospec=True
    )
    def test_launcher_on_custom_job_type_calls_custom_job_remote_runner(
        self, mock_custom_job_remote_runner
    ):
        launcher.main(self._input_args)
        mock_custom_job_remote_runner.assert_called_once_with(
            type='CustomJob',
            gcp_project='test_project',
            gcp_region='us_central1',
            payload='test_payload',
            gcp_resources='test_file_path/test_file.txt'
        )
