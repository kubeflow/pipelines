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
    dependency_map = build_dependency_map(tasks)
    return topological_sort(dependency_map)


def build_dependency_map(
    tasks: Dict[str,
                pipeline_spec_pb2.PipelineTaskSpec]) -> Dict[str, List[str]]:
    """Builds a dictionary of task name to all upstream task names
    (dependencies). This is a data structure simplification step, which allows
    for a general topological_sort sort implementation.

    Args:
        tasks: The tasks in the pipeline.

    Returns:
        An dictionary of task name to all upstream tasks. The key task depends on all value tasks being executed first.
    """
    return {
        task_name: task_details.dependent_tasks
        for task_name, task_details in tasks.items()
    }


def topological_sort(dependency_map: Dict[str, List[str]]) -> List[str]:
    """Topologically sorts a dictionary of task names to upstream tasks.

    Args:
        dependency_map: A dictionary of tasks name to a list of upstream tasks. The key task depends on all value tasks being executed first.

    Returns:
        A totally ordered stack of tasks. Tasks should be executed in the order they are popped off the right side of the stack.
    """

    def dfs(node: str) -> None:
        visited.add(node)
        for neighbor in dependency_map[node]:
            if neighbor not in visited:
                dfs(neighbor)
        result.append(node)

    # sort lists to force deterministic result
    dependency_map = {k: sorted(v) for k, v in dependency_map.items()}
    visited: Set[str] = set()
    result = []
    for node in dependency_map:
        if node not in visited:
            dfs(node)
    return result[::-1]
