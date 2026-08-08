# Copyright 2026 The Kubeflow Authors
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


def add_pod_affinity(
    task: PipelineTask,
    match_pod_expressions: Optional[List[dict]] = None,
    match_pod_labels: Optional[dict] = None,
    topology_key: Optional[str] = None,
    namespaces: Optional[List[str]] = None,
    match_namespace_expressions: Optional[List[dict]] = None,
    match_namespace_labels: Optional[dict] = None,
    weight: Optional[int] = None,
    anti: Optional[bool] = None,
) -> PipelineTask:
    """Add a constraint to the task Pod's `podAffinity` or `podAntiAffinity`
    <https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#inter-pod-affinity-and-anti-affinity>`_.

    Args:
        task: Pipeline task.
        match_pod_expressions: List of dicts for matchPodExpressions (keys: key, operator, values).
        match_pod_labels: Dict of key-value pairs for matchPodLabels.
        topology_key: The topology key.
        namespaces: List of namespaces.
        match_namespace_expressions: List of dicts for matchNamespaceExpressions (keys: key, operator, values).
        match_namespace_labels: Dict of key-value pairs for matchNamespaceLabels.
        weight: If set, this affinity is preferred (K8s weight 1-100); otherwise required.
        anti: If set to True, adds a PodAntiAffinity. Otherwise, adds a PodAffinity.

    Returns:
        Task object with added pod affinity.
    """
    VALID_OPERATORS = {"In", "NotIn", "Exists", "DoesNotExist"}
    msg = common.get_existing_kubernetes_config_as_message(task)
    affinity_term = pb.PodAffinityTerm()

    _add_affinity_terms(affinity_term.match_pod_expressions, match_pod_expressions, "match_pod_expression", VALID_OPERATORS)
    
    if match_pod_labels:
        affinity_term.match_pod_labels.update(match_pod_labels)
        
    if topology_key:
        affinity_term.topology_key = topology_key
        
    if namespaces:
        affinity_term.namespaces.extend(namespaces)
        
    _add_affinity_terms(affinity_term.match_namespace_expressions, match_namespace_expressions, "match_namespace_expression", VALID_OPERATORS)

    if match_namespace_labels:
        affinity_term.match_namespace_labels.update(match_namespace_labels)

    if weight is not None:
        if not (1 <= weight <= 100):
            raise ValueError(f"weight must be between 1 and 100, got {weight}.")
        affinity_term.weight = weight
        
    if anti is not None:
        affinity_term.anti = anti
        
    msg.pod_affinity.append(affinity_term)
    task.platform_config["kubernetes"] = json_format.MessageToDict(msg)
    return task

def validate_pod_affinity(pod_affinity_json: dict):
    """
    Validates a Python dictionary against the Kubernetes V1PodAffinityTerm model.

    Args:
        pod_affinity_json: A dictionary representing a Kubernetes PodAffinityTerm object.

    Returns:
        True if the dictionary is a valid V1PodAffinityTerm.

    Raises:
        ValueError: If the dictionary does not conform to the V1PodAffinityTerm schema.
    """
    from kubernetes import client

    try:
        k8s_model_dict = common.deserialize_dict_to_k8s_model_keys(pod_affinity_json)
        client.V1PodAffinityTerm(**k8s_model_dict)
    except (TypeError, ValueError) as e:
        raise ValueError(f"Invalid V1PodAffinityTerm JSON: {e}")

def add_pod_affinity_json(
    task: PipelineTask,
    pod_affinity_json: Union[pipeline_channel.PipelineParameterChannel, dict],
    anti: Optional[bool] = None,
) -> PipelineTask:
    """Add a pod affinity constraint to the task Pod's `podAffinity` or `podAntiAffinity`
    using a JSON struct or pipeline parameter.

    This allows parameterized pod affinity to be specified,
    matching the Kubernetes PodAffinityTerm schema:
    https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#PodAffinityTerm

    Args:
        task: Pipeline task.
        pod_affinity_json: Dict or pipeline parameter for pod affinity. Should match K8s PodAffinityTerm schema.
        anti: If set to True, adds a PodAntiAffinity. Otherwise, adds a PodAffinity.

    Returns:
        Task object with added pod affinity.
    """
   
    if isinstance(pod_affinity_json, dict):
        validate_pod_affinity(pod_affinity_json)
    msg = common.get_existing_kubernetes_config_as_message(task)
    
    # Remove any previous JSON-based pod affinity terms with the same 'anti' setting
    for i in range(len(msg.pod_affinity) - 1, -1, -1):
        if msg.pod_affinity[i].HasField("pod_affinity_json") and msg.pod_affinity[i].anti == bool(anti):
            del msg.pod_affinity[i]
            
    affinity_term = pb.PodAffinityTerm()
    input_param_spec = common.parse_k8s_parameter_input(pod_affinity_json, task)
    affinity_term.pod_affinity_json.CopyFrom(input_param_spec)
    if anti is not None:
        affinity_term.anti = anti
        
    msg.pod_affinity.append(affinity_term)
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
