# Copyright 2019 Google LLC
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

from typing import Callable, Dict, Optional, Text
from ..dsl._container_op import BaseOp, ContainerOp

# Pod label indicating the SDK type from which the pipeline is
# generated. By default it's set to kfp.
_SDK_ENV_LABEL = 'pipelines.kubeflow.org/pipeline-sdk-type'
_SDK_ENV_DEFAULT = 'kfp'


def get_default_telemetry_labels() -> Dict[Text, Text]:
    """Returns the default pod labels for telemetry purpose."""
    result = {
        _SDK_ENV_LABEL: _SDK_ENV_DEFAULT,
    }
    return result


def add_pod_env(op: BaseOp) -> BaseOp:
    """Adds pod environment info to ContainerOp.
    """
    if isinstance(op, ContainerOp) and op.pod_labels and op.pod_labels.get('add-pod-env', None) == 'true':
        from kubernetes import client as k8s_client
        op.container.add_env_variable(
            k8s_client.V1EnvVar(
                name='KFP_POD_NAME',
                value_from=k8s_client.V1EnvVarSource(
                    field_ref=k8s_client.V1ObjectFieldSelector(
                        field_path='metadata.name'
                    )
                )
            )
        ).add_env_variable(
            k8s_client.V1EnvVar(
                name='KFP_NAMESPACE',
                value_from=k8s_client.V1EnvVarSource(
                    field_ref=k8s_client.V1ObjectFieldSelector(
                        field_path='metadata.namespace'
                    )
                )
            )
        )
    return op


def add_pod_labels(labels: Optional[Dict[Text, Text]] = None) -> Callable:
    """Adds provided pod labels to each pod."""

    def _add_pod_labels(task):
        for k, v in labels.items():
            # Only append but not update.
            # This is needed to bypass TFX pipelines/components.
            if k not in task.pod_labels:
                task.add_pod_label(k, v)
        return task

    return _add_pod_labels
