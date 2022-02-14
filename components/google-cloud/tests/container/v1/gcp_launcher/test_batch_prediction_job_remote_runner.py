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
"""Test Vertex AI Batch Prediction Job Remote Runner Client module."""

import json
from logging import raiseExceptions
import os
import time
import unittest
from unittest import mock

from google.cloud import aiplatform
from google.cloud.aiplatform.compat.types import job_state as gca_job_state
from google.protobuf import json_format
from google_cloud_pipeline_components.container.utils.execution_context import ExecutionContext
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
from google_cloud_pipeline_components.container.v1.gcp_launcher import batch_prediction_job_remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher import job_remote_runner
import requests
import google.auth
import google.auth.transport.requests


class BatchPredictionJobRemoteRunnerUtilsTests(unittest.TestCase):

    def setUp(self):
        super(BatchPredictionJobRemoteRunnerUtilsTests, self).setUp()
        self._payload = (
            '{"batchPredictionJob": {"displayName": '
            '"BatchPredictionComponentName", "model": '
            '"projects/test/locations/test/models/test-model","inputConfig":'
            ' {"instancesFormat": "CSV","gcsSource": {"uris": '
            '["test_gcs_source"]}}, "outputConfig": {"predictionsFormat": '
            '"CSV", "gcsDestination": {"outputUriPrefix": '
            '"test_gcs_destination"}}}}')
        self._job_type = 'BatchPredictionJob'
        self._project = 'test_project'
        self._location = 'test_region'
        self._batch_prediction_job_name = '/projects/{self._project}/locations/{self._location}/jobs/test_job_id'
        self._gcp_resources = os.path.join(os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), "gcp_resources")
        self._batch_prediction_job_uri_prefix = f'https://{self._location}-aiplatform.googleapis.com/v1/'
        self._output_file_path = os.path.join(os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), "localpath/foo")
        self._executor_input = '{"outputs":{"artifacts":{"batchpredictionjob":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.VertexBatchPredictionJob"},"uri":"gs://abc"}]}},"outputFile":"'+self._output_file_path+'"}}'

    def tearDown(self):
        if os.path.exists(self._gcp_resources):
            os.remove(self._gcp_resources)

    @mock.patch.object(aiplatform.gapic, 'JobServiceClient', autospec=True)
    @mock.patch.object(os.path, 'exists', autospec=True)
    def test_batch_prediction_job_remote_runner_succeeded(
            self, mock_path_exists, mock_job_service_client):
        job_client = mock.Mock()
        mock_job_service_client.return_value = job_client

        create_batch_prediction_job_response = mock.Mock()
        job_client.create_batch_prediction_job.return_value = create_batch_prediction_job_response
        create_batch_prediction_job_response.name = self._batch_prediction_job_name

        get_batch_prediction_job_response = mock.Mock()
        job_client.get_batch_prediction_job.return_value = get_batch_prediction_job_response
        get_batch_prediction_job_response.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED
        get_batch_prediction_job_response.name = 'job1'
        get_batch_prediction_job_response.output_info.bigquery_output_table = 'bigquery_output_table'
        get_batch_prediction_job_response.output_info.bigquery_output_dataset = 'bigquery_output_dataset'
        get_batch_prediction_job_response.output_info.gcs_output_directory = 'gcs_output_directory'

        mock_path_exists.return_value = False

        batch_prediction_job_remote_runner.create_batch_prediction_job(
            self._job_type, self._project, self._location, self._payload,
            self._gcp_resources,self._executor_input)

        mock_job_service_client.assert_called_once_with(
            client_options={
                'api_endpoint': 'test_region-aiplatform.googleapis.com'
            },
            client_info=mock.ANY)

        expected_parent = f'projects/{self._project}/locations/{self._location}'
        expected_job_spec = json.loads(self._payload, strict=False)

        job_client.create_batch_prediction_job.assert_called_once_with(
            parent=expected_parent, batch_prediction_job=expected_job_spec)

        with open(self._gcp_resources) as f:
            serialized_gcp_resources = f.read()

            # Instantiate GCPResources Proto
            batch_prediction_job_resources = json_format.Parse(
                serialized_gcp_resources, GcpResources())

            self.assertEqual(len(batch_prediction_job_resources.resources), 1)
            batch_prediction_job_name = batch_prediction_job_resources.resources[
                0].resource_uri[len(self._batch_prediction_job_uri_prefix):]
            self.assertEqual(batch_prediction_job_name,
                             self._batch_prediction_job_name)

        with open(self._output_file_path) as f:
            executor_output = json.load(f, strict=False)
            self.assertEqual(
                executor_output,
                json.loads(
                    '{"artifacts": {"batchpredictionjob": {"artifacts": [{"metadata": {"resourceName": "job1", "bigqueryOutputDataset": "bigquery_output_dataset","bigqueryOutputTable": "bigquery_output_table","gcsOutputDirectory": "gcs_output_directory"}, "name": "foobar", "type": {"schemaTitle": "google.VertexBatchPredictionJob"}, "uri": "https://test_region-aiplatform.googleapis.com/v1/job1"}]}}}'
                ))

    @mock.patch.object(aiplatform.gapic, 'JobServiceClient', autospec=True)
    @mock.patch.object(os.path, 'exists', autospec=True)
    def test_batch_prediction_job_remote_runner_raises_exception_on_error(
            self, mock_path_exists, mock_job_service_client):

        job_client = mock.Mock()
        mock_job_service_client.return_value = job_client

        create_batch_prediction_job_response = mock.Mock()
        job_client.create_batch_prediction_job.return_value = create_batch_prediction_job_response
        create_batch_prediction_job_response.name = self._batch_prediction_job_name

        get_batch_prediction_job_response = mock.Mock()
        job_client.get_batch_prediction_job.return_value = get_batch_prediction_job_response
        get_batch_prediction_job_response.state = gca_job_state.JobState.JOB_STATE_FAILED

        mock_path_exists.return_value = False

        with self.assertRaises(RuntimeError):
            batch_prediction_job_remote_runner.create_batch_prediction_job(
                self._job_type, self._project, self._location, self._payload,
                self._gcp_resources,self._executor_input)

    @mock.patch.object(aiplatform.gapic, 'JobServiceClient', autospec=True)
    @mock.patch.object(os.path, 'exists', autospec=True)
    @mock.patch.object(time, 'sleep', autospec=True)
    def test_batch_prediction_job_remote_runner_retries_to_get_status_on_non_completed_job(
            self, mock_time_sleep, mock_path_exists, mock_job_service_client):
        job_client = mock.Mock()
        mock_job_service_client.return_value = job_client

        create_batch_prediction_job_response = mock.Mock()
        job_client.create_batch_prediction_job.return_value = create_batch_prediction_job_response
        create_batch_prediction_job_response.name = self._batch_prediction_job_name

        get_batch_prediction_job_response_success = mock.Mock()
        get_batch_prediction_job_response_success.name = 'job1'
        get_batch_prediction_job_response_success.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED
        get_batch_prediction_job_response_success.output_info.bigquery_output_table = 'bigquery_output_table'
        get_batch_prediction_job_response_success.output_info.bigquery_output_dataset = 'bigquery_output_dataset'
        get_batch_prediction_job_response_success.output_info.gcs_output_directory = 'gcs_output_directory'

        get_batch_prediction_job_response_running = mock.Mock()
        get_batch_prediction_job_response_running.state = gca_job_state.JobState.JOB_STATE_RUNNING

        job_client.get_batch_prediction_job.side_effect = [
            get_batch_prediction_job_response_running,
            get_batch_prediction_job_response_success
        ]

        mock_path_exists.return_value = False

        batch_prediction_job_remote_runner.create_batch_prediction_job(
            self._job_type, self._project, self._location, self._payload,
            self._gcp_resources,self._executor_input)
        mock_time_sleep.assert_called_once_with(
            job_remote_runner._POLLING_INTERVAL_IN_SECONDS)
        self.assertEqual(job_client.get_batch_prediction_job.call_count, 2)

    @mock.patch.object(aiplatform.gapic, 'JobServiceClient', autospec=True)
    @mock.patch.object(os.path, 'exists', autospec=True)
    @mock.patch.object(google.auth, 'default', autospec=True)
    @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
    @mock.patch.object(requests, 'post', autospec=True)
    @mock.patch.object(ExecutionContext, '__init__', autospec=True)
    def test_batch_prediction_job_remote_runner_cancel(self,
                                                       mock_execution_context,
                                                       mock_post_requests,
                                                       _, mock_auth,
                                                       mock_path_exists,
                                                       mock_job_service_client):
        creds = mock.Mock()
        creds.token = 'fake_token'
        mock_auth.return_value = [creds, "project"]

        job_client = mock.Mock()
        mock_job_service_client.return_value = job_client

        create_batch_prediction_job_response = mock.Mock()
        job_client.create_batch_prediction_job.return_value = create_batch_prediction_job_response
        create_batch_prediction_job_response.name = self._batch_prediction_job_name

        get_batch_prediction_job_response = mock.Mock()
        job_client.get_batch_prediction_job.return_value = get_batch_prediction_job_response
        get_batch_prediction_job_response.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED
        get_batch_prediction_job_response.name = 'job1'
        get_batch_prediction_job_response.output_info.bigquery_output_table = 'bigquery_output_table'
        get_batch_prediction_job_response.output_info.bigquery_output_dataset = 'bigquery_output_dataset'
        get_batch_prediction_job_response.output_info.gcs_output_directory = 'gcs_output_directory'

        mock_path_exists.return_value = False
        mock_execution_context.return_value = None

        batch_prediction_job_remote_runner.create_batch_prediction_job(
            self._job_type, self._project, self._location, self._payload,
            self._gcp_resources,self._executor_input)

        # Call cancellation handler
        mock_execution_context.call_args[1]['on_cancel']()
        mock_post_requests.assert_called_once_with(
            url=f'{self._batch_prediction_job_uri_prefix}{self._batch_prediction_job_name}:cancel',
            data='',
            headers={
                'Content-type': 'application/json',
                'Authorization': 'Bearer fake_token',
            })
