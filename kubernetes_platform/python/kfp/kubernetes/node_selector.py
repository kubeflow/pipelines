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
from typing import Optional, Union

from google.protobuf import json_format
from kfp.dsl import PipelineTask, pipeline_channel
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


def add_node_selector_json(
        task: PipelineTask,
        node_selector_json: Union[pipeline_channel.PipelineParameterChannel, dict],
) -> PipelineTask:
    """Add a pod nodeSelector constraint to the task Pod's `nodeSelector'

    Args:
        task: Pipeline task.
        node_selector_json:
            node selector provided as dict or input parameter. Takes
            precedence over label_key and label_value. Only one
            node_selector_json is applicable to a task and can contain
            multiple key/value pairs.
    Returns:
        Task object with an added nodeSelector constraint.
    """

    msg = common.get_existing_kubernetes_config_as_message(task)

    input_param_spec = common.parse_k8s_parameter_input(node_selector_json, task)
    msg.node_selector.node_selector_json.CopyFrom(input_param_spec)

    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task
