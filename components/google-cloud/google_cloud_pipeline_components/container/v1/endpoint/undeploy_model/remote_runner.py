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
import re

from google_cloud_pipeline_components.container.v1.gcp_launcher import lro_remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import error_util
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import json_util


_MODEL_NAME_TEMPLATE = r'(projects/(?P<project>.*)/locations/(?P<location>.*)/models/(?P<modelid>.*))'
_ENDPOINT_NAME_TEMPLATE = r'(projects/(?P<project>.*)/locations/(?P<location>.*)/endpoints/(?P<endpointid>.*))'


def undeploy_model(
    type,
    project,
    location,
    payload,
    gcp_resources,
):
  """
  Undeploy a model from the endpoint and poll the LongRunningOperator till it reaches a final state.
  """
  # TODO(IronPan) temporarily remove the empty fields from the spec
  undeploy_model_request = json_util.recursive_remove_empty(
      json.loads(payload, strict=False))
  endpoint_name = undeploy_model_request['endpoint']

  # Get the endpoint where the model is deployed to
  endpoint_uri_pattern = re.compile(_ENDPOINT_NAME_TEMPLATE)
  match = endpoint_uri_pattern.match(endpoint_name)
  try:
    location = match.group('location')
  except AttributeError as err:
    # TODO(ruifang) propagate the error.
    raise ValueError('Invalid endpoint name: {}. Expect: {}.'.format(
        endpoint_name,
        'projects/[project_id]/locations/[location]/endpoints/[endpoint_id]'))
  api_endpoint = location + '-aiplatform.googleapis.com'
  vertex_uri_prefix = f'https://{api_endpoint}/v1/'

  get_endpoint_url = f'{vertex_uri_prefix}{endpoint_name}'

  try:
    get_endpoint_remote_runner = lro_remote_runner.LroRemoteRunner(location)
    endpoint = get_endpoint_remote_runner.request(get_endpoint_url, '', 'get')

    # Get the deployed_model_id
    deployed_model_id = ''
    model_name = undeploy_model_request['model']

    for deployed_model in endpoint['deployedModels']:
      if deployed_model['model'] == model_name:
        deployed_model_id = deployed_model['id']
        break

    if not deployed_model_id:
      # TODO(ruifang) propagate the error.
      raise ValueError('Model {} not found at endpoint {}.'.format(
          model_name, endpoint_name))

    # Undeploy the model
    undeploy_model_lro_request = {
        'deployed_model_id': deployed_model_id,
    }
    if 'traffic_split' in undeploy_model_request:
      undeploy_model_lro_request['traffic_split'] = undeploy_model_request[
          'traffic_split']

    model_uri_pattern = re.compile(_MODEL_NAME_TEMPLATE)
    match = model_uri_pattern.match(model_name)
    try:
      location = match.group('location')
    except AttributeError as err:
      # TODO(ruifang) propagate the error.
      raise ValueError('Invalid model name: {}. Expect: {}.'.format(
          model_name,
          'projects/[project_id]/locations/[location]/models/[model_id]'))
    api_endpoint = location + '-aiplatform.googleapis.com'
    vertex_uri_prefix = f'https://{api_endpoint}/v1/'

    undeploy_model_url = f'{vertex_uri_prefix}{endpoint_name}:undeployModel'
    undeploy_model_remote_runner = lro_remote_runner.LroRemoteRunner(location)
    undeploy_model_lro = undeploy_model_remote_runner.create_lro(
        undeploy_model_url, json.dumps(undeploy_model_lro_request),
        gcp_resources)
    undeploy_model_lro = undeploy_model_remote_runner.poll_lro(
        lro=undeploy_model_lro)
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
