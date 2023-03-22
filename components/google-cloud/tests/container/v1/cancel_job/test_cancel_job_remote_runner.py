# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
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
import requests
import google.auth
import unittest
from google.cloud import aiplatform
from unittest import mock
from google.cloud.aiplatform.compat.types import job_state as gca_job_state
from google_cloud_pipeline_components.container.utils.execution_context import ExecutionContext
from google_cloud_pipeline_components.container.v1.cancel_job import remote_runner as cancel_job_remote_runner
from google_cloud_pipeline_components.container.v1.endpoint.create_endpoint import remote_runner as create_endpoint_remote_runner
from google_cloud_pipeline_components.container.v1.wait_gcp_resources import remote_runner as wait_gcp_resources_remote_runner
from google_cloud_pipeline_components.container.v1.bigquery.query_job import remote_runner as query_job_remote_runner
from google_cloud_pipeline_components.container.v1.dataproc.create_spark_sql_batch import remote_runner as dataproc_batch_remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import gcp_labels_util
import googleapiclient.discovery as discovery
from google_cloud_pipeline_components.container.v1.custom_job import remote_runner as custom_job_remote_runner

_SYSTEM_LABELS = {'key1': 'value1', 'key2': 'value2'}


class CancelJobTests(unittest.TestCase):

  def setUp(self):
    super(CancelJobTests, self).setUp()
    self._dataflow_payload = (
        '{"resources": [{"resourceType": "DataflowJob",'
        '"resourceUri": "https://dataflow.googleapis.com/'
        'v1b3/projects/foo/locations/us-central1/jobs/job123"}]}'
    )

    self._custom_job_payload = (
        '{"resources": [{"resourceType": "CustomJob",'
        '"resourceUri": "https://us-aiplatform.googleapis.com/v1/test_job"}]}'
    )

    self._big_query_payload = (
        '{"resources": [{"resourceType": "CustomJob",'
        '"resourceUri": "https://www.googleapis.com/bigquery/v2/projects/'
        'test_project/jobs/fake_job?location=US"}]}'
    )

    self._project = 'project1'
    self._location = 'us-central1'
    self._gcp_resources_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'gcp_resources')
    self._type = 'DataflowJob'

    self._custom_job_name = 'test_job'
    self._custom_job_uri_prefix = 'https://us-aiplatform.googleapis.com/v1/'

    self._big_query_job_type = 'BigqueryQueryJob'
    self._big_query_project = 'test_project'
    self._big_query_location = 'US'
    self._big_query_job_configuration_query_override = '{}'
    self._big_query_output_file_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'localpath/foo'
    )
    self._big_query_job_uri = 'https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US'

    self.vertexlro_type = 'CreateEndpoint'
    self.vertexlro_project = 'test_project'
    self.vertexlro_location = 'test_region'
    self.vertexlro_payload = '{"display_name": "ContainerComponent"}'
    self.vertexlro_endpoint_name = f'projects/{self.vertexlro_project}/locations/{self.vertexlro_location}/endpoints/123'
    self.vertexlro_uri_prefix = (
        f'https://{self.vertexlro_location}-aiplatform.googleapis.com/v1/'
    )
    self.vertexlro_lro_name = f'projects/{self.vertexlro_project}/locations/{self.vertexlro_location}/operations/123'
    self.vertexlro_output_file_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'localpath/foo'
    )
    self.vertexlro_executor_input = (
        '{"outputs":{"artifacts":{"endpoint":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"system.Endpoint"},"uri":"gs://abc"}]}},"outputFile":"'
        + self.vertexlro_output_file_path
        + '"}}'
    )

    self.dataproc_uri_prefix = 'https://dataproc.googleapis.com/v1'
    self.dataproc_project = 'test-project'
    self.dataproc_location = 'test-location'
    self.dataproc_batch_id = 'test-batch-id'
    self.dataproc_creds_token = 'fake-token'
    self.dataproc_operation_id = 'fake-operation-id'
    self.dataproc_operation_name = f'projects/{self.dataproc_project}/regions/{self.dataproc_location}/operations/{self.dataproc_operation_id}'
    self.dataproc_operation_uri = (
        f'{self.dataproc_uri_prefix}/{self.dataproc_operation_name}'
    )
    self.dataproc_batch_name = f'projects/{self.dataproc_project}/locations/{self.dataproc_location}/batches/{self.dataproc_batch_id}'
    self.dataproc_batch_uri = (
        f'{self.dataproc_uri_prefix}/{self.dataproc_batch_name}'
    )
    os.environ[gcp_labels_util.SYSTEM_LABEL_ENV_VAR] = json.dumps(
        _SYSTEM_LABELS
    )

    self.dataproc_test_batch = {
        'labels': {'foo': 'bar', 'fizz': 'buzz'},
        'runtime_config': {
            'version': 'test-version',
            'container_image': 'test-container-image',
            'properties': {'foo': 'bar', 'fizz': 'buzz'},
        },
        'environment_config': {
            'execution_config': {
                'service_account': 'test-service-account',
                'network_tags': ['test-tag-1', 'test-tag-2'],
                'kms_key': 'test-kms-key',
                'network_uri': 'test-network-uri',
                'subnetwork_uri': 'test-subnetwork-uri',
            },
            'peripherals_config': {
                'metastore_service': 'test-metastore-service',
                'spark_history_server_config': {
                    'dataproc_cluster': 'test-dataproc-cluster'
                },
            },
        },
    }
    self.dataproc_test_spark_sql_batch = {
        **self.dataproc_test_batch,
        'spark_sql_batch': {
            'query_file_uri': 'test-query-file',
            'query_variables': {'foo': 'bar', 'fizz': 'buzz'},
            'jar_file_uris': ['test-jar-file-uri-1', 'test-jar-file-uri-2'],
        },
    }
    self.dataproc_expected_spark_sql_batch = (
        self.dataproc_test_spark_sql_batch.copy()
    )
    self.dataproc_expected_spark_sql_batch['labels'] = {
        'key1': 'value1',
        'key2': 'value2',
        'foo': 'bar',
        'fizz': 'buzz',
    }
    self.dataproc_test_spark_sql_empty_fields_batch = {
        'runtime_config': {'container_image': ''},
        'spark_sql_batch': {
            'query_file_uri': 'test-query-file',
            'query_variables': {},
            'jar_file_uris': [],
        },
    }
    self.dataproc_expected_spark_sql_empty_fields_batch = (
        self.dataproc_test_spark_sql_empty_fields_batch.copy()
    )
    self.dataproc_expected_spark_sql_empty_fields_batch['labels'] = (
        _SYSTEM_LABELS
    )
    self.dataproc_test_spark_sql_empty_fields_removed_batch = {
        'spark_sql_batch': {'query_file_uri': 'test-query-file'}
    }
    self.dataproc_expected_spark_sql_empty_fields_removed_batch = (
        self.dataproc_test_spark_sql_empty_fields_removed_batch.copy()
    )
    self.dataproc_expected_spark_sql_empty_fields_removed_batch['labels'] = (
        _SYSTEM_LABELS
    )

  def tearDown(self):
    if os.path.exists(self._gcp_resources_path):
      os.remove(self._gcp_resources_path)

  @mock.patch.object(discovery, 'build', autospec=True)
  @mock.patch.object(ExecutionContext, '__init__', autospec=True)
  def test_cancel_job_on_non_completed_data_flow_job_succeeds(
      self, mock_execution_context, mock_build
  ):
    df_client = mock.Mock()
    mock_build.return_value = df_client
    expected_done_job = {'id': 'job-1', 'currentState': 'JOB_STATE_DONE'}
    expected_running_job = {'id': 'job-1', 'currentState': 'JOB_STATE_RUNNING'}
    get_request = mock.Mock()
    df_client.projects().locations().jobs().get.return_value = get_request
    # The first done_job is to end the polling loop.
    get_request.execute.side_effect = [
        expected_done_job,
        expected_running_job,
        expected_running_job,
    ]
    update_request = mock.Mock()
    df_client.projects().locations().jobs().update.return_value = update_request
    mock_execution_context.return_value = None
    expected_cancel_job = {
        'id': 'job-1',
        'currentState': 'JOB_STATE_RUNNING',
        'requestedState': 'JOB_STATE_CANCELLED',
    }

    wait_gcp_resources_remote_runner.wait_gcp_resources(
        self._type,
        self._project,
        self._location,
        self._dataflow_payload,
        self._gcp_resources_path,
    )

    cancel_job_remote_runner.cancel_job(
        self._dataflow_payload,
    )

    df_client.projects().locations().jobs().update.assert_called_once_with(
        projectId='foo',
        jobId='job123',
        location='us-central1',
        body=expected_cancel_job,
    )

  @mock.patch.object(aiplatform.gapic, 'JobServiceClient', autospec=True)
  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(ExecutionContext, '__init__', autospec=True)
  def test_cancel_job_on_non_completed_custom_job_succeeds(
      self,
      mock_execution_context,
      mock_post_requests,
      _,
      mock_auth,
      mock_job_service_client,
  ):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']

    job_client = mock.Mock()
    mock_job_service_client.return_value = job_client

    create_custom_job_response = mock.Mock()
    job_client.create_custom_job.return_value = create_custom_job_response
    create_custom_job_response.name = self._custom_job_name

    get_custom_job_response = mock.Mock()
    job_client.get_custom_job.return_value = get_custom_job_response
    get_custom_job_response.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED
    mock_execution_context.return_value = None

    custom_job_remote_runner.create_custom_job(
        self._type,
        self._project,
        self._location,
        self._custom_job_payload,
        self._gcp_resources_path,
    )

    # Call cancellation handler
    cancel_job_remote_runner.cancel_job(
        self._custom_job_payload,
    )
    mock_post_requests.assert_called_once_with(
        url=f'{self._custom_job_uri_prefix}{self._custom_job_name}:cancel',
        data='',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
        },
    )

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  @mock.patch.object(ExecutionContext, '__init__', autospec=True)
  def test_cancel_job_on_non_completed_big_query_succeeds(
      self,
      mock_execution_context,
      mock_time_sleep,
      mock_get_requests,
      mock_post_requests,
      _,
      mock_auth,
  ):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {
        'selfLink': self._big_query_job_uri
    }
    mock_post_requests.return_value = mock_created_bq_job
    mock_execution_context.return_value = None

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._big_query_job_uri,
        'status': {'state': 'DONE'},
        'configuration': {
            'query': {
                'destinationTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table',
                }
            }
        },
    }

    mock_get_requests.return_value = mock_polled_bq_job
    payload = (
        '{"configuration": {"query": {"query": "SELECT * FROM '
        '`bigquery-public-data.ml_datasets.penguins`"}}}'
    )
    executor_input = (
        '{"outputs":{"artifacts":{"destination_table":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"'
        + self._big_query_output_file_path
        + '"}}'
    )
    query_job_remote_runner.bigquery_query_job(
        self._big_query_job_type,
        self._big_query_project,
        self._big_query_location,
        payload,
        self._big_query_job_configuration_query_override,
        self._gcp_resources_path,
        executor_input,
    )

    gcp_resource = (
        '{"resources": [{"resourceType": "BigQueryJob","resourceUri":'
        ' "https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US"}]}'
    )

    # Call cancellation handler
    cancel_job_remote_runner.cancel_job(gcp_resource)

    self.assertEqual(mock_post_requests.call_count, 2)
    mock_post_requests.assert_called_with(
        url='https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job/cancel',
        data='',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
        },
    )

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'get', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'post', autospec=True)
  def test_cancel_job_on_non_completed_dataproc_job_succeeds(
      self,
      mock_post_requests,
      mock_get_requests,
      _,
      mock_auth_default,
  ):
    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self.dataproc_creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_operation = mock.Mock(spec=requests.models.Response)
    mock_operation.json.return_value = {
        'name': self.dataproc_operation_name,
        'metadata': {'batch': self.dataproc_batch_name},
    }
    mock_post_requests.return_value = mock_operation

    mock_polled_lro = mock.Mock(spec=requests.models.Response)
    mock_polled_lro.json.return_value = {
        'name': self.dataproc_operation_name,
        'done': True,
        'response': {'name': self.dataproc_batch_name, 'state': 'SUCCEEDED'},
    }
    mock_get_requests.return_value = mock_polled_lro

    dataproc_batch_remote_runner.create_spark_sql_batch(
        type='SparkSqlBatch',
        project=self.dataproc_project,
        location=self.dataproc_location,
        batch_id=self.dataproc_batch_id,
        payload=json.dumps(self.dataproc_test_spark_sql_batch),
        gcp_resources=self._gcp_resources_path,
    )

    mock_post_requests.assert_called_with(
        self=mock.ANY,
        url=f'{self.dataproc_uri_prefix}/projects/{self.dataproc_project}/locations/{self.dataproc_location}/batches/?batchId={self.dataproc_batch_id}',
        data=json.dumps(self.dataproc_expected_spark_sql_batch),
        headers={'Authorization': f'Bearer {self.dataproc_creds_token}'},
    )

    gcp_resources = (
        '{"resources": [{"resourceType": "DataprocLro","resourceUri": "'
        + self.dataproc_operation_uri
        + '"}]}'
    )

    cancel_job_remote_runner.cancel_job(gcp_resources)
    mock_post_requests.assert_called_with(
        self=mock.ANY,
        url=f'{self.dataproc_uri_prefix}/projects/{self.dataproc_project}/regions/{self.dataproc_location}/operations/{self.dataproc_operation_id}:cancel',
        data='',
        headers={'Authorization': f'Bearer {self.dataproc_creds_token}'},
    )

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  def test_cancel_job_on_non_completed_vertex_lro_job_succeeds(
      self, mock_post_requests, _, mock_auth
  ):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    create_endpoint_lro = mock.Mock()
    create_endpoint_lro.json.return_value = {
        'name': self.vertexlro_lro_name,
        'done': True,
        'response': {'name': self.vertexlro_endpoint_name},
    }
    mock_post_requests.return_value = create_endpoint_lro

    create_endpoint_remote_runner.create_endpoint(
        self.vertexlro_type,
        self.vertexlro_project,
        self.vertexlro_location,
        self.vertexlro_payload,
        self._gcp_resources_path,
        self.vertexlro_executor_input,
    )
    gcp_resources = (
        '{"resources": [{"resourceType": "VertexLro","resourceUri": "'
        + self.vertexlro_uri_prefix
        + self.vertexlro_lro_name
        + '"}]}'
    )

    # Call cancellation handler
    cancel_job_remote_runner.cancel_job(gcp_resources)
    self.assertEqual(mock_post_requests.call_count, 2)
    mock_post_requests.assert_called_with(
        url=f'{self.vertexlro_uri_prefix}{self.vertexlro_lro_name}:cancel',
        data='',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
        },
    )

  @mock.patch.object(discovery, 'build', autospec=True)
  def test_cancel_job_on_invalid_gcp_resource_type_fails(self, mock_build):
    invalid_payload = (
        '{"resources": [{"resourceType": "InvalidType",'
        '"resourceUri": "https://dataflow.googleapis.com/v1b3/'
        'projects/foo/locations/us-central1/jobs/job123"}]}'
    )
    with self.assertRaisesRegex(ValueError, r'Invalid gcp_resources:'):
      cancel_job_remote_runner.cancel_job(
          invalid_payload,
      )

  @mock.patch.object(discovery, 'build', autospec=True)
  def test_cancel_job_on_empty_gcp_resource_fails(self, mock_build):
    invalid_payload = '{"resources": [{}]}'
    with self.assertRaisesRegex(ValueError, r'Invalid gcp_resources:'):
      cancel_job_remote_runner.cancel_job(
          invalid_payload,
      )

  @mock.patch.object(discovery, 'build', autospec=True)
  def test_cancel_job_on_invalid_gcp_resource_uri_fails(self, mock_build):
    invalid_payload = (
        '{"resources": [{"resourceType": "DataflowJob",'
        '"resourceUri": "https://dataflow.googleapis.com/'
        'v1b3/projects/abc"}]}'
    )
    with self.assertRaisesRegex(ValueError, r'Invalid dataflow resource URI:'):
      cancel_job_remote_runner.cancel_job(invalid_payload)
