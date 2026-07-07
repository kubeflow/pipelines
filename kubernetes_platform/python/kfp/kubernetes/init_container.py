# Copyright 2026 The Kubeflow Authors
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

from typing import Dict, List, Optional

from google.protobuf import json_format
from kfp.dsl import PipelineTask
from kfp.kubernetes import common
from kfp.kubernetes import kubernetes_executor_config_pb2 as pb


def add_init_container(
    task: PipelineTask,
    name: str,
    image: str,
    command: Optional[List[str]] = None,
    args: Optional[List[str]] = None,
    env: Optional[Dict[str, str]] = None,
    volume_mounts: Optional[Dict[str, str]] = None,
) -> PipelineTask:
    """Add an `init container
    <https://kubernetes.io/docs/concepts/workloads/pods/init-containers/>`_ to
    the task's pod.

    Init containers run to completion, in call order, before the task's main
    container starts. The backend applies a Pod Security Standards compliant
    security context to each one.

    Args:
        task: Pipeline task.
        name: Name of the init container. Must be unique within the pod.
        image: Container image of the init container.
        command: Entrypoint array. Corresponds to the ``command`` field.
        args: Arguments to the entrypoint. Corresponds to the ``args`` field.
        env: Mapping of environment variable names to values. Insertion order
            is preserved when serializing to the pod spec, which matters for
            ``$(VAR)`` references; duplicate names are not representable.
        volume_mounts: Mapping of volume name to mount path. Each volume must be
            defined on the pod, for example via
            :func:`~kfp.kubernetes.empty_dir_mount` or
            :func:`~kfp.kubernetes.mount_pvc`.

    Returns:
        Task object with the added init container.
    """
    if not name:
        raise ValueError('Argument for "name" must be a non-empty string.')
    if name == 'kfp-launcher':
        raise ValueError(
            'Init container name "kfp-launcher" is reserved by Kubeflow Pipelines.'
        )
    if not image:
        raise ValueError('Argument for "image" must be a non-empty string.')

    msg = common.get_existing_kubernetes_config_as_message(task)

    for existing_init_container in msg.init_containers:
        if existing_init_container.name == name:
            raise ValueError(
                f'An init container named "{name}" was already added to this task.'
            )

    init_container = pb.InitContainer(
        name=name,
        image=image,
        command=command or [],
        args=args or [],
    )
    for env_name, env_value in (env or {}).items():
        init_container.env.append(
            pb.InitContainer.EnvVar(name=env_name, value=env_value))
    for volume_name, mount_path in (volume_mounts or {}).items():
        init_container.volume_mounts.append(
            pb.InitContainer.VolumeMount(
                volume_name=volume_name, mount_path=mount_path))

    msg.init_containers.append(init_container)

    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task
