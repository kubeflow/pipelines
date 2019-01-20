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

from collections import OrderedDict
from typing import Any, List, Mapping, NamedTuple
from ._structures import ConcatPlaceholder, IfPlaceholder, InputValuePlaceholder, InputPathPlaceholder, IsPresentPlaceholder, OutputPathPlaceholder, TaskSpec
from ._components import _generate_input_file_name, _generate_output_file_name, _default_component_name
from kfp.dsl._metadata import ComponentMeta, ParameterMeta, TypeMeta, _annotation_to_typemeta

ResolvedContainerTask = NamedTuple(
    'ResolvedContainerTask', [
        ('component_name', str),                        #Name of the component
        ('container_image', str),                       #Docker container image name
        ('command', List[str]),                         #Command-line
        ('arguments', str),                             #Command-line arguments
        ('env', str),                                   #Environment variable values
        ('input_paths', Mapping[str, str]),             #Mapping between input names and local paths where the data will be put for use by the containerized program
        ('input_path_arguments', Mapping[str, Any]),    #Whatever was passed as argument to the artifact inputs of the component
        ('output_paths', Mapping[str, str]),            #Mapping between input names and local paths where the data will be put for use by the containerized program
        ('component_spec', TaskSpec),                   #Original ComponentSpec instance
        ('task', TaskSpec),                             #Original TaskSpec instance
    ]
)


#Holds the stack of transformation functions. The last function is called each time ResolvedContainerTask instance is created from TaskSpec.
_created_resolved_container_task_transformation_handler = []


def resolve_container_task(task_spec: TaskSpec):
    argument_values = task_spec.arguments
    component_spec = task_spec.component_ref._component_spec

    if hasattr(component_spec.implementation, 'graph'):
        raise TypeError('Cannot convert graph component to ContainerOp')

    inputs_dict = {input_spec.name: input_spec for input_spec in component_spec.inputs or []}
    container_spec = component_spec.implementation.container

    #Preserving the order to make the kubernetes output names deterministic
    input_paths = OrderedDict()
    input_path_arguments = OrderedDict()
    output_paths = OrderedDict()
    unconfigurable_output_paths = container_spec.file_outputs or {}
    for output in component_spec.outputs or []:
        if output.name in unconfigurable_output_paths:
            output_paths[output.name] = unconfigurable_output_paths[output.name]

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
            input_path = _generate_input_file_name(input_name)
            input_paths[input_name] = input_path
            argument_value = argument_values.get(input_name, None)
            if argument_value is not None:
                input_path_arguments[input_name] = argument_value
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

    resolved_task = ResolvedContainerTask(
        component_name=component_spec.name or _default_component_name,
        container_image=container_spec.image,
        command=expanded_command,
        arguments=expanded_args,
        input_paths=input_paths,
        input_path_arguments=input_path_arguments,
        output_paths=output_paths,
        env=container_spec.env,
        component_spec=component_spec,
        task=task_spec,
    )

    #Applying optional transformation to the newly created ResolvedContainerTask
    if _created_resolved_container_task_transformation_handler:
        resolved_task = _created_resolved_container_task_transformation_handler[-1](resolved_task)

    return resolved_task


_dummy_pipeline=None

def _create_container_op_from_resolved_task(resolved_task: ResolvedContainerTask):
    from .. import dsl
    global _dummy_pipeline
    need_dummy = dsl.Pipeline._default_pipeline is None
    if need_dummy:
        if _dummy_pipeline == None:
            _dummy_pipeline = dsl.Pipeline('dummy pipeline')
        _dummy_pipeline.__enter__()
    
    if resolved_task.input_paths:
        raise ValueError('ContainerOp does not support input artifacts "{}"'.format(resolved_task.input_paths))

    #Renaming outputs to conform with ContainerOp/Argo
    from ._naming import _sanitize_python_function_name, generate_unique_name_conversion_table
    output_names = (resolved_task.output_paths or {}).keys()
    output_name_to_kubernetes = generate_unique_name_conversion_table(output_names, _sanitize_python_function_name)
    output_paths_for_container_op = {output_name_to_kubernetes[name]: path for name, path in resolved_task.output_paths.items()}


    # Construct the ComponentMeta
    component_meta = ComponentMeta(name=resolved_task.component_spec.name, description=resolved_task.component_spec.description)
    # Inputs
    if resolved_task.component_spec.inputs is not None:
        for input in resolved_task.component_spec.inputs:
            component_meta.inputs.append(ParameterMeta(name=input.name, description=input.description, param_type=_annotation_to_typemeta(input.type), default=input.default))
    if resolved_task.component_spec.outputs is not None:
        for output in resolved_task.component_spec.outputs:
            component_meta.outputs.append(ParameterMeta(name=output.name, description=output.description, param_type=_annotation_to_typemeta(output.type)))

    task = dsl.ContainerOp(
        name=resolved_task.component_name,
        image=resolved_task.container_image,
        command=resolved_task.command,
        arguments=resolved_task.arguments,
        file_outputs=output_paths_for_container_op,
    )

    task._set_metadata(component_meta)

    if resolved_task.env:
        from kubernetes import client as k8s_client
        for name, value in resolved_task.env.items():
            task.container.add_env_variable(k8s_client.V1EnvVar(name=name, value=value))
  
    if need_dummy:
        _dummy_pipeline.__exit__()

    return task
