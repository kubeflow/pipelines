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
import re
from google_cloud_pipeline_components.container.v1.gcp_launcher import lro_remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import error_util
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import json_util


_MODEL_NAME_TEMPLATE = r'(projects/(?P<project>.*)/locations/(?P<location>.*)/models/(?P<modelid>.*))'


def export_model(type, project, location, payload, gcp_resources, output_info):
  """Export model and poll the LongRunningOperator till it reaches a final state."""
  # TODO(IronPan) temporarily remove the empty fields from the spec
  export_model_request = json_util.recursive_remove_empty(
      json.loads(payload, strict=False))
  model_name = export_model_request['name']

  uri_pattern = re.compile(_MODEL_NAME_TEMPLATE)
  match = uri_pattern.match(model_name)
  try:
    location = match.group('location')
  except AttributeError as err:
    # TODO(ruifang) propagate the error.
    raise ValueError('Invalid model name: {}. Expect: {}.'.format(
        model_name,
        'projects/[project_id]/locations/[location]/models/[model_id]'))

  api_endpoint = location + '-aiplatform.googleapis.com'
  vertex_uri_prefix = f'https://{api_endpoint}/v1/'
  export_model_url = f'{vertex_uri_prefix}{model_name}:export'

  try:
    remote_runner = lro_remote_runner.LroRemoteRunner(location)
    export_model_lro = remote_runner.create_lro(
        export_model_url, json.dumps(export_model_request), gcp_resources)
    export_model_lro = remote_runner.poll_lro(lro=export_model_lro)
    output_info_content = export_model_lro['metadata']['outputInfo']
    with open(output_info, 'w') as f:
      f.write(json.dumps(output_info_content))
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
