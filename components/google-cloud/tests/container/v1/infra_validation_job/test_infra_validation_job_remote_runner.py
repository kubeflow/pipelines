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
"""Test Vertex AI Infra Validation module."""

import json
import os

from google.cloud import aiplatform
from google.cloud.aiplatform.compat.types import job_state as gca_job_state
from google_cloud_pipeline_components.container.v1.infra_validation_job import remote_runner as infra_validation_job_remote_runner

import unittest
from unittest import mock


class InfraValidationJobRunnerUtilsTests(unittest.TestCase):

  def setUp(self):
    super(InfraValidationJobRunnerUtilsTests, self).setUp()
    self._project = 'test_project'
    self._location = 'test_region'
    self._type = 'InfraValidationJob'

    self._output_file_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'localpath', 'foo')

    self._executor_input_dict = {
        'inputs': {
            'artifacts': {
                'unmanaged_container_model': {
                    'artifacts': [{
                        'metadata': {
                            'containerSpec': {
                                'imageUri': 'image_foo'
                            }
                        },
                        'name': 'unmanaged_container_model',
                        'type': {
                            'schemaTitle': 'google.UnmanagedContainerModel'
                        },
                        'uri': 'gs://abc'
                    }]
                }
            }
        },
        'outputs': {
            'outputFile': self._output_file_path
        }
    }
    self._executor_input = json.dumps(self._executor_input_dict)
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'gcp_resources')

    self._infra_validation_example_path = 'test_infra_validation_example_path'
    self._machine_type = 'n1-standard-8'

    self._payload = json.dumps(
        {'infra_validation_example_path': self._infra_validation_example_path,
         'machine_type': self._machine_type})

    self._expected_args = ['--model_base_path', 'gs://abc']
    if self._infra_validation_example_path:
      self._expected_args.extend([
          '--infra_validation_example_path', self._infra_validation_example_path
      ])

    self._expected_env_variables = [{
        'name': 'INFRA_VALIDATION_MODE',
        'value': '1'
    }]

    self._expected_job_spec = {
        'display_name': 'infra-validation',
        'job_spec': {
            'worker_pool_specs': [{
                'machine_spec': {
                    'machine_type': self._machine_type
                },
                'replica_count': 1,
                'container_spec': {
                    'image_uri': 'image_foo',
                    'args': self._expected_args,
                    'env': self._expected_env_variables
                },
            }]
        },
    }

  def tearDown(self):
    if os.path.exists(self._gcp_resources):
      os.remove(self._gcp_resources)

  def test_construct_infra_validation_job_spec(self):
    actual_job_spec = infra_validation_job_remote_runner.construct_infra_validation_job_payload(self._executor_input, self._payload)
    self.assertDictEqual(self._expected_job_spec, json.loads(actual_job_spec))

  @mock.patch.object(aiplatform.gapic, 'JobServiceClient', autospec=True)
  def test_infra_validation_job_remote_runner_succeeded(
      self, mock_job_service_client):

    job_client = mock.Mock()
    mock_job_service_client.return_value = job_client

    create_custom_job_response = mock.Mock()
    job_client.create_custom_job.return_value = create_custom_job_response
    create_custom_job_response.name = 'infra_validation'

    get_custom_job_response = mock.Mock()
    job_client.get_custom_job.return_value = get_custom_job_response
    get_custom_job_response.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

    infra_validation_job_remote_runner.create_infra_validation_job(
        self._type, self._project, self._location, self._gcp_resources,
        self._executor_input, self._payload)

    job_client.create_custom_job.assert_called_once_with(
        parent=f'projects/{self._project}/locations/{self._location}',
        custom_job=self._expected_job_spec)
