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
"""Common module for creating GCP launchers based on the AI Platform SDK."""

import logging
from os import path
import re
import time
from typing import Any

import requests
import google.auth
import google.auth.transport.requests
from google.protobuf import json_format
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources

_POLLING_INTERVAL_IN_SECONDS = 20


class LroRemoteRunner():
    """Common module for creating and poll LRO."""

    def __init__(self, location) -> None:
        self.api_endpoint = location + '-aiplatform.googleapis.com'
        self.vertex_uri_prefix = f"https://{self.api_endpoint}/v1/"
        self.creds, _ = google.auth.default()

    def create_lro(self, create_url: str, request_body: str,
                   gcp_resources: str, http_request: str = 'post') -> Any:
        """call the create API and get a LRO"""

        # Currently we don't check if operation already exists and continue from there
        # If this is desirable to the user and improves the reliability, we could do the following
        # ```
        # from google.api_core import operations_v1, grpc_helpers
        # channel = grpc_helpers.create_channel(location + '-aiplatform.googleapis.com')
        # api = operations_v1.OperationsClient(channel)
        # current_status = api.get_operation(lro.operation.name)
        # ```

        self.creds.refresh(google.auth.transport.requests.Request())
        headers = {
            'Content-type': 'application/json',
            'Authorization': 'Bearer ' + self.creds.token,
            'User-Agent': 'google-cloud-pipeline-components'
        }
        http_request_fn = getattr(requests, http_request)
        lro = http_request_fn(
            url=create_url, data=request_body, headers=headers).json()

        if "error" in lro and lro["error"]["code"]:
            raise RuntimeError(
                "Failed to create the resource. Error: {}".format(lro["error"]))

        lro_name = lro['name']
        get_operation_uri = f"{self.vertex_uri_prefix}{lro_name}"

        # Write the lro to the gcp_resources output parameter
        long_running_operations = GcpResources()
        long_running_operation = long_running_operations.resources.add()
        long_running_operation.resource_type = "VertexLro"
        long_running_operation.resource_uri = get_operation_uri
        with open(gcp_resources, 'w') as f:
            f.write(json_format.MessageToJson(long_running_operations))

        return lro

    def poll_lro(self, lro: Any) -> Any:
        """Poll the LRO till it reaches a final state."""
        while (not "done" in lro) or (not lro['done']):
            time.sleep(_POLLING_INTERVAL_IN_SECONDS)
            logging.info('The resource is creating...')
            if not self.creds.valid:
                self.creds.refresh(google.auth.transport.requests.Request())
            headers = {
                'Content-type': 'application/json',
                'Authorization': 'Bearer ' + self.creds.token
            }
            lro_name = lro['name']
            lro = requests.get(
                f"{self.vertex_uri_prefix}{lro_name}", headers=headers).json()

        if "error" in lro and lro["error"]["code"]:
            raise RuntimeError(
                "Failed to create the resource. Error: {}".format(lro["error"]))
        else:
            logging.info('Create resource complete. %s.', lro)
            return lro
