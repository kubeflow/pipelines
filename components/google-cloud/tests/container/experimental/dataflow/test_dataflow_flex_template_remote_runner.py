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
"""Test Dataflow Flex Template Remote Runner."""
import json
import os

import google.auth
from google_cloud_pipeline_components.container.experimental.dataflow.flex_template import remote_runner as dataflow_flex_template_remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import gcp_labels_util
from google_cloud_pipeline_components.proto import gcp_resources_pb2
import re
import requests

from google.protobuf import json_format
import unittest
from unittest import mock

_SYSTEM_LABELS = {'system_label1': 'value1', 'system_label2': 'value2'}


class DataflowFlexTemplateRemoteRunnerTests(unittest.TestCase):
  """Tests for Dataflow Flex Template Remote Runner."""

  def setUp(self):
    super(DataflowFlexTemplateRemoteRunnerTests, self).setUp()
    self._dataflow_uri_prefix = 'https://dataflow.googleapis.com/v1b3'
    self._job_type = 'DataflowJob'
    self._project = 'test-project'
    self._location = 'us-central1'
    self._creds_token = 'fake-token'
    self._test_bucket_name = 'test_bucket_name'
    self._temp_blob_path = 'temp_blob_path'
    self._staging_blob_path = 'staging_blob_path'
    self._gcs_temp_path = f'gs://{self._test_bucket_name}/{self._temp_blob_path}'
    self._gcs_staging_path = f'gs://{self._test_bucket_name}/{self._staging_blob_path}'
    self._job_id = '2023-01-03_16_06_07-12446786984340643222'
    self._job_name = 'test_job_name'
    self._container_spec_gcs_path = 'test_container_spec_gcs_path'
    self._job_uri = f'{self._dataflow_uri_prefix}/projects/{self._project}/locations/{self._location}/jobs/{self._job_id}'
    self._args = ['test_arg']
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'gcp_resources')
    self._local_file_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'local_file')
    os.environ[gcp_labels_util.SYSTEM_LABEL_ENV_VAR] = json.dumps(
        _SYSTEM_LABELS)

    self._test_payload = {
        'launch_parameter': {
            'container_spec_gcs_path': self._container_spec_gcs_path,
            'parameters': {
                'param1': 'value1',
                'param2': 'value2',
            },
            'launch_options': {
                'param1': 'value1',
                'param2': 'value2',
            },
            'environment': {
                'num_workers': 1,
                'max_workers': 1000,
                'service_account_email': 'service-account-email',
                'temp_location': 'gs://test-bucket-123/temp',
                'machine_type': 'n1-standard-1',
                'additional_experiments': 'additional_experiment_1',
                'network': 'test-network-uri',
                'additional_user_labels': {
                    'label1': 'value1',
                    'label2': 'value2',
                },
                'kms_key_name': 'test-key-name',
                'ip_configuration': 'WORKER_IP_PRIVATE',
                'worker_region': self._location,
                'worker_zone': 'us-central1-b',
                'staging_location': self._gcs_staging_path,
            }
        }
    }

    self._expected_payload = self._test_payload.copy()
    self._expected_payload['launch_parameter']['environment']['additional_user_labels'] = {
        **_SYSTEM_LABELS,
        **self._expected_payload['launch_parameter']['environment']['additional_user_labels'],
    }

  def tearDown(self):
    super(DataflowFlexTemplateRemoteRunnerTests, self).tearDown()
    if os.path.exists(self._gcp_resources):
      os.remove(self._gcp_resources)

  def _validate_gcp_resources_succeeded(self):
    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      job_resources = json_format.Parse(serialized_gcp_resources,
                                        gcp_resources_pb2.GcpResources())

      # Validate number of resources.
      self.assertLen(job_resources.resources, 1)

      # Validate that gcp_resources contains a valid DataflowJob resourcce.
      test_job_resources = gcp_resources_pb2.GcpResources()
      job_resource = test_job_resources.resources.add()
      job_resource.resource_type = 'DataflowJob'
      job_resource.resource_uri = self._job_uri
      self.assertEqual(job_resources, test_job_resources)

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'post', autospec=True)
  def test_dataflow_flex_template_remote_runner_launch_succeeded(self,
                                                                 mock_post_requests,
                                                                 _,
                                                                 mock_auth_default):
    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_job = mock.Mock(spec=requests.models.Response)
    mock_job.json.return_value = {
        'job': {
            'id': self._job_id,
            'projectId': self._project,
            'location': self._location,
        }
    }
    mock_post_requests.return_value = mock_job

    test_payload = self._test_payload.copy()
    test_payload['launch_parameter']['job_name'] = self._job_name

    expected_payload = test_payload.copy()
    expected_payload['launch_parameter']['environment']['additional_user_labels'] = {
        **_SYSTEM_LABELS,
        **expected_payload['launch_parameter']['environment']['additional_user_labels'],
    }

    dataflow_flex_template_remote_runner.launch_flex_template(
        type=self._job_type,
        project=self._project,
        location=self._location,
        payload=json.dumps(test_payload),
        gcp_resources=self._gcp_resources,
    )

    mock_post_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataflow_uri_prefix}/projects/{self._project}/locations/{self._location}/flexTemplates:launch',
        data=json.dumps(expected_payload),
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )
    self._validate_gcp_resources_succeeded()

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'post', autospec=True)
  def test_dataflow_flex_template_remote_runner_launch_bad_request(self,
                                                                   mock_post_requests,
                                                                   _,
                                                                   mock_auth_default):
    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_request = mock.Mock(spec=requests.models.Request)
    mock_request.url = f'{self._dataflow_uri_prefix}/projects/{self._project}/locations/{self._location}/flexTemplates:launch'
    mock_request.body = self._test_payload
    mock_response = mock.MagicMock(spec=requests.models.Response)
    mock_response.status_code = 400
    mock_response.json.return_value = {
        'error': {
            'code': 400,
            'message': 'JobName invalid; the name must consist of only the characters [-a-z0-9], starting with a letter and ending with a letter or number',
            'status': 'INVALID_ARGUMENT',
        }
    }
    mock_response.raise_for_status.side_effect = requests.exceptions.HTTPError(
        request=mock_request, response=mock_response,
    )
    mock_post_requests.return_value = mock_response

    test_payload = self._test_payload.copy()
    test_payload['launch_parameter']['job_name'] = self._job_name

    expected_payload = test_payload.copy()
    expected_payload['launch_parameter']['environment']['additional_user_labels'] = {
        **_SYSTEM_LABELS,
        **expected_payload['launch_parameter']['environment']['additional_user_labels'],
    }

    with self.assertRaises(RuntimeError) as err:
      dataflow_flex_template_remote_runner.launch_flex_template(
          type=self._job_type,
          project=self._project,
          location=self._location,
          payload=json.dumps(test_payload),
          gcp_resources=self._gcp_resources,
      )

    mock_post_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataflow_uri_prefix}/projects/{self._project}/locations/{self._location}/flexTemplates:launch',
        data=json.dumps(expected_payload),
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )

    exception_msg = (
        f'Error 400 returned from POST: {mock_request.url}'
        '. Status: INVALID_ARGUMENT'
        ', Message: JobName invalid; the name must consist of only the characters [-a-z0-9], starting with a letter and ending with a letter or number'
    )
    self.assertEqual(exception_msg, str(err.exception))

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'post', autospec=True)
  def test_dataflow_flex_template_remote_runner_generate_job_name(self,
                                                                  mock_post_requests,
                                                                  _,
                                                                  mock_auth_default):
    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    mock_job = mock.Mock(spec=requests.models.Response)
    mock_job.json.return_value = {
        'job': {
            'id': self._job_id,
            'projectId': self._project,
            'location': self._location,
        }
    }
    mock_post_requests.return_value = mock_job

    dataflow_flex_template_remote_runner.launch_flex_template(
        type=self._job_type,
        project=self._project,
        location=self._location,
        payload=json.dumps(self._test_payload),
        gcp_resources=self._gcp_resources,
    )

    post_data = json.loads(mock_post_requests.call_args[1]['data'])
    job_name = post_data['launch_parameter']['job_name']

    self.assertEqual(len(re.findall('dataflowjob-[0-9]{14}-[a-f0-9]{8}', job_name)), 1)

    expected_payload = self._test_payload.copy()
    expected_payload['launch_parameter']['job_name'] = job_name
    expected_payload['launch_parameter']['environment']['additional_user_labels'] = {
        **_SYSTEM_LABELS,
        **expected_payload['launch_parameter']['environment']['additional_user_labels'],
    }

    mock_post_requests.assert_called_once_with(
        self=mock.ANY,
        url=f'{self._dataflow_uri_prefix}/projects/{self._project}/locations/{self._location}/flexTemplates:launch',
        data=json.dumps(expected_payload),
        headers={'Authorization': f'Bearer {self._creds_token}'}
    )

    self._validate_gcp_resources_succeeded()

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests.sessions.Session, 'post', autospec=True)
  def test_dataflow_flex_template_remote_runner_existing_job_found(self,
                                                                   mock_post_requests,
                                                                   _,
                                                                   mock_auth_default):
    with open(self._gcp_resources, 'w') as f:
      f.write(
          json.dumps({
              'resources': [
                  {
                      'resource_type': 'DataflowJob',
                      'resource_uri': self._job_uri,
                  },
              ]
          })
      )

    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    dataflow_flex_template_remote_runner.launch_flex_template(
        type=self._job_type,
        project=self._project,
        location=self._location,
        payload=json.dumps(self._test_payload),
        gcp_resources=self._gcp_resources,
    )
    mock_post_requests.assert_not_called()

  @mock.patch.object(google.auth, 'default', autospec=True)
  def test_dataflow_flex_template_remote_runner_job_exists_wrong_format(self,
                                                                        mock_auth_default):
    with open(self._gcp_resources, 'w') as f:
      f.write(
          json.dumps({
              'resources': [
                  {
                      'resourceType': 'DataflowJob',
                      'resourceUri': 'invalid_resource_uri',
                  },
              ]
          })
      )

    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    with self.assertRaises(ValueError):
      dataflow_flex_template_remote_runner.launch_flex_template(
          type=self._job_type,
          project=self._project,
          location=self._location,
          payload=json.dumps(self._test_payload),
          gcp_resources=self._gcp_resources,
      )

  @mock.patch.object(google.auth, 'default', autospec=True)
  def test_dataflow_flex_template_remote_runner_exception_with_multiple_job_resources_in_gcp_resources(self, mock_auth_default):
    with open(self._gcp_resources, 'w') as f:
      f.write(
          (
              f'{{"resources": ['
              f'{{"resourceType": "DataflowJob", "resourceUri": "{self._job_uri}"}},'
              f'{{"resourceType": "DataflowJob", "resourceUri": "{self._job_uri}"}}'
              f']}}'
          )
      )

    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = self._creds_token
    mock_auth_default.return_value = [mock_creds, 'project']

    with self.assertRaises(ValueError):
      dataflow_flex_template_remote_runner.launch_flex_template(
          type=self._job_type,
          project=self._project,
          location=self._location,
          payload=json.dumps(self._test_payload),
          gcp_resources=self._gcp_resources,
      )
