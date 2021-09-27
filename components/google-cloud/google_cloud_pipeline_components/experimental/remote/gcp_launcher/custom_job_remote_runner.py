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

import json
import logging
import os
import re
import time
from os import path
from google.api_core import gapic_v1
from google.cloud import aiplatform
from google.cloud.aiplatform.compat.types import job_state as gca_job_state
from google_cloud_pipeline_components.experimental.proto.gcp_resources_pb2 import GcpResources
from google.protobuf import json_format

_POLLING_INTERVAL_IN_SECONDS = 20
_CONNECTION_ERROR_RETRY_LIMIT = 5

_JOB_COMPLETE_STATES = (
    gca_job_state.JobState.JOB_STATE_SUCCEEDED,
    gca_job_state.JobState.JOB_STATE_FAILED,
    gca_job_state.JobState.JOB_STATE_CANCELLED,
    gca_job_state.JobState.JOB_STATE_PAUSED,
)

_JOB_ERROR_STATES = (
    gca_job_state.JobState.JOB_STATE_FAILED,
    gca_job_state.JobState.JOB_STATE_CANCELLED,
    gca_job_state.JobState.JOB_STATE_PAUSED,
)


def create_custom_job(
    type,
    project,
    location,
    payload,
    gcp_resources,
):
    """
  Create and poll custom job status till it reaches a final state.
  This follows the typical launching logic
  1. Read if the custom job already exists in gcp_resources
     - If already exists, jump to step 3 and poll the job status. This happens if the
       launcher container experienced unexpected termination, such as preemption
  2. Deserialize the payload into the job spec and create the custom job.
  3. Poll the custom job status every _POLLING_INTERVAL_IN_SECONDS seconds
     - If the custom job is succeeded, return succeeded
     - If the custom job is cancelled/paused, it's an unexpected scenario so return failed
     - If the custom job is running, continue polling the status

  Also retry on ConnectionError up to _CONNECTION_ERROR_RETRY_LIMIT times during the poll.
  """
    client_options = {"api_endpoint": location + '-aiplatform.googleapis.com'}
    client_info = gapic_v1.client_info.ClientInfo(
        user_agent="google-cloud-pipeline-components",)

    custom_job_uri_prefix = f"https://{client_options['api_endpoint']}/v1/"

    # Initialize client that will be used to create and send requests.
    job_client = aiplatform.gapic.JobServiceClient(
        client_options=client_options, client_info=client_info)

    # Instantiate GCPResources Proto
    custom_job_resources = GcpResources()

    # Check if the Custom job already exists
    if path.exists(gcp_resources) and os.stat(gcp_resources).st_size != 0:
        with open(gcp_resources) as f:
            serialized_gcp_resources = f.read()
            custom_job_resources = json_format.Parse(serialized_gcp_resources,
                                                     GcpResources())
            # Resouces should only contain one item.
            if len(custom_job_resources.resources) != 1:
                raise ValueError(
                    f"gcp_resouces should contain one resouce, found {len(custom_job_resources.resources)}"
                )

            custom_job_name_group = re.findall(
                custom_job_resources.resources[0].resource_uri,
                f"{custom_job_uri_prefix}(.*)")

            if not custom_job_name_group or not custom_job_name_group[0]:
                raise ValueError(
                    "Custom Job Name in gcp_resouce is not formatted correctly or is empty."
                )
            custom_job_name = custom_job_name_group[0]

            logging.info(
                'CustomJob name already exists: %s. Continue polling the status',
                custom_job_name)
    else:
        parent = f"projects/{project}/locations/{location}"
        job_spec = json.loads(payload, strict=False)
        create_custom_job_response = job_client.create_custom_job(
            parent=parent, custom_job=job_spec)
        custom_job_name = create_custom_job_response.name

        # Write the job proto to output
        custom_job_resource = custom_job_resources.resources.add()
        custom_job_resource.resource_type = "CustomJob"
        custom_job_resource.resource_uri = f"{custom_job_uri_prefix}{custom_job_name}"

        with open(gcp_resources, 'w') as f:
            f.write(json_format.MessageToJson(custom_job_resources))

    # Poll the job status
    retry_count = 0
    while True:
        try:
            get_custom_job_response = job_client.get_custom_job(
                name=custom_job_name)
            retry_count = 0
        # Handle transient connection error.
        except ConnectionError as err:
            retry_count += 1
            if retry_count < _CONNECTION_ERROR_RETRY_LIMIT:
                logging.warning(
                    'ConnectionError (%s) encountered when polling job: %s. Trying to '
                    'recreate the API client.', err, custom_job_name)
                # Recreate the Python API client.
                job_client = aiplatform.gapic.JobServiceClient(
                    client_options=client_options)
            else:
                logging.error('Request failed after %s retries.',
                              _CONNECTION_ERROR_RETRY_LIMIT)
                # TODO(ruifang) propagate the error.
                raise

        if get_custom_job_response.state == gca_job_state.JobState.JOB_STATE_SUCCEEDED:
            logging.info('GetCustomJob response state =%s',
                         get_custom_job_response.state)
            return
        elif get_custom_job_response.state in _JOB_ERROR_STATES:
            # TODO(ruifang) propagate the error.
            raise RuntimeError("Job failed with error state: {}.".format(
                get_custom_job_response.state))
        else:
            logging.info(
                'Job %s is in a non-final state %s.'
                ' Waiting for %s seconds for next poll.', custom_job_name,
                get_custom_job_response.state, _POLLING_INTERVAL_IN_SECONDS)
            time.sleep(_POLLING_INTERVAL_IN_SECONDS)
