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

# import json
# import logging
import re

from typing import List, Union

# import google.auth
from google_cloud_pipeline_components.container.v1.bigquery.utils.bigquery_util import _send_cancel_request as send_cancel_bigquery_job_request
from google_cloud_pipeline_components.container.v1.dataproc.utils.dataproc_util import _cancel_batch as send_cancel_dataproc_job_request
from google_cloud_pipeline_components.container.v1.gcp_launcher.job_remote_runner import send_cancel_request as send_cancel_vertex_job_request
from google_cloud_pipeline_components.container.v1.wait_gcp_resources.remote_runner import _send_cancel_request as send_cancel_dataflow_job_request
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
# import googleapiclient.discovery as discovery
# import requests

from google.protobuf.json_format import Parse

_SUPPORTED_JOB_TYPES = [
    'DataflowJob',
    'VertexLro',
    'BigQueryJob',
    'BatchPredictionJob',
    'HyperparameterTuningJob',
    'CustomJob',
    'DataprocLro',
]
_JOB_CANCELLED_STATE = 'JOB_STATE_CANCELLED'
_DATAFLOW_URI_TEMPLATE = r'(https://dataflow.googleapis.com/v1b3/projects/(?P<project>.*)/locations/(?P<location>.*)/jobs/(?P<jobid>.*))'
_JOB_URI_TEMPLATE = (
    r'https://(?P<location>.*)-aiplatform.googleapis.com/v1/(?P<jobname>.*)'
)
_DATAPROC_URI_TEMPLATE = r'https://dataproc.googleapis.com/v1/(?P<jobname>.*)'


def cancel_dataflow_job(job_uri) -> None:
  """Cancels a running dataflow job."""
  uri_pattern = re.compile(_DATAFLOW_URI_TEMPLATE)
  match = uri_pattern.match(job_uri)
  try:
    project = match.group('project')
    location = match.group('location')
    job_id = match.group('jobid')
  except AttributeError as err:
    expected_template = 'https://dataflow.googleapis.com/v1b3/projects/{project_id}/locations/{location}/jobs/{job_id}'
    raise ValueError(
        f'Invalid dataflow resource URI: {job_uri}. Expect:'
        f' {expected_template}.'
    ) from err

  send_cancel_dataflow_job_request(project, job_id, location)


def cancel_biquery_job(job_uri) -> None:
  """Cancels a running bigquery job using the job_uri."""
  send_cancel_bigquery_job_request(job_uri)


def cancel_vertex_job(job_uri) -> None:
  """Cancels a running vertex job."""
  uri_pattern = re.compile(_JOB_URI_TEMPLATE)
  match = uri_pattern.match(job_uri)
  try:
    location = match.group('location')
    job_name = match.group('jobname')
  except AttributeError as err:
    expected_template = (
        'https://{location})-aiplatform.googleapis.com/v1/{jobname}'
    )
    raise ValueError(
        f'Invalid custom Job resource URI: {job_uri}. Expect:'
        f' {expected_template}.'
    ) from err
  send_cancel_vertex_job_request(location, job_name)


def cancel_vertex_lro_job(job_uri) -> None:
  """Cancels a running vertex Lro job."""
  uri_pattern = re.compile(_JOB_URI_TEMPLATE)
  match = uri_pattern.match(job_uri)
  try:
    location = match.group('location')
    job_name = match.group('jobname')
  except AttributeError as err:
    expected_template = (
        'https://{location})-aiplatform.googleapis.com/v1/{jobname}'
    )
    raise ValueError(
        f'Invalid custom Job resource URI: {job_uri}. Expect:'
        f' {expected_template}.'
    ) from err
  send_cancel_vertex_job_request(location, job_name)


def cancel_dataproc_job(job_uri) -> None:
  """Cancels a running data proc job."""
  uri_pattern = re.compile(_DATAPROC_URI_TEMPLATE)
  match = uri_pattern.match(job_uri)
  try:
    job_name = match.group('jobname')
  except AttributeError as err:
    expected_template = 'https://dataproc.googleapis.com/v1/{jobname}'
    raise ValueError(
        f'Invalid dataproc Job resource URI: {job_uri}. Expect:'
        f' {expected_template}.'
    ) from err
  send_cancel_dataproc_job_request(job_name)


def cancel_job(gcp_resources) -> None:
  """Cancels a running job."""
  if gcp_resources == 'default':
    raise ValueError(
        'No gcp_resources provided, Job may have already completed'
    )

  try:
    gcp_resources = Parse(gcp_resources, GcpResources())
  except ValueError as err:
    raise ValueError(f'Invalid gcp_resources: {gcp_resources}') from err

  if len(gcp_resources.resources) != 1:
    raise ValueError(
        f'Invalid gcp_resources: {gcp_resources}. Cancel job component supports'
        ' cancelling only one resource at this moment.'
    )
  resource_type = gcp_resources.resources[0].resource_type
  if resource_type not in _SUPPORTED_JOB_TYPES:
    raise ValueError(
        f'Invalid gcp_resources: {gcp_resources}. Resource type not supported'
    )

  job_uri = gcp_resources.resources[0].resource_uri

  if resource_type == 'DataflowJob':
    cancel_dataflow_job(job_uri)
  elif resource_type == 'BatchPredictionJob':
    cancel_vertex_job(job_uri)
  elif resource_type == 'CustomJob':
    cancel_vertex_job(job_uri)
  elif resource_type == 'HyperparameterTuningJob':
    cancel_vertex_job(job_uri)
  elif resource_type == 'VertexLro':
    cancel_vertex_lro_job(job_uri)
  elif resource_type == 'DataprocLro':
    cancel_dataproc_job(job_uri)
  elif resource_type == 'BigQueryJob':
    cancel_biquery_job(job_uri)
  else:
    raise ValueError(
        f'Invalid gcp_resources: {gcp_resources}. Resource type not supported'
    )
