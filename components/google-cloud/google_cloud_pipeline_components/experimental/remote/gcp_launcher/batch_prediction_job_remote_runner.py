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
"""GCP launcher for batch prediction jobs based on the AI Platform SDK."""

from . import job_remote_runner


def create_batch_prediction_job(job_type, project, location, payload,
                                gcp_resources):
  """Create and poll batch prediction job status till it reaches a final state.

  This follows the typical launching logic:
  1. Read if the batch prediction job already exists in gcp_resources
     - If already exists, jump to step 3 and poll the job status. This happens
     if the launcher container experienced unexpected termination, such as
     preemption
  2. Deserialize the payload into the job spec and create the batch prediction
  job.
  3. Poll the batch prediction job status every _POLLING_INTERVAL_IN_SECONDS
  seconds
     - If the batch prediction job is succeeded, return succeeded
     - If the batch prediction job is cancelled/paused, it's an unexpected
     scenario so return failed
     - If the batch prediction job is running, continue polling the status

  Also retry on ConnectionError up to _CONNECTION_ERROR_RETRY_LIMIT times during
  the poll.
  """
  job_remote_runner.create_job(job_type, project, location, payload,
                               gcp_resources)
