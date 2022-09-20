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
"""GCP launcher for infra validation jobs based on the AI Platform SDK."""

import json
import logging

from . import custom_job_remote_runner
from . import job_remote_runner
from .utils import error_util


ARTIFACT_PROPERTY_KEY_UNMANAGED_CONTAINER_MODEL = 'unmanaged_container_model'
API_KEY_CONTAINER_SPEC = 'containerSpec'


def construct_infra_validation_job_spec(executor_input,
                                        payload):
  """Construct infra validation payload for CustomJob."""
  # Extract artifact uri and prediction server uri
  artifact = json.loads(executor_input).get('inputs', {}).get(
      'artifacts', {}).get(ARTIFACT_PROPERTY_KEY_UNMANAGED_CONTAINER_MODEL,
                           {}).get('artifacts')
  if artifact:
    model_artifact_path = artifact[0].get('uri')
    prediction_server_image_uri = artifact[0].get('metadata', {}).get(
        API_KEY_CONTAINER_SPEC, {}).get('imageUri', {})
  else:
    raise ValueError('unmanaged_container_model not found in executor_input.')

  # Set the general contract args and env_var for infra validation job
  payload_json = json.loads(payload)
  args = ['--model_base_path', model_artifact_path]
  infra_validation_example_path = payload_json.get(
      'infra_validation_example_path')
  if infra_validation_example_path:
    args.extend(
        ['--infra_validation_example_path', infra_validation_example_path])

  env_variables = [{'name': 'INFRA_VALIDATION_MODE', 'value': '1'}]

  # Construct CustomJob payload
  custom_job_spec = {
      'display_name': 'infra-validation',
      'job_spec': {
          'worker_pool_specs': [{
              'machine_spec': {
                  'machine_type': 'n1-standard-4'
              },
              'replica_count': 1,
              'container_spec': {
                  'image_uri': prediction_server_image_uri,
                  'args': args,
                  'env': env_variables
              },
          }]
      },
  }

  return json.dumps(custom_job_spec)


def create_infra_validation_job(
    type,
    project,
    location,
    gcp_resources,
    executor_input,
    payload
):
  """Create and poll custom job status till it reaches a final state.

  This follows the typical launching logic:
  1. Read if the custom job already exists in gcp_resources
     - If already exists, jump to step 3 and poll the job status. This happens
       if the launcher container experienced unexpected termination, such as
       preemption
  2. parse the executor_input into the job spec and create the custom job
  3. Poll the custom job status every _POLLING_INTERVAL_IN_SECONDS seconds
     - If the custom job is succeeded, return succeeded
     - If the custom job is cancelled/paused, it's an unexpected scenario so
     return failed
     - If the custom job is running, continue polling the status
  Also retry on ConnectionError up to
  job_remote_runner._CONNECTION_ERROR_RETRY_LIMIT times during the poll.
  """
  remote_runner = job_remote_runner.JobRemoteRunner(type, project, location,
                                                    gcp_resources)

  try:
    # Create custom job if it does not exist
    job_name = remote_runner.check_if_job_exists()
    job_spec = construct_infra_validation_job_spec(
        executor_input, payload)
    if job_name is None:
      job_name = remote_runner.create_job(
          custom_job_remote_runner.create_custom_job_with_client, job_spec)

    # Poll custom job status until "JobState.JOB_STATE_SUCCEEDED"
    remote_runner.poll_job(custom_job_remote_runner.get_custom_job_with_client,
                           job_name)
  except (ConnectionError, RuntimeError, ValueError) as err:
    error_util.exit_with_internal_error(err.args[0])
