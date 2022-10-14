# Copyright 2022 The Kubeflow Authors. All Rights Reserved.
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

from google_cloud_pipeline_components.container.v1.custom_job.remote_runner import create_custom_job
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import error_util

ARTIFACT_PROPERTY_KEY_UNMANAGED_CONTAINER_MODEL = 'unmanaged_container_model'
API_KEY_CONTAINER_SPEC = 'containerSpec'


def construct_infra_validation_job_payload(executor_input, payload):
  """Construct infra validation payload for CustomJob."""
  # Extract artifact uri and prediction server uri
  artifact = json.loads(executor_input).get('inputs', {}).get(
      'artifacts', {}).get(ARTIFACT_PROPERTY_KEY_UNMANAGED_CONTAINER_MODEL,
                           {}).get('artifacts')
  if artifact:
    model_artifact_path = artifact[0].get('uri')
    prediction_server_image_uri = artifact[0].get('metadata', {}).get(
        API_KEY_CONTAINER_SPEC, {}).get('imageUri', '')
  else:
    raise ValueError('unmanaged_container_model not found in executor_input.')

  # Set the general contract args and env_var for infra validation job
  payload_json = json.loads(payload)
  args = ['--model_base_path', model_artifact_path]

  # extract infra validation example path
  infra_validation_example_path = payload_json.get(
      'infra_validation_example_path')
  if infra_validation_example_path:
    args.extend(
        ['--infra_validation_example_path', infra_validation_example_path])

  env_variables = [{'name': 'INFRA_VALIDATION_MODE', 'value': '1'}]

  # Construct CustomJob payload
  custom_job_payload = {
      'display_name': 'infra-validation',
      'job_spec': {
          'worker_pool_specs': [{
              'machine_spec': {
                  'machine_type': payload_json.get('machine_type')
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

  return json.dumps(custom_job_payload)


def create_infra_validation_job(
    type,
    project,
    location,
    gcp_resources,
    executor_input,
    payload
):
  """Create and poll infra validation job status till it reaches a final state.

  This follows the typical launching logic:
  1. Read if the infra validation job already exists in gcp_resources
     - If already exists, jump to step 3 and poll the job status. This happens
       if the launcher container experienced unexpected termination, such as
       preemption
  2. parse the executor_input into the infra validation custom job spec and
  create the custom job
  3. Poll the custom job status every _POLLING_INTERVAL_IN_SECONDS seconds
     - If the custom job is succeeded, return succeeded
     - If the custom job is cancelled/paused, it's an unexpected scenario so
     return failed
     - If the custom job is running, continue polling the status
  Also retry on ConnectionError up to
  job_remote_runner._CONNECTION_ERROR_RETRY_LIMIT times during the poll.
  """
  try:
    infra_validation_job_payload = construct_infra_validation_job_payload(
        executor_input, payload)
    create_custom_job(type, project, location, infra_validation_job_payload,
                      gcp_resources)
  except (ConnectionError, RuntimeError, ValueError) as err:
    error_util.exit_with_internal_error(err.args[0])

