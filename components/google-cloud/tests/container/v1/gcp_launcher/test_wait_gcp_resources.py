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
from google_cloud_pipeline_components.container.utils.execution_context import ExecutionContext
from google_cloud_pipeline_components.container.v1.gcp_launcher import wait_gcp_resources
import googleapiclient.discovery as discovery


class WaitGcpResourcesTests(unittest.TestCase):

    def setUp(self):
        super(WaitGcpResourcesTests, self).setUp()
        self._payload = '{"resources": [{"resourceType": "DataflowJob","resourceUri": "https://dataflow.googleapis.com/v1b3/projects/foo/locations/us-central1/jobs/job123"}]}'
        self._project = 'project1'
        self._location = 'us-central1'
        self._gcp_resources_path = os.path.join(os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), "gcp_resources")
        self._type = 'DataflowJob'

    def tearDown(self):
        if os.path.exists(self._gcp_resources_path):
            os.remove(self._gcp_resources_path)

    @mock.patch.object(discovery, 'build', autospec=True)
    def test_wait_gcp_resources_on_getting_succeeded_dataflow(self, mock_build):
        df_client = mock.Mock()
        mock_build.return_value = df_client
        expected_job = {'id': 'job-1', 'currentState': 'JOB_STATE_DONE'}
        get_request = mock.Mock()
        df_client.projects().locations().jobs().get.return_value = get_request
        get_request.execute.return_value = expected_job

        wait_gcp_resources.wait_gcp_resources(self._type, self._project,
                                              self._location, self._payload,
                                              self._gcp_resources_path)
        df_client.projects().locations().jobs().get.assert_called_once_with(
            projectId='foo', jobId='job123', location='us-central1', view=None)

    @mock.patch.object(discovery, 'build', autospec=True)
    def test_wait_gcp_resources_on_getting_failed_dataflow(self, mock_build):
        df_client = mock.Mock()
        mock_build.return_value = df_client
        expected_job = {'id': 'job-1', 'currentState': 'JOB_STATE_FAILED'}
        get_request = mock.Mock()
        df_client.projects().locations().jobs().get.return_value = get_request
        get_request.execute.return_value = expected_job

        with self.assertRaises(RuntimeError):
            wait_gcp_resources.wait_gcp_resources(self._type, self._project,
                                                  self._location, self._payload,
                                                  self._gcp_resources_path)

    @mock.patch.object(discovery, 'build', autospec=True)
    def test_wait_gcp_resources_on_invalid_gcp_resource_type(self, mock_build):
        invalid_payload = '{"resources": [{"resourceType": "BigQuery","resourceUri": "https://dataflow.googleapis.com/v1b3/projects/foo/locations/us-central1/jobs/job123"}]}'
        with self.assertRaises(ValueError):
            wait_gcp_resources.wait_gcp_resources(self._type, self._project,
                                                  self._location,
                                                  invalid_payload,
                                                  self._gcp_resources_path)

    @mock.patch.object(discovery, 'build', autospec=True)
    def test_wait_gcp_resources_on_empty_gcp_resource(self, mock_build):
        invalid_payload = '{"resources": [{}]}'
        with self.assertRaises(ValueError):
            wait_gcp_resources.wait_gcp_resources(self._type, self._project,
                                                  self._location,
                                                  invalid_payload,
                                                  self._gcp_resources_path)

    @mock.patch.object(discovery, 'build', autospec=True)
    def test_wait_gcp_resources_on_invalid_gcp_resource_uri(self, mock_build):
        invalid_payload = '{"resources": [{"resourceType": "DataflowJob","resourceUri": "https://dataflow.googleapis.com/v1b3/projects/abc"}]}'
        with self.assertRaises(ValueError):
            wait_gcp_resources.wait_gcp_resources(self._type, self._project,
                                                  self._location,
                                                  invalid_payload,
                                                  self._gcp_resources_path)

    @mock.patch.object(discovery, 'build', autospec=True)
    @mock.patch.object(time, "sleep", autospec=True)
    def test_wait_gcp_resources_retries_to_get_status_on_non_completed_job(
            self, mock_time_sleep, mock_build):
        df_client = mock.Mock()
        mock_build.return_value = df_client
        expected_job_running = {
            'id': 'job-1',
            'currentState': 'JOB_STATE_RUNNING'
        }
        expected_job_succeeded = {
            'id': 'job-1',
            'currentState': 'JOB_STATE_DONE'
        }
        get_request = mock.Mock()
        df_client.projects().locations().jobs().get.return_value = get_request
        get_request.execute.side_effect = [
            expected_job_running, expected_job_succeeded
        ]

        wait_gcp_resources.wait_gcp_resources(self._type, self._project,
                                              self._location, self._payload,
                                              self._gcp_resources_path)
        mock_time_sleep.assert_called_once_with(
            wait_gcp_resources._POLLING_INTERVAL_IN_SECONDS)
        self.assertEqual(df_client.projects().locations().jobs().get.call_count,
                         2)

    @mock.patch.object(discovery, 'build', autospec=True)
    @mock.patch.object(ExecutionContext, '__init__', autospec=True)
    def test_wait_gcp_resources_cancel_dataflow(self,
                                                mock_execution_context,
                                                mock_build):
        df_client = mock.Mock()
        mock_build.return_value = df_client
        expected_done_job = {'id': 'job-1', 'currentState': 'JOB_STATE_DONE'}
        expected_running_job = {'id': 'job-1', 'currentState': 'JOB_STATE_RUNNING'}
        get_request = mock.Mock()
        df_client.projects().locations().jobs().get.return_value = get_request
        # The first done_job is to end the polling loop.
        get_request.execute.side_effect = [
            expected_done_job, expected_running_job, expected_running_job
        ]
        update_request = mock.Mock()
        df_client.projects().locations().jobs().update.return_value = update_request
        mock_execution_context.return_value = None
        expected_cancel_job = {
            'id': 'job-1',
            'currentState': 'JOB_STATE_RUNNING',
            'requestedState': 'JOB_STATE_CANCELLED'
        }

        wait_gcp_resources.wait_gcp_resources(self._type, self._project,
                                              self._location, self._payload,
                                              self._gcp_resources_path)

        # Call cancellation handler
        mock_execution_context.call_args[1]['on_cancel']()
        df_client.projects().locations().jobs().update.assert_called_once_with(
            projectId='foo', jobId='job123', location='us-central1',
            body=expected_cancel_job)
