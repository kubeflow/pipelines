# Copyright 2023 The Kubeflow Authors
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
import inspect
from typing import Any, Dict, List, Optional

from kfp.dsl import pipeline_task_state_abc


class FinalTaskState(pipeline_task_state_abc.PipelineTaskState):
    """A pipeline task which has not been executed.

    The corresponding PipelineTask's state can be updated via methods.
    """

    @property
    def outputs(self) -> Dict[str, Any]:
        return self.task._outputs

    @property
    def output(self) -> Any:
        if len(self.task._outputs) != 1:
            raise AttributeError(
                'The task has multiple outputs. Please reference the output by its name.'
            )
        return list(self.task._outputs.values())[0]

    @property
    def platform_spec(self) -> Any:
        raise NotImplementedError(
            'Platform-specific features are not supported for local execution.')

    @property
    def name(self) -> str:
        return self.task._task_spec.name

    @property
    def inputs(self) -> Dict[str, Any]:
        return self.task._inputs

    @property
    def dependent_tasks(self) -> List['PipelineTask']:
        raise NotImplementedError(
            'Task has no dependent tasks since it is executed independently.')

    def set_caching_options(self, enable_caching: bool) -> None:
        self._raise_task_configuration_not_implemented()

    def set_cpu_request(self, cpu: str) -> None:
        self._raise_task_configuration_not_implemented()

    def set_cpu_limit(self, cpu: str) -> None:
        self._raise_task_configuration_not_implemented()

    def set_accelerator_limit(self, limit: int) -> None:
        self._raise_task_configuration_not_implemented()

    def set_memory_request(self, memory: str) -> None:
        self._raise_task_configuration_not_implemented()

    def set_memory_limit(self, memory: str) -> None:
        self._raise_task_configuration_not_implemented()

    def set_retry(self,
                  num_retries: int,
                  backoff_duration: Optional[str] = None,
                  backoff_factor: Optional[float] = None,
                  backoff_max_duration: Optional[str] = None) -> None:
        self._raise_task_configuration_not_implemented()

    def set_display_name(self, name: str) -> None:
        self._raise_task_configuration_not_implemented()

    def set_env_variable(self, name: str, value: str) -> None:
        self._raise_task_configuration_not_implemented()

    def after(self, *tasks) -> None:
        self._raise_task_configuration_not_implemented()

    def ignore_upstream_failure(self) -> None:
        self._raise_task_configuration_not_implemented()

    def _raise_task_configuration_not_implemented(self) -> None:
        stack = inspect.stack()
        caller_name = stack[1].function
        raise NotImplementedError(
            f"Task configuration methods are not supported for local execution. Got call to '.{caller_name}()'."
        )
