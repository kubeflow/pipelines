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
"""GCP launcher for Dataflow Flex Templates."""

import datetime
import json
import logging
import os
from os import path
import re
from typing import Any, Dict, Union
import uuid

import google.auth.transport.requests
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import gcp_labels_util
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import json_util
from google_cloud_pipeline_components.proto import gcp_resources_pb2
import requests
from requests.adapters import HTTPAdapter
from requests.sessions import Session
from urllib3.util.retry import Retry

from google.protobuf import json_format


_CONNECTION_ERROR_RETRY_LIMIT = 5
_CONNECTION_RETRY_BACKOFF_FACTOR = 2.0

_DATAFLOW_URI_PREFIX = 'https://dataflow.googleapis.com/v1b3'
_DATAFLOW_JOB_URI_TEMPLATE = rf'({_DATAFLOW_URI_PREFIX}/projects/(?P<project>.*)/locations/(?P<location>.*)/jobs/(?P<job>.*))'


def insert_system_labels_into_payload(payload):
  job_spec = json.loads(payload)
  try:
    labels = job_spec['launch_parameter']['environment'][
        'additional_user_labels'
    ]
  except KeyError:
    labels = {}

  if 'launch_parameter' not in job_spec.keys():
    job_spec['launch_parameter'] = {}
  if 'environment' not in job_spec['launch_parameter'].keys():
    job_spec['launch_parameter']['environment'] = {}
  if (
      'additional_user_labels'
      not in job_spec['launch_parameter']['environment'].keys()
  ):
    job_spec['launch_parameter']['environment']['additional_user_labels'] = {}

  labels = gcp_labels_util.attach_system_labels(labels)
  job_spec['launch_parameter']['environment']['additional_user_labels'] = labels
  return json.dumps(job_spec)


