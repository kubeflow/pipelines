# Copyright 2024 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from google.protobuf import json_format
from kfp.dsl import PipelineTask
from kfp.kubernetes import common
from kfp.kubernetes import kubernetes_executor_config_pb2 as pb
from kubernetes.client import V1Toleration


def add_toleration(
    task: PipelineTask,
    toleration: V1Toleration,
) -> PipelineTask:
    """Add a `toleration<https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/>`_. to a task.

    Args:
        task: Pipeline task.
        toleration: A V1Tolerations defined using the Kubernetes Python Client.

    Returns:
        Task object with added tolerations.
    """
    msg = common.get_existing_kubernetes_config_as_message(task)
    msg.tolerations.append(
        pb.Toleration(
            key=toleration.key,
            operator=toleration.operator,
            value=toleration.value,
            effect=toleration.effect,
            toleration_seconds=toleration.toleration_seconds,
        ))
    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task
