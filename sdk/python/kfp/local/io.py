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

from kfp.local import status as status_module


class IOStore:

    def __init__(self):

        self._task_output_data: Dict[str,
                                     Dict[str,
                                          Any]] = collections.defaultdict(dict)
        self._parent_input_data: Dict[str, Any] = {}
        self._task_status_data: Dict[str, status_module.Status] = {}
        self._task_error_data: Dict[str, Optional[str]] = {}

    def put_parent_input(
        self,
        key: str,
        value: Any,
    ) -> None:
        """Persist the value of a parent component (i.e., parent pipeline)
        input.

        Args:
            key: Parent component input name.
            value: Value associated with key.
        """
        self._parent_input_data[key] = value

    def get_parent_input(
        self,
        key: str,
    ) -> None:
        """Get the value of the parent component (i.e., parent pipeline) input
        named key.

        Args:
            key: Parent component input name.

        Returns:
            The output value.
        """
        if key in self._parent_input_data:
            return self._parent_input_data[key]
        raise ValueError(f"Parent pipeline input argument '{key}' not found.")

    def put_task_output(
        self,
        task_name: str,
        key: str,
        value: Any,
    ) -> None:
        """Persist the value of an upstream task output.

        Args:
            task_name: Upstream task name.
            key: Output name.
            value: Value associated with key.
        """
        self._task_output_data[task_name][key] = value

    def get_task_output(
        self,
        task_name: str,
        key: str,
    ) -> Any:
        """Get the value of an upstream task output.

        Args:
            task_name: Upstream task name.
            key: Output name.

        Returns:
            The output value.
        """
        common_exception_string = f"Tried to get output '{key}' from task '{task_name}'"
        if task_name in self._task_output_data:
            outputs = self._task_output_data[task_name]
        else:
            raise ValueError(
                f"{common_exception_string}, but task '{task_name}' not found.")

        if key in outputs:
            return outputs[key]
        else:
            raise ValueError(
                f"{common_exception_string}, but task '{task_name}' has no output named '{key}'."
            )

    def put_task_status(
        self,
        task_name: str,
        task_status: status_module.Status,
        error_message: Optional[str] = None,
    ) -> None:
        """Persist the status of a task execution.

        Args:
            task_name: Task name.
            task_status: The status of the task (SUCCESS or FAILURE).
            error_message: Optional error message if task failed.
        """
        self._task_status_data[task_name] = task_status
        if error_message:
            self._task_error_data[task_name] = error_message

    def get_task_status(
        self,
        task_name: str,
    ) -> status_module.Status:
        """Get the status of a task.

        Args:
            task_name: Task name.

        Returns:
            The task status.
        """
        if task_name in self._task_status_data:
            return self._task_status_data[task_name]
        raise ValueError(f"Task '{task_name}' status not found.")

    def get_task_error(
        self,
        task_name: str,
    ) -> Optional[str]:
        """Get the error message of a task, if any.

        Args:
            task_name: Task name.

        Returns:
            The error message or None.
        """
        return self._task_error_data.get(task_name)
