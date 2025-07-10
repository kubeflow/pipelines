# Copyright 2025 The Kubeflow Authors
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

from typing import List, Optional, Union
from google.protobuf import json_format
from kfp.dsl import PipelineTask, pipeline_channel
from kfp.kubernetes import common
from kfp.kubernetes import kubernetes_executor_config_pb2 as pb
from kubernetes import client


def add_node_affinity(
    task: PipelineTask,
    match_expressions: Optional[List[dict]] = None,
    match_fields: Optional[List[dict]] = None,
    weight: Optional[int] = None,
) -> PipelineTask:
    """Add a constraint to the task Pod's `nodeAffinity
    <https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity>`_.

    Each constraint is specified as a match expression (key, operator, values) or match field,
    corresponding to the PodSpec's `nodeAffinity <https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#NodeAffinity>`_ field.

    Args:
        task: Pipeline task.
        match_expressions: List of dicts for matchExpressions (keys: key, operator, values).
        match_fields: List of dicts for matchFields (keys: key, operator, values).
        weight: If set, this affinity is preferred (K8s weight 1-100); otherwise required.

    Returns:
        Task object with added node affinity.
    """
    VALID_OPERATORS = {"In", "NotIn", "Exists", "DoesNotExist", "Gt", "Lt"}
    msg = common.get_existing_kubernetes_config_as_message(task)
    affinity_term = pb.NodeAffinityTerm()

    _add_affinity_terms(affinity_term.match_expressions, match_expressions, "match_expression", VALID_OPERATORS)
    _add_affinity_terms(affinity_term.match_fields, match_fields, "match_field", VALID_OPERATORS)

    if weight is not None:
        if not (1 <= weight <= 100):
            raise ValueError(f"weight must be between 1 and 100, got {weight}.")
        affinity_term.weight = weight
    msg.node_affinity.append(affinity_term)
    task.platform_config["kubernetes"] = json_format.MessageToDict(msg)
    return task

def validate_node_affinity(node_affinity_json: dict):
    """
    Validates a Python dictionary against the Kubernetes V1NodeAffinity model.

    Args:
        node_affinity_json: A dictionary representing a Kubernetes NodeAffinity object.

    Returns:
        True if the dictionary is a valid V1NodeAffinity.

    Raises:
        ValueError: If the dictionary does not conform to the V1NodeAffinity schema.
    """
    from kubernetes import client

    try:
        k8s_model_dict = common.deserialize_dict_to_k8s_model_keys(node_affinity_json)
        client.V1NodeAffinity(**k8s_model_dict)
    except (TypeError, ValueError) as e:
        raise ValueError(f"Invalid V1NodeAffinity JSON: {e}")

def add_node_affinity_json(
    task: PipelineTask,
    node_affinity_json: Union[pipeline_channel.PipelineParameterChannel, dict],
) -> PipelineTask:
    """Add a node affinity constraint to the task Pod's `nodeAffinity`
    using a JSON struct or pipeline parameter.

    This allows parameterized node affinity to be specified,
    matching the Kubernetes NodeAffinity schema:
    https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#nodeaffinity

    Args:
        task: Pipeline task.
        node_affinity_json: Dict or pipeline parameter for node affinity. Should match K8s NodeAffinity schema.

    Returns:
        Task object with added node affinity.
    """
   
    if isinstance(node_affinity_json, dict):
        validate_node_affinity(node_affinity_json)
    msg = common.get_existing_kubernetes_config_as_message(task)
    # Remove any previous JSON-based node affinity terms
    for i in range(len(msg.node_affinity) - 1, -1, -1):
        if msg.node_affinity[i].HasField("node_affinity_json"):
            del msg.node_affinity[i]
    affinity_term = pb.NodeAffinityTerm()
    input_param_spec = common.parse_k8s_parameter_input(node_affinity_json, task)
    affinity_term.node_affinity_json.CopyFrom(input_param_spec)
    msg.node_affinity.append(affinity_term)
    task.platform_config["kubernetes"] = json_format.MessageToDict(msg)
    return task

def _add_affinity_terms(
    affinity_list,
    selector_terms: Optional[List[dict]],
    term_kind: str,
    valid_operators: set,
):
    for selector in selector_terms or []:
        key = selector.get("key")
        operator = selector.get("operator")

        if not key:
            raise ValueError(f"Each {term_kind} must have a non-empty 'key'.")
        if not operator:
            raise ValueError(f"Each {term_kind} for key '{key}' must have a non-empty 'operator'.")
        if operator not in valid_operators:
            raise ValueError(
                f"Invalid operator '{operator}' for key '{key}' in {term_kind}. "
                f"Must be one of {sorted(valid_operators)}."
            )

        affinity_list.add(
            key=key,
            operator=operator,
            values=selector.get("values", []),
        )