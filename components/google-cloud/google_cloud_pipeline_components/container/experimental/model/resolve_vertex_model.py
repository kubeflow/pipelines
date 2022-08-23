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
"""Module for resolving a google.VertexModel or model name to the existing google.VertexModel MLMD artifact connected to the managed Vertex model and version."""
import argparse
import os
import sys

from google.api_core import gapic_v1
from google.cloud import aiplatform
from google.cloud.aiplatform.metadata.schema.google import artifact_schema
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import artifact_util
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources

from google.protobuf import json_format

RESOURCE_TYPE = 'ResolveVertexModel'


def _make_parent_dirs_and_return_path(file_path: str):
  os.makedirs(os.path.dirname(file_path), exist_ok=True)
  return file_path


parser = argparse.ArgumentParser(
    prog='Vertex Model Service Get/Resolve Model', description='')
group_args = parser.add_mutually_exclusive_group(required=True)
group_args.add_argument(
    '--model_artifact_resource_name',
    dest='model_artifact_resource_name',
    type=str,
    default=None)
group_args.add_argument(
    '--model_name', dest='model_name', type=str, default=None)
parser.add_argument(
    '--model_version', dest='model_version', type=str, default=None)
parser.add_argument(
    '--executor_input',
    dest='executor_input',
    type=str,
    required=True,
    default=argparse.SUPPRESS)
parser.add_argument(
    '--gcp_resources',
    dest='gcp_resources',
    type=_make_parent_dirs_and_return_path,
    required=True,
    default=argparse.SUPPRESS)


def main(argv):
  """Calls ModelService.GetModel."""
  parsed_args, _ = parser.parse_known_args(argv)

  if parsed_args.model_version:
    model_version_suffix = f'@{parsed_args.model_version}'
  else:
    model_version_suffix = ''

  if parsed_args.model_name:
    model_artifact_resource_name = f'{parsed_args.model_name}{model_version_suffix}'
  elif parsed_args.model_version:
    model_artifact_resource_name = f'{parsed_args.model_artifact_resource_name.split("@", 1)[0]}{model_version_suffix}'
  else:
    model_artifact_resource_name = parsed_args.model_artifact_resource_name

  location = model_artifact_resource_name.split('/')[3]
  api_endpoint = f'{location}-aiplatform.googleapis.com'
  resource_uri_prefix = f'https://{api_endpoint}/v1/'

  client = aiplatform.gapic.ModelServiceClient(
      client_info=gapic_v1.client_info.ClientInfo(
          user_agent='google-cloud-pipeline-components'),
      client_options={
          'api_endpoint': api_endpoint,
      })
  get_model_response = client.get_model(parent=model_artifact_resource_name)

  model_name_without_version = get_model_response.name.split('@', 1)[0]
  model_name = f'{model_name_without_version}@{get_model_response.version_id}'

  # Write the model resource to GcpResources output.
  resources = GcpResources()
  model_resource = resources.resources.add()
  model_resource.resource_type = RESOURCE_TYPE
  model_resource.resource_uri = f'{resource_uri_prefix}{model_name}'
  with open(parsed_args.gcp_resources, 'w') as f:
    f.write(json_format.MessageToJson(resources))

  vertex_model_artifact = artifact_schema.VertexModel(
      vertex_model_name=model_name)
  artifact_util.update_output_artifacts(parsed_args.executor_input,
                                        [vertex_model_artifact])


if __name__ == '__main__':
  print(sys.argv)
  main(sys.argv[1:])
