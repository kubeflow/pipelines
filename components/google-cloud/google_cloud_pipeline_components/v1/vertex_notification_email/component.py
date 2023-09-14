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
  """Send notification email(s) when an upstream task/DAG completes.

  This component can only be used as an [ExitHandler](https://www.kubeflow.org/docs/components/pipelines/v2/pipelines/control-flow/#exit-handling-dslexithandler)'s exit task. Note that the [PipelineTaskFinalStatus](https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.PipelineTaskFinalStatus) is provided automatically by Vertex Pipelines at runtime. You should not provide any input to this parameter when you instantiate this component as a task. This component works only on Vertex Pipelines. This component raises an exception when run on Kubeflow Pipelines. See a [usage example](https://cloud.google.com/vertex-ai/docs/pipelines/email-notifications).

  Args:
    recipients: A list of email addresses to send a notification to.
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
