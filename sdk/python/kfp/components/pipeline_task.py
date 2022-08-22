# Copyright 2021-2022 The Kubeflow Authors
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
"""Pipeline task class and operations."""

import copy
import re
from typing import Any, List, Mapping, Optional, Union

from kfp.components import constants
from kfp.components import pipeline_channel
from kfp.components import placeholders
from kfp.components import structures
from kfp.components import utils
from kfp.components.types import type_utils


def create_pipeline_task(
    component_spec: structures.ComponentSpec,
    args: Mapping[str, Any],
) -> 'PipelineTask':  # pytype: disable=name-error
    return PipelineTask(component_spec=component_spec, args=args)


class PipelineTask:
    """Represents a pipeline task (instantiated component).

    **Note:** ``PipelineTask`` should not be constructed by pipeline authors directly, but instead obtained via an instantiated component (see example).

    Replaces ``ContainerOp`` from ``kfp`` v1. Holds operations available on a task object, such as
    ``.after()``, ``.set_memory_limit()``, ``.enable_caching()``, etc.

    Args:
        component_spec: The component definition.
        args: The dictionary of arguments on which the component was called to instantiate this task.

    Example:
      ::

        @dsl.component
        def identity(message: str) -> str:
            return message

        @dsl.pipeline(name='my_pipeline')
        def my_pipeline():
            # task is an instance of PipelineTask
            task = identity(message='my string')
    """

    # Fallback behavior for compiling a component. This should be overriden by
    # pipeline `register_task_and_generate_id` if compiling a pipeline (more
    # than one component).
    _register_task_handler = lambda task: utils.maybe_rename_for_k8s(
        task.component_spec.name)

    def __init__(
        self,
        component_spec: structures.ComponentSpec,
        args: Mapping[str, Any],
    ):
        """Initilizes a PipelineTask instance."""
        # import within __init__ to avoid circular import
        from kfp.components.tasks_group import TasksGroup

        self.parent_task_group: Union[None, TasksGroup] = None
        args = args or {}

        for input_name, argument_value in args.items():

            if input_name not in component_spec.inputs:
                raise ValueError(
                    f'Component "{component_spec.name}" got an unexpected input:'
                    f' {input_name}.')

            input_type = component_spec.inputs[input_name].type
            argument_type = None

            if isinstance(argument_value, pipeline_channel.PipelineChannel):
                argument_type = argument_value.channel_type
            elif isinstance(argument_value, str):
                argument_type = 'String'
            elif isinstance(argument_value, bool):
                argument_type = 'Boolean'
            elif isinstance(argument_value, int):
                argument_type = 'Integer'
            elif isinstance(argument_value, float):
                argument_type = 'Float'
            elif isinstance(argument_value, dict):
                argument_type = 'Dict'
            elif isinstance(argument_value, list):
                argument_type = 'List'
            else:
                raise ValueError(
                    'Input argument supports only the following types: '
                    'str, int, float, bool, dict, and list. Got: '
                    f'"{argument_value}" of type "{type(argument_value)}".')

            type_utils.verify_type_compatibility(
                given_type=argument_type,
                expected_type=input_type,
                error_message_prefix=(
                    'Incompatible argument passed to the input '
                    f'"{input_name}" of component "{component_spec.name}": '),
            )

        self.component_spec = component_spec

        self._task_spec = structures.TaskSpec(
            name=self._register_task_handler(),
            inputs={input_name: value for input_name, value in args.items()},
            dependent_tasks=[],
            component_ref=component_spec.name,
            enable_caching=True,
        )

        self.importer_spec = None
        self.container_spec = None

        if component_spec.implementation.container is not None:

            self.container_spec = self._resolve_command_line_and_arguments(
                component_spec=component_spec,
                args=args,
            )
        elif component_spec.implementation.importer is not None:
            self.importer_spec = component_spec.implementation.importer
            self.importer_spec.artifact_uri = args['uri']

        self._outputs = {
            output_name: pipeline_channel.create_pipeline_channel(
                name=output_name,
                channel_type=output_spec.type,
                task_name=self._task_spec.name,
            ) for output_name, output_spec in (
                component_spec.outputs or {}).items()
        }

        self._inputs = args

        self._channel_inputs = [
            value for _, value in args.items()
            if isinstance(value, pipeline_channel.PipelineChannel)
        ] + pipeline_channel.extract_pipeline_channels_from_any([
            value for _, value in args.items()
            if not isinstance(value, pipeline_channel.PipelineChannel)
        ])

    @property
    def name(self) -> str:
        """The name of the task.

        Unique within its parent group.
        """
        return self._task_spec.name

    @property
    def inputs(
        self
    ) -> List[Union[type_utils.PARAMETER_TYPES,
                    pipeline_channel.PipelineChannel]]:
        """The list of actual inputs passed to the task."""
        return self._inputs

    @property
    def channel_inputs(self) -> List[pipeline_channel.PipelineChannel]:
        """The list of all channel inputs passed to the task.

        :meta private:
        """
        return self._channel_inputs

    @property
    def output(self) -> pipeline_channel.PipelineChannel:
        """The single output of the task.

        Used when a task has exactly one output parameter.
        """
        if len(self._outputs) != 1:
            raise AttributeError
        return list(self._outputs.values())[0]

    @property
    def outputs(self) -> Mapping[str, pipeline_channel.PipelineChannel]:
        """The dictionary of outputs of the task.

        Used when a task has more the one output or uses an
        ``OutputPath`` or ``Output[Artifact]`` type annotation.
        """
        return self._outputs

    @property
    def dependent_tasks(self) -> List[str]:
        """A list of the dependent task names."""
        return self._task_spec.dependent_tasks

    def _resolve_command_line_and_arguments(
        self,
        component_spec: structures.ComponentSpec,
        args: Mapping[str, str],
    ) -> structures.ContainerSpecImplementation:
        """Resolves the command line argument placeholders in a
        ContainerSpecImplementation.

        Args:
            component_spec: The component definition.
            args: The dictionary of component arguments.
        """
        argument_values = args

        component_inputs = component_spec.inputs or {}
        inputs_dict = {
            input_name: input_spec
            for input_name, input_spec in component_inputs.items()
        }
        component_outputs = component_spec.outputs or {}
        outputs_dict = {
            output_name: output_spec
            for output_name, output_spec in component_outputs.items()
        }

        def check_input_type_and_convert_to_placeholder(
                arg) -> Union[str, List[str], None]:
            # TODO: separate input type-checking logic from placeholder-conversion/.to_placeholder() logic
            if arg is None:
                return None

            elif isinstance(arg, (str, int, float, bool)):
                return str(arg)

            elif isinstance(arg, placeholders.InputValuePlaceholder):
                input_name = arg.input_name
                if not type_utils.is_parameter_type(
                        inputs_dict[input_name].type):
                    raise TypeError(
                        f'Input "{input_name}" with type '
                        f'"{inputs_dict[input_name].type}" cannot be paired with '
                        'InputValuePlaceholder.')

                if input_name in args or type_utils.is_task_final_status_type(
                        inputs_dict[input_name].type):
                    return arg.to_placeholder_string()

                input_spec = inputs_dict[input_name]
                if input_spec.default is None:
                    raise ValueError(
                        f'No value provided for input: {input_name}.')

                else:
                    return None

            elif isinstance(arg, placeholders.InputUriPlaceholder):
                input_name = arg.input_name
                if type_utils.is_parameter_type(inputs_dict[input_name].type):
                    raise TypeError(
                        f'Input "{input_name}" with type '
                        f'"{inputs_dict[input_name].type}" cannot be paired with '
                        'InputUriPlaceholder.')

                if input_name in args:
                    return arg.to_placeholder_string()
                input_spec = inputs_dict[input_name]
                if input_spec.default is None:
                    raise ValueError(
                        f'No value provided for input: {input_name}.')

                else:
                    return None

            elif isinstance(arg, placeholders.InputPathPlaceholder):
                input_name = arg.input_name
                if type_utils.is_parameter_type(inputs_dict[input_name].type):
                    raise TypeError(
                        f'Input "{input_name}" with type '
                        f'"{inputs_dict[input_name].type}" cannot be paired with '
                        'InputPathPlaceholder.')

                if input_name in args:
                    return arg.to_placeholder_string()
                input_spec = inputs_dict[input_name]
                if input_spec._optional:
                    return None
                else:
                    raise ValueError(
                        f'No value provided for input: {input_name}.')

            elif isinstance(arg, placeholders.OutputUriPlaceholder):
                output_name = arg.output_name
                if type_utils.is_parameter_type(outputs_dict[output_name].type):
                    raise TypeError(
                        f'Onput "{output_name}" with type '
                        f'"{outputs_dict[output_name].type}" cannot be paired with '
                        'OutputUriPlaceholder.')

                return arg.to_placeholder_string()

            elif isinstance(arg, (placeholders.OutputPathPlaceholder,
                                  placeholders.OutputParameterPlaceholder)):
                output_name = arg.output_name
                return placeholders.OutputParameterPlaceholder(
                    arg.output_name).to_placeholder_string(
                    ) if type_utils.is_parameter_type(
                        outputs_dict[output_name].type
                    ) else placeholders.OutputPathPlaceholder(
                        arg.output_name).to_placeholder_string()

            elif isinstance(arg, placeholders.Placeholder):
                return arg.to_placeholder_string()

            else:
                raise TypeError(f'Unrecognized argument type: {arg}.')

        def expand_argument_list(argument_list) -> Optional[List[str]]:
            if argument_list is None:
                return None

            expanded_list = []
            for part in argument_list:
                expanded_part = check_input_type_and_convert_to_placeholder(
                    part)
                if expanded_part is not None:
                    if isinstance(expanded_part, list):
                        expanded_list.extend(expanded_part)
                    else:
                        expanded_list.append(str(expanded_part))
            return expanded_list

        container_spec = component_spec.implementation.container

        resolved_container_spec = copy.deepcopy(container_spec)
        resolved_container_spec.command = expand_argument_list(
            container_spec.command)
        resolved_container_spec.args = expand_argument_list(container_spec.args)

        return resolved_container_spec

    def set_caching_options(self, enable_caching: bool) -> 'PipelineTask':
        """Sets caching options for the task.

        Args:
            enable_caching: Whether to enable caching.

        Returns:
            Self return to allow chained setting calls.
        """
        self._task_spec.enable_caching = enable_caching
        return self

    def set_cpu_limit(self, cpu: str) -> 'PipelineTask':
        """Sets CPU limit (maximum) for the task.

        Args:
            cpu: Maximum CPU requests allowed. This string should be a number or a number followed by an "m" to indicate millicores (1/1000). For more information, see `Specify a CPU Request and a CPU Limit <https://kubernetes.io/docs/tasks/configure-pod-container/assign-cpu-resource/#specify-a-cpu-request-and-a-cpu-limit>`_.

        Returns:
            Self return to allow chained setting calls.
        """
        if re.match(r'([0-9]*[.])?[0-9]+m?$', cpu) is None:
            raise ValueError(
                'Invalid cpu string. Should be float or integer, or integer'
                ' followed by "m".')

        if cpu.endswith('m'):
            cpu = float(cpu[:-1]) / 1000
        else:
            cpu = float(cpu)

        if self.container_spec is None:
            raise ValueError(
                'There is no container specified in implementation')

        if self.container_spec.resources is not None:
            self.container_spec.resources.cpu_limit = cpu
        else:
            self.container_spec.resources = structures.ResourceSpec(
                cpu_limit=cpu)

        return self

    def set_gpu_limit(self, gpu: str) -> 'PipelineTask':
        """Sets GPU limit (maximum) for the task.

        Args:
            gpu: The maximum GPU reuqests allowed. This string should be a positive integer number of GPUs.

        Returns:
            Self return to allow chained setting calls.
        """
        if re.match(r'[1-9]\d*$', gpu) is None:
            raise ValueError('GPU must be positive integer.')

        gpu = int(gpu)

        if self.container_spec is None:
            raise ValueError(
                'There is no container specified in implementation')

        if self.container_spec.resources is not None:
            self.container_spec.resources.accelerator_count = gpu
        else:
            self.container_spec.resources = structures.ResourceSpec(
                accelerator_count=gpu)

        return self

    def set_memory_limit(self, memory: str) -> 'PipelineTask':
        """Sets memory limit (maximum) for the task.

        Args:
            memory: The maximum memory requests allowed. This string should be a number or a number followed by one of "E", "Ei", "P", "Pi", "T", "Ti", "G", "Gi", "M", "Mi", "K", or "Ki".

        Returns:
            Self return to allow chained setting calls.
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

        if self.container_spec is None:
            raise ValueError(
                'There is no container specified in implementation')

        if self.container_spec.resources is not None:
            self.container_spec.resources.memory_limit = memory
        else:
            self.container_spec.resources = structures.ResourceSpec(
                memory_limit=memory)

        return self

    def set_retry(self,
                  num_retries: int,
                  backoff_duration: Optional[str] = None,
                  backoff_factor: Optional[float] = None,
                  backoff_max_duration: Optional[str] = None) -> 'PipelineTask':
        """Sets task retry parameters.

        Args:
            num_retries : Number of times to retry on failure.
            backoff_duration: Number of seconds to wait before triggering a retry. Defaults to ``'0s'`` (immediate retry).
            backoff_factor: Exponential backoff factor applied to ``backoff_duration``. For example, if ``backoff_duration="60"`` (60 seconds) and ``backoff_factor=2``, the first retry will happen after 60 seconds, then again after 120, 240, and so on. Defaults to ``2.0``.
            backoff_max_duration: Maximum duration during which the task will be retried. Maximum duration is 1 hour (3600s). Defaults to ``'3600s'``.

        Returns:
            Self return to allow chained setting calls.
        """
        self._task_spec.retry_policy = structures.RetryPolicy(
            max_retry_count=num_retries,
            backoff_duration=backoff_duration,
            backoff_factor=backoff_factor,
            backoff_max_duration=backoff_max_duration,
        )
        return self

    def add_node_selector_constraint(self, accelerator: str) -> 'PipelineTask':
        """Sets accelerator type to use when executing this task.

        Args:
            value: The name of the accelerator. Available values include
                ``'NVIDIA_TESLA_K80'`` and ``'TPU_V3'``.

        Returns:
            Self return to allow chained setting calls.
        """
        if self.container_spec is None:
            raise ValueError(
                'There is no container specified in implementation')

        if self.container_spec.resources is not None:
            self.container_spec.resources.accelerator_type = accelerator
            if self.container_spec.resources.accelerator_count is None:
                self.container_spec.resources.accelerator_count = 1
        else:
            self.container_spec.resources = structures.ResourceSpec(
                accelerator_count=1, accelerator_type=accelerator)

        return self

    def set_display_name(self, name: str) -> 'PipelineTask':
        """Sets display name for the task.

        Args:
            name: Display name.

        Returns:
            Self return to allow chained setting calls.
        """
        self._task_spec.display_name = name
        return self

    def set_env_variable(self, name: str, value: str) -> 'PipelineTask':
        """Sets environment variable for the task.

        Args:
            name: Environment variable name.
            value: Environment variable value.

        Returns:
            Self return to allow chained setting calls.
        """
        if self.container_spec.env is not None:
            self.container_spec.env[name] = value
        else:
            self.container_spec.env = {name: value}
        return self

    def after(self, *tasks) -> 'PipelineTask':
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
            if task.parent_task_group is not self.parent_task_group:
                raise ValueError(
                    f'Cannot use .after() across inner pipelines or DSL control flow features. Tried to set {self.name} after {task.name}, but these tasks do not belong to the same pipeline or are not enclosed in the same control flow content manager.'
                )
            self._task_spec.dependent_tasks.append(task.name)
        return self
