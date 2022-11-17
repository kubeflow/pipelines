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
"""GCP launcher for custom jobs based on the AI Platform SDK."""

from google.api_core import retry
from google_cloud_pipeline_components.container.v1.gcp_launcher import job_remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import error_util

_CUSTOM_JOB_RETRY_DEADLINE_SECONDS = 10.0 * 60.0


def create_custom_job_with_client(job_client, parent, job_spec):
  create_custom_job_fn = None
  try:
    create_custom_job_fn = job_client.create_custom_job(
        parent=parent, custom_job=job_spec)
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
  return create_custom_job_fn


def get_custom_job_with_client(job_client, job_name):
  get_custom_job_fn = None
  try:
    get_custom_job_fn = job_client.get_custom_job(
        name=job_name,
        retry=retry.Retry(deadline=_CUSTOM_JOB_RETRY_DEADLINE_SECONDS))
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
  return get_custom_job_fn


def create_custom_job(
    type,
    project,
    location,
    payload,
    gcp_resources,
):
  """Create and poll custom job status till it reaches a final state.

  This follows the typical launching logic:
  1. Read if the custom job already exists in gcp_resources
     - If already exists, jump to step 3 and poll the job status. This happens
       if the launcher container experienced unexpected termination, such as
       preemption
  2. Deserialize the payload into the job spec and create the custom job
  3. Poll the custom job status every _POLLING_INTERVAL_IN_SECONDS seconds
     - If the custom job is succeeded, return succeeded
     - If the custom job is cancelled/paused, it's an unexpected scenario so
     return failed
     - If the custom job is running, continue polling the status
  Also retry on ConnectionError up to
  job_remote_runner._CONNECTION_ERROR_RETRY_LIMIT times during the poll.
  """
  remote_runner = job_remote_runner.JobRemoteRunner(type, project, location,
                                                    gcp_resources)

  try:
    # Create custom job if it does not exist
    job_name = remote_runner.check_if_job_exists()
    if job_name is None:
      job_name = remote_runner.create_job(create_custom_job_with_client,
                                          payload)

    # Poll custom job status until "JobState.JOB_STATE_SUCCEEDED"
    remote_runner.poll_job(get_custom_job_with_client, job_name)
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
