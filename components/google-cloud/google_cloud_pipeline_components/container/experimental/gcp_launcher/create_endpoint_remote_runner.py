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
from .utils import json_util
from . import lro_remote_runner


def create_endpoint(
    type,
    project,
    location,
    payload,
    gcp_resources,
    executor_input,
):
    """
  Create endpoint and poll the LongRunningOperator till it reaches a final state.
  """
    api_endpoint = location + '-aiplatform.googleapis.com'
    vertex_uri_prefix = f"https://{api_endpoint}/v1/"
    create_endpoint_url = f"{vertex_uri_prefix}projects/{project}/locations/{location}/endpoints"
    endpoint_spec = json.loads(payload, strict=False)
    # TODO(IronPan) temporarily remove the empty fields from the spec
    create_endpoint_request = json_util.recursive_remove_empty(endpoint_spec)

    remote_runner = lro_remote_runner.LroRemoteRunner(location)
    upload_model_lro = remote_runner.create_lro(create_endpoint_url,json.dumps(create_endpoint_request),gcp_resources)
    remote_runner.poll_lro(lro=upload_model_lro,executor_input=executor_input, artifact_name='endpoint',resource_name_key='endpoint')
 