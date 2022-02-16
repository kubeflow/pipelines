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
"""Test Vertex AI Hyperparameter Tuning Remote Runner Client module."""

import json
from logging import raiseExceptions
import os
import time
from unittest import mock

from google.cloud import aiplatform
from google.cloud.aiplatform.compat.types import job_state as gca_job_state
from google_cloud_pipeline_components.container.utils.execution_context import ExecutionContext
from google_cloud_pipeline_components.container.v1.gcp_launcher import hyperparameter_tuning_job_remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher import job_remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import json_util
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources

from google.protobuf import json_format
import unittest
import requests
import google.auth
import google.auth.transport.requests


class HyperparameterTuningJobRemoteRunnerUtilsTests(unittest.TestCase):

  def setUp(self):
    super(HyperparameterTuningJobRemoteRunnerUtilsTests, self).setUp()
    self._payload = (
        '{"display_name": "HPTuningJob", "study_spec": {"metrics": '
        '{"accuracy": "maximize"}, "parameters": {"parameterId": '
        '"learning_rate", "doubleValueSpec": {"minValue": 0.001, "maxValue": '
        '1.0}, "scaleType": 2, "conditionalParameterSpecs": []}, "algorithm": '
        '"ALGORITHM_UNSPECIFIED", "measurement_selection_type": '
        '"BEST_MEASUREMENT"}, "max_trial_count": 10.0, "parallel_trial_count":'
        ' 3.0, "max_failed_trial_count": 0.0, "trial_job_spec": '
        '{"worker_pool_specs": [{"container_spec": {"image_uri": '
        '"gcr.io/project_id/test"}, "machine_spec": {"accelerator_count": 1.0,'
        ' "accelerator_type": "NVIDIA_TESLA_T4", "machine_type": '
        '"n1-standard-4"}, "replica_count": 1.0}], "service_account": "", '
        '"network": "", "base_output_directory": "gs://my-bucket/blob"}, '
        '"encryption_spec": {"kms_key_name": ""}}')
    self._project = "test_project"
    self._location = "test_region"
    self._hptuning_job_name = f"/projects/{self._project}/locations/{self._location}/hyperparameterTuningJobs/test_job_id"
    self._gcp_resources = os.path.join(
        os.getenv("TEST_UNDECLARED_OUTPUTS_DIR"), "gcp_resources")
    self._type = "HyperparameterTuningJob"
    self._hptuning_job_uri_prefix = f"https://{self._location}-aiplatform.googleapis.com/v1/"

  def tearDown(self):
    super(HyperparameterTuningJobRemoteRunnerUtilsTests, self).tearDown()
    if os.path.exists(self._gcp_resources):
      os.remove(self._gcp_resources)

  @mock.patch.object(aiplatform.gapic, "JobServiceClient", autospec=True)
  def test_hptuning_job_remote_runner_on_region_is_set_correctly_in_client_options(
      self, mock_job_service_client):

    job_client = mock.Mock()
    mock_job_service_client.return_value = job_client

    create_hptuning_job_response = mock.Mock()
    job_client.create_hyperparameter_tuning_job.return_value = create_hptuning_job_response
    create_hptuning_job_response.name = self._hptuning_job_name

    get_hptuning_job_response = mock.Mock()
    job_client.get_hyperparameter_tuning_job.return_value = get_hptuning_job_response
    get_hptuning_job_response.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

    hyperparameter_tuning_job_remote_runner.create_hyperparameter_tuning_job(
        self._type, self._project, self._location, self._payload,
        self._gcp_resources)
    mock_job_service_client.assert_called_once_with(
        client_options={
            "api_endpoint": "test_region-aiplatform.googleapis.com"
        },
        client_info=mock.ANY)

  @mock.patch.object(aiplatform.gapic, "JobServiceClient", autospec=True)
  @mock.patch.object(os.path, "exists", autospec=True)
  def test_hptuning_job_remote_runner_on_payload_deserializes_correctly(
      self, mock_path_exists, mock_job_service_client):

    job_client = mock.Mock()
    mock_job_service_client.return_value = job_client

    create_hptuning_job_response = mock.Mock()
    job_client.create_hyperparameter_tuning_job.return_value = create_hptuning_job_response
    create_hptuning_job_response.name = self._hptuning_job_name

    get_hptuning_job_response = mock.Mock()
    job_client.get_hyperparameter_tuning_job.return_value = get_hptuning_job_response
    get_hptuning_job_response.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

    mock_path_exists.return_value = False

    hyperparameter_tuning_job_remote_runner.create_hyperparameter_tuning_job(
        self._type, self._project, self._location, self._payload,
        self._gcp_resources)

    expected_parent = f"projects/{self._project}/locations/{self._location}"
    expected_job_spec = json_util.recursive_remove_empty(
        json.loads(self._payload, strict=False))

    job_client.create_hyperparameter_tuning_job.assert_called_once_with(
        parent=expected_parent, hyperparameter_tuning_job=expected_job_spec)

  @mock.patch.object(aiplatform.gapic, "JobServiceClient", autospec=True)
  @mock.patch.object(os.path, "exists", autospec=True)
  def test_hptuning_job_remote_runner_raises_exception_on_error(
      self, mock_path_exists, mock_job_service_client):

    job_client = mock.Mock()
    mock_job_service_client.return_value = job_client

    create_hptuning_job_response = mock.Mock()
    job_client.create_hyperparameter_tuning_job.return_value = create_hptuning_job_response
    create_hptuning_job_response.name = self._hptuning_job_name

    get_hptuning_job_response = mock.Mock()
    job_client.get_hyperparameter_tuning_job.return_value = get_hptuning_job_response
    get_hptuning_job_response.state = gca_job_state.JobState.JOB_STATE_FAILED

    mock_path_exists.return_value = False

    with self.assertRaises(RuntimeError):
      hyperparameter_tuning_job_remote_runner.create_hyperparameter_tuning_job(
          self._type, self._project, self._location, self._payload,
          self._gcp_resources)

  @mock.patch.object(aiplatform.gapic, "JobServiceClient", autospec=True)
  @mock.patch.object(os.path, "exists", autospec=True)
  @mock.patch.object(time, "sleep", autospec=True)
  def test_hptuning_job_remote_runner_retries_to_get_status_on_non_completed_job(
      self, mock_time_sleep, mock_path_exists, mock_job_service_client):
    job_client = mock.Mock()
    mock_job_service_client.return_value = job_client

    create_hptuning_job_response = mock.Mock()
    job_client.create_hyperparameter_tuning_job.return_value = create_hptuning_job_response
    create_hptuning_job_response.name = self._hptuning_job_name

    get_hptuning_job_response_success = mock.Mock()
    get_hptuning_job_response_success.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

    get_hptuning_job_response_running = mock.Mock()
    get_hptuning_job_response_running.state = gca_job_state.JobState.JOB_STATE_RUNNING

    job_client.get_hyperparameter_tuning_job.side_effect = [
        get_hptuning_job_response_running, get_hptuning_job_response_success
    ]

    mock_path_exists.return_value = False

    hyperparameter_tuning_job_remote_runner.create_hyperparameter_tuning_job(
        self._type, self._project, self._location, self._payload,
        self._gcp_resources)
    mock_time_sleep.assert_called_once_with(
        job_remote_runner._POLLING_INTERVAL_IN_SECONDS)
    self.assertEqual(job_client.get_hyperparameter_tuning_job.call_count, 2)

  @mock.patch.object(aiplatform.gapic, "JobServiceClient", autospec=True)
  @mock.patch.object(os.path, "exists", autospec=True)
  @mock.patch.object(time, "sleep", autospec=True)
  def test_hptuning_job_remote_runner_returns_gcp_resources(
      self, mock_time_sleep, mock_path_exists, mock_job_service_client):
    job_client = mock.Mock()
    mock_job_service_client.return_value = job_client

    create_hptuning_job_response = mock.Mock()
    job_client.create_hyperparameter_tuning_job.return_value = create_hptuning_job_response
    create_hptuning_job_response.name = self._hptuning_job_name

    get_hptuning_job_response_success = mock.Mock()
    get_hptuning_job_response_success.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

    job_client.get_hyperparameter_tuning_job.side_effect = [
        get_hptuning_job_response_success
    ]

    mock_path_exists.return_value = False

    hyperparameter_tuning_job_remote_runner.create_hyperparameter_tuning_job(
        self._type, self._project, self._location, self._payload,
        self._gcp_resources)

    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()

      # Instantiate GCPResources Proto
      hptuning_job_resources = json_format.Parse(serialized_gcp_resources,
                                                 GcpResources())

      self.assertLen(hptuning_job_resources.resources, 1)
      hptuning_job_name = hptuning_job_resources.resources[0].resource_uri[
          len(self._hptuning_job_uri_prefix):]
      self.assertEqual(hptuning_job_name, self._hptuning_job_name)

  @mock.patch.object(aiplatform.gapic, "JobServiceClient", autospec=True)
  @mock.patch.object(time, "sleep", autospec=True)
  def test_hptuning_job_remote_runner_raises_exception_with_more_than_one_resource_in_gcp_resources(
      self, _, mock_job_service_client):
    job_client = mock.Mock()
    mock_job_service_client.return_value = job_client

    create_hptuning_job_response = mock.Mock()
    job_client.create_hyperparameter_tuning_job.return_value = create_hptuning_job_response
    create_hptuning_job_response.name = self._hptuning_job_name

    get_hptuning_job_response_success = mock.Mock()
    get_hptuning_job_response_success.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

    job_client.get_hyperparameter_tuning_job.side_effect = [
        get_hptuning_job_response_success
    ]

    # Write the job proto to output
    hptuning_job_resources = GcpResources()
    hptuning_job_resource_1 = hptuning_job_resources.resources.add()
    hptuning_job_resource_1.resource_type = "HyperparameterTuningJob"
    hptuning_job_resource_1.resource_uri = f"{self._hptuning_job_uri_prefix}{self._hptuning_job_name}"

    hptuning_job_resource_2 = hptuning_job_resources.resources.add()
    hptuning_job_resource_2.resource_type = "HyperparameterTuningJob"
    hptuning_job_resource_2.resource_uri = f"{self._hptuning_job_uri_prefix}{self._hptuning_job_name}"

    with open(self._gcp_resources, "w") as f:
      f.write(json_format.MessageToJson(hptuning_job_resources))

    with self.assertRaisesRegex(
        ValueError, "gcp_resources should contain one resource, found 2"):
      hyperparameter_tuning_job_remote_runner.create_hyperparameter_tuning_job(
          self._type, self._project, self._location, self._payload,
          self._gcp_resources)

  @mock.patch.object(aiplatform.gapic, "JobServiceClient", autospec=True)
  @mock.patch.object(time, "sleep", autospec=True)
  def test_hptuning_job_remote_runner_raises_exception_empty_uri_in_gcp_resources(
      self, _, mock_job_service_client):
    job_client = mock.Mock()
    mock_job_service_client.return_value = job_client

    create_hptuning_job_response = mock.Mock()
    job_client.create_hyperparameter_tuning_job.return_value = create_hptuning_job_response
    create_hptuning_job_response.name = self._hptuning_job_name

    get_hptuning_job_response_success = mock.Mock()
    get_hptuning_job_response_success.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED

    job_client.get_hyperparameter_tuning_job.side_effect = [
        get_hptuning_job_response_success
    ]

    # Write the job proto to output
    hptuning_job_resources = GcpResources()
    hptuning_job_resource_1 = hptuning_job_resources.resources.add()
    hptuning_job_resource_1.resource_type = "HyperparameterTuningJob"
    hptuning_job_resource_1.resource_uri = ""

    with open(self._gcp_resources, "w") as f:
      f.write(json_format.MessageToJson(hptuning_job_resources))

    with self.assertRaisesRegex(
        ValueError,
        "Job Name in gcp_resource is not formatted correctly or is empty."):
      hyperparameter_tuning_job_remote_runner.create_hyperparameter_tuning_job(
          self._type, self._project, self._location, self._payload,
          self._gcp_resources)


  @mock.patch.object(aiplatform.gapic, 'JobServiceClient', autospec=True)
  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(google.auth.transport.requests, 'Request', autospec=True)
  @mock.patch.object(requests, 'post', autospec=True)
  @mock.patch.object(ExecutionContext, '__init__', autospec=True)
  def test_hptuning_job_remote_runner_cancel(self,
                                             mock_execution_context,
                                             mock_post_requests,
                                             _, mock_auth,
                                             mock_job_service_client):
    creds = mock.Mock()
    creds.token = 'fake_token'
    mock_auth.return_value = [creds, "project"]

    job_client = mock.Mock()
    mock_job_service_client.return_value = job_client

    create_hptuning_job_response = mock.Mock()
    job_client.create_hyperparameter_tuning_job.return_value = create_hptuning_job_response
    create_hptuning_job_response.name = self._hptuning_job_name

    get_hptuning_job_response = mock.Mock()
    job_client.get_hyperparameter_tuning_job.return_value = get_hptuning_job_response
    get_hptuning_job_response.state = gca_job_state.JobState.JOB_STATE_SUCCEEDED
    mock_execution_context.return_value = None

    hyperparameter_tuning_job_remote_runner.create_hyperparameter_tuning_job(
        self._type, self._project, self._location, self._payload,
        self._gcp_resources)

    # Call cancellation handler
    mock_execution_context.call_args[1]['on_cancel']()
    mock_post_requests.assert_called_once_with(
        url=f'{self._hptuning_job_uri_prefix}{self._hptuning_job_name}:cancel',
        data='',
        headers={
            'Content-type': 'application/json',
            'Authorization': 'Bearer fake_token',
        })
