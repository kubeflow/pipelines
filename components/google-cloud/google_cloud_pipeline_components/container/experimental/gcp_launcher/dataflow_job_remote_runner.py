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
"""Common module for launching and managing the Vertex Job resources."""

import json
import logging
import os
from os import path
import re
import time
from typing import Optional
import proto

from .utils import json_util
from google.api_core import gapic_v1
from google.cloud import dataflow
from google.protobuf import json_format
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources

_POLLING_INTERVAL_IN_SECONDS = 20
_CONNECTION_ERROR_RETRY_LIMIT = 5

_JOB_COMPLETE_STATES = (
    dataflow.JobState.JOB_STATE_DONE,
    dataflow.JobState.JOB_STATE_FAILED, 
    dataflow.JobState.JOB_STATE_CANCELLED,
    # TODO() Confirm Job STOPPED is not an error state
    # dataflow.JobState.JOB_STATE_STOPPED,
)

_JOB_ERROR_STATES = (
    dataflow.JobState.JOB_STATE_FAILED, 
    dataflow.JobState.JOB_STATE_CANCELLED,
    # dataflow.JobState.JOB_STATE_STOPPED,
)


class DataflowJobRemoteRunner():
    """Common module for creating and poll Dataflow jobs."""

    def __init__(self, job_type, project, location, gcp_resources):
        """Initlizes a job client and other common attributes."""
        self.job_type = job_type
        self.project = project
        self.location = location
        self.gcp_resources = gcp_resources
        # TODO (chavoshi) Confirm if a regional end point is needed for dataflow. 
        # currently not included in the client build.
        client_options = {
            'api_endpoint': f'dataflow.googleapis.com/projects'
        }
        client_info = gapic_v1.client_info.ClientInfo(
            user_agent='google-cloud-pipeline-components')

        self.job_uri_prefix = f"https://{client_options['api_endpoint']}/v1b3/projects/{project}/locations/{location}/jobs/"

        # Choose the corresponding dataflow Gapic client based on the job_type.
        if job_type == 'DataflowLaunchPythonJob':
            self.job_client = dataflow.JobsV1Beta3Client(
                client_options=client_options, client_info=client_info)

        elif job_type == 'DataflowLaunchFlexTemplate':
            self.job_client = dataflow.FlexTemplatesServiceClient(
                client_options=client_options, client_info=client_info)

        elif job_type == 'DataflowLaunchClassicTemplate':
            self.job_client = dataflow.TemplatesServiceClient(
                client_options=client_options, client_info=client_info)
        else: 
            raise ValueError(f"job_type {job_type} is not supported.")

    def check_if_job_exists(self) -> Optional[str]:
        """Check if the job already exists."""
        if path.exists(self.gcp_resources) and os.stat(
                self.gcp_resources).st_size != 0:
            with open(self.gcp_resources) as f:
                serialized_gcp_resources = f.read()
                job_resources = json_format.Parse(serialized_gcp_resources,
                                                  GcpResources())
                # Resources should only contain one item.
                if len(job_resources.resources) != 1:
                    raise ValueError(
                        f'gcp_resources should contain one resource, found {len(job_resources.resources)}'
                    )

                job_id_group = re.findall(
                    job_resources.resources[0].resource_uri,
                    f'{self.job_uri_prefix}(.*)')

                if not job_id_group or not job_id_group[0]:
                    raise ValueError(
                        'Job ID in gcp_resource is not formatted correctly or is empty.'
                    )
                job_id_group = job_id_group[0]

                logging.info(
                    '%s name already exists: %s. Continue polling the status',
                    self.job_type, job_id_group)
            return job_id_group
        else:
            return None

    def create_python_job(self, create_job_fn, payload) -> str:
        """Create a python based dataflow job."""
        # TODO(chavoshi) Confirm how this parameter can be passed to Dataflow client.
        # note the warning in dataflow docs that without full parent string location 
        # defaults to us-central1 see
        # https://github.com/googleapis/python-dataflow-client/blob/9e9d4f05e67090c743764b9275669746dfc7933f/google/cloud/dataflow_v1beta3/services/jobs_v1_beta3/client.py#L352
        # parent = f'projects/{self.project}/locations/{self.location}'

        # TODO(kevinbnaughton) remove empty fields from the spec temporarily.
        job_proto = json_util.recursive_remove_empty(json.loads(payload, strict=False))

        create_job_response = create_job_fn(self.job_client, job_proto)
        job_id = create_job_response.id

        # Write the job proto to output.
        job_resources = GcpResources()
        job_resource = job_resources.resources.add()
        job_resource.resource_type = self.job_type
        job_resource.resource_uri = f'{self.job_uri_prefix}{job_id}'

        with open(self.gcp_resources, 'w') as f:
            f.write(json_format.MessageToJson(job_resources))

        return job_id

    def poll_job(self, get_job_fn, job_id: str) -> proto.Message:
        """Poll the job status."""
        retry_count = 0
        while True:
            try:
                get_job_response = get_job_fn(self.job_client, job_id)
                retry_count = 0
            # Handle transient connection error.
            except ConnectionError as err:
                retry_count += 1
                if retry_count < _CONNECTION_ERROR_RETRY_LIMIT:
                    logging.warning(
                        'ConnectionError (%s) encountered when polling job: %s. Trying to '
                        'recreate the API client.', err, job_id)
                    # Recreate the Python API client.
                    self.job_client = self.get_job_client(
                        self.client_options, self.client_info)
                else:
                    logging.error('Request failed after %s retries.',
                                  _CONNECTION_ERROR_RETRY_LIMIT)
                    # TODO(ruifang) propagate the error.
                    raise

            if get_job_response.current_state == dataflow.JobState.JOB_STATE_DONE:
                logging.info('Get%s response state =%s', self.job_type,
                             get_job_response.current_state)
                return get_job_response
            elif get_job_response.current_state in _JOB_ERROR_STATES:
                # TODO(ruifang) propagate the error.
                raise RuntimeError('Job failed with error state: {}.'.format(
                    get_job_response.current_state))
            else:
                logging.info(
                    'Job %s is in a non-final state %s.'
                    ' Waiting for %s seconds for next poll.', job_id,
                    get_job_response.current_state, _POLLING_INTERVAL_IN_SECONDS)
                time.sleep(_POLLING_INTERVAL_IN_SECONDS)
