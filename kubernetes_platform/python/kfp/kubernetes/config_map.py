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

from typing import Dict

from google.protobuf import json_format
from kfp.dsl import PipelineTask
from kfp.kubernetes import common
from kfp.kubernetes import kubernetes_executor_config_pb2 as pb


def use_config_map_as_env(
    task: PipelineTask,
    config_map_name: str,
    config_map_key_to_env: Dict[str, str],
) -> PipelineTask:
    """Use a Kubernetes ConfigMap as an environment variable as described by the `Kubernetes documentation
    https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#define-container-environment-variables-using-configmap-data` _.

    Args:
        task: Pipeline task.
        config_map_name: Name of the ConfigMap.
        config_map_key_to_env: Dictionary of ConfigMap key to environment variable name. For example, ``{'foo': 'FOO'}`` sets the value of the ConfigMap's foo field to the environment variable ``FOO``.

    Returns:
        Task object with updated ConfigMap configuration.
    """

    msg = common.get_existing_kubernetes_config_as_message(task)

    key_to_env = [
        pb.ConfigMapAsEnv.ConfigMapKeyToEnvMap(
            config_map_key=config_map_key,
            env_var=env_var,
        ) for config_map_key, env_var in config_map_key_to_env.items()
    ]
    config_map_as_env = pb.ConfigMapAsEnv(
        config_map_name=config_map_name,
        key_to_env=key_to_env,
    )

    msg.config_map_as_env.append(config_map_as_env)

    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task


def use_config_map_as_volume(
    task: PipelineTask,
    config_map_name: str,
    mount_path: str,
) -> PipelineTask:
    """Use a Kubernetes ConfigMap by mounting its data to the task's container as
    described by the `Kubernetes documentation <https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#add-configmap-data-to-a-volume>`_.

    Args:
        task: Pipeline task.
        config_map_name: Name of the ConfigMap.
        mount_path: Path to which to mount the ConfigMap data.

    Returns:
        Task object with updated ConfigMap configuration.
    """

    msg = common.get_existing_kubernetes_config_as_message(task)

    config_map_as_vol = pb.ConfigMapAsVolume(
        config_map_name=config_map_name,
        mount_path=mount_path,
    )
    msg.config_map_as_volume.append(config_map_as_vol)

    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task
