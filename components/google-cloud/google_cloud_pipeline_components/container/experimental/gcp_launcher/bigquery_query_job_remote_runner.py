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

import logging
import json
import re
import os
import requests
import time
import google.auth
import google.auth.transport.requests

from .utils import json_util
from .utils import artifact_util
from google.cloud import bigquery
from google.protobuf import json_format
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
from os import path
from typing import Optional

_POLLING_INTERVAL_IN_SECONDS = 20
_BQ_JOB_NAME_TEMPLATE = r'(https://www.googleapis.com/bigquery/v2/projects/(?P<project>.*)/jobs/(?P<job>.*)\?location=(?P<location>.*))'


def check_if_job_exists(gcp_resources) -> Optional[str]:
  """Check if the BigQuery job already created.

  Return the job url if created. Return None otherwise
  """
  if path.exists(gcp_resources) and os.stat(gcp_resources).st_size != 0:
    with open(gcp_resources) as f:
      serialized_gcp_resources = f.read()
      job_resources = json_format.Parse(serialized_gcp_resources,
                                        GcpResources())
      # Resources should only contain one item.
      if len(job_resources.resources) != 1:
        raise ValueError(
            f'gcp_resources should contain one resource, found {len(job_resources.resources)}'
        )
      # Validate the format of the resource uri.
      job_name_pattern = re.compile(_BQ_JOB_NAME_TEMPLATE)
      match = job_name_pattern.match(job_resources.resources[0].resource_uri)
      try:
        project = match.group('project')
        job = match.group('job')
      except AttributeError as err:
        raise ValueError('Invalid bigquery job uri: {}. Expect: {}.'.format(
            job_resources.resources[0].resource_uri,
            'https://www.googleapis.com/bigquery/v2/projects/[projectId]/jobs/[jobId]?location=[location]'
        ))

    return job_resources.resources[0].resource_uri
  else:
    return None


def create_job(job_type, project, location, payload, creds,
               gcp_resources) -> str:
  """Create a new BigQuery job"""
  job_configuration = json.loads(payload, strict=False)
  # Always use standard SQL instead of legacy SQL.
  job_configuration['query']['useLegacySql'] = False
  job_request = {
      # TODO(IronPan) temporarily remove the empty fields from the spec
      'configuration': json_util.recursive_remove_empty(job_configuration),
  }
  if location is not None:
    if 'jobReference' not in job_request:
      job_request['jobReference'] = {}
    job_request['jobReference']['location'] = location

  creds.refresh(google.auth.transport.requests.Request())
  headers = {
      'Content-type': 'application/json',
      'Authorization': 'Bearer ' + creds.token,
      'User-Agent': 'google-cloud-pipeline-components'
  }
  insert_job_url = f'https://www.googleapis.com/bigquery/v2/projects/{project}/jobs'
  job = requests.post(
      url=insert_job_url, data=json.dumps(job_request), headers=headers).json()
  if 'selfLink' not in job:
    raise RuntimeError(
        'BigQquery Job failed. Cannot retrieve the job name. Response: {}.'
        .format(job))

  # Write the bigquey job uri to gcp resource.
  job_uri = job['selfLink']
  job_resources = GcpResources()
  job_resource = job_resources.resources.add()
  job_resource.resource_type = job_type
  job_resource.resource_uri = job_uri
  with open(gcp_resources, 'w') as f:
    f.write(json_format.MessageToJson(job_resources))

  return job_uri


def poll_job(job_uri, creds) -> dict:
  """Poll the bigquery job till it reaches a final state."""
  job = {}
  while ('status' not in job) or ('state' not in job['status']) or (
      job['status']['state'].lower() != 'done'):
    time.sleep(_POLLING_INTERVAL_IN_SECONDS)
    logging.info('The job is running...')
    if not creds.valid:
      creds.refresh(google.auth.transport.requests.Request())
    headers = {
        'Content-type': 'application/json',
        'Authorization': 'Bearer ' + creds.token
    }
    job = requests.get(job_uri, headers=headers).json()
    if 'status' in job and 'errorResult' in job['status']:
      raise RuntimeError('The BigQuery job failed. Error: {}'.format(
          job['status']))

  logging.info('BigQuery Job completed succesesfully. Job: %s.', job)
  return job


def create_bigquery_job(
    type,
    project,
    location,
    payload,
    gcp_resources,
    executor_input,
):
  """Create and poll bigquery job status till it reaches a final state.

  This follows the typical launching logic:
  1. Read if the bigquery job already exists in gcp_resources
     - If already exists, jump to step 3 and poll the job status. This happens
     if the launcher container experienced unexpected termination, such as
     preemption
  2. Deserialize the payload into the job spec and create the bigquery job
  3. Poll the bigquery job status every
  job_remote_runner._POLLING_INTERVAL_IN_SECONDS seconds
     - If the bigquery job is succeeded, return succeeded
     - If the bigquery job is pending/running, continue polling the status

  Also retry on ConnectionError up to
  job_remote_runner._CONNECTION_ERROR_RETRY_LIMIT times during the poll.
  """
  creds, _ = google.auth.default()
  job_uri = check_if_job_exists(gcp_resources)
  if job_uri is None:
    job_uri = create_job(type, project, location, payload, creds, gcp_resources)

  # Poll bigquery job status until finished.
  job = poll_job(job_uri, creds)

  # write destination_table output artifact
  if 'destinationTable' in job['configuration']['query']:
    projectId = job['configuration']['query']['destinationTable']['projectId']
    datasetId = job['configuration']['query']['destinationTable']['datasetId']
    tableId = job['configuration']['query']['destinationTable']['tableId']
    artifact_util.update_output_artifact(
        executor_input, 'destinationTable',
        f'https://www.googleapis.com/bigquery/v2/projects/{projectId}/datasets/{datasetId}/tables/{tableId}',
        {})
