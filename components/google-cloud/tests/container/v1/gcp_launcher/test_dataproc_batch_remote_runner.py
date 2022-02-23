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
"""Test Dataproc Batch Remote Runner module."""

import json
import os
import time
from unittest import mock

import google.auth
from google_cloud_pipeline_components.container.utils.execution_context import ExecutionContext
from google_cloud_pipeline_components.container.v1.gcp_launcher import dataproc_batch_remote_runner
from google_cloud_pipeline_components.proto import gcp_resources_pb2
import requests

from google.protobuf import json_format
import unittest


class DataprocBatchRemoteRunnerUtilsTests(unittest.TestCase):
  """Test cases for Dataproc Batch remote runner."""

  def setUp(self):
    super(DataprocBatchRemoteRunnerUtilsTests, self).setUp()
    self._dataproc_uri_prefix = 'https://dataproc.googleapis.com/v1'
    self._project = 'test-project'
    self._location = 'test-location'
    self._batch_id = 'test-batch-id'
    self._creds_token = 'fake-token'
    self._operation_id = 'fake-operation-id'
    self._batch_name = f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}'
    self._batch_uri = f'{self._dataproc_uri_prefix}/{self._batch_name}'
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'gcp_resources')

    self._test_batch = {
        'labels': {'foo': 'bar', 'fizz': 'buzz'},
        'runtime_config': {
            'version': 'test-version',
            'container_image': 'test-container-image',
            'properties': {'foo': 'bar', 'fizz': 'buzz'}
        },
        'environment_config': {
            'execution_config': {
                'service_account': 'test-service-account',
                'network_tags': ['test-tag-1', 'test-tag-2'],
                'kms_key': 'test-kms-key',
                'network_uri': 'test-network-uri',
                'subnetwork_uri': 'test-subnetwork-uri'
            },
            'peripherals_config': {
                'metastore_service': 'test-metastore-service',
                'spark_history_server_config': {
                    'dataproc_cluster': 'test-dataproc-cluster'
                }
            }
        }
    }
    self._test_spark_batch = {
        **self._test_batch,
        'spark_batch': {
            'main_class': 'test-main-class',
            'jar_file_uris': ['test-jar-file-uri-1', 'test-jar-file-uri-2'],
            'file_uris': ['test-file-uri-1', 'test-file-uri-2'],
            'archive_uris': ['test-archive-uri-1', 'test-archive-uri-2'],
            'args': ['arg1', 'arg2']
        }
    }
    self._test_pyspark_batch = {
        **self._test_batch,
        'pyspark_batch': {
            'main_python_file_uri': 'test-python-file',
            'python_file_uris': ['test-python-file-uri-1', 'test-python-file-uri-2'],
            'jar_file_uris': ['test-jar-file-uri-1', 'test-jar-file-uri-2'],
            'file_uris': ['test-file-uri-1', 'test-file-uri-2'],
            'archive_uris': ['test-archive-uri-1', 'test-archive-uri-2'],
            'args': ['arg1', 'arg2']
        }
    }
    self._test_spark_r_batch = {
        **self._test_batch,
        'spark_r_batch': {
            'main_r_file_uri': 'test-r-file',
            'file_uris': ['test-file-uri-1', 'test-file-uri-2'],
            'archive_uris': ['test-archive-uri-1', 'test-archive-uri-2'],
            'args': ['arg1', 'arg2']
        }
    }
    self._test_spark_sql_batch = {
        **self._test_batch,
        'spark_sql_batch': {
            'query_file_uri': 'test-query-file',
            'query_variables': {'foo': 'bar', 'fizz': 'buzz'},
            'jar_file_uris': ['test-jar-file-uri-1', 'test-jar-file-uri-2'],
        }
    }
    self._test_spark_sql_empty_fields_batch = {
        'runtime_config': {
            'container_image': ''
        },
        'spark_sql_batch': {
            'query_file_uri': 'test-query-file',
            'query_variables': {},
            'jar_file_uris': [],
        }
    }
    self._test_spark_sql_empty_fields_removed_batch = {
        'spark_sql_batch': {
            'query_file_uri': 'test-query-file'
        }
    }

  def tearDown(self):
    super(DataprocBatchRemoteRunnerUtilsTests, self).tearDown()
    if os.path.exists(self._gcp_resources):
      os.remove(self._gcp_resources)

  def _validate_gcp_resources_succeeded(self):
    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      # Instantiate GCPResources Proto
      operations = json_format.Parse(serialized_gcp_resources,
                                          gcp_resources_pb2.GcpResources())
      self.assertLen(operations.resources, 1)
      self.assertEqual(
          operations.resources[0].resource_uri,
          f'{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}'
      )

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'get', autospec=True)
  def test_dataproc_batch_remote_runner_existing_operation_succeeded(self, mock_get_requests, _,
                                         mock_auth_default):
    with open(self._gcp_resources, 'w') as f:
      f.write(
          f'{{"resources": [{{"resourceType": "DataprocLro", "resourceUri": "{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}"}}]}}'
      )

    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_polled_lro = mock.Mock(spec=requests.models.Response)
    mock_polled_lro.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'done': True,
        'response': {
            'name': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}',
            'state': 'SUCCEEDED'
        }
    }
    mock_get_requests.return_value = mock_polled_lro

    dataproc_batch_remote_runner.create_spark_batch(
        type='SparkBatch',
        project=self._project,
        location=self._location,
        batch_id=self._batch_id,
        payload=json.dumps(self._test_spark_batch),
        gcp_resources=self._gcp_resources
    )

    mock_get_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'get', autospec=True)
  def test_dataproc_batch_remote_runner_existing_operation_failed(self, mock_get_requests, _,
                                                mock_auth_default):
    with open(self._gcp_resources, 'w') as f:
      f.write(
          f'{{"resources": [{{"resourceType": "DataprocLro", "resourceUri": "{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}"}}]}}'
      )

    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_polled_lro = mock.Mock(spec=requests.models.Response)
    mock_polled_lro.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'done': True,
        'error': {
            'code': 10
        }
    }
    mock_get_requests.return_value = mock_polled_lro

    with self.assertRaises(RuntimeError):
      dataproc_batch_remote_runner.create_spark_batch(
          type='SparkBatch',
          project=self._project,
          location=self._location,
          batch_id=self._batch_id,
          payload=json.dumps(self._test_spark_batch),
          gcp_resources=self._gcp_resources
      )

    mock_get_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'get', autospec=True)
  def test_dataproc_batch_remote_runner_existing_operation_not_found(self, mock_get_requests, _,
                                                   mock_auth_default):
    with open(self._gcp_resources, 'w') as f:
      f.write(
          f'{{"resources": [{{"resourceType": "DataprocLro", "resourceUri": "{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}"}}]}}'
      )

    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_polled_lro = mock.Mock(spec=requests.models.Response)
    mock_polled_lro.raise_for_status.side_effect = requests.exceptions.HTTPError(mock.Mock(status=404), 'Not found')
    mock_get_requests.return_value = mock_polled_lro

    with self.assertRaises(RuntimeError):
      dataproc_batch_remote_runner.create_spark_batch(
          type='SparkBatch',
          project=self._project,
          location=self._location,
          batch_id=self._batch_id,
          payload=json.dumps(self._test_spark_batch),
          gcp_resources=self._gcp_resources
      )

    mock_get_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )

  @mock.patch.object(google.auth, 'default', autospec=True)
  def test_dataproc_batch_remote_runner_operation_exists_wrong_format(self, mock_auth_default):
    with open(self._gcp_resources, 'w') as f:
      f.write(
          f'{{"resources": [{{"resourceType": "DataprocLro", "resourceUri": "{self._dataproc_uri_prefix}/v1/projects/test-project/regions/test-location/operations"}}]}}'
      )

    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    with self.assertRaises(ValueError):
      dataproc_batch_remote_runner.create_spark_batch(
          type='SparkBatch',
          project=self._project,
          location=self._location,
          batch_id=self._batch_id,
          payload=json.dumps(self._test_spark_batch),
          gcp_resources=self._gcp_resources
      )

  @mock.patch.object(google.auth, 'default', autospec=True)
  def test_dataproc_batch_remote_runner_exception_with_more_than_one_resource_in_gcp_resources(self, mock_auth_default):
    with open(self._gcp_resources, 'w') as f:
      f.write(
          (
              f'{{"resources": ['
              f'{{"resourceType": "DataprocLro", "resourceUri": "{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}"}},'
              f'{{"resourceType": "DataprocLro", "resourceUri": "{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}"}}'
              f']}}'
          )
      )

    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    with self.assertRaises(ValueError):
      dataproc_batch_remote_runner.create_spark_batch(
          type='SparkBatch',
          project=self._project,
          location=self._location,
          batch_id=self._batch_id,
          payload=json.dumps(self._test_spark_batch),
          gcp_resources=self._gcp_resources
      )

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'get', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'post', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_dataproc_batch_remote_runner_spark_batch_succeeded(self, mock_time_sleep,
                                                            mock_post_requests, mock_get_requests,
                                                            _, mock_auth_default):
    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_operation = mock.Mock(spec=requests.models.Response)
    mock_operation.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'metadata': {
            'batch': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}'
        }
    }
    mock_post_requests.return_value = mock_operation

    mock_polled_lro = mock.Mock(spec=requests.models.Response)
    mock_polled_lro.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'done': True,
        'response': {
            'name': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}',
            'state': 'SUCCEEDED'
        }
    }
    mock_get_requests.return_value = mock_polled_lro

    dataproc_batch_remote_runner.create_spark_batch(
        type='SparkBatch',
        project=self._project,
        location=self._location,
        batch_id=self._batch_id,
        payload=json.dumps(self._test_spark_batch),
        gcp_resources=self._gcp_resources
    )

    mock_post_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/locations/{self._location}/batches/?batchId={self._batch_id}',
        data=json.dumps(self._test_spark_batch),
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
    mock_get_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
    mock_time_sleep.assert_called_once()
    self._validate_gcp_resources_succeeded()

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'get', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'post', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_dataproc_batch_remote_runner_pyspark_batch_succeeded(self, mock_time_sleep,
                                                                mock_post_requests, mock_get_requests,
                                                                _, mock_auth_default):
    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_operation = mock.Mock(spec=requests.models.Response)
    mock_operation.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'metadata': {
            'batch': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}'
        }
    }
    mock_post_requests.return_value = mock_operation

    mock_polled_lro = mock.Mock(spec=requests.models.Response)
    mock_polled_lro.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'done': True,
        'response': {
            'name': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}',
            'state': 'SUCCEEDED'
        }
    }
    mock_get_requests.return_value = mock_polled_lro

    dataproc_batch_remote_runner.create_pyspark_batch(
        type='PySparkBatch',
        project=self._project,
        location=self._location,
        batch_id=self._batch_id,
        payload=json.dumps(self._test_pyspark_batch),
        gcp_resources=self._gcp_resources
    )

    mock_post_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/locations/{self._location}/batches/?batchId={self._batch_id}',
        data=json.dumps(self._test_pyspark_batch),
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
    mock_get_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
    mock_time_sleep.assert_called_once()
    self._validate_gcp_resources_succeeded()

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'get', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'post', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_dataproc_batch_remote_runner_spark_r_batch_succeeded(self, mock_time_sleep,
                                                                mock_post_requests, mock_get_requests,
                                                                _, mock_auth_default):
    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_operation = mock.Mock(spec=requests.models.Response)
    mock_operation.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'metadata': {
            'batch': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}'
        }
    }
    mock_post_requests.return_value = mock_operation

    mock_polled_lro = mock.Mock(spec=requests.models.Response)
    mock_polled_lro.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'done': True,
        'response': {
            'name': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}',
            'state': 'SUCCEEDED'
        }
    }
    mock_get_requests.return_value = mock_polled_lro

    dataproc_batch_remote_runner.create_spark_r_batch(
        type='SparkRBatch',
        project=self._project,
        location=self._location,
        batch_id=self._batch_id,
        payload=json.dumps(self._test_spark_r_batch),
        gcp_resources=self._gcp_resources
    )

    mock_post_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/locations/{self._location}/batches/?batchId={self._batch_id}',
        data=json.dumps(self._test_spark_r_batch),
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
    mock_get_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
    mock_time_sleep.assert_called_once()
    self._validate_gcp_resources_succeeded()

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'get', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'post', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_dataproc_batch_remote_runner_spark_sql_batch_succeeded(self, mock_time_sleep,
                                                                  mock_post_requests, mock_get_requests,
                                                                  _, mock_auth_default):
    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_operation = mock.Mock(spec=requests.models.Response)
    mock_operation.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'metadata': {
            'batch': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}'
        }
    }
    mock_post_requests.return_value = mock_operation

    mock_polled_lro = mock.Mock(spec=requests.models.Response)
    mock_polled_lro.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'done': True,
        'response': {
            'name': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}',
            'state': 'SUCCEEDED'
        }
    }
    mock_get_requests.return_value = mock_polled_lro

    dataproc_batch_remote_runner.create_spark_sql_batch(
        type='SparkSqlBatch',
        project=self._project,
        location=self._location,
        batch_id=self._batch_id,
        payload=json.dumps(self._test_spark_sql_batch),
        gcp_resources=self._gcp_resources
    )

    mock_post_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/locations/{self._location}/batches/?batchId={self._batch_id}',
        data=json.dumps(self._test_spark_sql_batch),
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
    mock_get_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
    mock_time_sleep.assert_called_once()
    self._validate_gcp_resources_succeeded()

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'get', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'post', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_dataproc_batch_remote_runner_spark_sql_batch_with_empty_fields_succeeded(self, mock_time_sleep,
                                                                                    mock_post_requests, mock_get_requests,
                                                                                    _, mock_auth_default):
    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_operation = mock.Mock(spec=requests.models.Response)
    mock_operation.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'metadata': {
            'batch': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}'
        }
    }
    mock_post_requests.return_value = mock_operation

    mock_polled_lro = mock.Mock(spec=requests.models.Response)
    mock_polled_lro.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'done': True,
        'response': {
            'name': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}',
            'state': 'SUCCEEDED'
        }
    }
    mock_get_requests.return_value = mock_polled_lro

    dataproc_batch_remote_runner.create_spark_sql_batch(
        type='SparkSqlBatch',
        project=self._project,
        location=self._location,
        batch_id=self._batch_id,
        payload=json.dumps(self._test_spark_sql_empty_fields_batch),
        gcp_resources=self._gcp_resources
    )

    mock_post_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/locations/{self._location}/batches/?batchId={self._batch_id}',
        data=json.dumps(self._test_spark_sql_empty_fields_removed_batch),
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
    mock_get_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
    mock_time_sleep.assert_called_once()
    self._validate_gcp_resources_succeeded()

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'get', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'post', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_dataproc_batch_remote_runner_spark_sql_batch_with_empty_fields_succeeded(self, mock_time_sleep,
                                                                                    mock_post_requests, mock_get_requests,
                                                                                    _, mock_auth_default):
    mock_creds = mock.Mock()
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_operation = mock.Mock(spec=requests.models.Response)
    mock_operation.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'metadata': {
            'batch': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}'
        }
    }
    mock_post_requests.return_value = mock_operation

    mock_polled_lro = mock.Mock()
    mock_polled_lro.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'done': True,
        'response': {
            'name': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}',
            'state': 'SUCCEEDED'
        }
    }
    mock_get_requests.return_value = mock_polled_lro

    dataproc_batch_remote_runner.create_spark_sql_batch(
        type='SparkSqlBatch',
        project=self._project,
        location=self._location,
        batch_id=self._batch_id,
        payload=json.dumps(self._test_spark_sql_empty_fields_batch),
        gcp_resources=self._gcp_resources
    )

    mock_post_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'https://dataproc.googleapis.com/v1/projects/{self._project}/locations/{self._location}/batches/?batchId={self._batch_id}',
        data=json.dumps(self._test_spark_sql_empty_fields_removed_batch),
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
    mock_get_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'https://dataproc.googleapis.com/v1/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
    mock_time_sleep.assert_called_once()
    self._validate_gcp_resources_succeeded()

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'get', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'post', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_dataproc_batch_remote_runner_spark_batch_exception_on_error(self, mock_time_sleep,
                                                                       mock_post_requests, mock_get_requests,
                                                                       _, mock_auth_default):
    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_operation = mock.Mock(spec=requests.models.Response)
    mock_operation.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'metadata': {
            'batch': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}'
        }
    }
    mock_post_requests.return_value = mock_operation

    mock_polled_lro = mock.Mock(spec=requests.models.Response)
    mock_polled_lro.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'done': True,
        'error': {
            'code': 10
        }
    }
    mock_get_requests.return_value = mock_polled_lro

    with self.assertRaises(RuntimeError):
      dataproc_batch_remote_runner.create_spark_batch(
          type='SparkBatch',
          project=self._project,
          location=self._location,
          batch_id=self._batch_id,
          payload=json.dumps(self._test_spark_batch),
          gcp_resources=self._gcp_resources
      )

    mock_post_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/locations/{self._location}/batches/?batchId={self._batch_id}',
        data=json.dumps(self._test_spark_batch),
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
    mock_get_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
    mock_time_sleep.assert_called_once()
    self._validate_gcp_resources_succeeded()

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_dataproc_batch_remote_runner_auth_token_refreshed(self, mock_time_sleep,
                                                             mock_get_requests,
                                                             _, mock_auth_default):
    with open(self._gcp_resources, 'w') as f:
      f.write(
          f'{{"resources": [{{"resourceType": "DataprocLro", "resourceUri": "{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}"}}]}}'
      )

    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = 'updated-auth-token'
    mock_creds.valid = False
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_polled_lro = mock.Mock(spec=requests.models.Response)
    mock_polled_lro.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'done': True,
        'response': {
            'name': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}',
            'state': 'SUCCEEDED'
        }
    }
    mock_get_requests.return_value = mock_polled_lro

    dataproc_batch_remote_runner.create_spark_sql_batch(
        type='SparkSqlBatch',
        project=self._project,
        location=self._location,
        batch_id=self._batch_id,
        payload=json.dumps(self._test_spark_sql_batch),
        gcp_resources=self._gcp_resources
    )
    mock_creds.refresh.assert_called_once_with(mock.ANY)
    mock_get_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        headers={'Authorization': f'Bearer updated-auth-token'})

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'get', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'post', autospec=True)
  @mock.patch.object(ExecutionContext, '__init__', autospec=True)
  def test_dataproc_batch_remote_runner_batch_cancel_succeeded(self,
                                                               mock_execution_context,
                                                               mock_post_requests,
                                                               mock_get_requests,
                                                               _, mock_auth_default):
    mock_execution_context.return_value = None

    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_operation = mock.Mock(spec=requests.models.Response)
    mock_operation.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'metadata': {
            'batch': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}'
        }
    }
    mock_post_requests.return_value = mock_operation

    mock_polled_lro = mock.Mock(spec=requests.models.Response)
    mock_polled_lro.json.return_value = {
        'name': f'projects/{self._project}/regions/{self._location}/operations/{self._operation_id}',
        'done': True,
        'response': {
            'name': f'projects/{self._project}/locations/{self._location}/batches/{self._batch_id}',
            'state': 'SUCCEEDED'
        }
    }
    mock_get_requests.return_value = mock_polled_lro

    dataproc_batch_remote_runner.create_spark_sql_batch(
        type='SparkSqlBatch',
        project=self._project,
        location=self._location,
        batch_id=self._batch_id,
        payload=json.dumps(self._test_spark_sql_batch),
        gcp_resources=self._gcp_resources
    )

    mock_post_requests.assert_called_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/locations/{self._location}/batches/?batchId={self._batch_id}',
        data=json.dumps(self._test_spark_sql_batch),
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )

    mock_execution_context.call_args[1]['on_cancel']()
    mock_post_requests.assert_called_with(
        self=mock.ANY,
        url=f'{self._dataproc_uri_prefix}/projects/{self._project}/regions/{self._location}/operations/{self._operation_id}:cancel',
        data='',
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
