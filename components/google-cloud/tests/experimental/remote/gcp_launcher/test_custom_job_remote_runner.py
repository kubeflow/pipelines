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
"""Test Vertex AI Custom Job Remote Runner Client module."""

import json
from logging import raiseExceptions
from os import path
import time
import unittest
from unittest import mock
from google_cloud_pipeline_components.experimental.remote.gcp_launcher import custom_job_remote_runner
from google.cloud import aiplatform
from google.cloud.aiplatform.compat.types import job_state as gca_job_state


class CustomJobRemoteRunnerUtilsTests(unittest.TestCase):

    def setUp(self):
        super(CustomJobRemoteRunnerUtilsTests, self).setUp()
        self._payload = '{"display_name": "ContainerComponent", "job_spec": {"worker_pool_specs": [{"machine_spec": {"machine_type": "n1-standard-4"}, "replica_count": 1, "container_spec": {"image_uri": "google/cloud-sdk:latest", "command": ["sh", "-c", "set -e -x\\necho \\"$0, this is an output parameter\\"\\n", "{{$.inputs.parameters[\'input_text\']}}", "{{$.outputs.parameters[\'output_value\'].output_file}}"]}}]}}'
        self._gcp_project = 'test_gcp_project'
        self._gcp_region = 'test_region'
        self._gcp_resouces_path = 'gcp_resouces'
        self._type = 'CustomJob'

    @mock.patch.object(aiplatform.gapic, 'JobServiceClient', autospec=True)
    def test_custom_job_remote_runner_on_region_is_set_correctly_in_client_options(
        self, mock_job_service_client
    ):

        job_client = mock.Mock()
        mock_job_service_client.return_value = job_client

        create_custom_job_response = mock.Mock()
        job_client.create_custom_job.return_value = create_custom_job_response
        create_custom_job_response.name = 'test_custom_job'

        get_custom_job_response = mock.Mock()
        job_client.get_custom_job.return_value = get_custom_job_response
        get_custom_job_response.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

        custom_job_remote_runner.create_custom_job(
            self._type, self._gcp_project, self._gcp_region, self._payload,
            self._gcp_resouces_path
        )
        mock_job_service_client.assert_called_once_with(
            client_options={
                'api_endpoint': 'test_region-aiplatform.googleapis.com'
            }
        )

    @mock.patch.object(aiplatform.gapic, 'JobServiceClient', autospec=True)
    @mock.patch.object(path, "exists", autospec=True)
    def test_custom_job_remote_runner_on_payload_deserializes_correctly(
        self, mock_path_exists, mock_job_service_client
    ):

        job_client = mock.Mock()
        mock_job_service_client.return_value = job_client

        create_custom_job_response = mock.Mock()
        job_client.create_custom_job.return_value = create_custom_job_response
        create_custom_job_response.name = 'test_custom_job'

        get_custom_job_response = mock.Mock()
        job_client.get_custom_job.return_value = get_custom_job_response
        get_custom_job_response.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

        mock_path_exists.return_value = False

        custom_job_remote_runner.create_custom_job(
            self._type, self._gcp_project, self._gcp_region, self._payload,
            self._gcp_resouces_path
        )

        expected_parent = f"projects/{self._gcp_project}/locations/{self._gcp_region}"
        expected_job_spec = json.loads(self._payload, strict=False)

        job_client.create_custom_job.assert_called_once_with(
            parent=expected_parent, custom_job=expected_job_spec
        )

    @mock.patch.object(aiplatform.gapic, 'JobServiceClient', autospec=True)
    @mock.patch.object(path, "exists", autospec=True)
    def test_custom_job_remote_runner_raises_exception_on_error(
        self, mock_path_exists, mock_job_service_client
    ):

        job_client = mock.Mock()
        mock_job_service_client.return_value = job_client

        create_custom_job_response = mock.Mock()
        job_client.create_custom_job.return_value = create_custom_job_response
        create_custom_job_response.name = 'test_custom_job'

        get_custom_job_response = mock.Mock()
        job_client.get_custom_job.return_value = get_custom_job_response
        get_custom_job_response.state = gca_job_state.JobState.JOB_STATE_FAILED

        mock_path_exists.return_value = False

        with self.assertRaises(RuntimeError):
            custom_job_remote_runner.create_custom_job(
                self._type, self._gcp_project, self._gcp_region, self._payload,
                self._gcp_resouces_path
            )

    @mock.patch.object(aiplatform.gapic, 'JobServiceClient', autospec=True)
    @mock.patch.object(path, "exists", autospec=True)
    @mock.patch.object(time, "sleep", autospec=True)
    def test_custom_job_remote_runner_retries_to_get_status_on_non_completed_job(
        self, mock_time_sleep, mock_path_exists, mock_job_service_client
    ):
        job_client = mock.Mock()
        mock_job_service_client.return_value = job_client

        create_custom_job_response = mock.Mock()
        job_client.create_custom_job.return_value = create_custom_job_response
        create_custom_job_response.name = 'test_custom_job'

        get_custom_job_response_success = mock.Mock()
        get_custom_job_response_success.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

        get_custom_job_response_running = mock.Mock()
        get_custom_job_response_running.state = gca_job_state.JobState.JOB_STATE_RUNNING

        job_client.get_custom_job.side_effect = [
            get_custom_job_response_running, get_custom_job_response_success
        ]

        mock_path_exists.return_value = False

        custom_job_remote_runner.create_custom_job(
            self._type, self._gcp_project, self._gcp_region, self._payload,
            self._gcp_resouces_path
        )
        mock_time_sleep.assert_called_once_with(
            custom_job_remote_runner._POLLING_INTERVAL_IN_SECONDS
        )
        self.assertEqual(job_client.get_custom_job.call_count, 2)
