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

import collections
import copy
from typing import Any, List, Mapping, Optional, Union

from kfp.dsl import _component_bridge
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
