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

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp import dsl
from kfp.dsl import Input


@dsl.container_component
def model_delete(model: Input[VertexModel], gcp_resources: dsl.OutputPath(str)):
  # fmt: off
  """[Deletes](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models/delete) a Google Cloud Vertex [Model](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models). See the [Model delete](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models/delete) method for more information. Note that the full model is deleted, NOT only the model version.

  Args:
      model: The name of the Model resource to be deleted. Format: `projects/{project}/locations/{location}/models/{model}`. [More information](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models/delete#path-parameters).

  Returns:
    gcp_resources: Serialized JSON of `gcp_resources` [proto](https://github.com/kubeflow/pipelines/tree/master/components/google-cloud/google_cloud_pipeline_components/proto) which tracks the delete Model's long-running operation.
  """
  # fmt: on
  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.model.delete_model.launcher',
      ],
      args=[
          '--type',
          'DeleteModel',
          '--payload',
          dsl.ConcatPlaceholder([
              '{',
              '"model": "',
              model.metadata['resourceName'],
              '"',
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
