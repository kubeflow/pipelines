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
"""Common module for launching and managing the Vertex Pipeline resources."""

import json
import logging
import os
from os import path
import re
import time
from typing import Any, Callable, Optional, Union

from google.api_core import client_options
from google.api_core import gapic_v1
import google.auth
import google.auth.transport.requests
from google.cloud import aiplatform
from google.cloud.aiplatform_v1.types import pipeline_job
from google.cloud.aiplatform_v1.types import pipeline_state
from google.cloud.aiplatform_v1.types import training_pipeline
from google_cloud_pipeline_components.container.utils import execution_context
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import error_util
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import json_util
from google_cloud_pipeline_components.proto import gcp_resources_pb2
import requests

from google.rpc import code_pb2
from google.protobuf import json_format


_POLLING_INTERVAL_IN_SECONDS = 20
_CONNECTION_ERROR_RETRY_LIMIT = 5

_PIPELINE_COMPLETE_STATES = (
    pipeline_state.PipelineState.PIPELINE_STATE_SUCCEEDED,
    pipeline_state.PipelineState.PIPELINE_STATE_FAILED,
    pipeline_state.PipelineState.PIPELINE_STATE_CANCELLED,
    pipeline_state.PipelineState.PIPELINE_STATE_PAUSED,
)

_PIPELINE_ERROR_STATES = (
    pipeline_state.PipelineState.PIPELINE_STATE_FAILED,
    pipeline_state.PipelineState.PIPELINE_STATE_CANCELLED,
    pipeline_state.PipelineState.PIPELINE_STATE_PAUSED,
)

_PIPELINE_USER_ERROR_CODES = (
    code_pb2.INVALID_ARGUMENT,
    code_pb2.NOT_FOUND,
    code_pb2.PERMISSION_DENIED,
    code_pb2.ALREADY_EXISTS,
    code_pb2.FAILED_PRECONDITION,
    code_pb2.OUT_OF_RANGE,
    code_pb2.UNIMPLEMENTED,
)


