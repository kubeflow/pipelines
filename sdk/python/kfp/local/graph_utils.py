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
"""Graph algorithms which are useful for working with PipelineSpec."""

from typing import Dict, List, Set

from kfp.pipeline_spec import pipeline_spec_pb2


def topological_sort_tasks(
        tasks: Dict[str, pipeline_spec_pb2.PipelineTaskSpec]) -> List[str]:
    """Given a dictionary of task name to PipelineTaskSpec, obtains a
    topologically sorted stack of task names.

    Args:
        tasks: The tasks in the pipeline.

    Returns:
        A totally ordered stack of tasks. Tasks should be executed in the order they are popped off the right side of the stack.
    """
    adj_list = build_adjacency_list(tasks)
    return topological_sort(adj_list)


def build_adjacency_list(
    tasks: Dict[str,
                pipeline_spec_pb2.PipelineTaskSpec]) -> Dict[str, List[str]]:
    """Builds an adjacency list from a dictionary of task name to
    PipelineTaskSpec. This is a data structure simplification step, which
    allows for a general topological_sort sort implementation.

    Args:
        tasks: The tasks in the pipeline.

    Returns:
        An adjacency list of tasks name to a list of upstream tasks. The key task depends on all value tasks being executed first.
    """
    return {
        task_name: task_details.dependent_tasks
        for task_name, task_details in tasks.items()
    }


def topological_sort(adj_list: Dict[str, List[str]]) -> List[str]:
    """Topologicall sorts an adjacency list.

    Args:
        adj_list: An adjacency list of tasks name to a list of upstream tasks. The key task depends on all value tasks being executed first.

    Returns:
        A totally ordered stack of tasks. Tasks should be executed in the order they are popped off the right side of the stack.
    """

    def dfs(node: str) -> None:
        visited.add(node)
        for neighbor in adj_list[node]:
            if neighbor not in visited:
                dfs(neighbor)
        result.append(node)

    # sort lists to force deterministic result
    adj_list = {k: sorted(v) for k, v in adj_list.items()}
    visited: Set[str] = set()
    result = []
    for node in adj_list:
        if node not in visited:
            dfs(node)
    return result[::-1]
