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


from typing import Optional

from google_cloud_pipeline_components import _image
from kfp import dsl


@dsl.container_component
def tensorboard_experiment_creator(
    tensorboard_resource_name: str,
    gcp_resources: dsl.OutputPath(str),
    tensorboard_experiment_id: Optional[str] = '{{$.pipeline_task_name}}',
    tensorboard_experiment_display_name: Optional[str] = None,
    tensorboard_experiment_description: Optional[str] = None,
    tensorboard_experiment_labels: Optional[dict] = None,
):
  # fmt: off
  """Create a TensorboardExperiment for Pipeline usage.

  Args:
      tensorboad_resource_name: Resource name used to retrieve tensorboard instances. Format example: projects/{project_number}/locations/{location}/tensorboards/{tensorboard_id}
      tensorboard_experiment_id: The tensorboard experiment id.
          If not set, default to task name.
      tensorboard_experiment_display_name: The display name of the tensorboard experiment. If not set, default to None.
      tensorboard_experiment_description: The description of the tensorboard experiment. If not set, default to None.
      tensorboard_experiment_labels: The labels of the tensorboard experiment. If not set, default to None.
  Returns:
      gcp_resources: Gcp_resource with Full resource name of the TensorboardExperiment as uri. Format example: projects/{project_number}/locations/{location}/tensorboards/{tensorboard_id}/experiments/{experiment}
  """
  # fmt: on

  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.experimental.tensorboard.tensorboard_experiment_creator',
      ],
      args=[
          '--task',
          'tensorboard_experiment_creator',
          '--display_name',
          'tensorboard_experiment_creator-run',
          '--tensorboard_resource_name',
          tensorboard_resource_name,
          '--tensorboard_experiment_id',
          tensorboard_experiment_id,
          dsl.IfPresentPlaceholder(
              input_name='tensorboard_experiment_display_name',
              then=[
                  '--tensorboard_experiment_display_name',
                  tensorboard_experiment_display_name,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='tensorboard_experiment_description',
              then=[
                  '--tensorboard_experiment_description',
                  tensorboard_experiment_description,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='tensorboard_experiment_labels',
              then=[
                  '--tensorboard_experiment_labels',
                  tensorboard_experiment_labels,
              ],
          ),
          '--gcp_resources',
          gcp_resources,
      ],
  )