class PipelineRemoteRunner:
  """Common module for creating and polling pipelines on the Vertex Platform."""

  def __init__(
      self, pipeline_type: str, project: str, location: str, gcp_resources: str
  ):
    """Initializes a pipeline client and other common attributes."""
    self.pipeline_type = pipeline_type
    self.project = project
    self.location = location
    self.gcp_resources = gcp_resources
    self.client_options = client_options.ClientOptions(
        api_endpoint=location + '-aiplatform.googleapis.com'
    )
    self.client_info = gapic_v1.client_info.ClientInfo(
        user_agent='google-cloud-pipeline-components'
    )
    self.pipeline_client = aiplatform.gapic.PipelineServiceClient(
        client_options=self.client_options, client_info=self.client_info
    )
    self.pipeline_uri_prefix = f'https://{self.client_options.api_endpoint}/v1/'
    self.poll_pipeline_name = ''

  def check_if_pipeline_exists(self) -> Optional[str]:
    """Checks if the pipeline already exists.

    Returns:
      The pipeline name if the pipeline already exists. Otherwise, None.
    """
    if (
        path.exists(self.gcp_resources)
        and os.stat(self.gcp_resources).st_size != 0
    ):
      with open(self.gcp_resources) as f:
        serialized_gcp_resources = f.read()
        pipeline_resources: gcp_resources_pb2.GcpResources = json_format.Parse(
            serialized_gcp_resources, gcp_resources_pb2.GcpResources()
        )
        # Resources should only contain one item.
        if len(pipeline_resources.resources) != 1:
          raise ValueError(
              'gcp_resources should contain one resource, found'
              f' {len(pipeline_resources.resources)}'
          )

        pipeline_name_group = re.findall(
            f'{self.pipeline_uri_prefix}(.*)',
            pipeline_resources.resources[0].resource_uri,
        )

        if not pipeline_name_group or not pipeline_name_group[0]:
          raise ValueError(
              'Pipeline Name in gcp_resource is not formatted correctly or is'
              ' empty.'
          )
        pipeline_name = pipeline_name_group[0]

        logging.info(
            '%s name already exists: %s. Continue polling the status',
            self.pipeline_type,
            pipeline_name,
        )
      return pipeline_name
    else:
      return None

  def create_pipeline(
      self,
      create_pipeline_fn: Callable[
          [aiplatform.gapic.PipelineServiceClient, str, Any],
          Union[training_pipeline.TrainingPipeline, pipeline_job.PipelineJob],
      ],
      payload: str,
  ) -> str:
    """Creates a pipeline job or training pipeline.

    Args:
      create_pipeline_fn: A function that creates a pipeline from the pipeline
        client and pipeline spec.
      payload: JSON string to parse the pipeline spec from.

    Returns:
      The created pipeline name.
    """
    parent = f'projects/{self.project}/locations/{self.location}'
    # TODO(kevinbnaughton) remove empty fields from the spec temporarily.
    pipeline_spec = json_util.recursive_remove_empty(
        json.loads(payload, strict=False)
    )
    create_pipeline_response = create_pipeline_fn(
        self.pipeline_client, parent, pipeline_spec
    )
    pipeline_name = create_pipeline_response.name

    # Write the pipeline proto to output.
    pipeline_resources = gcp_resources_pb2.GcpResources()
    pipeline_resource = pipeline_resources.resources.add()
    pipeline_resource.resource_type = self.pipeline_type
    pipeline_resource.resource_uri = (
        f'{self.pipeline_uri_prefix}{pipeline_name}'
    )

    with open(self.gcp_resources, 'w') as f:
      f.write(json_format.MessageToJson(pipeline_resources))

    return pipeline_name

  def poll_pipeline(
      self,
      get_pipeline_fn: Callable[
          [aiplatform.gapic.PipelineServiceClient, str],
          Union[training_pipeline.TrainingPipeline, pipeline_job.PipelineJob],
      ],
      pipeline_name: str,
  ) -> Union[training_pipeline.TrainingPipeline, pipeline_job.PipelineJob]:
    """Poll the pipeline status with automatic retry.

    Args:
      get_pipeline_fn: A function that polls the pipeline from the pipeline
        client and the pipeline name.
      pipeline_name: The name of the pipeline.

    Returns:
      The polled pipeline job or training pipeline.

    Raises:
      ValueError: The pipeline failed with a user error.
      RuntimeError: The pipeline failed with an unknown error.
    """
    with execution_context.ExecutionContext(
        on_cancel=lambda: self.send_cancel_request(pipeline_name)
    ):
      retry_count = 0
      while True:
        try:
          get_pipeline_response = get_pipeline_fn(
              self.pipeline_client, pipeline_name
          )
          retry_count = 0
        # Handle transient connection error.
        except ConnectionError as err:
          retry_count += 1
          if retry_count < _CONNECTION_ERROR_RETRY_LIMIT:
            logging.warning(
                (
                    'ConnectionError (%s) encountered when polling pipeline:'
                    ' %s. Trying to recreate the API client.'
                ),
                err,
                pipeline_name,
            )
            # Recreate the Python API client.
            self.pipeline_client = aiplatform.gapic.PipelineServiceClient(
                client_options=self.client_options, client_info=self.client_info
            )
            logging.info(
                'Waiting for %s seconds for next poll.',
                _POLLING_INTERVAL_IN_SECONDS,
            )
            time.sleep(_POLLING_INTERVAL_IN_SECONDS)
            continue
          else:
            # TODO(ruifang) propagate the error.
            # Exit with an internal error code.
            error_util.exit_with_internal_error(
                f'Request failed after {_CONNECTION_ERROR_RETRY_LIMIT} retries.'
            )
            return  # Not necessary, only to please the linter.

        if (
            get_pipeline_response.state
            == pipeline_state.PipelineState.PIPELINE_STATE_SUCCEEDED
        ):
          logging.info(
              'Get%s response state =%s',
              self.pipeline_type,
              get_pipeline_response.state,
          )
          return get_pipeline_response
        elif get_pipeline_response.state in _PIPELINE_ERROR_STATES:
          # TODO(ruifang) propagate the error.
          if get_pipeline_response.error.code in _PIPELINE_USER_ERROR_CODES:
            raise ValueError(
                'Pipeline failed with value error in error state: {}.'.format(
                    get_pipeline_response.state
                )
            )
          else:
            raise RuntimeError(
                'Pipeline failed with error state: {}.'.format(
                    get_pipeline_response.state
                )
            )
        else:
          logging.info(
              (
                  'Pipeline %s is in a non-final state %s.'
                  ' Waiting for %s seconds for next poll.'
              ),
              pipeline_name,
              get_pipeline_response.state,
              _POLLING_INTERVAL_IN_SECONDS,
          )
          time.sleep(_POLLING_INTERVAL_IN_SECONDS)

  def send_cancel_request(self, pipeline_name: str):
    """Cancels a pipeline with the given name."""
    if not pipeline_name:
      return
    creds, _ = google.auth.default(
        scopes=['https://www.googleapis.com/auth/cloud-platform']
    )
    if not creds.valid:
      creds.refresh(google.auth.transport.requests.Request())
    headers = {
        'Content-type': 'application/json',
        'Authorization': 'Bearer ' + creds.token,
    }
    # Vertex AI cancel APIs:
    # https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.pipelineJobs/cancel
    # https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.trainingPipelines/cancel
    requests.post(
        url=f'{self.pipeline_uri_prefix}{pipeline_name}:cancel',
        data='',
        headers=headers,
    )
