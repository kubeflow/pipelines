# Copyright 2018 Google LLC
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

import copy
from collections import OrderedDict
from typing import Mapping
from ._structures import ContainerImplementation, ConcatPlaceholder, IfPlaceholder, InputValuePlaceholder, InputPathPlaceholder, IsPresentPlaceholder, OutputPathPlaceholder, TaskSpec
from ._components import _generate_input_file_name, _generate_output_file_name, _default_component_name

def create_container_op_from_task(task_spec: TaskSpec):
    argument_values = task_spec.arguments
    component_spec = task_spec.component_ref.spec

    if not isinstance(component_spec.implementation, ContainerImplementation):
        raise TypeError('Only container component tasks can be converted to ContainerOp')

    inputs_dict = {input_spec.name: input_spec for input_spec in component_spec.inputs or []}
    container_spec = component_spec.implementation.container

    output_paths = OrderedDict() #Preserving the order to make the kubernetes output names deterministic
    unconfigurable_output_paths = container_spec.file_outputs or {}
    for output in component_spec.outputs or []:
        if output.name in unconfigurable_output_paths:
            output_paths[output.name] = unconfigurable_output_paths[output.name]

    input_paths = OrderedDict()
    artifact_arguments = OrderedDict()

    def expand_command_part(arg): #input values with original names
        #(Union[str,Mapping[str, Any]]) -> Union[str,List[str]]
        if arg is None:
            return None
        if isinstance(arg, (str, int, float, bool)):
            return str(arg)

        if isinstance(arg, InputValuePlaceholder):
            input_name = arg.input_name
            input_value = argument_values.get(input_name, None)
            if input_value is not None:
                return str(input_value)
            else:
                input_spec = inputs_dict[input_name]
                if input_spec.optional:
                    return None
                else:
                    raise ValueError('No value provided for input {}'.format(input_name))

        if isinstance(arg, InputPathPlaceholder):
            input_name = arg.input_name
            input_value = argument_values.get(input_name, None)
            if input_value is not None:
                input_path = _generate_input_file_name(input_name)
                input_paths[input_name] = input_path
                artifact_arguments[input_name] = input_value
                return input_path
            else:
                input_spec = inputs_dict[input_name]
                if input_spec.optional:
                    #Even when we support default values there is no need to check for a default here.
                    #In current execution flow (called by python task factory), the missing argument would be replaced with the default value by python itself.
                    return None
                else:
                    raise ValueError('No value provided for input {}'.format(input_name))

        elif isinstance(arg, OutputPathPlaceholder):
            output_name = arg.output_name
            output_filename = _generate_output_file_name(output_name)
            if arg.output_name in output_paths:
                if output_paths[output_name] != output_filename:
                    raise ValueError('Conflicting output files specified for port {}: {} and {}'.format(output_name, output_paths[output_name], output_filename))
            else:
                output_paths[output_name] = output_filename

            return output_filename

        elif isinstance(arg, ConcatPlaceholder):
            expanded_argument_strings = expand_argument_list(arg.items)
            return ''.join(expanded_argument_strings)

        elif isinstance(arg, IfPlaceholder):
            arg = arg.if_structure
            condition_result = expand_command_part(arg.condition)
            from distutils.util import strtobool
            condition_result_bool = condition_result and strtobool(condition_result) #Python gotcha: bool('False') == True; Need to use strtobool; Also need to handle None and []
            result_node = arg.then_value if condition_result_bool else arg.else_value
            if result_node is None:
                return []
            if isinstance(result_node, list):
                expanded_result = expand_argument_list(result_node)
            else:
                expanded_result = expand_command_part(result_node)
            return expanded_result

        elif isinstance(arg, IsPresentPlaceholder):
            argument_is_present = argument_values.get(arg.input_name, None) is not None
            return str(argument_is_present)
        else:
            raise TypeError('Unrecognized argument type: {}'.format(arg))
    
    def expand_argument_list(argument_list):
        expanded_list = []
        if argument_list is not None:
            for part in argument_list:
                expanded_part = expand_command_part(part)
                if expanded_part is not None:
                    if isinstance(expanded_part, list):
                        expanded_list.extend(expanded_part)
                    else:
                        expanded_list.append(str(expanded_part))
        return expanded_list

    expanded_command = expand_argument_list(container_spec.command)
    expanded_args = expand_argument_list(container_spec.args)

    return _task_object_factory(
        name=component_spec.name or _default_component_name,
        container_image=container_spec.image,
        command=expanded_command,
        arguments=expanded_args,
        input_paths=input_paths,
        output_paths=output_paths,
        artifact_arguments=artifact_arguments,
        env=container_spec.env,
        component_spec=component_spec,
    )


def _create_container_op_from_resolved_task(name:str, container_image:str, command=None, arguments=None, input_paths=None, artifact_arguments=None, output_paths=None, env : Mapping[str, str]=None, component_spec=None):
    from .. import dsl

    #Renaming outputs to conform with ContainerOp/Argo
    from ._naming import _sanitize_python_function_name, generate_unique_name_conversion_table
    output_names = (output_paths or {}).keys()
    output_name_to_kubernetes = generate_unique_name_conversion_table(output_names, _sanitize_python_function_name)
    output_paths_for_container_op = {output_name_to_kubernetes[name]: path for name, path in output_paths.items()}

    task = dsl.ContainerOp(
        name=name,
        image=container_image,
        command=command,
        arguments=arguments,
        file_outputs=output_paths_for_container_op,
        artifact_argument_paths=[dsl.InputArgumentPath(argument=artifact_arguments[input_name], input=input_name, path=path) for input_name, path in input_paths.items()],
    )
    # Fixing ContainerOp output types
    if component_spec.outputs:
        for output in component_spec.outputs:
            pythonic_name = output_name_to_kubernetes[output.name]
            if pythonic_name in task.outputs:
                task.outputs[pythonic_name].param_type = output.type

    component_meta = copy.copy(component_spec)
    component_meta.implementation = None
    task._set_metadata(component_meta)

    if env:
        from kubernetes import client as k8s_client
        for name, value in env.items():
            task.container.add_env_variable(k8s_client.V1EnvVar(name=name, value=value))

    if component_spec.metadata:
        for key, value in (component_spec.metadata.annotations or {}).items():
            task.add_pod_annotation(key, value)
        for key, value in (component_spec.metadata.labels or {}).items():
            task.add_pod_label(key, value)

    return task


_task_object_factory=_create_container_op_from_resolved_task
