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

import logging
import re
import time

from functools import partial
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
import googleapiclient.discovery as discovery
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import error_util
from google_cloud_pipeline_components.container.utils import execution_context

from google.protobuf.json_format import Parse

_POLLING_INTERVAL_IN_SECONDS = 20
_CONNECTION_ERROR_RETRY_LIMIT = 5

_JOB_SUCCESSFUL_STATES = ['JOB_STATE_DONE']
_JOB_CANCELLED_STATE = 'JOB_STATE_CANCELLED'
_JOB_FAILED_STATES = [
    'JOB_STATE_STOPPED', 'JOB_STATE_FAILED', _JOB_CANCELLED_STATE,
    'JOB_STATE_UPDATED', 'JOB_STATE_DRAINED'
]
_JOB_TERMINATED_STATES = _JOB_SUCCESSFUL_STATES + _JOB_FAILED_STATES
_DATAFLOW_URI_TEMPLATE = r'(https://dataflow.googleapis.com/v1b3/projects/(?P<project>.*)/locations/(?P<location>.*)/jobs/(?P<jobid>.*))'


def wait_gcp_resources(
    type,
    project,
    location,
    payload,
    gcp_resources,
):
  """
    Poll the gcp resources till it reaches a final state.
    """
  input_gcp_resources = Parse(payload, GcpResources())
  if len(input_gcp_resources.resources) != 1:
    raise ValueError(
        'Invalid payload: %s. Wait component support waiting on only one resource at this moment.'
        % payload)

  if input_gcp_resources.resources[0].resource_type != 'DataflowJob':
    raise ValueError(
        'Invalid payload: %s. Wait component only support waiting on Dataflow job at this moment.'
        % payload)

  dataflow_job_uri = input_gcp_resources.resources[0].resource_uri
  uri_pattern = re.compile(_DATAFLOW_URI_TEMPLATE)
  match = uri_pattern.match(dataflow_job_uri)
  # Get the project and location from the job URI instead from the parameter.
  try:
    project = match.group('project')
    location = match.group('location')
    job_id = match.group('jobid')
  except AttributeError as err:
    # TODO(ruifang) propagate the error.
    raise ValueError('Invalid dataflow resource URI: {}. Expect: {}.'.format(
        dataflow_job_uri,
        'https://dataflow.googleapis.com/v1b3/projects/[project_id]/locations/[location]/jobs/[job_id]'
    ))

  # Propagate the GCP resources as the output of the wait component
  with open(gcp_resources, 'w') as f:
    f.write(payload)

  with execution_context.ExecutionContext(
      on_cancel=partial(
          _send_cancel_request,
          project,
          job_id,
          location,
      )):
    # Poll the job status
    retry_count = 0
    while True:
      try:
        df_client = discovery.build('dataflow', 'v1b3', cache_discovery=False)
        job = df_client.projects().locations().jobs().get(
            projectId=project, jobId=job_id, location=location,
            view=None).execute()
        retry_count = 0
      except ConnectionError as err:
        retry_count += 1
        if retry_count <= _CONNECTION_ERROR_RETRY_LIMIT:
          logging.warning(
              'ConnectionError (%s) encountered when polling job: %s. Retrying.',
              err, job_id)
        else:
          error_util.exit_with_internal_error(
              'Request failed after %s retries.'.format(
                  _CONNECTION_ERROR_RETRY_LIMIT))

      job_state = job.get('currentState', None)
      # Write the job details as gcp_resources
      if job_state in _JOB_SUCCESSFUL_STATES:
        logging.info('GetDataflowJob response state =%s. Job completed',
                     job_state)
        return

      elif job_state in _JOB_TERMINATED_STATES:
        # TODO(ruifang) propagate the error.
        raise RuntimeError('Job {} failed with error state: {}.'.format(
            job_id, job_state))
      else:
        logging.info(
            'Job %s is in a non-final state %s. Waiting for %s seconds for next poll.',
            job_id, job_state, _POLLING_INTERVAL_IN_SECONDS)
        time.sleep(_POLLING_INTERVAL_IN_SECONDS)


def _send_cancel_request(project, job_id, location):
  logging.info('dataflow_cancelling_job_params: %s, %s, %s', project, job_id,
               location)
  df_client = discovery.build('dataflow', 'v1b3', cache_discovery=False)
  job = df_client.projects().locations().jobs().get(
      projectId=project, jobId=job_id, location=location, view=None).execute()
  # Dataflow cancel API:
  # https://cloud.google.com/dataflow/docs/guides/stopping-a-pipeline#stopping_a_job
  logging.info('Sending Dataflow cancel request')
  job['requestedState'] = _JOB_CANCELLED_STATE
  logging.info('dataflow_cancelling_job: %s', job)
  job = df_client.projects().locations().jobs().update(
      projectId=project,
      jobId=job_id,
      location=location,
      body=job,
  ).execute()
  logging.info('dataflow_cancelled_job: %s', job)
  job = df_client.projects().locations().jobs().get(
      projectId=project, jobId=job_id, location=location, view=None).execute()
  logging.info('dataflow_cancelled_job: %s', job)
