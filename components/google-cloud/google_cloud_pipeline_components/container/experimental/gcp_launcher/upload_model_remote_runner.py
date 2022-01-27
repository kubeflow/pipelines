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
from .utils import artifact_util


def upload_model(
    type,
    project,
    location,
    payload,
    gcp_resources,
    executor_input,
):
    """
  Upload model and poll the LongRunningOperator till it reaches a final state.
  """
    api_endpoint = location + '-aiplatform.googleapis.com'
    vertex_uri_prefix = f"https://{api_endpoint}/v1/"
    upload_model_url = f"{vertex_uri_prefix}projects/{project}/locations/{location}/models:upload"
    model_spec = json.loads(payload, strict=False)
    upload_model_request = {
        # TODO(IronPan) temporarily remove the empty fields from the spec
        'model': json_util.recursive_remove_empty(model_spec)
    }

    remote_runner = lro_remote_runner.LroRemoteRunner(location)
    upload_model_lro = remote_runner.create_lro(
        upload_model_url, json.dumps(upload_model_request), gcp_resources)
    upload_model_lro = remote_runner.poll_lro(lro=upload_model_lro)
    model_resource_name = upload_model_lro['response']['model']
    artifact_util.update_output_artifact(
        executor_input, 'model', vertex_uri_prefix + model_resource_name, {
            artifact_util.ARTIFACT_PROPERTY_KEY_RESOURCE_NAME:
                model_resource_name
        })
