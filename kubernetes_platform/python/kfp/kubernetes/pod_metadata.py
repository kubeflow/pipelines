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


def add_pod_label(
    task: PipelineTask,
    label_key: str,
    label_value: str,
) -> PipelineTask:
    """Add a label to the task Pod's `metadata
    <https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#Pod>`_.

    Each label is a key-value pair, corresponding to the metadata's `ObjectMeta <https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/object-meta/#ObjectMeta`_ field.

    Args:
        task: Pipeline task.
        label_key: Key of the metadata label.
        label_value: Value of the metadata label.

    Returns:
        Task object with an added metadata label.
    """

    msg = common.get_existing_kubernetes_config_as_message(task)
    msg.pod_metadata.labels.update({label_key: label_value})
    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task


def add_pod_annotation(
    task: PipelineTask,
    annotation_key: str,
    annotation_value: str,
) -> PipelineTask:
    """Add an annotation to the task Pod's `metadata
    <https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#Pod>`_.

    Each annotation is a key-value pair, corresponding to the metadata's `ObjectMeta <https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/object-meta/#ObjectMeta`_ field.

    Args:
        task: Pipeline task.
        annotation_key: Key of the metadata annotation.
        annotation_value: Value of the metadata annotation.

    Returns:
        Task object with an added metadata annotation.
    """

    msg = common.get_existing_kubernetes_config_as_message(task)
    msg.pod_metadata.annotations.update({annotation_key: annotation_value})
    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task
