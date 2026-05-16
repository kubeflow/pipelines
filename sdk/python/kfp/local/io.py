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
"""Object for storing task outputs in-memory during local execution."""

import collections
from typing import Any, Dict, Optional


class IOStore:
    """In-memory store of a DAG's parameter/artifact state.

    Each DAG invocation gets its own IOStore. A nested DAG receives a
    reference to its enclosing store via `parent`, and lookups that miss in
    the local scope walk up the chain. This lets a nested ParallelFor or
    Condition body resolve a pipeline-channel reference surfaced from an
    outer scope (e.g. an outer task's output) even when the compiler already
    bound it as a DAG input at the nested boundary.

    Writes always stay local to the current scope — a nested DAG never
    mutates its parent's state.
    """

    def __init__(self, parent: Optional['IOStore'] = None):
        self._task_output_data: Dict[str,
                                     Dict[str,
                                          Any]] = collections.defaultdict(dict)
        self._parent_input_data: Dict[str, Any] = {}
        self._task_status_data: Dict[str, str] = {}
        self._parent = parent

    def put_parent_input(
        self,
        key: str,
        value: Any,
    ) -> None:
        """Persist the value of a parent component (i.e., parent pipeline)
        input."""
        self._parent_input_data[key] = value

    def get_parent_input(
        self,
        key: str,
    ) -> Any:
        """Get the value of the parent component (i.e., parent pipeline) input
        named key.

        Walks up the enclosing DAG chain on miss — an inner DAG may
        reference a pipeline channel whose binding was resolved at an
        outer scope.
        """
        if key in self._parent_input_data:
            return self._parent_input_data[key]
        if self._parent is not None:
            return self._parent.get_parent_input(key)
        raise ValueError(f"Parent pipeline input argument '{key}' not found.")

    def put_task_output(
        self,
        task_name: str,
        key: str,
        value: Any,
    ) -> None:
        """Persist the value of an upstream task output."""
        self._task_output_data[task_name][key] = value

    def get_task_output(
        self,
        task_name: str,
        key: str,
    ) -> Any:
        """Get the value of an upstream task output.

        Walks up the enclosing DAG chain on miss. Compiled IR normally
        surfaces cross-scope task outputs as DAG inputs, but some items
        resolution paths (e.g. ParallelFor over an outer task's output)
        walk directly from a task name — the chain keeps that path
        working.
        """
        common_exception_string = (
            f"Tried to get output '{key}' from task '{task_name}'")

        if task_name in self._task_output_data:
            outputs = self._task_output_data[task_name]
            if key in outputs:
                return outputs[key]
            if self._parent is not None:
                try:
                    return self._parent.get_task_output(task_name, key)
                except ValueError:
                    pass
            raise ValueError(
                f"{common_exception_string}, but task '{task_name}' has no "
                f"output named '{key}'.")

        if self._parent is not None:
            try:
                return self._parent.get_task_output(task_name, key)
            except ValueError:
                pass
        raise ValueError(
            f"{common_exception_string}, but task '{task_name}' not found.")

    def put_task_status(
        self,
        task_name: str,
        task_status: str,
    ) -> None:
        """Persist the final status of a task."""
        self._task_status_data[task_name] = task_status

    def get_task_status(
        self,
        task_name: str,
    ) -> str:
        """Get the final status of a task.

        Walks up the enclosing chain on miss so exit handlers in nested
        scopes can see outer statuses.
        """
        if task_name in self._task_status_data:
            return self._task_status_data[task_name]
        if self._parent is not None:
            return self._parent.get_task_status(task_name)
        raise ValueError(f"Status for task '{task_name}' not found.")
