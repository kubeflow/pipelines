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
from typing import Optional

from google_cloud_pipeline_components.container.v1.gcp_launcher import lro_remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import artifact_util
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import error_util
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import json_util
from google_cloud_pipeline_components.types.artifact_types import VertexModel


ARTIFACT_PROPERTY_KEY_UNMANAGED_CONTAINER_MODEL = 'unmanaged_container_model'
API_KEY_PREDICT_SCHEMATA = 'predict_schemata'
API_KEY_CONTAINER_SPEC = 'container_spec'
API_KEY_ARTIFACT_URI = 'artifact_uri'


def append_unmanaged_model_artifact_into_payload(executor_input, model_spec):
  artifact = json.loads(executor_input).get('inputs', {}).get(
      'artifacts', {}).get(ARTIFACT_PROPERTY_KEY_UNMANAGED_CONTAINER_MODEL,
                           {}).get('artifacts')
  if artifact:
    model_spec[
        API_KEY_PREDICT_SCHEMATA] = json_util.camel_case_to_snake_case_recursive(
            artifact[0].get('metadata', {}).get('predictSchemata', {}))
    model_spec[
        API_KEY_CONTAINER_SPEC] = json_util.camel_case_to_snake_case_recursive(
            artifact[0].get('metadata', {}).get('containerSpec', {}))
    model_spec[API_KEY_ARTIFACT_URI] = artifact[0].get('uri')
  return model_spec


def upload_model(
    type,
    project,
    location,
    payload,
    gcp_resources,
    executor_input,
    parent_model_name: Optional[str] = None,
):
  """Upload model and poll the LongRunningOperator till it reaches a final state."""
  api_endpoint = location + '-aiplatform.googleapis.com'
  vertex_uri_prefix = f'https://{api_endpoint}/v1/'
  upload_model_url = f'{vertex_uri_prefix}projects/{project}/locations/{location}/models:upload'
  model_spec = json.loads(payload, strict=False)
  upload_model_request = {
      # TODO(IronPan) temporarily remove the empty fields from the spec
      'model':
          json_util.recursive_remove_empty(
              append_unmanaged_model_artifact_into_payload(
                  executor_input, model_spec))
  }
  if parent_model_name:
    upload_model_request['parent_model'] = parent_model_name.rsplit('@', 1)[0]

  # Add explanation_spec details back into the request if metadata is non-empty, as sklearn/xgboost input features can be empty.
  if (('explanation_spec' in model_spec) and
      ('metadata' in model_spec['explanation_spec']) and
      model_spec['explanation_spec']['metadata']):
    upload_model_request['model']['explanation_spec']['metadata'] = model_spec[
        'explanation_spec']['metadata']

  try:
    remote_runner = lro_remote_runner.LroRemoteRunner(location)
    upload_model_lro = remote_runner.create_lro(
        upload_model_url, json.dumps(upload_model_request), gcp_resources)
    upload_model_lro = remote_runner.poll_lro(lro=upload_model_lro)
    model_resource_name = upload_model_lro['response']['model']
    if 'model_version_id' in upload_model_lro['response']:
      model_resource_name += f'@{upload_model_lro["response"]["model_version_id"]}'

    vertex_model = VertexModel('model', vertex_uri_prefix + model_resource_name,
                               model_resource_name)
    artifact_util.update_output_artifacts(executor_input, [vertex_model])
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
