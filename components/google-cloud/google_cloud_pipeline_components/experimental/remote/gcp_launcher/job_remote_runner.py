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
"""Common module for creating GCP launchers based on the AI Platform SDK."""

import json
import logging
import os
from os import path
import time

from google.api_core import gapic_v1
from google.cloud import aiplatform
from google.cloud.aiplatform.compat.types import job_state as gca_job_state
from google.protobuf import json_format
from google_cloud_pipeline_components.experimental.proto.gcp_resources_pb2 import GcpResources

_POLLING_INTERVAL_IN_SECONDS = 20
_CONNECTION_ERROR_RETRY_LIMIT = 5

_JOB_COMPLETE_STATES = (
    gca_job_state.JobState.JOB_STATE_SUCCEEDED,
    gca_job_state.JobState.JOB_STATE_FAILED,
    gca_job_state.JobState.JOB_STATE_CANCELLED,
    gca_job_state.JobState.JOB_STATE_PAUSED,
)

_JOB_ERROR_STATES = (
    gca_job_state.JobState.JOB_STATE_FAILED,
    gca_job_state.JobState.JOB_STATE_CANCELLED,
    gca_job_state.JobState.JOB_STATE_PAUSED,
)


def create_job(job_type, project, location, payload, gcp_resources):
  """Create and poll job status till it reaches a final state.

  This follows the typical launching logic:
  1. Read if the job already exists in gcp_resources
     - If already exists, jump to step 3 and poll the job status. This happens
     if the launcher container experienced unexpected termination, such as
     preemption
  2. Deserialize the payload into the job spec and create the job.
  3. Poll the job status every _POLLING_INTERVAL_IN_SECONDS seconds
     - If the job is succeeded, return succeeded
     - If the job is cancelled/paused, it's an unexpected scenario so return
     failed
     - If the job is running, continue polling the status

  Also retry on ConnectionError up to _CONNECTION_ERROR_RETRY_LIMIT times during
  the poll.
  """
  if job_type != 'CustomJob' and job_type != 'BatchPredictionJob':
    logging.error('JobType must be type "CustomJob" or "BatchPredictionJob".')

  client_options = {'api_endpoint': location + '-aiplatform.googleapis.com'}
  client_info = gapic_v1.client_info.ClientInfo(
      user_agent='google-cloud-pipeline-components',)

  job_uri_prefix = f"https://{client_options['api_endpoint']}/v1/"

  # Initialize client that will be used to create and send requests.
  job_client = aiplatform.gapic.JobServiceClient(
      client_options=client_options, client_info=client_info)

  # Instantiate GCPResources Proto
  job_resources = GcpResources()
  job_resource = job_resources.resources.add()

  # Check if the job already exists
  if path.exists(gcp_resources) and os.stat(gcp_resources).st_size != 0:
    with open(gcp_resources) as f:
      serialized_gcp_resources = f.read()
      job_resource = json_format.Parse(serialized_gcp_resources, job_resource)
      job_name = job_resource.resource_uri[len(job_uri_prefix):]

      logging.info('%s name already exists: %s. Continue polling the status',
                   job_type, job_name)
  else:
    parent = f'projects/{project}/locations/{location}'
    job_spec = json.loads(payload, strict=False)
    if job_type == 'CustomJob':
      create_job_response = job_client.create_custom_job(
          parent=parent, custom_job=job_spec)
    elif job_type == 'BatchPredictionJob':
      create_job_response = job_client.create_batch_prediction_job(
          parent=parent, batch_prediction_job=job_spec)
    job_name = create_job_response.name

    # Write the job proto to output
    job_resource.resource_type = job_type
    job_resource.resource_uri = f'{job_uri_prefix}{job_name}'

    with open(gcp_resources, 'w') as f:
      f.write(json_format.MessageToJson(job_resource))

  # Poll the job status
  retry_count = 0
  while True:
    try:
      if job_type == 'CustomJob':
        get_job_response = job_client.get_custom_job(name=job_name)
      elif job_type == 'BatchPredictionJob':
        get_job_response = job_client.get_batch_prediction_job(name=job_name)
      retry_count = 0
    # Handle transient connection error.
    except ConnectionError as err:
      retry_count += 1
      if retry_count < _CONNECTION_ERROR_RETRY_LIMIT:
        logging.warning(
            'ConnectionError (%s) encountered when polling job: %s. Trying to '
            'recreate the API client.', err, job_name)
        # Recreate the Python API client.
        job_client = aiplatform.gapic.JobServiceClient(
            client_options=client_options)
      else:
        logging.error('Request failed after %s retries.',
                      _CONNECTION_ERROR_RETRY_LIMIT)
        # TODO(ruifang) propagate the error.
        raise

    if get_job_response.state == gca_job_state.JobState.JOB_STATE_SUCCEEDED:
      logging.info('Get%s response state =%s', job_type, get_job_response.state)
      return
    elif get_job_response.state in _JOB_ERROR_STATES:
      # TODO(ruifang) propagate the error.
      raise RuntimeError('Job failed with error state: {}.'.format(
          get_job_response.state))
    else:
      logging.info(
          'Job %s is in a non-final state %s.'
          ' Waiting for %s seconds for next poll.', job_name,
          get_job_response.state, _POLLING_INTERVAL_IN_SECONDS)
      time.sleep(_POLLING_INTERVAL_IN_SECONDS)
