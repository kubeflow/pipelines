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

from typing import Optional, List
from dataclasses import dataclass

from google.protobuf import json_format
from kfp.dsl import PipelineTask
from kfp.kubernetes import common
from kfp.kubernetes import kubernetes_executor_config_pb2 as pb

try:
    from typing import Literal
except ImportError:
    from typing_extensions import Literal


@dataclass
class SelectorRequirement:
    """Used to define the requirements of an affinity.
    key: either the field (if used with match_fields) or the label key (match_expressions) to match on
    operator: One of: In, NotIn, Exists, DoesNotExist. For nodeAffinity, Gt and Lt are also legal. More info: `https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#operators`
    values: List of string values to match on.
    """
    key: str
    operator: Literal["In", "NotIn", "Exists", "DoesNotExist", "Gt", "Lt"]
    values: List[str]

def add_node_affinity(
    task: PipelineTask,
    match_expressions: List[SelectorRequirement] = [],
    match_fields: List[SelectorRequirement] = [],
    weight: Optional[int] = None
    
):
    """Add a `node affinity<https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity>`_. to a task.
    Args:
        task:
            Pipeline task.
        match_expressions:
            A list of requirements of the affinity that will match node labels. 
        match_fields:
            A list of requirements of the affinity that will match node's other fields
        weight:
            affinity weight indicates that the affinity rule is preferred/soft, not required/hard.
    Returns:
        Task object with added node affinity terms.
    """
    match_expressions_list = [
        pb.SelectorRequirement(key = requirement.key, operator= requirement.operator, values = requirement.values)
        for requirement in match_expressions
    ]
    match_fields_list = [
        pb.SelectorRequirement(key = requirement.key, operator= requirement.operator, values = requirement.values)
        for requirement in match_fields
    ]
    
    if weight is not None and not (1 <= weight <= 100):
        raise ValueError("If weight is set, it should be an integer between 1 and 100") 
    
    msg = common.get_existing_kubernetes_config_as_message(task)
    msg.node_affinity.append(
        pb.NodeAffinityTerm(
            match_expressions=match_expressions_list,
            match_fields=match_fields_list,
            weight=weight
        )
    )
    task.platform_config["kubernetes"] = json_format.MessageToDict(msg)

    return task