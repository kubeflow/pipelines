# Copyright 2021 The Kubeflow Authors
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

import re
import copy
from typing import Any, List, Mapping, Optional, Union

from kfp.dsl import _component_bridge
from kfp.v2.components.experimental import constants
from kfp.v2.components.experimental import pipeline_channel
from kfp.v2.components.experimental import placeholders
from kfp.v2.components.experimental import structures
from kfp.v2.components.types import type_utils


# TODO(chensun): return PipelineTask object instead of ContainerOp object.
def create_pipeline_task(
    component_spec: structures.ComponentSpec,
    arguments: Mapping[str, Any],
) -> "ContainerOp":  # pytype: disable=name-error
    return _component_bridge._create_container_op_from_component_and_arguments(
        component_spec=component_spec.to_v1_component_spec(),
        arguments=arguments,
    )


class PipelineTask:
    """Represents a pipeline task -- an instantiated component.

    Replaces `ContainerOp`. Holds operations available on a task object, such as
    `.after()`, `.set_memory_limit()`, `enable_caching()`, etc.

    Attributes:
        task_spec: The task spec of the task.
        component_spec: The component spec of the task.
        container_spec: The resolved container spec of the task.
    """

    def __init__(
        self,
        component_spec: structures.ComponentSpec,
        arguments: Mapping[str, Any],
    ):
        """Initilizes a PipelineTask instance.

        Args:
            component_spec: The component definition.
            arguments: The dictionary of component arguments.
        """
        for input_name, argument_value in arguments.items():

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
            elif isinstance(argument_value, int):
                argument_type = 'Integer'
            elif isinstance(argument_value, float):
                argument_type = 'Float'
            elif isinstance(argument_value, bool):
                argument_type = 'Boolean'
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

        self.task_spec = structures.TaskSpec(
            # The name of the task is subject to change due to component reuse.
            name=component_spec.name,
            inputs={
                input_name: value for input_name, value in arguments.items()
            },
            dependent_tasks=[],
            component_ref=component_spec.name,
            enable_caching=True,
        )

        self.container_spec = self._resolve_command_line_and_arguments(
            component_spec=component_spec,
            arguments=arguments,
        )

    def _resolve_command_line_and_arguments(
        self,
        component_spec: structures.ComponentSpec,
        arguments: Mapping[str, str],
    ) -> structures.ContainerSpec:
        """Resolves the command line argument placeholders in a container spec.

        Args:
            component_spec: The component definition.
            arguments: The dictionary of component arguments.
        """
        argument_values = arguments

        if not component_spec.implementation.container:
            raise TypeError(
                'Only container components have command line to resolve')

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

        def expand_command_part(arg) -> Union[str, List[str], None]:
            if arg is None:
                return None

            if isinstance(arg, (str, int, float, bool)):
                return str(arg)

            elif isinstance(arg, (dict, list)):
                return json.dumps(arg)

            elif isinstance(arg, structures.InputValuePlaceholder):
                input_name = arg.input_name
                if not type_utils.is_parameter_type(
                        inputs_dict[input_name].type):
                    raise TypeError(
                        f'Input "{input_name}" with type '
                        f'"{inputs_dict[input_name].type}" cannot be paired with '
                        'InputValuePlaceholder.')

                if input_name in arguments:
                    return placeholders.input_parameter_placeholder(input_name)
                else:
                    input_spec = inputs_dict[input_name]
                    if input_spec.default is not None:
                        return None
                    else:
                        raise ValueError(
                            f'No value provided for input: {input_name}.')

            elif isinstance(arg, structures.InputUriPlaceholder):
                input_name = arg.input_name
                if type_utils.is_parameter_type(inputs_dict[input_name].type):
                    raise TypeError(
                        f'Input "{input_name}" with type '
                        f'"{inputs_dict[input_name].type}" cannot be paired with '
                        'InputUriPlaceholder.')

                if input_name in arguments:
                    input_uri = placeholders.input_artifact_uri_placeholder(
                        input_name)
                    return input_uri
                else:
                    input_spec = inputs_dict[input_name]
                    if input_spec.default is not None:
                        return None
                    else:
                        raise ValueError(
                            f'No value provided for input: {input_name}.')

            elif isinstance(arg, structures.InputPathPlaceholder):
                input_name = arg.input_name
                if type_utils.is_parameter_type(inputs_dict[input_name].type):
                    raise TypeError(
                        f'Input "{input_name}" with type '
                        f'"{inputs_dict[input_name].type}" cannot be paired with '
                        'InputPathPlaceholder.')

                if input_name in arguments:
                    input_path = placeholders.input_artifact_path_placeholder(
                        input_name)
                    return input_path
                else:
                    input_spec = inputs_dict[input_name]
                    if input_spec.optional:
                        return None
                    else:
                        raise ValueError(
                            f'No value provided for input: {input_name}.')

            elif isinstance(arg, structures.OutputUriPlaceholder):
                output_name = arg.output_name
                if type_utils.is_parameter_type(outputs_dict[output_name].type):
                    raise TypeError(
                        f'Onput "{output_name}" with type '
                        f'"{outputs_dict[output_name].type}" cannot be paired with '
                        'OutputUriPlaceholder.')

                output_uri = placeholders.output_artifact_uri_placeholder(
                    output_name)
                return output_uri

            elif isinstance(arg, structures.OutputPathPlaceholder):
                output_name = arg.output_name

                if type_utils.is_parameter_type(outputs_dict[output_name].type):
                    output_path = placeholders.output_parameter_path_placeholder(
                        output_name)
                else:
                    output_path = placeholders.output_artifact_path_placeholder(
                        output_name)
                return output_path

            elif isinstance(arg, structures.ConcatPlaceholder):
                expanded_argument_strings = expand_argument_list(arg.items)
                return ''.join(expanded_argument_strings)

            elif isinstance(arg, structures.IfPresentPlaceholder):
                if arg.if_structure.input_name in argument_values:
                    result_node = arg.if_structure.then
                else:
                    result_node = arg.if_structure.otherwise

                if result_node is None:
                    return []

                if isinstance(result_node, list):
                    expanded_result = expand_argument_list(result_node)
                else:
                    expanded_result = expand_command_part(result_node)
                return expanded_result

            else:
                raise TypeError('Unrecognized argument type: {}'.format(arg))

        def expand_argument_list(argument_list) -> Optional[List[str]]:
            if argument_list is None:
                return None

            expanded_list = []
            for part in argument_list:
                expanded_part = expand_command_part(part)
                if expanded_part is not None:
                    if isinstance(expanded_part, list):
                        expanded_list.extend(expanded_part)
                    else:
                        expanded_list.append(str(expanded_part))
            return expanded_list

        container_spec = component_spec.implementation.container
        resolved_container_spec = copy.deepcopy(container_spec)

        resolved_container_spec.commands = expand_argument_list(
            container_spec.commands)
        resolved_container_spec.arguments = expand_argument_list(
            container_spec.arguments)

        return resolved_container_spec

    def set_caching_options(self, enable_caching: bool) -> 'PipelineTask':
        """Sets caching options for the Pipeline task.

        Args:
            enable_caching: Whether or not to enable caching for this task.

        Returns:
            Self return to allow chained setting calls.
        """
        self.task_spec.enable_caching = enable_caching
        return self

    def set_cpu_limit(self, cpu: str) -> 'PipelineTask':
        """Set cpu limit (maximum) for this operator.

        Args:
            cpu(str): A string which can be a
                number or a number followed by "m", whichmeans 1/1000.

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

        if self.component_spec.implementation.container is not None:
            if self.component_spec.implementation.container.resources is not None:
                self.component_spec.implementation.container.resources.cpu_limit = cpu
            else:
                self.component_spec.implementation.container.resources = structures.ResourceSpec(
                    cpu_limit=cpu)
        else:
            raise ValueError(
                'There is no container specified in implementation')

        return self

    def set_gpu_limit(self, gpu: str) -> 'PipelineTask':
        """Set gpu limit (maximum) for this operator.

        Args:
            gpu(str): Positive number required for number of GPUs.

        Returns:
            Self return to allow chained setting calls.
        """
        if re.match(r'[1-9]\d*$', gpu) is None:
            raise ValueError('GPU must be positive integer.')

        gpu = int(gpu)

        if self.component_spec.implementation.container is not None:
            if self.component_spec.implementation.container.resources is not None:
                self.component_spec.implementation.container.resources.accelerator_count = gpu
            else:
                self.component_spec.implementation.container.resources = structures.ResourceSpec(
                    accelerator_count=gpu)
        else:
            raise ValueError(
                'There is no container specified in implementation')
        return self

    def set_memory_limit(self, memory: str) -> 'PipelineTask':
        """Set memory limit (maximum) for this operator.

        Args:
            memory(str): a string which can be a number or a number followed by
                one of "E", "Ei", "P", "Pi", "T", "Ti", "G", "Gi", "M", "Mi",
                "K", "Ki".

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

        if self.component_spec.implementation.container is not None:
            if self.component_spec.implementation.container.resources is not None:
                self.component_spec.implementation.container.resources.memory_limit = memory
            else:
                self.component_spec.implementation.container.resources = structures.ResourceSpec(
                    memory_limit=memory)
        else:
            raise ValueError(
                'There is no container specified in implementation')

        return self

    def add_node_selector_constraint(self, accelerator: str) -> 'PipelineTask':
        """Sets accelerator type requirement for this task.

        Args:
            value(str): The name of the accelerator. Available values include
                'NVIDIA_TESLA_K80', 'TPU_V3'.

        Returns:
            Self return to allow chained setting calls.
        """
        if self.component_spec.implementation.container is not None:
            if self.component_spec.implementation.container.resources is not None:
                self.component_spec.implementation.container.resources.accelerator_type = accelerator
                if self.component_spec.implementation.container.resources.accelerator_count is None:
                    self.component_spec.implementation.container.resources.accelerator_count = 1
            else:
                self.component_spec.implementation.container.resources = structures.ResourceSpec(
                    accelerator_count=1, accelerator_type=accelerator)
        else:
            raise ValueError(
                'There is no container specified in implementation')

        return self

    def set_display_name(self, name: str) -> 'PipelineTask':
        """Set display name for the pipelineTask.

        Args:
            name(str): display name for the task.

        Returns:
            Self return to allow chained setting calls.
        """
        self.task_spec.name = name
        return self

    def after(self, *tasks) -> 'PipelineTask':
        """Specify explicit dependency on other tasks.

        Args:
            name(tasks): dependent tasks.

        Returns:
            Self return to allow chained setting calls.
        """
        for task in tasks:
            self.task_spec.dependent_tasks.append(task.name)
        return self
