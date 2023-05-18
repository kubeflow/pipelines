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

from typing import List

from google_cloud_pipeline_components import _image
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import PipelineTaskFinalStatus


@container_component
def vertex_pipelines_notification_email(
    recipients: List[str],
    pipeline_task_final_status: PipelineTaskFinalStatus,
):
  # fmt: off
  """When this component is included as an exit handler, sends a notification
  email with the status of the upstream DAG to the specified recipients.

  This component works only on Vertex Pipelines. This component raises an
  exception when run on Kubeflow Pipelines.

  Args:
    recipients: A list of email addresses to send this notification
      to.
    pipeline_task_final_status: The task final status
      of the upstream DAG that this component will use in the notification.
  """
  # fmt: on
  return ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.vertex_notification_email.executor',
      ],
      args=[
          '--type',
          'VertexNotificationEmail',
          '--payload',
          '',
      ],
  )
