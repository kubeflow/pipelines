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


def add_node_selector(
    task: PipelineTask,
    label_key: str,
    label_value: str,
) -> PipelineTask:
    """Add a constraint to the task Pod's `nodeSelector
    <https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector>`_.

    Each constraint is a key-value pair, corresponding to the PodSpec's `nodeSelector <https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling>`_ field.

    For the task's Pod to be eligible to run on a node, the node's labels must satisfy the constraint.

    Args:
        task: Pipeline task.
        label_key: Key of the nodeSelector label.
        label_value: Value of the nodeSelector label.

    Returns:
        Task object with an added nodeSelector constraint.
    """

    msg = common.get_existing_kubernetes_config_as_message(task)
    msg.node_selector.labels.update({label_key: label_value})
    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task
