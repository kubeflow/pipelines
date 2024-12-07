# Copyright 2023 The Kubeflow Authors
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


def enable_shared_memory(
    task: PipelineTask, volume_name: str = "shm", size: str = ""
) -> PipelineTask:
    """Add shared memory configuration to the task's Kubernetes configuration.

    This function adds a shared memory volume with optional name and size
    parameters to the task's Kubernetes Executor config. Size should be
    specified in standard Kubernetes format (e.g., '1Gi', '500Mi').

    Args:
        task: Pipeline task.
        volume_name: Name of the shared memory volume, defaults to 'shm'.
        size: Size of the shared memory, defaults to an empty string, ''.

    Returns:
        Task object with updated shared memory configuration.
    """

    # get existing k8s config
    msg = common.get_existing_kubernetes_config_as_message(task)

    # set new values
    msg.enabled_shared_memory.size = size if size is not None else ""
    msg.enabled_shared_memory.volume_name = volume_name

    # update task specific k8s config
    task.platform_config["kubernetes"] = json_format.MessageToDict(msg)

    return task
