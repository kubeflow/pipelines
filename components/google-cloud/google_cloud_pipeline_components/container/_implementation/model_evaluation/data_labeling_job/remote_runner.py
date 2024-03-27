# Copyright 2024 The Kubeflow Authors. All Rights Reserved.
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
"""GCP launcher for data labeling jobs based on the AI Platform SDK."""

import json

from google.api_core import retry
from google_cloud_pipeline_components.container.v1.gcp_launcher import job_remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import error_util

_DATA_LABELING_JOB_RETRY_DEADLINE_SECONDS = 10.0 * 60.0


def create_data_labeling_job_with_client(job_client, parent, job_spec):
  create_data_labeling_job_fn = None
  try:
    create_data_labeling_job_fn = job_client.create_data_labeling_job(
        parent=parent, data_labeling_job=job_spec
    )
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
  return create_data_labeling_job_fn


def get_data_labeling_job_with_client(job_client, job_name):
  get_data_labeling_job_fn = None
  try:
    get_data_labeling_job_fn = job_client.get_data_labeling_job(
        name=job_name,
        retry=retry.Retry(deadline=_DATA_LABELING_JOB_RETRY_DEADLINE_SECONDS),
    )
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
  return get_data_labeling_job_fn


def create_data_labeling_job(
    type,
    project,
    location,
    gcp_resources,
    job_display_name,
    dataset_name,
    instruction_uri,
    inputs_schema_uri,
    annotation_spec,
    labeler_count,
    annotation_label,
):
  """Create data labeling job.

  This follows the typical launching logic:
  1. Read if the data labeling job already exists in gcp_resources
     - If already exists, jump to step 3 and poll the job status. This happens
     if the launcher container experienced unexpected termination, such as
     preemption
  2. Deserialize the params into the job spec and create the data labeling
  job
  3. Poll the data labeling job status every
  job_remote_runner._POLLING_INTERVAL_IN_SECONDS seconds
     - If the data labeling job is succeeded, return succeeded
     - If the data labeling job is cancelled/paused, it's an unexpected
     scenario so return failed
     - If the data labeling job is running, continue polling the status

  Also retry on ConnectionError up to
  job_remote_runner._CONNECTION_ERROR_RETRY_LIMIT times during the poll.
  """
  remote_runner = job_remote_runner.JobRemoteRunner(
      type, project, location, gcp_resources
  )

  job_spec = {
      'display_name': job_display_name,
      'datasets': [dataset_name],
      'instruction_uri': instruction_uri,
      'inputs_schema_uri': inputs_schema_uri,
      'inputs': annotation_spec,
      'annotation_labels': {
          'aiplatform.googleapis.com/annotation_set_name': annotation_label
      },
      'labeler_count': labeler_count,
  }

  try:
    # Create data labeling job if it does not exist
    job_name = remote_runner.check_if_job_exists()
    if job_name is None:
      job_name = remote_runner.create_job(
          create_data_labeling_job_with_client,
          json.dumps(job_spec),
      )

    # Poll data labeling job status until "JobState.JOB_STATE_SUCCEEDED"
    remote_runner.poll_job(get_data_labeling_job_with_client, job_name)
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
