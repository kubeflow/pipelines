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
"""GCP launcher for Dataproc Batch workloads."""

import json
import logging
import os
from os import path
import re
import time
from typing import Any, Dict, Optional, Union

import google.auth.transport.requests
from google_cloud_pipeline_components.proto import gcp_resources_pb2
import requests
from requests.adapters import HTTPAdapter
from requests.sessions import Session
from urllib3.util.retry import Retry
from .utils import json_util
from ...utils import execution_context

from google.protobuf import json_format


_POLL_INTERVAL_SECONDS = 20
_CONNECTION_ERROR_RETRY_LIMIT = 5
_CONNECTION_RETRY_BACKOFF_FACTOR = 2.

_DATAPROC_URI_PREFIX = 'https://dataproc.googleapis.com/v1'
_DATAPROC_OPERATION_URI_TEMPLATE = rf'({_DATAPROC_URI_PREFIX}/projects/(?P<project>.*)/regions/(?P<region>.*)/operations/(?P<operation>.*))'


class DataprocBatchRemoteRunner():
  """Common module for creating and polling Dataproc Serverless Batch workloads."""

  def __init__(
      self,
      type: str,
      project: str,
      location: str,
      gcp_resources: str
  ) -> None:
    """Initializes a DataprocBatchRemoteRunner object."""
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
        method_whitelist=['GET', 'POST']
    )
    adapter = HTTPAdapter(max_retries=retry)
    session = requests.Session()
    session.headers.update({
        'Content-Type': 'application/json',
        'User-Agent': 'google-cloud-pipeline-components'
    })
    session.mount('https://', adapter)
    return session

  def _get_resource(self, url: str) -> Dict[str, Any]:
    """GET a http request.

    Args:
      url: The resource url.

    Returns:
      Dict of the JSON payload returned in the http response.

    Raises:
      RuntimeError: Failed to get or parse the http response.
    """
    if not self._creds.valid:
      self._creds.refresh(google.auth.transport.requests.Request())
    headers = {'Authorization': 'Bearer '+ self._creds.token}

    result = self._session.get(url, headers=headers)
    try:
      result.raise_for_status()
      return result.json()
    except requests.exceptions.HTTPError as err:
      raise RuntimeError('Failed to GET resource: {}. Error response:\n{}'
                         .format(url, result)) from err
    except json.decoder.JSONDecodeError as err:
      raise RuntimeError('Failed to decode JSON from response: {}'
                         .format(result)) from err

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
    headers = {'Authorization': 'Bearer '+ self._creds.token}

    result = self._session.post(url=url, data=post_data, headers=headers)
    try:
      result.raise_for_status()
      json_data = result.json()
      return json_data
    except requests.exceptions.HTTPError as err:
      raise RuntimeError('Failed to POST resource: {}. Error response:\n{}'
                         .format(url, result)) from err
    except json.decoder.JSONDecodeError as err:
      raise RuntimeError('Failed to decode JSON from response: {}'
                         .format(result)) from err

  def _cancel_batch(self, lro_name) -> None:
    """Cancels a Dataproc batch workload."""
    if not lro_name:
      return
    # Dataproc Operation Cancel API:
    # https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.operations/cancel
    lro_uri = f'{_DATAPROC_URI_PREFIX}/{lro_name}:cancel'
    self._post_resource(lro_uri, '')

  def check_if_operation_exists(self) -> Union[Dict[str, Any], None]:
    """Check if a Dataproc Batch operation already exists.

    Returns:
      Dict of the long-running Operation resource if it exists. For more details, see:
         https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.operations#resource:-operation
      None if the Operation resource does not exist.

    Raises:
      ValueError: Operation resource uri format is invalid.
    """
    if path.exists(self._gcp_resources) and os.stat(self._gcp_resources).st_size != 0:
      with open(self._gcp_resources) as f:
        serialized_gcp_resources = f.read()

      job_resources = json_format.Parse(serialized_gcp_resources,
                                        gcp_resources_pb2.GcpResources())
      # Resources should only contain one item.
      if len(job_resources.resources) != 1:
        raise ValueError(
            f'gcp_resources should contain one resource, found {len(job_resources.resources)}'
        )

      # Validate the format of the Operation resource uri.
      job_name_pattern = re.compile(_DATAPROC_OPERATION_URI_TEMPLATE)
      match = job_name_pattern.match(job_resources.resources[0].resource_uri)
      try:
        matched_project = match.group('project')
        matched_region = match.group('region')
        matched_operation_id = match.group('operation')
      except AttributeError as err:
        raise ValueError('Invalid Resource uri: {}. Expect: {}.'.format(
            job_resources.resources[0].resource_uri,
            'https://dataproc.googleapis.com/v1/projects/[projectId]/regions/[region]/operations/[operationId]'
        )) from err

      # Get the long-running Operation resource.
      lro = self._get_resource(job_resources.resources[0].resource_uri)
      return lro
    else:
      return None

  def wait_for_batch(
      self,
      lro: Dict[str, Any],
      poll_interval_seconds: int
  ) -> Dict[str, Any]:
    """Waits for a Dataproc batch workload to reach a final state.

    Args:
      lro: Dict of the long-running Operation resource. For more details, see:
        https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.operations#resource:-operation
      poll_interval_seconds: Seconds to wait between polls.

    Returns:
      Dict of the long-running Operation resource. For more details, see:
         https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.operations#resource:-operation

    Raises:
      RuntimeError: Operation resource indicates an error.
    """
    lro_name = lro['name']
    lro_uri = f'{_DATAPROC_URI_PREFIX}/{lro_name}'
    with execution_context.ExecutionContext(
        on_cancel=lambda: self._cancel_batch(lro_name)):
      while ('done' not in lro) or (not lro['done']):
        time.sleep(poll_interval_seconds)
        lro = self._get_resource(lro_uri)
        logging.info('Polled operation: %s', lro_name)

      if 'error' in lro and lro['error']['code']:
        raise RuntimeError(
            'Operation failed. Error: {}'.format(lro['error']))
      else:
        logging.info('Operation complete: %s', lro)
        return lro

  def create_batch(
      self,
      batch_id: str,
      batch_request: Dict[str, Any]
  ) -> Dict[str, Any]:
    """Common function for creating a batch workload.

    Args:
      batch_id: Dataproc batch id.
      batch_request: Dict of the Batch resource. For more details, see:
        https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.locations.batches#:-batch

    Returns:
       Dict of the long-running Operation resource. For more details, see:
         https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.operations#resource:-operation
    """
    # Create the Batch resource.
    create_batch_url = f'https://dataproc.googleapis.com/v1/projects/{self._project}/locations/{self._location}/batches/?batchId={batch_id}'
    lro = self._post_resource(create_batch_url, json.dumps(batch_request))
    lro_name = lro['name']

    # Write the Operation resource uri to the gcp_resources output file.
    job_resources = gcp_resources_pb2.GcpResources()
    job_resource = job_resources.resources.add()
    job_resource.resource_type = 'DataprocLro'
    job_resource.resource_uri = f'{_DATAPROC_URI_PREFIX}/{lro_name}'
    with open(self._gcp_resources, 'w') as f:
      f.write(json_format.MessageToJson(job_resources))

    return lro


