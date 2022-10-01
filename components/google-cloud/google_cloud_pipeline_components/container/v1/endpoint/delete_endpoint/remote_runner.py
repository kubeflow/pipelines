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
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import error_util
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import json_util
from google_cloud_pipeline_components.container.v1.gcp_launcher import lro_remote_runner

_ENDPOINT_NAME_TEMPLATE = r'(projects/(?P<project>.*)/locations/(?P<location>.*)/endpoints/(?P<endpointid>.*))'

def delete_endpoint(
    type,
    project,
    location,
    payload,
    gcp_resources,
):
  """
    Delete endpoint and poll the LongRunningOperator till it reaches a final
    state.
    """
  # TODO(IronPan) temporarily remove the empty fields from the spec
  delete_endpoint_request = json_util.recursive_remove_empty(
      json.loads(payload, strict=False))
  endpoint_name = delete_endpoint_request['endpoint']

  uri_pattern = re.compile(_ENDPOINT_NAME_TEMPLATE)
  match = uri_pattern.match(endpoint_name)
  try:
    location = match.group('location')
  except AttributeError as err:
    # TODO(ruifang) propagate the error.
    raise ValueError('Invalid endpoint name: {}. Expect: {}.'.format(
        endpoint_name,
        'projects/[project_id]/locations/[location]/endpoints/[endpoint_id]'))
  delete_endpoint_url = (f'https://{location}-aiplatform.googleapis.com/v1/'
                         f'{endpoint_name}')

  try:
    remote_runner = lro_remote_runner.LroRemoteRunner(location)
    delete_endpoint_lro = remote_runner.create_lro(delete_endpoint_url, '',
                                                   gcp_resources, 'delete')
    delete_endpoint_lro = remote_runner.poll_lro(lro=delete_endpoint_lro)
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
