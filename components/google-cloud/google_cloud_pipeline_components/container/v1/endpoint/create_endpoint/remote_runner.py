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

import json
from google_cloud_pipeline_components.container.v1.gcp_launcher import lro_remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import artifact_util
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import json_util
from google_cloud_pipeline_components.types.artifact_types import VertexEndpoint


def create_endpoint(
    type,
    project,
    location,
    payload,
    gcp_resources,
    executor_input,
):
  """Create endpoint and poll the LongRunningOperator till it reaches a final state."""
  api_endpoint = location + '-aiplatform.googleapis.com'
  vertex_uri_prefix = f'https://{api_endpoint}/v1/'
  create_endpoint_url = f'{vertex_uri_prefix}projects/{project}/locations/{location}/endpoints'
  endpoint_spec = json.loads(payload, strict=False)
  # TODO(IronPan) temporarily remove the empty fields from the spec
  create_endpoint_request = json_util.recursive_remove_empty(endpoint_spec)

  remote_runner = lro_remote_runner.LroRemoteRunner(location)
  create_endpoint_lro = remote_runner.create_lro(
      create_endpoint_url, json.dumps(create_endpoint_request), gcp_resources)
  create_endpoint_lro = remote_runner.poll_lro(lro=create_endpoint_lro)
  endpoint_resource_name = create_endpoint_lro['response']['name']

  vertex_endpoint = VertexEndpoint('endpoint',
                                   vertex_uri_prefix + endpoint_resource_name,
                                   endpoint_resource_name)
  artifact_util.update_output_artifacts(executor_input, [vertex_endpoint])
