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


def set_timeout(
    task: PipelineTask,
    seconds: int,
) -> PipelineTask:
    """Add timeout to the task Pod's `active_deadline_seconds
    <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#podspec-v1-core>`_.

    Timeout an integer greater than 0, corresponding to the podspec active_deadline_seconds <https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#podspec-v1-core`_ field.
    Integer 0 means removing the timeout fields from previous functions.

    Args:
        task: Pipeline task.
        seconds: Value of the active_deadline_seconds.

    Returns:
        Task object with an updated active_deadline_seconds.
    """

    msg = common.get_existing_kubernetes_config_as_message(task)
    if seconds >= 0:
        msg.active_deadline_seconds = seconds
    else:
        raise ValueError(
            f'Argument for "seconds" must be an integer greater or equals to 0. Got invalid input: {seconds}. '
        )
    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task