def _create_batch(
    type: str,
    project: str,
    location: str,
    batch_id: str,
    payload: Optional[str],
    gcp_resources: str,
) -> Dict[str, Any]:
  """Common function for creating Dataproc Batch workloads.

  Args:
    type: Dataproc job type that is written to gcp_resources.
    project: Project to launch the batch workload.
    location: Location to launch the batch workload.
    batch_id: Dataproc batch id.
    payload: A json serialized Batch resource. For more details, see:
      https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.locations.batches#:-batch
    gcp_resources: File path for storing `gcp_resources` output parameter.

  Returns:
    Dict of the completed Batch resource. For more details, see:
      https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.locations.batches#:-batch
  """
  try:
    batch_request = json_util.recursive_remove_empty(
        json.loads(payload, strict=False))
  except json.decoder.JSONDecodeError as err:
    raise RuntimeError('Failed to decode JSON from payload: {}'
                       .format(payload)) from err

  remote_runner = DataprocBatchRemoteRunner(type, project, location, gcp_resources)
  lro = remote_runner.check_if_operation_exists()
  if not lro:
    lro = remote_runner.create_batch(batch_id, batch_request)
  # Wait for the Batch workload to finish.
  return remote_runner.wait_for_batch(lro, _POLL_INTERVAL_SECONDS)


# A common function can be used for all Batch workload types.
create_pyspark_batch = _create_batch
create_spark_batch = _create_batch
create_spark_r_batch = _create_batch
create_spark_sql_batch = _create_batch
