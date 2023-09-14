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

from typing import Dict

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components.types.artifact_types import VertexEndpoint
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp import dsl
from kfp.dsl import Input
from kfp.dsl import OutputPath


@dsl.container_component
def model_undeploy(
    model: Input[VertexModel],
    endpoint: Input[VertexEndpoint],
    gcp_resources: OutputPath(str),
    traffic_split: Dict[str, str] = {},
):
  # fmt: off
  """[Undeploys](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints/undeployModel) a Google Cloud Vertex [DeployedModel](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints#deployedmodel) within an [Endpoint](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints). See the [undeploy Model](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints/undeployModel) method for more information.

  Args:
      model: The model that was deployed to the Endpoint.
      endpoint: The Endpoint for the DeployedModel to be undeployed from.
      traffic_split: If this field is provided, then the Endpoint's trafficSplit will be overwritten with it. If last DeployedModel is being undeployed from the Endpoint, the [Endpoint.traffic_split] will always end up empty when this call returns. A DeployedModel will be successfully undeployed only if it doesn't have any traffic assigned to it when this method executes, or if this field unassigns any traffic to it.

  Returns:
      gcp_resources: Serialized JSON of `gcp_resources` [proto](https://github.com/kubeflow/pipelines/tree/master/components/google-cloud/google_cloud_pipeline_components/proto) which tracks the undeploy Model's long-running operation.
  """
  # fmt: on
  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.endpoint.undeploy_model.launcher',
      ],
      args=[
          '--type',
          'UndeployModel',
          '--payload',
          dsl.ConcatPlaceholder([
              '{',
              '"endpoint": "',
              endpoint.metadata['resourceName'],
              '"',
              ', "model": "',
              model.metadata['resourceName'],
              '"',
              ', "traffic_split": ',
              traffic_split,
              '}',
          ]),
          '--project',
          '',
          '--location',
          '',
          '--gcp_resources',
          gcp_resources,
      ],
  )
