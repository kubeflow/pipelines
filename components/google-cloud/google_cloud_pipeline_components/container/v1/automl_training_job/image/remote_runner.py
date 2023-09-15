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
"""GCP remote runner for AutoML image training pipelines based on the AI Platform SDK."""

import logging
from typing import Any

from google.api_core import retry
from google.cloud.aiplatform import gapic
from google_cloud_pipeline_components.container.v1.gcp_launcher import pipeline_remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import error_util


_GET_PIPELINE_RETRY_DEADLINE_SECONDS = 10.0 * 60.0


def create_pipeline_with_client(
    pipeline_client: gapic.PipelineServiceClient,
    parent,
    pipeline_spec: Any,
):
  """Creates a training pipeline with the client."""
  created_pipeline = None
  try:
    logging.info(
        'Creating AutoML Vision training pipeline with sanitized pipeline'
        ' spec: %s',
        pipeline_spec,
    )
    created_pipeline = pipeline_client.create_training_pipeline(
        parent=parent, training_pipeline=pipeline_spec
    )
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
  return created_pipeline


def get_pipeline_with_client(
    pipeline_client: gapic.PipelineServiceClient, pipeline_name: str
):
  """Gets training pipeline state with the client."""
  get_automl_vision_training_pipeline = None
  try:
    get_automl_vision_training_pipeline = pipeline_client.get_training_pipeline(
        name=pipeline_name,
        retry=retry.Retry(deadline=_GET_PIPELINE_RETRY_DEADLINE_SECONDS),
    )
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
  return get_automl_vision_training_pipeline


def create_pipeline(
    type: str,  # pylint: disable=redefined-builtin
    project: str,
    location: str,
    payload: str,
    gcp_resources: str,
):
  """Create and poll AutoML Vision training pipeline status till it reaches a final state.

  This follows the typical launching logic:
  1. Read if the training pipeline already exists in gcp_resources
     - If already exists, jump to step 3 and poll the pipeline status. This
     happens
     if the launcher container experienced unexpected termination, such as
     preemption
  2. Deserialize the payload into the pipeline spec and create the training
  pipeline
  3. Poll the training pipeline status every
  pipeline_remote_runner._POLLING_INTERVAL_IN_SECONDS seconds
     - If the training pipeline is succeeded, return succeeded
     - If the training pipeline is cancelled/paused, it's an unexpected
     scenario so return failed
     - If the training pipeline is running, continue polling the status

  Also retry on ConnectionError up to
  pipeline_remote_runner._CONNECTION_ERROR_RETRY_LIMIT times during the poll.

  Args:
    type: Job type.
    project: Project name.
    location: Location to start the training job.
    payload: Serialized JSON payload.
    gcp_resources: URI for storing GCP resources.
  """
  remote_runner = pipeline_remote_runner.PipelineRemoteRunner(
      type, project, location, gcp_resources
  )

  try:
    # Create AutoML vision training pipeline if it does not exist
    pipeline_name = remote_runner.check_if_pipeline_exists()
    if pipeline_name is None:
      logging.info(
          'AutoML Vision training payload formatted: %s',
          payload,
      )
      pipeline_name = remote_runner.create_pipeline(
          create_pipeline_with_client,
          payload,
      )

    # Poll AutoML Vision training pipeline status until
    # "PipelineState.PIPELINE_STATE_SUCCEEDED"
    remote_runner.poll_pipeline(get_pipeline_with_client, pipeline_name)

  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
