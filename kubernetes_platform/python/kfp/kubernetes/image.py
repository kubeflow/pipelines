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

from typing import List

from google.protobuf import json_format
from kfp.dsl import PipelineTask
from kfp.kubernetes import common
from kfp.kubernetes import kubernetes_executor_config_pb2 as pb


def set_image_pull_secrets(
    task: PipelineTask,
    secret_names: List[str],
) -> PipelineTask:
    """Set image pull secrets for a Kubernetes task.

    Args:
        task: Pipeline task.
        secret_names: List of image pull secret names.

    Returns:
        Task object with updated image pull secret configuration.
    """

    msg = common.get_existing_kubernetes_config_as_message(task)

    # Assuming secret_names is a list of strings
    image_pull_secret = [
        pb.ImagePullSecret(secret_name=secret_name)
        for secret_name in secret_names
    ]

    msg.image_pull_secret.extend(image_pull_secret)

    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task


def set_image_pull_policy(task: PipelineTask, policy: str) -> PipelineTask:
    """Set image pull policy for the container.

    Args:
        task: Pipeline task.
        policy: One of `Always`, `Never`, `IfNotPresent`.

    Returns:
        Task object with an added ImagePullPolicy specification.
    """
    if policy not in ['Always', 'Never', 'IfNotPresent']:
        raise ValueError(
            'Invalid imagePullPolicy. Must be one of `Always`, `Never`, `IfNotPresent`.'
        )
    msg = common.get_existing_kubernetes_config_as_message(task)
    msg.image_pull_policy = policy
    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task
