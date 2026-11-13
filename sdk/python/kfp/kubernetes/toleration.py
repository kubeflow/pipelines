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
from typing import Optional, Union

from google.protobuf import json_format

from kfp.dsl import PipelineTask, pipeline_channel
from kfp.kubernetes import common
from kfp.kubernetes import kubernetes_executor_config_pb2 as pb

try:
    from typing import Literal
except ImportError:
    from typing_extensions import Literal


_VALID_TOLERATION_OPERATORS = ('Equal', 'Exists', 'Gt', 'Lt')
_VALID_TOLERATION_EFFECTS = ('NoExecute', 'NoSchedule', 'PreferNoSchedule')


def _validate_toleration_field(name: str, value, valid_values: tuple):
    # None and '' mean "omitted": Kubernetes defaults the operator to Equal
    # and an empty effect matches all effects.
    if value is None or value == '':
        return
    if value not in valid_values:
        raise ValueError(
            f'Invalid toleration {name} {value!r}. Must be one of: '
            f'{", ".join(valid_values)} (or None/empty string to omit).'
        )


def add_toleration(
    task: PipelineTask,
    key: Optional[str] = None,
    operator: Optional[Literal['Equal', 'Exists', 'Gt', 'Lt']] = None,
    value: Optional[str] = None,
    effect: Optional[
        Literal['NoExecute', 'NoSchedule', 'PreferNoSchedule']
    ] = None,
    toleration_seconds: Optional[int] = None,
):
    """Add a `toleration <https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/>`__ to a task.

    Args:
        task:
            Pipeline task.
        key:
            key is the taint key that the toleration applies to. Empty means
            match all taint keys. If the key is empty, operator must be "Exists";
            this combination means to match all values and all keys.
        operator:
            operator represents a key's relationship to the value. Valid
            operators are "Exists", "Equal", "Gt" and "Lt". Defaults to
            "Equal". "Exists" is equivalent to wildcard for value, so that a
            pod can tolerate all taints of a particular category. "Gt" and
            "Lt" are numeric comparisons and need a Kubernetes version that
            supports them. Any other value raises ``ValueError``.
        value:
            value is the taint value the toleration matches to. If the operator
            is "Exists", the value should be empty, otherwise just a regular
            string.
        effect:
            effect indicates the taint effect to match. Empty means match all
            taint effects. When specified, allowed values are NoSchedule,
            PreferNoSchedule and NoExecute. Any other value raises
            ``ValueError``.
        toleration_seconds:
            toleration_seconds represents the period of time the toleration
            (which must be of effect NoExecute, otherwise this field is ignored)
            tolerates the taint. By default, it is not set, which means tolerate
            the taint forever (do not evict). Zero and negative values will be
            treated as 0 (evict immediately) by the system.

    Returns:
        Task object with added toleration.
    """

    _validate_toleration_field(
        'operator', operator, _VALID_TOLERATION_OPERATORS
    )
    _validate_toleration_field('effect', effect, _VALID_TOLERATION_EFFECTS)

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
    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task


def add_toleration_json(
    task: PipelineTask,
    toleration_json: Union[
        pipeline_channel.PipelineParameterChannel, list, dict
    ],
):
    """Add a Pod Toleration in the form of a Pipeline Input JSON to a task.

    Args:
        task:
            Pipeline task.
        toleration_json:
            a toleration that is a pipeline input parameter, dict, or list.
            The input parameter must be of type dict or list.

            If it is an  input parameter of type dict, it must be a single toleration object.
            For example a pipeline input parameter in this case could be::

                {
                  "key": "key1",
                  "operator": "Equal",
                  "value": "value1",
                  "effect": "NoSchedule"
                }

            If it is an input parameter of type list, it must be list of toleration objects.
            For example a pipeline input parameter in this case could be::

                [
                    {
                      "key": "key1",
                      "operator": "Equal",
                      "value": "value1",
                      "effect": "NoSchedule"
                    },
                    {
                      "key": "key2",
                      "operator": "Exists",
                      "effect": "NoExecute"
                    }
                ]

            In the case of static list or dicts, the call wraps add_toleration.

    Returns:
        Task object with added toleration.
    """

    if isinstance(toleration_json, pipeline_channel.PipelineParameterChannel):
        msg = common.get_existing_kubernetes_config_as_message(task)
        toleration = pb.Toleration()
        toleration.toleration_json.CopyFrom(
            common.parse_k8s_parameter_input(toleration_json, task)
        )
        msg.tolerations.append(toleration)
        task.platform_config['kubernetes'] = json_format.MessageToDict(msg)
    elif isinstance(toleration_json, list):
        for toleration in toleration_json:
            _add_dict_toleration(task, toleration)
    elif isinstance(toleration_json, dict):
        _add_dict_toleration(task, toleration_json)
    else:
        raise ValueError(
            'toleration_json must be a dict, list, or input parameter'
        )

    return task


def _add_dict_toleration(task: PipelineTask, toleration: dict):
    # Accepts camelCase keys (e.g. tolerationSeconds) and returns a new dict
    # so the caller's input is not mutated.
    toleration = common.deserialize_dict_to_k8s_model_keys(toleration)
    add_toleration(
        task,
        key=toleration.get('key'),
        value=toleration.get('value'),
        operator=toleration.get('operator'),
        effect=toleration.get('effect'),
        toleration_seconds=toleration.get('toleration_seconds'),
    )
