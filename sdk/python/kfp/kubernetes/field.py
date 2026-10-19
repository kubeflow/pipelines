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


def use_field_path_as_env(
    task: PipelineTask,
    env_name: str,
    field_path: str,
) -> PipelineTask:
    """Use a Kubernetes Field Path as an environment variable as described in
    https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information

    Args:
        task: Pipeline task.
        env_name: Name of the enviornment variable.
        field_path: Kubernetes field path to expose as the enviornment variable.

    Returns:
        Task object with updated field path as the enviornment variable.
    """

    msg = common.get_existing_kubernetes_config_as_message(task)
    field_path_as_env = pb.FieldPathAsEnv(
        name=env_name,
        field_path=field_path,
    )
    msg.field_path_as_env.append(field_path_as_env)
    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task
