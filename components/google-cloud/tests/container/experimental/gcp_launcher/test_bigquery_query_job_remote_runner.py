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
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
from google_cloud_pipeline_components.container.experimental.gcp_launcher import bigquery_query_job_remote_runner
from google_cloud_pipeline_components.container.experimental.gcp_launcher import job_remote_runner


class BigqueryQueryJobRemoteRunnerUtilsTests(unittest.TestCase):

  def setUp(self):
    super(BigqueryQueryJobRemoteRunnerUtilsTests, self).setUp()
    self._payload = (
        '{"configuration": {"query": {"query": "CREATE OR REPLACE '
        'MODEL bqml_tutorial.penguins_model OPTIONS '
        '(model_type=\'linear_reg\', input_label_cols=[\'body_mass_g\']) AS '
        'SELECT * FROM `bigquery-public-data.ml_datasets.penguins` WHERE '
        'body_mass_g IS NOT NULL"}}}')
    self._job_configuration_query_override = '{}'
    self._job_type = 'BigqueryQueryJob'
    self._project = 'test_project'
    self._location = 'US'
    self._job_uri = 'https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job?location=US'
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'gcp_resources')
    self._output_file_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'localpath/foo')
    self._executor_input = '{"outputs":{"artifacts":{"destination_table":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.BQTable"}}]}},"outputFile":"' + self._output_file_path + '"}}'

  def tearDown(self):
    if os.path.exists(self._gcp_resources):
      os.remove(self._gcp_resources)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_bigquery_query_job_remote_runner_succeeded(self, mock_time_sleep,
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

    bigquery_query_job_remote_runner.create_bigquery_job(
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
          '{"artifacts": {"destination_table": {"artifacts": [{"metadata": {"projectId": "test_project", "datasetId": "test_dataset", "tableId": "test_table"}, "name": "foobar", "type": {"schemaTitle": "google.BQTable"}, "uri": "https://www.googleapis.com/bigquery/v2/projects/test_project/datasets/test_dataset/tables/test_table"}]}}}'
      )

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      # Instantiate GCPResources Proto
      bq_job_resources = json_format.Parse(serialized_gcp_resources,
                                           GcpResources())
      self.assertEqual(len(bq_job_resources.resources), 1)
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
  def test_bigquery_query_job_remote_runner_with_job_config_override_succeeded(
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

    job_configuration_query_override = ('{"query":"SELECT * FROM foo", '
                                        '"query_parameters": "abc"}')
    bigquery_query_job_remote_runner.create_bigquery_job(
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
      self.assertEqual(len(bq_job_resources.resources), 1)
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
  def test_bigquery_query_job_remote_runner_poll_existing_job_succeeded(
      self, mock_time_sleep, mock_get_requests, _, mock_auth):
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

    bigquery_query_job_remote_runner.create_bigquery_job(
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
  def test_bigquery_query_job_remote_runner_check_job_exists_wrong_format(
      self, _, mock_auth):
    # Mimic the case that self._gcp_resources already stores the job uri.
    with open(self._gcp_resources, 'w') as f:
      f.write(
          '{"resources": [{"resourceType": "BigqueryQueryJob", "resourceUri": "https://www.googleapis.com/bigquery/v2/projects/test_project/jobs/fake_job_no_location"}]}'
      )

    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']

    with self.assertRaises(ValueError):
      bigquery_query_job_remote_runner.create_bigquery_job(
          self._job_type, self._project, self._location, self._payload,
          self._job_configuration_query_override, self._gcp_resources,
          self._executor_input)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  def test_bigquery_query_job_remote_runner_failed_no_selflink(
      self, mock_post_requests, _, mock_auth):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, 'project']
    mock_created_bq_job = mock.Mock()
    mock_created_bq_job.json.return_value = {}
    mock_post_requests.return_value = mock_created_bq_job

    with self.assertRaises(RuntimeError):
      bigquery_query_job_remote_runner.create_bigquery_job(
          self._job_type, self._project, self._location, self._payload,
          self._job_configuration_query_override, self._gcp_resources,
          self._executor_input)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'get', autospec=True)
  @mock.patch.object(time, 'sleep', autospec=True)
  def test_bigquery_query_job_remote_runner_poll_existing_job_failed(
      self, mock_time_sleep, mock_get_requests, _, mock_auth):
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

    with self.assertRaises(RuntimeError):
      bigquery_query_job_remote_runner.create_bigquery_job(
          self._job_type, self._project, self._location, self._payload,
          self._job_configuration_query_override, self._gcp_resources,
          self._executor_input)

    self.assertEqual(mock_time_sleep.call_count, 1)
    self.assertEqual(mock_get_requests.call_count, 1)
