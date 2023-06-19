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


from google_cloud_pipeline_components.types.artifact_types import VertexDataset
from kfp import dsl
from kfp.dsl import Output


@dsl.container_component
def get_vertex_dataset(
    dataset_resource_name: str,
    dataset: Output[VertexDataset],
    gcp_resources: dsl.OutputPath(str),
):
  # fmt: off
  """Gets a `Dataset <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.datasets>`_ artifact as a Vertex Dataset artifact.

  Args:
    dataset_resource_name: Vertex Dataset resource name in the format of ``projects/{project}/locations/{location}/datasets/{dataset}``.

  Returns:
    dataset: Vertex Dataset artifact with a ``resourceName`` metadata field in the format of ``projects/{project}/locations/{location}/datasets/{dataset}``.
  """
  # fmt: on

  return dsl.ContainerSpec(
      image='python:3.8',
      command=[
          'sh',
          '-ec',
          (
              'program_path=$(mktemp)\nprintf "%s" "$0" >'
              ' "$program_path"\npython3 -u "$program_path" "$@"\n'
          ),
          (
              '\nimport os\n\ndef get_vertex_dataset(executor_input_json,'
              ' dataset_resource_name: str):\n    executor_output = {}\n   '
              " executor_output['artifacts'] = {}\n    for name, artifacts in"
              " executor_input_json.get('outputs',\n                           "
              "                       {}).get('artifacts',\n                   "
              '                                       {}).items():\n     '
              " artifacts_list = artifacts.get('artifacts')\n      if name =="
              " 'dataset' and artifacts_list:\n        updated_runtime_artifact"
              " = artifacts_list[0]\n        updated_runtime_artifact['uri'] ="
              ' dataset_resource_name\n       '
              " updated_runtime_artifact['metadata'] = {'resourceName':"
              " dataset_resource_name}\n        artifacts_list = {'artifacts':"
              ' [updated_runtime_artifact]}\n\n     '
              " executor_output['artifacts'][name] = artifacts_list\n\n    #"
              ' Update the output artifacts.\n    os.makedirs(\n       '
              " os.path.dirname(executor_input_json['outputs']['outputFile']),\n"
              '        exist_ok=True)\n    with'
              " open(executor_input_json['outputs']['outputFile'], 'w') as f:\n"
              '      f.write(json.dumps(executor_output))\n\nif __name__ =='
              " '__main__':\n  import argparse\n  import json\n\n  parser ="
              " argparse.ArgumentParser(description='get resource name')\n "
              " parser.add_argument('--dataset_resource_name', type=str)\n "
              " parser.add_argument('--executor_input', type=str)\n  args, _ ="
              ' parser.parse_known_args()\n  executor_input_json ='
              ' json.loads(args.executor_input)\n  dataset_resource_name ='
              ' args.dataset_resource_name\n '
              ' get_vertex_dataset(executor_input_json,'
              ' dataset_resource_name)\n'
          ),
      ],
      args=[
          '--dataset_resource_name',
          dataset_resource_name,
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ],
  )