class DataflowFlexTemplateRemoteRunner:
  """Common module for creating Dataproc Flex Template jobs."""

  def __init__(
      self,
      type: str,
      project: str,
      location: str,
      gcp_resources: str,
  ):
    """Initializes a DataflowFlexTemplateRemoteRunner object."""
    self._type = type
    self._project = project
    self._location = location
    self._creds, _ = google.auth.default()
    self._gcp_resources = gcp_resources
    self._session = self._get_session()

  def _get_session(self) -> Session:
    """Gets a http session."""
    retry = Retry(
        total=_CONNECTION_ERROR_RETRY_LIMIT,
        status_forcelist=[429, 503],
        backoff_factor=_CONNECTION_RETRY_BACKOFF_FACTOR,
        allowed_methods=['GET', 'POST'],
    )
    adapter = HTTPAdapter(max_retries=retry)
    session = requests.Session()
    session.headers.update({
        'Content-Type': 'application/json',
        'User-Agent': 'google-cloud-pipeline-components',
    })
    session.mount('https://', adapter)
    return session

  def _post_resource(self, url: str, post_data: str) -> Dict[str, Any]:
    """POST a http request.

    Args:
      url: The resource url.
      post_data: The POST data.

    Returns:
      Dict of the JSON payload returned in the http response.

    Raises:
      RuntimeError: Failed to get or parse the http response.
    """
    if not self._creds.valid:
      self._creds.refresh(google.auth.transport.requests.Request())
    headers = {'Authorization': 'Bearer ' + self._creds.token}

    result = self._session.post(url=url, data=post_data, headers=headers)
    json_data = {}
    try:
      json_data = result.json()
      result.raise_for_status()
      return json_data
    except requests.exceptions.HTTPError as err:
      try:
        err_msg = (
            'Dataflow service returned HTTP status {} from POST: {}. Status:'
            ' {}, Message: {}'.format(
                err.response.status_code,
                err.request.url,
                json_data['error']['status'],
                json_data['error']['message'],
            )
        )
      except (KeyError, TypeError):
        err_msg = err.response.text
      # Raise RuntimeError with the error returned from the Dataflow service.
      # Suppress HTTPError as it provides no actionable feedback.
      raise RuntimeError(err_msg) from None
    except json.decoder.JSONDecodeError as err:
      raise RuntimeError(
          'Failed to decode JSON from response:\n{}'.format(err.doc)
      ) from err

  def check_if_job_exists(self) -> Union[Dict[str, Any], None]:
    """Check if a Dataflow job already exists.

    Returns:
      Dict of the Job resource if it exists. For more details, see:
        https://cloud.google.com/dataflow/docs/reference/rest/v1b3/projects.jobs#Job
      None if the Job resource does not exist.

    Raises:
      ValueError: Job resource uri format is invalid.
    """
    if (
        path.exists(self._gcp_resources)
        and os.stat(self._gcp_resources).st_size != 0
    ):
      with open(self._gcp_resources) as f:
        serialized_gcp_resources = f.read()

      job_resources = json_format.Parse(
          serialized_gcp_resources, gcp_resources_pb2.GcpResources()
      )
      # Resources should only contain one item.
      if len(job_resources.resources) != 1:
        raise ValueError(
            'gcp_resources should contain one resource, found'
            f' {len(job_resources.resources)}'
        )

      # Validate the format of the Job resource uri.
      job_name_pattern = re.compile(_DATAFLOW_JOB_URI_TEMPLATE)
      match = job_name_pattern.match(job_resources.resources[0].resource_uri)
      try:
        matched_project = match.group('project')
        matched_location = match.group('location')
        matched_job_id = match.group('job')
      except AttributeError as err:
        raise ValueError(
            'Invalid Resource uri: {}. Expect: {}.'.format(
                job_resources.resources[0].resource_uri,
                'https://dataflow.googleapis.com/v1b3/projects/[projectId]/locations/[location]/jobs/[jobId]',
            )
        ) from err

      # Return the Job resource uri.
      return job_resources.resources[0].resource_uri

    return None

  def create_job(
      self,
      type: str,
      job_request: Dict[str, Any],
  ) -> None:
    """Create a job using a Dataflow Flex Template.

    Args:
      type: Job type that is written to gcp_resources.
      job_request: A json serialized Job resource. For more details, see:
        https://cloud.google.com/dataflow/docs/reference/rest/v1b3/projects.jobs#Job

    Raises:
      RuntimeError: The Job resource uri cannot be constructed by by parsing the
        response.

    Returns:
      None
    """
    launch_job_url = f'https://dataflow.googleapis.com/v1b3/projects/{self._project}/locations/{self._location}/flexTemplates:launch'
    response = self._post_resource(launch_job_url, json.dumps(job_request))

    if not job_request.get('validate_only', False):
      try:
        job = response['job']
        job_uri = f"{_DATAFLOW_URI_PREFIX}/projects/{job['projectId']}/locations/{job['location']}/jobs/{job['id']}"
      except KeyError as err:
        raise RuntimeError(
            'Dataflow Flex Template launch failed. '
            'Cannot determine the job resource uri from the response:\n'
            f'{response}'
        ) from err

      # Write the Job resource to the gcp_resources output file.
      job_resources = gcp_resources_pb2.GcpResources()
      job_resource = job_resources.resources.add()
      job_resource.resource_type = type
      job_resource.resource_uri = job_uri
      with open(self._gcp_resources, 'w') as f:
        f.write(json_format.MessageToJson(job_resources))
      logging.info('Created Dataflow job: %s', job_uri)
    else:
      logging.info('No Dataflow job is created for request validation.')


def launch_flex_template(
    type: str,
    project: str,
    location: str,
    payload: str,
    gcp_resources: str,
) -> None:
  """Main function for launching a Dataflow Flex Template.

  Args:
    type: Job type that is written to gcp_resources.
    project: Project to launch the job.
    location: Location to launch the job.
    payload: A json serialized Job resource. For more details, see:
      https://cloud.google.com/dataflow/docs/reference/rest/v1b3/projects.jobs#Job
    gcp_resources: File path for storing `gcp_resources` output parameter.

  Returns:
    None
  """
  try:
    job_spec = json_util.recursive_remove_empty(
        json.loads(insert_system_labels_into_payload(payload), strict=False)
    )
  except json.decoder.JSONDecodeError as err:
    raise RuntimeError(
        'Failed to decode JSON from payload: {}'.format(err.doc)
    ) from err

  remote_runner = DataflowFlexTemplateRemoteRunner(
      type, project, location, gcp_resources
  )
  if not remote_runner.check_if_job_exists():
    if 'launch_parameter' not in job_spec.keys():
      job_spec['launch_parameter'] = {}
    if 'job_name' not in job_spec['launch_parameter'].keys():
      now = datetime.datetime.now().strftime('%Y%m%d%H%M%S')
      job_spec['launch_parameter']['job_name'] = '-'.join(
          [type.lower(), now, uuid.uuid4().hex[:8]]
      )
    remote_runner.create_job(type, job_spec)
