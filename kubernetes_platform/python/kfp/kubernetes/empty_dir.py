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

from typing import Optional

from google.protobuf import json_format
from kfp.dsl import PipelineTask
from kfp.kubernetes import common
from kfp.kubernetes import kubernetes_executor_config_pb2 as pb


def empty_dir_mount(
    task: PipelineTask,
    volume_name: str,
    mount_path: str,
    medium: Optional[str] = None,
    size_limit: Optional[str] = None,
) -> PipelineTask:
    """Mount an EmptyDir volume to the task's container.

    Args:
        task: Pipeline task.
        volume_name: Name of the EmptyDir volume. 
        mount_path: Path within the container at which the EmptyDir should be mounted.
        medium: Storage medium to back the EmptyDir. Must be one of `Memory` or `HugePages`. Defaults to `None`.
        size_limit: Maximum size of the EmptyDir. For example, `5Gi`. Defaults to `None`.

    Returns:
        Task object with updated EmptyDir mount configuration.
    """

    msg = common.get_existing_kubernetes_config_as_message(task)

    empty_dir_mount = pb.EmptyDirMount(
        volume_name=volume_name,
        mount_path=mount_path,
        medium=medium,
        size_limit=size_limit,
    )

    msg.empty_dir_mounts.append(empty_dir_mount)

    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task
