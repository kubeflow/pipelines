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
import copy
import inspect
import re
from typing import Dict, List, Mapping, Optional, Union

from kfp.dsl import constants
from kfp.dsl import pipeline_channel
from kfp.dsl import pipeline_task_state_abc
from kfp.dsl import placeholders
from kfp.dsl import structures
from kfp.dsl.types import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2


class FutureTaskState(pipeline_task_state_abc.PipelineTaskState):
    """A pipeline task which is final and immutable.

    Only exists when a task is executed locally.
    """

    @property
    def platform_spec(self) -> pipeline_spec_pb2.PlatformSpec:
        """PlatformSpec for all tasks in the pipeline as task.

        Only for use on tasks created from GraphComponents.
        """
        if self.task.pipeline_spec:
            return self.task.component_spec.platform_spec

        # can only create primitive task platform spec at compile-time, since the executor label is not known until then
        raise ValueError(
            f'Can only access {".platform_spec"!r} property on a tasks created from pipelines. Use {".platform_config"!r} for tasks created from primitive components.'
        )

    @property
    def name(self) -> str:
        """The name of the task.

        Unique within its parent group.
        """
        return self.task._task_spec.name

    @property
    def inputs(
        self
    ) -> Dict[str, Union[type_utils.PARAMETER_TYPES,
                         pipeline_channel.PipelineChannel]]:
        """The inputs passed to the task."""
        return self.task._inputs

    @property
    def output(self) -> pipeline_channel.PipelineChannel:
        """The single output of the task.

        Used when a task has exactly one output parameter.
        """
        if len(self.task._outputs) != 1:
            raise AttributeError(
                'The task has multiple outputs. Please reference the output by its name.'
            )
        return list(self.task._outputs.values())[0]

    @property
    def outputs(self) -> Mapping[str, pipeline_channel.PipelineChannel]:
        """The dictionary of outputs of the task.

        Used when a task has more the one output or uses an
        ``OutputPath`` or ``Output[Artifact]`` type annotation.
        """
        return self.task._outputs

    @property
    def dependent_tasks(self) -> List[str]:
        """A list of the dependent task names."""
        return self.task._task_spec.dependent_tasks

    def _extract_container_spec_and_convert_placeholders(
        self, component_spec: structures.ComponentSpec
    ) -> structures.ContainerSpecImplementation:
        """Extracts a ContainerSpec from a ComponentSpec and converts
        placeholder objects to strings.

        Args:
            component_spec: The component definition.
        """
        container_spec = copy.deepcopy(component_spec.implementation.container)
        if container_spec is None:
            raise ValueError(
                '_extract_container_spec_and_convert_placeholders used incorrectly. ComponentSpec.implementation.container is None.'
            )
        container_spec.command = [
            placeholders.convert_command_line_element_to_string(e)
            for e in container_spec.command or []
        ]
        container_spec.args = [
            placeholders.convert_command_line_element_to_string(e)
            for e in container_spec.args or []
        ]
        return container_spec

    def set_caching_options(self, enable_caching: bool) -> None:
        """Sets caching options for the task.

        Args:
            enable_caching: Whether to enable caching.

        Returns:
            Self return to allow chained setting calls.
        """
        self.task._task_spec.enable_caching = enable_caching

    def _ensure_container_spec_exists(self) -> None:
        """Ensures that the task has a container spec."""
        caller_method_name = inspect.stack()[1][3]

        if self.task.container_spec is None:
            raise ValueError(
                f'{caller_method_name} can only be used on single-step components, not pipelines used as components, or special components like importers.'
            )

    def _validate_cpu_request_limit(self, cpu: str) -> float:
        """Validates cpu request/limit string and converts to its numeric
        value.

        Args:
            cpu: CPU requests or limits. This string should be a number or a
                number followed by an "m" to indicate millicores (1/1000). For
                more information, see `Specify a CPU Request and a CPU Limit
                <https://kubernetes.io/docs/tasks/configure-pod-container/assign-cpu-resource/#specify-a-cpu-request-and-a-cpu-limit>`_.

        Raises:
            ValueError if the cpu request/limit string value is invalid.

        Returns:
            The numeric value (float) of the cpu request/limit.
        """
        if re.match(r'([0-9]*[.])?[0-9]+m?$', cpu) is None:
            raise ValueError(
                'Invalid cpu string. Should be float or integer, or integer'
                ' followed by "m".')

        return float(cpu[:-1]) / 1000 if cpu.endswith('m') else float(cpu)

    def set_cpu_request(self, cpu: str) -> None:
        """Sets CPU request (minimum) for the task.

        Args:
            cpu: Minimum CPU requests required. This string should be a number
                or a number followed by an "m" to indicate millicores (1/1000).
                For more information, see `Specify a CPU Request and a CPU Limit
                <https://kubernetes.io/docs/tasks/configure-pod-container/assign-cpu-resource/#specify-a-cpu-request-and-a-cpu-limit>`_.

        Returns:
            Self return to allow chained setting calls.
        """
        self._ensure_container_spec_exists()

        cpu = self._validate_cpu_request_limit(cpu)

        if self.task.container_spec.resources is not None:
            self.task.container_spec.resources.cpu_request = cpu
        else:
            self.task.container_spec.resources = structures.ResourceSpec(
                cpu_request=cpu)

    def set_cpu_limit(self, cpu: str) -> None:
        """Sets CPU limit (maximum) for the task.

        Args:
            cpu: Maximum CPU requests allowed. This string should be a number
                or a number followed by an "m" to indicate millicores (1/1000).
                For more information, see `Specify a CPU Request and a CPU Limit
                <https://kubernetes.io/docs/tasks/configure-pod-container/assign-cpu-resource/#specify-a-cpu-request-and-a-cpu-limit>`_.

        Returns:
            Self return to allow chained setting calls.
        """
        self._ensure_container_spec_exists()

        cpu = self._validate_cpu_request_limit(cpu)

        if self.task.container_spec.resources is not None:
            self.task.container_spec.resources.cpu_limit = cpu
        else:
            self.task.container_spec.resources = structures.ResourceSpec(
                cpu_limit=cpu)

    def set_accelerator_limit(self, limit: int) -> None:
        """Sets accelerator limit (maximum) for the task. Only applies if
        accelerator type is also set via .set_accelerator_type().

        Args:
            limit: Maximum number of accelerators allowed.

        Returns:
            Self return to allow chained setting calls.
        """
        self._ensure_container_spec_exists()

        if isinstance(limit, str):
            if re.match(r'[1-9]\d*$', limit) is None:
                raise ValueError(f'{"limit"!r} must be positive integer.')
            limit = int(limit)

        if self.task.container_spec.resources is not None:
            self.task.container_spec.resources.accelerator_count = limit
        else:
            self.task.container_spec.resources = structures.ResourceSpec(
                accelerator_count=limit)

    def _validate_memory_request_limit(self, memory: str) -> float:
        """Validates memory request/limit string and converts to its numeric
        value.

        Args:
            memory: Memory requests or limits. This string should be a number or
               a number followed by one of "E", "Ei", "P", "Pi", "T", "Ti", "G",
               "Gi", "M", "Mi", "K", or "Ki".

        Raises:
            ValueError if the memory request/limit string value is invalid.

        Returns:
            The numeric value (float) of the memory request/limit.
        """
        if re.match(r'^[0-9]+(E|Ei|P|Pi|T|Ti|G|Gi|M|Mi|K|Ki){0,1}$',
                    memory) is None:
            raise ValueError(
                'Invalid memory string. Should be a number or a number '
                'followed by one of "E", "Ei", "P", "Pi", "T", "Ti", "G", '
                '"Gi", "M", "Mi", "K", "Ki".')

        if memory.endswith('E'):
            memory = float(memory[:-1]) * constants._E / constants._G
        elif memory.endswith('Ei'):
            memory = float(memory[:-2]) * constants._EI / constants._G
        elif memory.endswith('P'):
            memory = float(memory[:-1]) * constants._P / constants._G
        elif memory.endswith('Pi'):
            memory = float(memory[:-2]) * constants._PI / constants._G
        elif memory.endswith('T'):
            memory = float(memory[:-1]) * constants._T / constants._G
        elif memory.endswith('Ti'):
            memory = float(memory[:-2]) * constants._TI / constants._G
        elif memory.endswith('G'):
            memory = float(memory[:-1])
        elif memory.endswith('Gi'):
            memory = float(memory[:-2]) * constants._GI / constants._G
        elif memory.endswith('M'):
            memory = float(memory[:-1]) * constants._M / constants._G
        elif memory.endswith('Mi'):
            memory = float(memory[:-2]) * constants._MI / constants._G
        elif memory.endswith('K'):
            memory = float(memory[:-1]) * constants._K / constants._G
        elif memory.endswith('Ki'):
            memory = float(memory[:-2]) * constants._KI / constants._G
        else:
            # By default interpret as a plain integer, in the unit of Bytes.
            memory = float(memory) / constants._G

        return memory

    def set_memory_request(self, memory: str) -> None:
        """Sets memory request (minimum) for the task.

        Args:
            memory: The minimum memory requests required. This string should be
                a number or a number followed by one of "E", "Ei", "P", "Pi",
                "T", "Ti", "G", "Gi", "M", "Mi", "K", or "Ki".

        Returns:
            Self return to allow chained setting calls.
        """
        self._ensure_container_spec_exists()

        memory = self._validate_memory_request_limit(memory)

        if self.task.container_spec.resources is not None:
            self.task.container_spec.resources.memory_request = memory
        else:
            self.task.container_spec.resources = structures.ResourceSpec(
                memory_request=memory)

    def set_memory_limit(self, memory: str) -> None:
        """Sets memory limit (maximum) for the task.

        Args:
            memory: The maximum memory requests allowed. This string should be
                a number or a number followed by one of "E", "Ei", "P", "Pi",
                "T", "Ti", "G", "Gi", "M", "Mi", "K", or "Ki".

        Returns:
            Self return to allow chained setting calls.
        """
        self._ensure_container_spec_exists()

        memory = self._validate_memory_request_limit(memory)

        if self.task.container_spec.resources is not None:
            self.task.container_spec.resources.memory_limit = memory
        else:
            self.task.container_spec.resources = structures.ResourceSpec(
                memory_limit=memory)

    def set_retry(self,
                  num_retries: int,
                  backoff_duration: Optional[str] = None,
                  backoff_factor: Optional[float] = None,
                  backoff_max_duration: Optional[str] = None) -> None:
        """Sets task retry parameters.

        Args:
            num_retries : Number of times to retry on failure.
            backoff_duration: Number of seconds to wait before triggering a retry. Defaults to ``'0s'`` (immediate retry).
            backoff_factor: Exponential backoff factor applied to ``backoff_duration``. For example, if ``backoff_duration="60"`` (60 seconds) and ``backoff_factor=2``, the first retry will happen after 60 seconds, then again after 120, 240, and so on. Defaults to ``2.0``.
            backoff_max_duration: Maximum duration during which the task will be retried. Maximum duration is 1 hour (3600s). Defaults to ``'3600s'``.

        Returns:
            Self return to allow chained setting calls.
        """
        self.task._task_spec.retry_policy = structures.RetryPolicy(
            max_retry_count=num_retries,
            backoff_duration=backoff_duration,
            backoff_factor=backoff_factor,
            backoff_max_duration=backoff_max_duration,
        )

    def set_accelerator_type(self, accelerator: str) -> None:
        """Sets accelerator type to use when executing this task.

        Args:
            accelerator: The name of the accelerator, such as ``'NVIDIA_TESLA_K80'``, ``'TPU_V3'``, ``'nvidia.com/gpu'`` or ``'cloud-tpus.google.com/v3'``.

        Returns:
            Self return to allow chained setting calls.
        """
        self._ensure_container_spec_exists()

        if self.task.container_spec.resources is not None:
            self.task.container_spec.resources.accelerator_type = accelerator
            if self.task.container_spec.resources.accelerator_count is None:
                self.task.container_spec.resources.accelerator_count = 1
        else:
            self.task.container_spec.resources = structures.ResourceSpec(
                accelerator_count=1, accelerator_type=accelerator)

    def set_display_name(self, name: str) -> None:
        """Sets display name for the task.

        Args:
            name: Display name.

        Returns:
            Self return to allow chained setting calls.
        """
        self.task._task_spec.display_name = name

    def set_env_variable(self, name: str, value: str) -> None:
        """Sets environment variable for the task.

        Args:
            name: Environment variable name.
            value: Environment variable value.

        Returns:
            Self return to allow chained setting calls.
        """
        self._ensure_container_spec_exists()

        if self.task.container_spec.env is not None:
            self.task.container_spec.env[name] = value
        else:
            self.task.container_spec.env = {name: value}

    def after(self, *tasks) -> None:
        """Specifies an explicit dependency on other tasks by requiring this
        task be executed after other tasks finish completion.

        Args:
            *tasks: Tasks after which this task should be executed.

        Returns:
            Self return to allow chained setting calls.

        Example:
          ::

            @dsl.pipeline(name='my-pipeline')
            def my_pipeline():
                task1 = my_component(text='1st task')
                task2 = my_component(text='2nd task').after(task1)
        """
        for task in tasks:
            self.task._run_after.append(task.name)
            self.task._task_spec.dependent_tasks.append(task.name)

    def ignore_upstream_failure(self) -> None:
        """If called, the pipeline task will run when any specified upstream
        tasks complete, even if unsuccessful.

        This method effectively turns the caller task into an exit task
        if the caller task has upstream dependencies.

        If the task has no upstream tasks, either via data exchange or an explicit dependency via .after(), this method has no effect.

        Returns:
            Self return to allow chained setting calls.

        Example:
          ::

            @dsl.pipeline()
            def my_pipeline(text: str = 'message'):
                task = fail_op(message=text)
                clean_up_task = print_op(
                    message=task.output).ignore_upstream_failure()
        """

        for input_spec_name, input_spec in (self.task.component_spec.inputs or
                                            {}).items():
            if type_utils.is_task_final_status_type(input_spec.type):
                continue
            argument_value = self.task._inputs[input_spec_name]
            if (isinstance(argument_value, pipeline_channel.PipelineChannel)
               ) and (not input_spec.optional) and (argument_value.task_name
                                                    is not None):
                raise ValueError(
                    f'Tasks can only use .ignore_upstream_failure() if all input parameters that accept arguments created by an upstream task have a default value, in case the upstream task fails to produce its output. Input parameter task {self.task.name!r}`s {input_spec_name!r} argument is an output of an upstream task {argument_value.task_name!r}, but {input_spec_name!r} has no default value.'
                )

        self.task._ignore_upstream_failure_tag = True
