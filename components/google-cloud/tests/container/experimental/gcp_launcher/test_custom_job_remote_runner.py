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
import os
import time
import unittest
from unittest import mock

from google.cloud import aiplatform
from google.cloud.aiplatform.compat.types import job_state as gca_job_state
from google.protobuf import json_format
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
from google_cloud_pipeline_components.container.experimental.gcp_launcher import custom_job_remote_runner
from google_cloud_pipeline_components.container.experimental.gcp_launcher import job_remote_runner


class CustomJobRemoteRunnerUtilsTests(unittest.TestCase):

    def setUp(self):
        super(CustomJobRemoteRunnerUtilsTests, self).setUp()
        self._payload = (
            '{"display_name": "ContainerComponent", "job_spec": '
            '{"worker_pool_specs": [{"machine_spec": {"machine_type": '
            '"n1-standard-4"}, "replica_count": 1, "container_spec": '
            '{"image_uri": "google/cloud-sdk:latest", "command": '
            '["sh", "-c", "set -e -x\\necho \\"$0, this is an output '
            'parameter\\"\\n", '
            '"{{$.inputs.parameters[\'input_text\']}}", '
            '"{{$.outputs.parameters[\'output_value\'].output_file}}"]}}]}}')
        self._project = "test_project"
        self._location = "test_region"
        self._custom_job_name = f"/projects/{self._project}/locations/{self._location}/jobs/test_job_id"
        self._gcp_resources = os.path.join(os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), "gcp_resources")
        self._type = "CustomJob"
        self._custom_job_uri_prefix = f"https://{self._location}-aiplatform.googleapis.com/v1/"

    def tearDown(self):
        if os.path.exists(self._gcp_resources):
            os.remove(self._gcp_resources)

    @mock.patch.object(aiplatform.gapic, "JobServiceClient", autospec=True)
    def test_custom_job_remote_runner_on_region_is_set_correctly_in_client_options(
            self, mock_job_service_client):

        job_client = mock.Mock()
        mock_job_service_client.return_value = job_client

        create_custom_job_response = mock.Mock()
        job_client.create_custom_job.return_value = create_custom_job_response
        create_custom_job_response.name = self._custom_job_name

        get_custom_job_response = mock.Mock()
        job_client.get_custom_job.return_value = get_custom_job_response
        get_custom_job_response.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

        custom_job_remote_runner.create_custom_job(self._type, self._project,
                                                   self._location,
                                                   self._payload,
                                                   self._gcp_resources)
        mock_job_service_client.assert_called_once_with(
            client_options={
                "api_endpoint": "test_region-aiplatform.googleapis.com"
            },
            client_info=mock.ANY)

    @mock.patch.object(aiplatform.gapic, "JobServiceClient", autospec=True)
    @mock.patch.object(os.path, "exists", autospec=True)
    def test_custom_job_remote_runner_on_payload_deserializes_correctly(
            self, mock_path_exists, mock_job_service_client):

        job_client = mock.Mock()
        mock_job_service_client.return_value = job_client

        create_custom_job_response = mock.Mock()
        job_client.create_custom_job.return_value = create_custom_job_response
        create_custom_job_response.name = self._custom_job_name

        get_custom_job_response = mock.Mock()
        job_client.get_custom_job.return_value = get_custom_job_response
        get_custom_job_response.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

        mock_path_exists.return_value = False

        custom_job_remote_runner.create_custom_job(self._type, self._project,
                                                   self._location,
                                                   self._payload,
                                                   self._gcp_resources)

        expected_parent = f"projects/{self._project}/locations/{self._location}"
        expected_job_spec = json.loads(self._payload, strict=False)

        job_client.create_custom_job.assert_called_once_with(
            parent=expected_parent, custom_job=expected_job_spec)

    @mock.patch.object(aiplatform.gapic, "JobServiceClient", autospec=True)
    @mock.patch.object(os.path, "exists", autospec=True)
    def test_custom_job_remote_runner_raises_exception_on_error(
            self, mock_path_exists, mock_job_service_client):

        job_client = mock.Mock()
        mock_job_service_client.return_value = job_client

        create_custom_job_response = mock.Mock()
        job_client.create_custom_job.return_value = create_custom_job_response
        create_custom_job_response.name = self._custom_job_name

        get_custom_job_response = mock.Mock()
        job_client.get_custom_job.return_value = get_custom_job_response
        get_custom_job_response.state = gca_job_state.JobState.JOB_STATE_FAILED

        mock_path_exists.return_value = False

        with self.assertRaises(RuntimeError):
            custom_job_remote_runner.create_custom_job(self._type,
                                                       self._project,
                                                       self._location,
                                                       self._payload,
                                                       self._gcp_resources)

    @mock.patch.object(aiplatform.gapic, "JobServiceClient", autospec=True)
    @mock.patch.object(os.path, "exists", autospec=True)
    @mock.patch.object(time, "sleep", autospec=True)
    def test_custom_job_remote_runner_retries_to_get_status_on_non_completed_job(
            self, mock_time_sleep, mock_path_exists, mock_job_service_client):
        job_client = mock.Mock()
        mock_job_service_client.return_value = job_client

        create_custom_job_response = mock.Mock()
        job_client.create_custom_job.return_value = create_custom_job_response
        create_custom_job_response.name = self._custom_job_name

        get_custom_job_response_success = mock.Mock()
        get_custom_job_response_success.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

        get_custom_job_response_running = mock.Mock()
        get_custom_job_response_running.state = gca_job_state.JobState.JOB_STATE_RUNNING

        job_client.get_custom_job.side_effect = [
            get_custom_job_response_running, get_custom_job_response_success
        ]

        mock_path_exists.return_value = False

        custom_job_remote_runner.create_custom_job(self._type, self._project,
                                                   self._location,
                                                   self._payload,
                                                   self._gcp_resources)
        mock_time_sleep.assert_called_once_with(
            job_remote_runner._POLLING_INTERVAL_IN_SECONDS)
        self.assertEqual(job_client.get_custom_job.call_count, 2)

    @mock.patch.object(aiplatform.gapic, "JobServiceClient", autospec=True)
    @mock.patch.object(os.path, "exists", autospec=True)
    @mock.patch.object(time, "sleep", autospec=True)
    def test_custom_job_remote_runner_returns_gcp_resources(
            self, mock_time_sleep, mock_path_exists, mock_job_service_client):
        job_client = mock.Mock()
        mock_job_service_client.return_value = job_client

        create_custom_job_response = mock.Mock()
        job_client.create_custom_job.return_value = create_custom_job_response
        create_custom_job_response.name = self._custom_job_name

        get_custom_job_response_success = mock.Mock()
        get_custom_job_response_success.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

        job_client.get_custom_job.side_effect = [
            get_custom_job_response_success
        ]

        mock_path_exists.return_value = False

        custom_job_remote_runner.create_custom_job(self._type, self._project,
                                                   self._location,
                                                   self._payload,
                                                   self._gcp_resources)

        with open(self._gcp_resources) as f:
            serialized_gcp_resources = f.read()

            # Instantiate GCPResources Proto
            custom_job_resources = json_format.Parse(serialized_gcp_resources,
                                                     GcpResources())

            self.assertEqual(len(custom_job_resources.resources), 1)
            custom_job_name = custom_job_resources.resources[0].resource_uri[
                len(self._custom_job_uri_prefix):]
            self.assertEqual(custom_job_name, self._custom_job_name)

    @mock.patch.object(aiplatform.gapic, "JobServiceClient", autospec=True)
    @mock.patch.object(time, "sleep", autospec=True)
    def test_custom_job_remote_runner_raises_exception_with_more_than_one_resources_in_gcp_resources(
            self, mock_time_sleep, mock_job_service_client):
        job_client = mock.Mock()
        mock_job_service_client.return_value = job_client

        create_custom_job_response = mock.Mock()
        job_client.create_custom_job.return_value = create_custom_job_response
        create_custom_job_response.name = self._custom_job_name

        get_custom_job_response_success = mock.Mock()
        get_custom_job_response_success.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

        job_client.get_custom_job.side_effect = [
            get_custom_job_response_success
        ]

        # Write the job proto to output
        custom_job_resources = GcpResources()
        custom_job_resource_1 = custom_job_resources.resources.add()
        custom_job_resource_1.resource_type = "CustomJob"
        custom_job_resource_1.resource_uri = f"{self._custom_job_uri_prefix}{self._custom_job_name}"

        custom_job_resource_2 = custom_job_resources.resources.add()
        custom_job_resource_2.resource_type = "CustomJob"
        custom_job_resource_2.resource_uri = f"{self._custom_job_uri_prefix}{self._custom_job_name}"

        with open(self._gcp_resources, "w") as f:
            f.write(json_format.MessageToJson(custom_job_resources))

        with self.assertRaisesRegex(
                ValueError,
                "gcp_resources should contain one resource, found 2"):
            custom_job_remote_runner.create_custom_job(self._type,
                                                       self._project,
                                                       self._location,
                                                       self._payload,
                                                       self._gcp_resources)

    @mock.patch.object(aiplatform.gapic, "JobServiceClient", autospec=True)
    @mock.patch.object(time, "sleep", autospec=True)
    def test_custom_job_remote_runner_raises_exception_empty_URI_in_gcp_resources(
            self, mock_time_sleep, mock_job_service_client):
        job_client = mock.Mock()
        mock_job_service_client.return_value = job_client

        create_custom_job_response = mock.Mock()
        job_client.create_custom_job.return_value = create_custom_job_response
        create_custom_job_response.name = self._custom_job_name

        get_custom_job_response_success = mock.Mock()
        get_custom_job_response_success.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

        job_client.get_custom_job.side_effect = [
            get_custom_job_response_success
        ]

        # Write the job proto to output
        custom_job_resources = GcpResources()
        custom_job_resource_1 = custom_job_resources.resources.add()
        custom_job_resource_1.resource_type = "CustomJob"
        custom_job_resource_1.resource_uri = ""

        with open(self._gcp_resources, "w") as f:
            f.write(json_format.MessageToJson(custom_job_resources))

        with self.assertRaisesRegex(
                ValueError,
                "Job Name in gcp_resource is not formatted correctly or is empty."
        ):
            custom_job_remote_runner.create_custom_job(self._type,
                                                       self._project,
                                                       self._location,
                                                       self._payload,
                                                       self._gcp_resources)
