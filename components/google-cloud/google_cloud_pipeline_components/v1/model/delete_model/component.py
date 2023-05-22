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
  """Deletes a Google Cloud Vertex Model. For more details, see
  https://cloud.google.com/vertex-
  ai/docs/reference/rest/v1/projects.locations.models/delete.

  Args:
      model: The model to be deleted.

  Returns:
      gcp_resources: Serialized gcp_resources proto tracking the delete model's long running operation.

          For more details, see https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
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
