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
"""Pipeline task state classes."""
import abc
from typing import Any, Dict, List, Optional


class PipelineTaskState(abc.ABC):
    """PipelineTaskState subclasses must implement all of the public properties
    and methods on PipelineTask."""

    def __init__(self, pipeline_task: 'pipeline_task.PipelineTask') -> None:
        self.task = pipeline_task

    @property
    @abc.abstractmethod
    def platform_spec(self) -> Any:
        pass

    @property
    @abc.abstractmethod
    def name(self) -> str:
        pass

    @property
    @abc.abstractmethod
    def inputs(self) -> Dict[str, Any]:
        pass

    @property
    @abc.abstractmethod
    def output(self) -> Any:
        pass

    @property
    @abc.abstractmethod
    def outputs(self) -> Dict[str, Any]:
        pass

    @property
    @abc.abstractmethod
    def dependent_tasks(self) -> List['PipelineTaskBase']:
        pass

    @abc.abstractmethod
    def set_caching_options(self, enable_caching: bool) -> 'PipelineTaskBase':
        pass

    @abc.abstractmethod
    def set_cpu_request(self, cpu: str) -> 'PipelineTaskBase':
        pass

    @abc.abstractmethod
    def set_cpu_limit(self, cpu: str) -> 'PipelineTaskBase':
        pass

    @abc.abstractmethod
    def set_accelerator_limit(self, limit: int) -> 'PipelineTaskBase':
        pass

    @abc.abstractmethod
    def set_memory_request(self, memory: str) -> 'PipelineTaskBase':
        pass

    @abc.abstractmethod
    def set_memory_limit(self, memory: str) -> 'PipelineTaskBase':
        pass

    @abc.abstractmethod
    def set_retry(
            self,
            num_retries: int,
            backoff_duration: Optional[str] = None,
            backoff_factor: Optional[float] = None,
            backoff_max_duration: Optional[str] = None) -> 'PipelineTaskBase':
        pass

    @abc.abstractmethod
    def set_display_name(self, name: str) -> 'PipelineTaskBase':
        pass

    @abc.abstractmethod
    def set_env_variable(self, name: str, value: str) -> 'PipelineTaskBase':
        pass

    @abc.abstractmethod
    def after(self, *tasks) -> 'PipelineTaskBase':
        pass

    @abc.abstractmethod
    def ignore_upstream_failure(self) -> 'PipelineTaskBase':
        pass
