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

try:
    from typing import Literal
except ImportError:
    from typing_extensions import Literal


def add_toleration(
    task: PipelineTask,
    key: Optional[str] = None,
    operator: Optional[Literal["Equal", "Exists"]] = None,
    value: Optional[str] = None,
    effect: Optional[Literal["NoExecute", "NoSchedule", "PreferNoSchedule"]] = None,
    toleration_seconds: Optional[int] = None,
):
    """Add a `toleration<https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/>`_. to a task.

    Args:
        task:
            Pipeline task.
        key:
            key is the taint key that the toleration applies to. Empty means
            match all taint keys. If the key is empty, operator must be Exists;
            this combination means to match all values and all keys.
        operator:
            operator represents a key's relationship to the value. Valid
            operators are Exists and Equal. Defaults to Equal. Exists is
            equivalent to wildcard for value, so that a pod can tolerate all
            taints of a particular category.
        value:
            value is the taint value the toleration matches to. If the operator
            is Exists, the value should be empty, otherwise just a regular
            string.
        effect:
            effect indicates the taint effect to match. Empty means match all
            taint effects. When specified, allowed values are NoSchedule,
            PreferNoSchedule and NoExecute.
        toleration_seconds:
            toleration_seconds represents the period of time the toleration
            (which must be of effect NoExecute, otherwise this field is ignored)
            tolerates the taint. By default, it is not set, which means tolerate
            the taint forever (do not evict). Zero and negative values will be
            treated as 0 (evict immediately) by the system.

    Returns:
        Task object with added toleration.
    """

    msg = common.get_existing_kubernetes_config_as_message(task)
    msg.tolerations.append(
        pb.Toleration(
            key=key,
            operator=operator,
            value=value,
            effect=effect,
            toleration_seconds=toleration_seconds,
        )
    )
    task.platform_config["kubernetes"] = json_format.MessageToDict(msg)

    return task
