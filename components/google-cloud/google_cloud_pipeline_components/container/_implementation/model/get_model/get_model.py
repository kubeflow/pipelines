# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
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
"""GCP launcher for Get Model based on the AI Platform SDK."""

import argparse
import os
import sys
from typing import Optional

from google.api_core import gapic_v1
from google.cloud import aiplatform
from google_cloud_pipeline_components.container.utils import artifact_utils
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
from google_cloud_pipeline_components.types.artifact_types import VertexModel

from google.protobuf import json_format


RESOURCE_TYPE = 'Model'


def _make_parent_dirs_and_return_path(file_path: str):
  os.makedirs(os.path.dirname(file_path), exist_ok=True)
  return file_path


def _parse_args(args):
  """Parse command line arguments."""
  parser = argparse.ArgumentParser(
      prog='Vertex Pipelines get model launcher', description=''
  )
  parser.add_argument(
      '--executor_input',
      dest='executor_input',
      type=str,
      # executor_input is only needed for components that emit output artifacts.
      required=True,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--gcp_resources',
      dest='gcp_resources',
      type=_make_parent_dirs_and_return_path,
      required=True,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--model_version', dest='model_version', type=str, default=None
  )

  group_args = parser.add_mutually_exclusive_group(required=True)
  group_args.add_argument(
      '--model_name', dest='model_name', type=str, default=None
  )
  parsed_args, _ = parser.parse_known_args(args)
  return vars(parsed_args)


def _get_model(
    executor_input,
    gcp_resources,
    model_name: str,
    model_version: Optional[str] = None,
):
  """Get model."""
  if model_version:
    model_resource_name = f'{model_name.split("@", 1)[0]}@{model_version}'
  else:
    model_resource_name = model_name

  if '/publishers/' not in model_resource_name:
    segments_dict = aiplatform.gapic.ModelServiceClient.parse_model_path(
        model_resource_name
    )
    location = segments_dict.get('location', '')
    if not location:
      raise ValueError(
          'Model resource name must be in the format'
          ' projects/{project}/locations/{location}/models/{model} or'
          ' projects/{project}/locations/{location}/models/{model}@{model_version}'
      )
    api_endpoint = location + '-aiplatform.googleapis.com'
    vertex_uri_prefix = f'https://{api_endpoint}/v1/'

    client = aiplatform.gapic.ModelServiceClient(
        client_info=gapic_v1.client_info.ClientInfo(
            user_agent='google-cloud-pipeline-components'
        ),
        client_options={
            'api_endpoint': api_endpoint,
        },
    )

    get_model_response = client.get_model(name=model_resource_name)

    resp_model_name_without_version = get_model_response.name.split('@', 1)[0]
    model_resource_name = (
        f'{resp_model_name_without_version}@{get_model_response.version_id}'
    )
  else:
    # If "/publisher/" is in resource_name, it is a Model Garden model. We can
    # skip and directly build the artifact if it is a model garden model.
    # TODO(b/279978677): Use SDK function when supported to retrieve location.
    location = model_resource_name.split('/')[3]
    api_endpoint = location + '-aiplatform.googleapis.com'
    vertex_uri_prefix = f'https://{api_endpoint}/v1/'

  vertex_model = VertexModel.create(
      'model', vertex_uri_prefix + model_resource_name, model_resource_name
  )
  # TODO(b/266848949): Output Artifact should use correct MLMD artifact.
  artifact_utils.update_output_artifacts(executor_input, [vertex_model])

  resources = GcpResources()
  model_resource = resources.resources.add()
  model_resource.resource_type = RESOURCE_TYPE
  model_resource.resource_uri = f'{vertex_uri_prefix}{model_resource_name}'
  with open(gcp_resources, 'w') as f:
    f.write(json_format.MessageToJson(resources))


def main(argv):
  """Main entry.

  Expected input args are as follows:
    gcp_resources - placeholder output for returning model_id.
    model_name - Required. Provided string resource name to create a model
    artifact.
    model_version - Optional. The models version, overrides any versions in
    model_name.

  Args:
    argv: A list of system arguments.
  """
  parsed_args = _parse_args(argv)
  _get_model(**parsed_args)


if __name__ == '__main__':
  main(sys.argv[1:])
