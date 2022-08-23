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
"""Test BigQuery Query Job Remote Runner module."""

import json
from logging import raiseExceptions
import os
import time
import unittest
from unittest import mock
import requests
import google.auth
import google.auth.transport.requests

from google.protobuf import json_format
from google_cloud_pipeline_components.container.utils.execution_context import ExecutionContext
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
from google_cloud_pipeline_components.container.v1.gcp_launcher import bigquery_job_remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher import job_remote_runner


class BigqueryQueryJobRemoteRunnerUtilsTests(unittest.TestCase):

  def setUp(self):
    super(BigqueryQueryJobRemoteRunnerUtilsTests, self).setUp()
    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._job_configuration_query_override = '{}'
    self._job_type = 'BigqueryQueryJob'
    self._project = 'test_project'
    self._location = 'US'
    self._job_uri = 'https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US'
    self._model_name = 'testproject.testdataset.testmodel'
    self._model_destination_path = 'gs://testproject/testmodelpah'
    self._exported_model_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'exported_model_path')
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'gcp_resources')
    self._output_file_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'localpath/foo')
    self._executor_input = '{"outputs":{"artifacts":{"destination_table":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' + self._output_file_path + '"}}'

  def tearDown(self):
    if os.path.exists(self._gcp_resources):
      os.remove(self._gcp_resources)

    super(BigqueryQueryJobRemoteRunnerUtilsTests, self).tearDown()

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_query_job_succeeded(self, mock_time_sleep, mock_get_requests,
                               mock_post_requests, _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_created_bq_job

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'configuration': {
            'query': {
                'destinationTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_bq_job
    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"destination_table":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' + self._output_file_path + '"}}'
    bigquery_job_remote_runner.bigquery_query_job(
        self._job_type, self._project, self._location, self._payload,
        self._job_configuration_query_override, self._gcp_resources,
        self._executor_input)
    mock_post_requests.assert_called_once_with(
        url=f'https://www.googleapis.com/bigquery/v2/projects/{self._project}/jobs',
        data=(
            '{"configuration": {"query": {"query": "SELECT * FROM `bigquery-public-data.ml_datasets.penguins`", "useLegacySql": false}}, "jobReference": {"location": "US"}}'
        ),
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
            'User-Agent': 'google-cloud-pipeline-components'
        })

    with open(self._output_file_path) as f:
      self.assertEqual(
          f.read(),
          '{"artifacts": {"destination_table": {"artifacts": [{"metadata": {"projectId": "test_project", "datasetId": "test_dataset", "tableId": "test_table"}, "name": "foobar", "type": {"schemaTitle": "google.BQTable"}, "uri": "https://www.googleapis.com/bigquery/v2/projects/test_project/datasets/test_dataset/tables/test_table"}]}}}'
      )

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      # Instantiate GCPResources Proto
      bq_job_resources = json_format.Parse(serialized_gcp_resources,
                                           GcpResources())
      self.assertLen(bq_job_resources.resources, 1)
      self.assertEqual(
          bq_job_resources.resources[0].resource_uri,
          'https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US'
      )

    self.assertEqual(mock_post_requests.call_count, 1)
    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_query_job_with_job_config_override_succeeded(self, mock_time_sleep,
                                                        mock_get_requests,
                                                        mock_post_requests, _,
                                                        mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_created_bq_job

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'configuration': {
            'query': {
                'destinationTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_bq_job
    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"destination_table":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    job_configuration_query_override = ('{"query":"SELECT * FROM foo", '
                                        '"query_parameters": "abc"}')
    bigquery_job_remote_runner.bigquery_query_job(
        self._job_type, self._project, self._location, self._payload,
        job_configuration_query_override, self._gcp_resources,
        self._executor_input)
    mock_post_requests.assert_called_once_with(
        url=f'https://www.googleapis.com/bigquery/v2/projects/{self._project}/jobs',
        data=(
            '{"configuration": {"query": {"query": "SELECT * FROM foo", "query_parameters": "abc", "useLegacySql": false}}, "jobReference": {"location": "US"}}'
        ),
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
            'User-Agent': 'google-cloud-pipeline-components'
        })

    with open(self._output_file_path) as f:
      self.assertEqual(
          f.read(),
          '{"artifacts": {"destination_table": {"artifacts": [{"metadata": {"projectId": "test_project", "datasetId": "test_dataset", "tableId": "test_table"}, "name": "foobar", "type": {"schemaTitle": "google.BQTable"}, "uri": "https://www.googleapis.com/bigquery/v2/projects/test_project/datasets/test_dataset/tables/test_table"}]}}}'
      )

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      # Instantiate GCPResources Proto
      bq_job_resources = json_format.Parse(serialized_gcp_resources,
                                           GcpResources())
      self.assertLen(bq_job_resources.resources, 1)
      self.assertEqual(
          bq_job_resources.resources[0].resource_uri,
          'https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US'
      )

    self.assertEqual(mock_post_requests.call_count, 1)
    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_query_job_poll_existing_job_succeeded(self, mock_time_sleep,
                                                 mock_get_requests, _,
                                                 mock_auth):
    # Mimic the case that self._gcp_resources already stores the job uri.
    with open(self._gcp_resources, 'w') as f:
      f.write(
          '{"resources": [{"resourceType": "BigqueryQueryJob", "resourceUri": "https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US"}]}'
      )

    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'configuration': {
            'query': {
                'destinationTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_bq_job
    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"destination_table":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    bigquery_job_remote_runner.bigquery_query_job(
        self._job_type, self._project, self._location, self._payload,
        self._job_configuration_query_override, self._gcp_resources,
        self._executor_input)

    with open(self._output_file_path) as f:
      self.assertEqual(
          f.read(),
          '{"artifacts": {"destination_table": {"artifacts": [{"metadata": {"projectId": "test_project", "datasetId": "test_dataset", "tableId": "test_table"}, "name": "foobar", "type": {"schemaTitle": "google.BQTable"}, "uri": "https://www.googleapis.com/bigquery/v2/projects/test_project/datasets/test_dataset/tables/test_table"}]}}}'
      )

    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  def test_query_job_check_job_exists_wrong_format(self, _, mock_auth):
    # Mimic the case that self._gcp_resources already stores the job uri.
    with open(self._gcp_resources, 'w') as f:
      f.write(
          '{"resources": [{"resourceType": "BigqueryQueryJob", "resourceUri": "https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job_no_location"}]}'
      )

    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"destination_table":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    with self.assertRaises(ValueError):
      bigquery_job_remote_runner.bigquery_query_job(
          self._job_type, self._project, self._location, self._payload,
          self._job_configuration_query_override, self._gcp_resources,
          self._executor_input)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  def test_query_job_failed_no_selflink(self, mock_post_requests, _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {}
    mock_post_requests.return_value = mock_created_bq_job
    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"destination_table":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    with self.assertRaises(RuntimeError):
      bigquery_job_remote_runner.bigquery_query_job(
          self._job_type, self._project, self._location, self._payload,
          self._job_configuration_query_override, self._gcp_resources,
          self._executor_input)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_query_job_poll_existing_job_warning(self, mock_time_sleep,
                                               mock_get_requests, _, mock_auth):
    # Mimic the case that self._gcp_resources already stores the job uri.
    with open(self._gcp_resources, 'w') as f:
      f.write(
          '{"resources": [{"resourceType": "BigqueryQueryJob", "resourceUri": "https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US"}]}'
      )

    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state':
                'DONE',
            'errors': [{
                'reason': 'invalidQuery',
                'location': 'query',
                'message':
                    'The input data has NULL values in one or more columns: '
                    'sex. BQML automatically handles null values (See '
                    'https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-create#imputation).'
                    ' If null values represent a special value in the data, '
                    'replace them with the desired value before training and '
                    'then retry.'
            }],
        },
        'configuration': {
            'query': {
                'destinationTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_bq_job
    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"destination_table":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    bigquery_job_remote_runner.bigquery_query_job(
        self._job_type, self._project, self._location, self._payload,
        self._job_configuration_query_override, self._gcp_resources,
        self._executor_input)

    with open(self._output_file_path) as f:
      self.assertEqual(
          f.read(),
          '{"artifacts": {"destination_table": {"artifacts": [{"metadata": {"projectId": "test_project", "datasetId": "test_dataset", "tableId": "test_table"}, "name": "foobar", "type": {"schemaTitle": "google.BQTable"}, "uri": "https://www.googleapis.com/bigquery/v2/projects/test_project/datasets/test_dataset/tables/test_table"}]}}}'
      )

    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_query_job_poll_existing_job_failed(self, mock_time_sleep,
                                              mock_get_requests, _, mock_auth):
    # Mimic the case that self._gcp_resources already stores the job uri.
    with open(self._gcp_resources, 'w') as f:
      f.write(
          '{"resources": [{"resourceType": "BigqueryQueryJob", "resourceUri": "https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US"}]}'
      )

    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE',
            'errorResult': {
                'foo': 'bar'
            }
        },
        'configuration': {
            'query': {
                'destinationTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_bq_job
    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"destination_table":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    with self.assertRaises(RuntimeError):
      bigquery_job_remote_runner.bigquery_query_job(
          self._job_type, self._project, self._location, self._payload,
          self._job_configuration_query_override, self._gcp_resources,
          self._executor_input)

    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  @mock.patch.object(ExecutionContext, '__init__', autospec=True)
  def test_query_job_cancel(self, mock_execution_context, mock_time_sleep,
                            mock_get_requests, mock_post_requests, _,
                            mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_created_bq_job
    mock_execution_context.return_value = None

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'configuration': {
            'query': {
                'destinationTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_bq_job
    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"destination_table":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' + self._output_file_path + '"}}'
    bigquery_job_remote_runner.bigquery_query_job(
        self._job_type, self._project, self._location, self._payload,
        self._job_configuration_query_override, self._gcp_resources,
        self._executor_input)
    # Call cancellation handler
    mock_execution_context.call_args[1]['on_cancel']()
    self.assertEqual(mock_post_requests.call_count, 2)
    mock_post_requests.assert_called_with(
        url='https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job/cancel',
        data='',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
        })

  # Tests for create model
  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_create_model_job_succeeded(self, mock_time_sleep, mock_get_requests,
                                      mock_post_requests, _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_created_bq_job

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'statistics': {
            'query': {
                'statementType': 'CREATE_MODEL',
                'ddlOperationPerformed': 'REPLACE',
                'ddlTargetTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_bq_job
    self._payload = (
        '{"configuration": {"query": {"query": "CREATE OR REPLACE MODEL '
        'bqml_tutorial.penguins_model OPTIONS (model_type=\'linear_reg\', '
        'input_label_cols=[\'body_mass_g\']) AS SELECT * FROM '
        '`bigquery-public-data.ml_datasets.penguins` WHERE body_mass_g IS NOT '
        'NULL"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"model":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQMLModel"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    bigquery_job_remote_runner.bigquery_create_model_job(
        self._job_type, self._project, self._location, self._payload,
        self._job_configuration_query_override, self._gcp_resources,
        self._executor_input)
    mock_post_requests.assert_called_once_with(
        url=f'https://www.googleapis.com/bigquery/v2/projects/{self._project}/jobs',
        data=(
            '{"configuration": {"query": {"query": "CREATE OR REPLACE MODEL bqml_tutorial.penguins_model OPTIONS (model_type=\'linear_reg\', input_label_cols=[\'body_mass_g\']) AS SELECT * FROM `bigquery-public-data.ml_datasets.penguins` WHERE body_mass_g IS NOT NULL", "useLegacySql": false}}, "jobReference": {"location": "US"}}'
        ),
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
            'User-Agent': 'google-cloud-pipeline-components'
        })

    with open(self._output_file_path) as f:
      self.assertEqual(
          f.read(),
          '{"artifacts": {"model": {"artifacts": [{"metadata": {"projectId": "test_project", "datasetId": "test_dataset", "modelId": "test_table"}, "name": "foobar", "type": {"schemaTitle": "google.BQMLModel"}, "uri": "https://www.googleapis.com/bigquery/v2/projects/test_project/datasets/test_dataset/models/test_table"}]}}}'
      )

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      # Instantiate GCPResources Proto
      bq_job_resources = json_format.Parse(serialized_gcp_resources,
                                           GcpResources())
      self.assertLen(bq_job_resources.resources, 1)
      self.assertEqual(
          bq_job_resources.resources[0].resource_uri,
          'https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US'
      )

    self.assertEqual(mock_post_requests.call_count, 1)
    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_create_model_job_with_job_config_override_succeeded(
      self, mock_time_sleep, mock_get_requests, mock_post_requests, _,
      mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_created_bq_job

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'statistics': {
            'query': {
                'statementType': 'CREATE_MODEL',
                'ddlOperationPerformed': 'REPLACE',
                'ddlTargetTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_bq_job
    self._payload = (
        '{"configuration": {"query": {"query": "CREATE OR REPLACE MODEL '
        'bqml_tutorial.penguins_model OPTIONS (model_type=\'linear_reg\', '
        'input_label_cols=[\'body_mass_g\']) AS SELECT * FROM '
        '`bigquery-public-data.ml_datasets.penguins` WHERE body_mass_g IS NOT '
        'NULL"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"model":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQMLModel"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    job_configuration_query_override = ('{"query":"SELECT * FROM foo", '
                                        '"query_parameters": "abc"}')
    bigquery_job_remote_runner.bigquery_create_model_job(
        self._job_type, self._project, self._location, self._payload,
        job_configuration_query_override, self._gcp_resources,
        self._executor_input)
    mock_post_requests.assert_called_once_with(
        url=f'https://www.googleapis.com/bigquery/v2/projects/{self._project}/jobs',
        data=(
            '{"configuration": {"query": {"query": "SELECT * FROM foo", "query_parameters": "abc", "useLegacySql": false}}, "jobReference": {"location": "US"}}'
        ),
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
            'User-Agent': 'google-cloud-pipeline-components'
        })

    with open(self._output_file_path) as f:
      self.assertEqual(
          f.read(),
          '{"artifacts": {"model": {"artifacts": [{"metadata": {"projectId": "test_project", "datasetId": "test_dataset", "modelId": "test_table"}, "name": "foobar", "type": {"schemaTitle": "google.BQMLModel"}, "uri": "https://www.googleapis.com/bigquery/v2/projects/test_project/datasets/test_dataset/models/test_table"}]}}}'
      )

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      # Instantiate GCPResources Proto
      bq_job_resources = json_format.Parse(serialized_gcp_resources,
                                           GcpResources())
      self.assertLen(bq_job_resources.resources, 1)
      self.assertEqual(
          bq_job_resources.resources[0].resource_uri,
          'https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US'
      )

    self.assertEqual(mock_post_requests.call_count, 1)
    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_create_model_job_poll_existing_job_succeeded(self, mock_time_sleep,
                                                        mock_get_requests, _,
                                                        mock_auth):
    # Mimic the case that self._gcp_resources already stores the job uri.
    with open(self._gcp_resources, 'w') as f:
      f.write(
          '{"resources": [{"resourceType": "BigqueryQueryJob", "resourceUri": "https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US"}]}'
      )

    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'statistics': {
            'query': {
                'statementType': 'CREATE_MODEL',
                'ddlOperationPerformed': 'REPLACE',
                'ddlTargetTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_bq_job
    self._payload = (
        '{"configuration": {"query": {"query": "CREATE OR REPLACE MODEL '
        'bqml_tutorial.penguins_model OPTIONS (model_type=\'linear_reg\', '
        'input_label_cols=[\'body_mass_g\']) AS SELECT * FROM '
        '`bigquery-public-data.ml_datasets.penguins` WHERE body_mass_g IS NOT '
        'NULL"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"model":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQMLModel"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    bigquery_job_remote_runner.bigquery_create_model_job(
        self._job_type, self._project, self._location, self._payload,
        self._job_configuration_query_override, self._gcp_resources,
        self._executor_input)

    with open(self._output_file_path) as f:
      self.assertEqual(
          f.read(),
          '{"artifacts": {"model": {"artifacts": [{"metadata": {"projectId": "test_project", "datasetId": "test_dataset", "modelId": "test_table"}, "name": "foobar", "type": {"schemaTitle": "google.BQMLModel"}, "uri": "https://www.googleapis.com/bigquery/v2/projects/test_project/datasets/test_dataset/models/test_table"}]}}}'
      )

    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  def test_create_model_job_check_job_exists_wrong_format(self, _, mock_auth):
    # Mimic the case that self._gcp_resources already stores the job uri.
    with open(self._gcp_resources, 'w') as f:
      f.write(
          '{"resources": [{"resourceType": "BigqueryQueryJob", "resourceUri": "https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job_no_location"}]}'
      )

    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    self._payload = (
        '{"configuration": {"query": {"query": "CREATE OR REPLACE MODEL '
        'bqml_tutorial.penguins_model OPTIONS (model_type=\'linear_reg\', '
        'input_label_cols=[\'body_mass_g\']) AS SELECT * FROM '
        '`bigquery-public-data.ml_datasets.penguins` WHERE body_mass_g IS NOT '
        'NULL"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"model":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQMLModel"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    with self.assertRaises(ValueError):
      bigquery_job_remote_runner.bigquery_create_model_job(
          self._job_type, self._project, self._location, self._payload,
          self._job_configuration_query_override, self._gcp_resources,
          self._executor_input)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  def test_create_model_job_failed_no_selflink(self, mock_post_requests, _,
                                               mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {}
    mock_post_requests.return_value = mock_created_bq_job
    self._payload = (
        '{"configuration": {"query": {"query": "CREATE OR REPLACE MODEL '
        'bqml_tutorial.penguins_model OPTIONS (model_type=\'linear_reg\', '
        'input_label_cols=[\'body_mass_g\']) AS SELECT * FROM '
        '`bigquery-public-data.ml_datasets.penguins` WHERE body_mass_g IS NOT '
        'NULL"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"model":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQMLModel"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    with self.assertRaises(RuntimeError):
      bigquery_job_remote_runner.bigquery_create_model_job(
          self._job_type, self._project, self._location, self._payload,
          self._job_configuration_query_override, self._gcp_resources,
          self._executor_input)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_create_model_job_poll_existing_job_failed(self, mock_time_sleep,
                                                     mock_get_requests, _,
                                                     mock_auth):
    # Mimic the case that self._gcp_resources already stores the job uri.
    with open(self._gcp_resources, 'w') as f:
      f.write(
          '{"resources": [{"resourceType": "BigqueryQueryJob", "resourceUri": "https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US"}]}'
      )

    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'errorResult': {
                'reason': 'invalidQuery',
                'location': 'query',
                'message':
                    'The input data has NULL values in one or more columns: '
                    'sex. BQML automatically handles null values (See '
                    'https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-create#imputation).'
                    ' If null values represent a special value in the data, '
                    'replace them with the desired value before training and '
                    'then retry.'
            },
            'state': 'DONE'
        },
        'statistics': {
            'query': {
                'statementType': 'CREATE_MODEL',
                'ddlOperationPerformed': 'REPLACE',
                'ddlTargetTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_bq_job
    self._payload = (
        '{"configuration": {"query": {"query": "CREATE OR REPLACE MODEL '
        'bqml_tutorial.penguins_model OPTIONS (model_type=\'linear_reg\', '
        'input_label_cols=[\'body_mass_g\']) AS SELECT * FROM '
        '`bigquery-public-data.ml_datasets.penguins` WHERE body_mass_g IS NOT '
        'NULL"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"model":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQMLModel"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    with self.assertRaises(RuntimeError):
      bigquery_job_remote_runner.bigquery_create_model_job(
          self._job_type, self._project, self._location, self._payload,
          self._job_configuration_query_override, self._gcp_resources,
          self._executor_input)

    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_create_model_job_failed_no_query_result(self, mock_time_sleep,
                                                   mock_get_requests,
                                                   mock_post_requests, _,
                                                   mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_created_bq_job

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'statistics': {
            'query': {
                'statementType': 'CREATE_MODEL',
                'ddlOperationPerformed': 'REPLACE',
            }
        }
    }
    mock_get_requests.return_value = mock_polled_bq_job
    self._payload = (
        '{"configuration": {"query": {"query": "CREATE OR REPLACE MODEL '
        'bqml_tutorial.penguins_model OPTIONS (model_type=\'linear_reg\', '
        'input_label_cols=[\'body_mass_g\']) AS SELECT * FROM '
        '`bigquery-public-data.ml_datasets.penguins` WHERE body_mass_g IS NOT '
        'NULL"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"model":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQMLModel"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    with self.assertRaises(RuntimeError):
      bigquery_job_remote_runner.bigquery_create_model_job(
          self._job_type, self._project, self._location, self._payload,
          self._job_configuration_query_override, self._gcp_resources,
          self._executor_input)

    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_create_model_job_failed_not_create_model(self, mock_time_sleep,
                                                    mock_get_requests,
                                                    mock_post_requests, _,
                                                    mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_created_bq_job

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'statistics': {
            'query': {
                'statementType': 'CREATE_TABLE',
                'ddlOperationPerformed': 'REPLACE',
                'ddlTargetTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_bq_job
    self._payload = (
        '{"configuration": {"query": {"query": "CREATE OR REPLACE MODEL '
        'bqml_tutorial.penguins_model OPTIONS (model_type=\'linear_reg\', '
        'input_label_cols=[\'body_mass_g\']) AS SELECT * FROM '
        '`bigquery-public-data.ml_datasets.penguins` WHERE body_mass_g IS NOT '
        'NULL"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"model":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQMLModel"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    with self.assertRaises(RuntimeError):
      bigquery_job_remote_runner.bigquery_create_model_job(
          self._job_type, self._project, self._location, self._payload,
          self._job_configuration_query_override, self._gcp_resources,
          self._executor_input)

    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  @mock.patch.object(ExecutionContext, '__init__', autospec=True)
  def test_create_model_job_cancel(self, mock_execution_context, mock_time_sleep,
                            mock_get_requests, mock_post_requests, _,
                            mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_created_bq_job
    mock_execution_context.return_value = None

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'statistics': {
            'query': {
                'statementType': 'CREATE_MODEL',
                'ddlOperationPerformed': 'REPLACE',
                'ddlTargetTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_bq_job
    self._payload = (
        '{"configuration": {"query": {"query": "CREATE OR REPLACE MODEL '
        'bqml_tutorial.penguins_model OPTIONS (model_type=\'linear_reg\', '
        'input_label_cols=[\'body_mass_g\']) AS SELECT * FROM '
        '`bigquery-public-data.ml_datasets.penguins` WHERE body_mass_g IS NOT '
        'NULL"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"model":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQMLModel"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    bigquery_job_remote_runner.bigquery_create_model_job(
        self._job_type, self._project, self._location, self._payload,
        self._job_configuration_query_override, self._gcp_resources,
        self._executor_input)

    # Call cancellation handler
    mock_execution_context.call_args[1]['on_cancel']()
    self.assertEqual(mock_post_requests.call_count, 2)
    mock_post_requests.assert_called_with(
        url='https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job/cancel',
        data='',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
        })

  # Tests for predict model job using query.
  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_predict_model_job_query_succeeded(self, mock_time_sleep,
                                             mock_get_requests,
                                             mock_post_requests, _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_predict_model_job = mock.Mock()
    mock_predict_model_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_predict_model_job

    mock_polled_predict_model_job = mock.Mock()
    mock_polled_predict_model_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'configuration': {
            'query': {
                'destinationTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_predict_model_job
    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"destination_table":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' + self._output_file_path + '"}}'
    self._model_name = 'bqml_tutorial.penguins_model'
    self._table_name = None
    self._query_statement = ('SELECT * FROM '
                             '`bigquery-public-data.ml_datasets.penguins`')
    self._threshold = None
    bigquery_job_remote_runner.bigquery_predict_model_job(
        self._job_type, self._project, self._location, self._model_name,
        self._table_name, self._query_statement, self._threshold, self._payload,
        self._job_configuration_query_override, self._gcp_resources,
        self._executor_input)
    mock_post_requests.assert_called_once_with(
        url=f'https://www.googleapis.com/bigquery/v2/projects/{self._project}/jobs',
        data=(
            '{"configuration": {"query": {"query": "SELECT * FROM ML.PREDICT(MODEL `bqml_tutorial.penguins_model`, (SELECT * FROM `bigquery-public-data.ml_datasets.penguins`))", "useLegacySql": false}}, "jobReference": {"location": "US"}}'
        ),
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
            'User-Agent': 'google-cloud-pipeline-components'
        })

    with open(self._output_file_path) as f:
      self.assertEqual(
          f.read(),
          '{"artifacts": {"destination_table": {"artifacts": [{"metadata": {"projectId": "test_project", "datasetId": "test_dataset", "tableId": "test_table"}, "name": "foobar", "type": {"schemaTitle": "google.BQTable"}, "uri": "https://www.googleapis.com/bigquery/v2/projects/test_project/datasets/test_dataset/tables/test_table"}]}}}'
      )

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      # Instantiate GCPResources Proto
      bq_job_resources = json_format.Parse(serialized_gcp_resources,
                                           GcpResources())
      self.assertLen(bq_job_resources.resources, 1)
      self.assertEqual(
          bq_job_resources.resources[0].resource_uri,
          'https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US'
      )

    self.assertEqual(mock_post_requests.call_count, 1)
    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  # Tests for predict model job using table.
  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_predict_model_table_job_succeeded(self, mock_time_sleep,
                                             mock_get_requests,
                                             mock_post_requests, _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_predict_model_job = mock.Mock()
    mock_predict_model_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_predict_model_job

    mock_polled_predict_model_job = mock.Mock()
    mock_polled_predict_model_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'configuration': {
            'query': {
                'destinationTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_predict_model_job
    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"destination_table":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' + self._output_file_path + '"}}'
    self._model_name = 'bqml_tutorial.penguins_model'
    self._table_name = 'bigquery-public-data.ml_datasets.penguins'
    self._query_statement = None
    self._threshold = None
    bigquery_job_remote_runner.bigquery_predict_model_job(
        self._job_type, self._project, self._location, self._model_name,
        self._table_name, self._query_statement, self._threshold, self._payload,
        self._job_configuration_query_override, self._gcp_resources,
        self._executor_input)
    mock_post_requests.assert_called_once_with(
        url=f'https://www.googleapis.com/bigquery/v2/projects/{self._project}/jobs',
        data=(
            '{"configuration": {"query": {"query": "SELECT * FROM ML.PREDICT(MODEL `bqml_tutorial.penguins_model`, TABLE `bigquery-public-data.ml_datasets.penguins`)", "useLegacySql": false}}, "jobReference": {"location": "US"}}'
        ),
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
            'User-Agent': 'google-cloud-pipeline-components'
        })

    with open(self._output_file_path) as f:
      self.assertEqual(
          f.read(),
          '{"artifacts": {"destination_table": {"artifacts": [{"metadata": {"projectId": "test_project", "datasetId": "test_dataset", "tableId": "test_table"}, "name": "foobar", "type": {"schemaTitle": "google.BQTable"}, "uri": "https://www.googleapis.com/bigquery/v2/projects/test_project/datasets/test_dataset/tables/test_table"}]}}}'
      )

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      # Instantiate GCPResources Proto
      bq_job_resources = json_format.Parse(serialized_gcp_resources,
                                           GcpResources())
      self.assertLen(bq_job_resources.resources, 1)
      self.assertEqual(
          bq_job_resources.resources[0].resource_uri,
          'https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US'
      )

    self.assertEqual(mock_post_requests.call_count, 1)
    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  @mock.patch.object(ExecutionContext, '__init__', autospec=True)
  def test_predict_model_job_query_cancel(self, mock_execution_context,
                                          mock_time_sleep,
                                          mock_get_requests, mock_post_requests,
                                          _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_predict_model_job = mock.Mock()
    mock_predict_model_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_predict_model_job
    mock_execution_context.return_value = None

    mock_polled_predict_model_job = mock.Mock()
    mock_polled_predict_model_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'configuration': {
            'query': {
                'destinationTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_predict_model_job
    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"destination_table":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' + self._output_file_path + '"}}'
    self._model_name = 'bqml_tutorial.penguins_model'
    self._table_name = None
    self._query_statement = ('SELECT * FROM '
                             '`bigquery-public-data.ml_datasets.penguins`')
    self._threshold = None
    bigquery_job_remote_runner.bigquery_predict_model_job(
        self._job_type, self._project, self._location, self._model_name,
        self._table_name, self._query_statement, self._threshold, self._payload,
        self._job_configuration_query_override, self._gcp_resources,
        self._executor_input)

    # Call cancellation handler
    mock_execution_context.call_args[1]['on_cancel']()
    self.assertEqual(mock_post_requests.call_count, 2)
    mock_post_requests.assert_called_with(
        url='https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job/cancel',
        data='',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
        })

  # Tests for Export Model
  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_export_model_job_succeeded_non_XGBoost(self, mock_time_sleep,
                                                  mock_get_requests,
                                                  mock_post_requests, _,
                                                  mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_created_bq_job

    mock_get_model_resp = mock.Mock()
    mock_get_model_resp.json.return_value = {'modelType': 'LINEAR_REGRESSION'}

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'configuration': {
            'extract': {
                'destinationUris': [self._model_destination_path],
            }
        }
    }
    mock_get_requests.side_effect = [mock_get_model_resp, mock_polled_bq_job]
    self._payload = '{"configuration": {"extract": {}, "labels": {}}}'
    self._executor_input = '{"outputs":{"artifacts":{"model_destination_path":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQMLModel"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    bigquery_job_remote_runner.bigquery_export_model_job(
        self._job_type, self._project, self._location, self._model_name,
        self._model_destination_path, self._payload,
        self._exported_model_path, self._gcp_resources,
        self._executor_input)

    mock_post_requests.assert_called_once_with(
        data='{"configuration": {"extract": {"sourceModel": {"projectId": "testproject", "datasetId": "testdataset", "modelId": "testmodel"}, "destinationUris": ["gs://testproject/testmodelpah"]}, "labels": {}}, "jobReference": {"location": "US"}}',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
            'User-Agent': 'google-cloud-pipeline-components'
        },
        url='https://www.googleapis.com/bigquery/v2/projects/test_project/jobs')

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      # Instantiate GCPResources Proto
      bq_job_resources = json_format.Parse(serialized_gcp_resources,
                                           GcpResources())
      self.assertLen(bq_job_resources.resources, 1)
      self.assertEqual(
          bq_job_resources.resources[0].resource_uri,
          'https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US'
      )

    with open(self._exported_model_path) as f:
      self.assertEqual(f.read(), self._model_destination_path)

    self.assertEqual(mock_post_requests.call_count, 1)
    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 2)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_export_model_job_succeeded_XGBoost(self, mock_time_sleep,
                                              mock_get_requests,
                                              mock_post_requests, _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_created_bq_job

    mock_get_model_resp = mock.Mock()
    mock_get_model_resp.json.return_value = {
        'modelType': 'BOOSTED_TREE_REGRESSOR'
    }

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'configuration': {
            'extract': {
                'destinationUris': [self._model_destination_path],
            }
        }
    }
    mock_get_requests.side_effect = [mock_get_model_resp, mock_polled_bq_job]
    self._payload = '{"configuration": {"extract": {}, "labels": {}}}'
    self._executor_input = '{"outputs":{"artifacts":{"model_destination_path":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQMLModel"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    bigquery_job_remote_runner.bigquery_export_model_job(
        self._job_type, self._project, self._location, self._model_name,
        self._model_destination_path, self._payload,
        self._exported_model_path, self._gcp_resources,
        self._executor_input)

    mock_post_requests.assert_called_once_with(
        data='{"configuration": {"extract": {"sourceModel": {"projectId": "testproject", "datasetId": "testdataset", "modelId": "testmodel"}, "destinationFormat": "ML_XGBOOST_BOOSTER", "destinationUris": ["gs://testproject/testmodelpah"]}, "labels": {}}, "jobReference": {"location": "US"}}',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
            'User-Agent': 'google-cloud-pipeline-components'
        },
        url='https://www.googleapis.com/bigquery/v2/projects/test_project/jobs')

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      # Instantiate GCPResources Proto
      bq_job_resources = json_format.Parse(serialized_gcp_resources,
                                           GcpResources())
      self.assertLen(bq_job_resources.resources, 1)
      self.assertEqual(
          bq_job_resources.resources[0].resource_uri,
          'https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US'
      )

    with open(self._exported_model_path) as f:
      self.assertEqual(f.read(), self._model_destination_path)

    self.assertEqual(mock_post_requests.call_count, 1)
    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 2)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_export_model_job_poll_existing_job_succeeded(self, mock_time_sleep,
                                                        mock_get_requests, _,
                                                        mock_auth):
    # Mimic the case that self._gcp_resources already stores the job uri.
    with open(self._gcp_resources, 'w') as f:
      f.write(
          '{"resources": [{"resourceType": "BigqueryQueryJob", "resourceUri": "https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US"}]}'
      )

    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'configuration': {
            'extract': {
                'destinationUris': [self._model_destination_path],
            }
        }
    }
    mock_get_requests.return_value = mock_polled_bq_job
    self._payload = '{"configuration": {"extract": {}, "labels": {}}}'
    self._executor_input = '{"outputs":{"artifacts":{"model_destination_path":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQMLModel"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    bigquery_job_remote_runner.bigquery_export_model_job(
        self._job_type, self._project, self._location, self._model_name,
        self._model_destination_path, self._payload,
        self._exported_model_path, self._gcp_resources,
        self._executor_input)

    with open(self._output_file_path) as f:
      self.assertEqual(
          f.read(),
          '{"artifacts": {"evaluation_metrics": {"artifacts": [{"metadata": {"schema": "mock_schema", "rows": "mock_rows"}, "name": "foobar", "type": {"schemaTitle": "system.Artifact"}, "uri": ""}]}}}'
      )

    with open(self._exported_model_path) as f:
      self.assertEqual(f.read(), self._model_destination_path)

    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  def test_export_model_job_failed_no_selflink(self, mock_post_requests, _,
                                               mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {}
    mock_post_requests.return_value = mock_created_bq_job
    self._payload = (
        '{"configuration": {"query": {"query": "CREATE OR REPLACE MODEL '
        'bqml_tutorial.penguins_model OPTIONS (model_type=\'linear_reg\', '
        'input_label_cols=[\'body_mass_g\']) AS SELECT * FROM '
        '`bigquery-public-data.ml_datasets.penguins` WHERE body_mass_g IS NOT '
        'NULL"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"model":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQMLModel"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    with self.assertRaises(RuntimeError):
      bigquery_job_remote_runner.bigquery_create_model_job(
          self._job_type, self._project, self._location, self._payload,
          self._job_configuration_query_override, self._gcp_resources,
          self._executor_input)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_export_model_job_poll_existing_job_failed(self, mock_time_sleep,
                                                     mock_get_requests, _,
                                                     mock_auth):
    # Mimic the case that self._gcp_resources already stores the job uri.
    with open(self._gcp_resources, 'w') as f:
      f.write(
          '{"resources": [{"resourceType": "BigqueryQueryJob", "resourceUri": "https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US"}]}'
      )

    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'errorResult': {
                'reason': 'invalidQuery',
                'location': 'query',
                'message':
                    'The input data has NULL values in one or more columns: '
                    'sex. BQML automatically handles null values (See '
                    'https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-create#imputation).'
                    ' If null values represent a special value in the data, '
                    'replace them with the desired value before training and '
                    'then retry.'
            },
            'state': 'DONE'
        },
        'statistics': {
            'query': {
                'statementType': 'CREATE_MODEL',
                'ddlOperationPerformed': 'REPLACE',
                'ddlTargetTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_bq_job
    self._payload = (
        '{"configuration": {"query": {"query": "CREATE OR REPLACE MODEL '
        'bqml_tutorial.penguins_model OPTIONS (model_type=\'linear_reg\', '
        'input_label_cols=[\'body_mass_g\']) AS SELECT * FROM '
        '`bigquery-public-data.ml_datasets.penguins` WHERE body_mass_g IS NOT '
        'NULL"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"model":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQMLModel"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    with self.assertRaises(RuntimeError):
      bigquery_job_remote_runner.bigquery_create_model_job(
          self._job_type, self._project, self._location, self._payload,
          self._job_configuration_query_override, self._gcp_resources,
          self._executor_input)

    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  @mock.patch.object(ExecutionContext, '__init__', autospec=True)
  def test_export_model_job_cancel(self, mock_execution_context,
                                   mock_time_sleep,
                                   mock_get_requests, mock_post_requests,
                                   _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_created_bq_job
    mock_execution_context.return_value = None

    mock_get_model_resp = mock.Mock()
    mock_get_model_resp.json.return_value = {'modelType': 'LINEAR_REGRESSION'}

    mock_polled_bq_job = mock.Mock()
    mock_polled_bq_job.json.return_value = {
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'configuration': {
            'extract': {
                'destinationUris': [self._model_destination_path],
            }
        }
    }
    mock_get_requests.side_effect = [mock_get_model_resp, mock_polled_bq_job]
    self._payload = '{"configuration": {"extract": {}, "labels": {}}}'
    self._executor_input = '{"outputs":{"artifacts":{"model_destination_path":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQMLModel"}}]}},"outputFile":"' + self._output_file_path + '"}}'

    bigquery_job_remote_runner.bigquery_export_model_job(
        self._job_type, self._project, self._location, self._model_name,
        self._model_destination_path, self._payload,
        self._exported_model_path, self._gcp_resources,
        self._executor_input)

    # Call cancellation handler
    mock_execution_context.call_args[1]['on_cancel']()
    self.assertEqual(mock_post_requests.call_count, 2)
    mock_post_requests.assert_called_with(
        url='https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job/cancel',
        data='',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
        })

# Tests for evaluate model job.

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_evaluate_model_job_succeeded(self, mock_time_sleep,
                                        mock_get_requests, mock_post_requests,
                                        _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_evaluate_model_job = mock.Mock()
    mock_evaluate_model_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_evaluate_model_job

    mock_polled_evaluate_model_job = mock.Mock()
    mock_polled_evaluate_model_job.json.return_value = {
        'id': 'bigquery-public-data:US.eval_query_id',
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        }
    }
    mock_get_query_results = mock.Mock()
    mock_get_query_results.json.return_value = {
        'schema': 'mock_schema',
        'rows': 'mock_rows',
    }
    mock_get_requests.side_effect = [
        mock_polled_evaluate_model_job, mock_get_query_results
    ]

    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"evaluation_metrics":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"system.Artifact"}}]}},"outputFile":"' + self._output_file_path + '"}}'
    self._model_name = 'bqml_tutorial.penguins_model'
    self._table_name = None
    self._query_statement = ('SELECT * FROM '
                             '`bigquery-public-data.ml_datasets.penguins`')
    self._threshold = None
    bigquery_job_remote_runner.bigquery_evaluate_model_job(
        self._job_type, self._project, self._location, self._model_name,
        self._table_name, self._query_statement, self._threshold, self._payload,
        self._job_configuration_query_override, self._gcp_resources,
        self._executor_input)
    mock_post_requests.assert_called_once_with(
        url=f'https://www.googleapis.com/bigquery/v2/projects/{self._project}/jobs',
        data=(
            '{"configuration": {"query": {"query": "SELECT * FROM ML.EVALUATE(MODEL `bqml_tutorial.penguins_model`, (SELECT * FROM `bigquery-public-data.ml_datasets.penguins`))", "useLegacySql": false}}, "jobReference": {"location": "US"}}'
        ),
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
            'User-Agent': 'google-cloud-pipeline-components'
        })

    with open(self._output_file_path) as f:
      self.assertEqual(
          f.read(),
          '{"artifacts": {"evaluation_metrics": {"artifacts": [{"metadata": {"schema": "mock_schema", "rows": "mock_rows"}, "name": "foobar", "type": {"schemaTitle": "system.Artifact"}, "uri": ""}]}}}'
      )

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      # Instantiate GCPResources Proto
      bq_job_resources = json_format.Parse(serialized_gcp_resources,
                                           GcpResources())
      self.assertLen(bq_job_resources.resources, 1)
      self.assertEqual(
          bq_job_resources.resources[0].resource_uri,
          'https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US'
      )

    self.assertEqual(mock_post_requests.call_count, 1)
    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 2)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  @mock.patch.object(ExecutionContext, '__init__', autospec=True)
  def test_evaluate_model_job_cancel(self, mock_execution_context,
                                     mock_time_sleep,
                                     mock_get_requests, mock_post_requests,
                                     _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_evaluate_model_job = mock.Mock()
    mock_evaluate_model_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_evaluate_model_job
    mock_execution_context.return_value = None

    mock_polled_evaluate_model_job = mock.Mock()
    mock_polled_evaluate_model_job.json.return_value = {
        'id': 'bigquery-public-data:US.eval_query_id',
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        }
    }
    mock_get_query_results = mock.Mock()
    mock_get_query_results.json.return_value = {
        'schema': 'mock_schema',
        'rows': 'mock_rows',
    }
    mock_get_requests.side_effect = [
        mock_polled_evaluate_model_job, mock_get_query_results
    ]

    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = '{"outputs":{"artifacts":{"evaluation_metrics":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"system.Artifact"}}]}},"outputFile":"' + self._output_file_path + '"}}'
    self._model_name = 'bqml_tutorial.penguins_model'
    self._table_name = None
    self._query_statement = ('SELECT * FROM '
                             '`bigquery-public-data.ml_datasets.penguins`')
    self._threshold = None
    bigquery_job_remote_runner.bigquery_evaluate_model_job(
        self._job_type, self._project, self._location, self._model_name,
        self._table_name, self._query_statement, self._threshold, self._payload,
        self._job_configuration_query_override, self._gcp_resources,
        self._executor_input)

    # Call cancellation handler
    mock_execution_context.call_args[1]['on_cancel']()
    self.assertEqual(mock_post_requests.call_count, 2)
    mock_post_requests.assert_called_with(
        url='https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job/cancel',
        data='',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
        })

  # Tests for ml reconstruction loss job using query.
  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_ml_reconstruction_loss_query_succeeded(self, mock_time_sleep,
                                                  mock_get_requests,
                                                  mock_post_requests, _,
                                                  mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_ml_reconstruction_loss_job = mock.Mock()
    mock_ml_reconstruction_loss_job.json.return_value = {
        'selfLink': self._job_uri
    }
    mock_post_requests.return_value = mock_ml_reconstruction_loss_job

    mock_polled_ml_reconstruction_loss_job = mock.Mock()
    mock_polled_ml_reconstruction_loss_job.json.return_value = {
        'id': 'bigquery-public-data:US.ml_reconstruction_loss_query_id',
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'configuration': {
            'query': {
                'destinationTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_ml_reconstruction_loss_job
    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = (
        '{"outputs":{"artifacts":{"destination_table": {"artifacts":'
        '[{"metadata":{},"name":"foobar",'
        '"type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' +
        self._output_file_path + '"}}')
    self._model_name = 'bqml_tutorial.penguins_model'
    self._table_name = None
    self._query_statement = ('SELECT * FROM '
                             '`bigquery-public-data.ml_datasets.penguins`')
    bigquery_job_remote_runner.bigquery_ml_reconstruction_loss_job(
        self._job_type, self._project, self._location, self._model_name,
        self._table_name, self._query_statement, self._payload,
        self._job_configuration_query_override, self._gcp_resources,
        self._executor_input)
    mock_post_requests.assert_called_once_with(
        url=f'https://www.googleapis.com/bigquery/v2/projects/{self._project}/jobs',
        data=('{"configuration": {"query": {"query": '
              '"SELECT * FROM ML.RECONSTRUCTION_LOSS('
              'MODEL `bqml_tutorial.penguins_model`, '
              '(SELECT * FROM `bigquery-public-data.ml_datasets.penguins`))", '
              '"useLegacySql": false}}, "jobReference": {"location": "US"}}'),
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
            'User-Agent': 'google-cloud-pipeline-components'
        })

    with open(self._output_file_path) as f:
      self.assertEqual(
          f.read(),
          '{"artifacts": {"destination_table": {"artifacts": [{"metadata": '
          '{"projectId": "test_project", "datasetId": "test_dataset", '
          '"tableId": "test_table"}, "name": "foobar", '
          '"type": {"schemaTitle": "google.BQTable"}, '
          '"uri": "https://www.googleapis.com/bigquery/v2/projects/test_project/datasets/test_dataset/tables/test_table"}]}}}'
      )

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      # Instantiate GCPResources Proto
      bq_job_resources = json_format.Parse(serialized_gcp_resources,
                                           GcpResources())
      self.assertLen(bq_job_resources.resources, 1)
      self.assertEqual(
          bq_job_resources.resources[0].resource_uri,
          'https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US'
      )

    self.assertEqual(mock_post_requests.call_count, 1)
    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  # Tests for ml_reconstruction_loss job using table.
  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_ml_reconstruction_loss_table_job_succeeded(self, mock_time_sleep,
                                                      mock_get_requests,
                                                      mock_post_requests, _,
                                                      mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_ml_reconstruction_loss_job = mock.Mock()
    mock_ml_reconstruction_loss_job.json.return_value = {
        'selfLink': self._job_uri
    }
    mock_post_requests.return_value = mock_ml_reconstruction_loss_job

    mock_polled_ml_reconstruction_loss_job = mock.Mock()
    mock_polled_ml_reconstruction_loss_job.json.return_value = {
        'id': 'bigquery-public-data:US.ml_reconstruction_loss_query_id',
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'configuration': {
            'query': {
                'destinationTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_ml_reconstruction_loss_job
    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = (
        '{"outputs":{"artifacts":{"destination_table":{ "artifacts":'
        '[{"metadata":{},"name":"foobar",'
        '"type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' +
        self._output_file_path + '"}}')
    self._model_name = 'bqml_tutorial.penguins_model'
    self._table_name = 'bigquery-public-data.ml_datasets.penguins'
    self._query_statement = None
    bigquery_job_remote_runner.bigquery_ml_reconstruction_loss_job(
        self._job_type, self._project, self._location, self._model_name,
        self._table_name, self._query_statement, self._payload,
        self._job_configuration_query_override, self._gcp_resources,
        self._executor_input)
    mock_post_requests.assert_called_once_with(
        url=f'https://www.googleapis.com/bigquery/v2/projects/{self._project}/jobs',
        data=('{"configuration": {"query": {'
              '"query": "SELECT * FROM ML.RECONSTRUCTION_LOSS('
              'MODEL `bqml_tutorial.penguins_model`, '
              'TABLE `bigquery-public-data.ml_datasets.penguins`)", '
              '"useLegacySql": false}}, "jobReference": {"location": "US"}}'),
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
            'User-Agent': 'google-cloud-pipeline-components'
        })

    with open(self._output_file_path) as f:
      self.assertEqual(
          f.read(),
          '{"artifacts": {"destination_table": {"artifacts": [{"metadata": '
          '{"projectId": "test_project", "datasetId": "test_dataset", '
          '"tableId": "test_table"}, "name": "foobar", '
          '"type": {"schemaTitle": "google.BQTable"}, '
          '"uri": "https://www.googleapis.com/bigquery/v2/projects/test_project/datasets/test_dataset/tables/test_table"}]}}}'
      )

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      # Instantiate GCPResources Proto
      bq_job_resources = json_format.Parse(serialized_gcp_resources,
                                           GcpResources())
      self.assertLen(bq_job_resources.resources, 1)
      self.assertEqual(
          bq_job_resources.resources[0].resource_uri,
          'https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US'
      )

    self.assertEqual(mock_post_requests.call_count, 1)
    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  @mock.patch.object(ExecutionContext, '__init__', autospec=True)
  def test_ml_reconstruction_loss_job_query_cancel(self, mock_execution_context,
                                                   mock_time_sleep,
                                                   mock_get_requests,
                                                   mock_post_requests, _,
                                                   mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_ml_reconstruction_loss_job = mock.Mock()
    mock_ml_reconstruction_loss_job.json.return_value = {
        'selfLink': self._job_uri
    }
    mock_post_requests.return_value = mock_ml_reconstruction_loss_job
    mock_execution_context.return_value = None

    mock_polled_ml_reconstruction_loss_job = mock.Mock()
    mock_polled_ml_reconstruction_loss_job.json.return_value = {
        'id': 'bigquery-public-data:US.ml_reconstruction_loss_query_id',
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        },
        'configuration': {
            'query': {
                'destinationTable': {
                    'projectId': 'test_project',
                    'datasetId': 'test_dataset',
                    'tableId': 'test_table'
                }
            }
        }
    }
    mock_get_requests.return_value = mock_polled_ml_reconstruction_loss_job
    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = (
        '{"outputs":{"artifacts":{"destination_table":{"artifacts":'
        '[{"metadata":{},"name":"foobar",'
        '"type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' +
        self._output_file_path + '"}}')
    self._model_name = 'bqml_tutorial.penguins_model'
    self._table_name = None
    self._query_statement = ('SELECT * FROM '
                             '`bigquery-public-data.ml_datasets.penguins`')
    bigquery_job_remote_runner.bigquery_ml_reconstruction_loss_job(
        self._job_type, self._project, self._location, self._model_name,
        self._table_name, self._query_statement, self._payload,
        self._job_configuration_query_override, self._gcp_resources,
        self._executor_input)

    # Call cancellation handler
    mock_execution_context.call_args[1]['on_cancel']()
    self.assertEqual(mock_post_requests.call_count, 2)
    mock_post_requests.assert_called_with(
        url='https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job/cancel',
        data='',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
        })

  # Tests for ml trial info job.
  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_ml_trial_info_job_succeeded(self, mock_time_sleep, mock_get_requests,
                                       mock_post_requests, _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_ml_trial_info_job = mock.Mock()
    mock_ml_trial_info_job.json.return_value = {'selfLink': self._job_uri}
    mock_post_requests.return_value = mock_ml_trial_info_job

    mock_polled_ml_trial_info_job = mock.Mock()
    mock_polled_ml_trial_info_job.json.return_value = {
        'id': 'bigquery-public-data:US.ml_trial_info_query_id',
        'selfLink': self._job_uri,
        'status': {
            'state': 'DONE'
        }
    }
    mock_get_query_results = mock.Mock()
    mock_get_query_results.json.return_value = {
        'schema': 'mock_schema',
        'rows': 'mock_rows',
    }
    mock_get_requests.side_effect = [
        mock_polled_ml_trial_info_job, mock_get_query_results
    ]

    self._payload = ('{"configuration": {"query": {"query": "SELECT * FROM '
                     '`bigquery-public-data.ml_datasets.penguins`"}}}')
    self._executor_input = (
        '{"outputs":{"artifacts":{"trial_info":{ "artifacts":'
        '[{"metadata":{},"name":"foobar",'
        '"type":{"schemaTitle":"system.Artifact"}}]}},"outputFile":"' +
        self._output_file_path + '"}}')
    self._model_name = 'bqml_tutorial.penguins_model'
    bigquery_job_remote_runner.bigquery_ml_trial_info_job(
        self._job_type, self._project, self._location, self._model_name,
        self._payload, self._job_configuration_query_override,
        self._gcp_resources, self._executor_input)
    mock_post_requests.assert_called_once_with(
        url=f'https://www.googleapis.com/bigquery/v2/projects/{self._project}/jobs',
        data=('{"configuration": {"query": {'
              '"query": "SELECT * FROM ML.TRIAL_INFO('
              'MODEL `bqml_tutorial.penguins_model`)", '
              '"useLegacySql": false}}, "jobReference": {"location": "US"}}'),
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
            'User-Agent': 'google-cloud-pipeline-components'
        })

    with open(self._output_file_path) as f:
      self.assertEqual(
          f.read(),
          ('{"artifacts": {"trial_info": {"artifacts": [{"metadata": '
           '{"schema": "mock_schema", "rows": "mock_rows"}, "name": "foobar", '
           '"type": {"schemaTitle": "system.Artifact"}, "uri": ""}]}}}'))

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      # Instantiate GCPResources Proto
      bq_job_resources = json_format.Parse(serialized_gcp_resources,
                                           GcpResources())
      self.assertLen(bq_job_resources.resources, 1)
      self.assertEqual(
          bq_job_resources.resources[0].resource_uri,
          'https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US'
      )

    self.assertEqual(mock_post_requests.call_count, 1)
    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 2)
