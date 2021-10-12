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
"""GCP launcher for Dataflow jobs."""

from .dataflow_discovery_client import client



def create_python_job(
    type,
    project,
    location,
    payload,
    gcp_resources,
):
    """Create and poll Dataflow job status untill it reaches a final state.

  This follows the typical launching logic:
  1. Read if the dataflow already exists in gcp_resources
     - If already exists, jump to step 3 and poll the job status. This happens
       if the launcher container experienced unexpected termination, such as
       preemption
  2. Deserialize the payload into the job proto and create the dataflow job.
  3. Poll the dataflow job status every _POLLING_INTERVAL_IN_SECONDS seconds
     - If the dataflow job is succeeded, return succeeded
     - If the dataflow job is cancelled/paused, it's an unexpected scenario so
     return failed
     - If the custom job is running, continue polling the status
  Also retry on ConnectionError up to
  job_remote_runner._CONNECTION_ERROR_RETRY_LIMIT times during the poll.
  """
    remote_runner = dataflow_remote_runner.DataflowJobRemoteRunner(type, project, location,
                                                      gcp_resources)

    # Create Dataflow job if it does not exist
    job_id = remote_runner.check_if_job_exists()
    if job_id is None:
        job_id = remote_runner.create_job(create_dataflow_python_with_client,
                                            payload)

    # Poll custom job status until "JobState.JOB_STATE_DONE"
    remote_runner.poll_job(get_dataflow_job_with_client, job_id)


def DataflowLaunchFlexTemplate(
    type,
    project,
    location,
    payload,
    gcp_resources,
):
    raise NotImplementedError("This feature has not been implmented yet")


def create_classic_template_job(
    type,
    project,
    location,
    payload,
    gcp_resources,
):
    raise NotImplementedError("This feature has not been implmented yet")
