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

__all__ = [
    'load_component',
    'load_component_from_text',
    'load_component_from_url',
    'load_component_from_file',
]

import sys
from ._yaml_utils import load_yaml, dump_yaml
from ._structures import ComponentSpec


_default_component_name = 'Component'


def load_component(filename=None, url=None, text=None):
    '''
    Loads component from text, file or URL and creates a task factory function
    
    Only one argument should be specified.

    Args:
        filename: Path of local file containing the component definition.
        url: The URL of the component file data
        text: A string containing the component file data.

    Returns:
        A factory function with a strongly-typed signature.
        Once called with the required arguments, the factory constructs a pipeline task instance (ContainerOp).
    '''
    #This function should be called load_task_factory since it returns a factory function.
    #The real load_component function should produce an object with component properties (e.g. name, description, inputs/outputs).
    #TODO: Change this function to return component spec object but it should be callable to construct tasks.
    non_null_args_count = len([name for name, value in locals().items() if value != None])
    if non_null_args_count != 1:
        raise ValueError('Need to specify exactly one source')
    if filename:
        return load_component_from_file(filename)
    elif url:
        return load_component_from_url(url)
    elif text:
        return load_component_from_text(text)
    else:
        raise ValueError('Need to specify a source')


def load_component_from_url(url):
    '''
    Loads component from URL and creates a task factory function
    
    Args:
        url: The URL of the component file data

    Returns:
        A factory function with a strongly-typed signature.
        Once called with the required arguments, the factory constructs a pipeline task instance (ContainerOp).
    '''
    if url is None:
        raise TypeError
    import requests
    resp = requests.get(url)
    resp.raise_for_status()
    return _create_task_factory_from_component_text(resp.content, url)


def load_component_from_file(filename):
    '''
    Loads component from file and creates a task factory function
    
    Args:
        filename: Path of local file containing the component definition.

    Returns:
        A factory function with a strongly-typed signature.
        Once called with the required arguments, the factory constructs a pipeline task instance (ContainerOp).
    '''
    if filename is None:
        raise TypeError
    with open(filename, 'r') as yaml_file:
        return _create_task_factory_from_component_text(yaml_file, filename)


def load_component_from_text(text):
    '''
    Loads component from text and creates a task factory function
    
    Args:
        text: A string containing the component file data.

    Returns:
        A factory function with a strongly-typed signature.
        Once called with the required arguments, the factory constructs a pipeline task instance (ContainerOp).
    '''
    if text is None:
        raise TypeError
    return _create_task_factory_from_component_text(text, None)


def _create_task_factory_from_component_text(text_or_file, component_filename=None):
    component_dict = load_yaml(text_or_file)
    return _create_task_factory_from_component_dict(component_dict, component_filename)


def _create_task_factory_from_component_dict(component_dict, component_filename=None):
    component_spec = ComponentSpec.from_struct(component_dict)
    return _create_task_factory_from_component_spec(component_spec, component_filename)


def _normalize_identifier_name(name):
    import re
    normalized_name = name.lower()
    normalized_name = re.sub(r'[\W_]', ' ', normalized_name)           #No non-word characters
    normalized_name = re.sub(' +', ' ', normalized_name).strip()    #No double spaces, leading or trailing spaces
    if re.match(r'\d', normalized_name):
        normalized_name = 'n' + normalized_name                     #No leading digits
    return normalized_name


def _sanitize_kubernetes_resource_name(name):
    return _normalize_identifier_name(name).replace(' ', '-')


def _sanitize_python_function_name(name):
    return _normalize_identifier_name(name).replace(' ', '_')


def _sanitize_file_name(name):
    import re
    return re.sub('[^-_.0-9a-zA-Z]+', '_', name)


def _generate_unique_suffix(data):
    import time
    import hashlib
    string_data = str( (data, time.time()) )
    return hashlib.sha256(string_data.encode()).hexdigest()[0:8]

_inputs_dir = '/inputs'
_outputs_dir = '/outputs'


def _generate_input_file_name(port_name):
    return _inputs_dir + '/' + _sanitize_file_name(port_name)


def _generate_output_file_name(port_name):
    return _outputs_dir + '/' + _sanitize_file_name(port_name)


def _try_get_object_by_name(obj_name):
    '''Locates any Python object (type, module, function, global variable) by name'''
    ##Might be heavy since locate searches all Python modules
    #from pydoc import locate
    #return locate(obj_name) or obj_name
    import builtins
    return builtins.__dict__.get(obj_name, obj_name)


def _make_name_unique_by_adding_index(name:str, collection, delimiter:str):
    unique_name = name
    if unique_name in collection:
        for i in range(2, sys.maxsize**10):
            unique_name = name + delimiter + str(i)
            if unique_name not in collection:
                break
    return unique_name


#TODO: Refactor the function to make it shorter
def _create_task_factory_from_component_spec(component_spec:ComponentSpec, component_filename=None):
    name = component_spec.name or _default_component_name
    description = component_spec.description
    
    inputs_list = component_spec.inputs or [] #List[InputSpec]
    outputs_list = component_spec.outputs or [] #List[OutputSpec]

    input_name_to_pythonic = {}
    output_name_to_pythonic = {}
    pythonic_name_to_original = {}

    input_name_to_kubernetes = {}
    output_name_to_kubernetes = {}
    kubernetes_name_to_input_name = {}
    kubernetes_name_to_output_name = {}

    for io_port in inputs_list:
        pythonic_name = _sanitize_python_function_name(io_port.name)
        pythonic_name = _make_name_unique_by_adding_index(pythonic_name, pythonic_name_to_original, '_')
        input_name_to_pythonic[io_port.name] = pythonic_name
        pythonic_name_to_original[pythonic_name] = io_port.name

        kubernetes_name = _sanitize_kubernetes_resource_name(io_port.name)
        kubernetes_name = _make_name_unique_by_adding_index(kubernetes_name, kubernetes_name_to_input_name, '-')
        input_name_to_kubernetes[io_port.name] = kubernetes_name
        kubernetes_name_to_input_name[kubernetes_name] = io_port.name

    for io_port in outputs_list:
        pythonic_name = _sanitize_python_function_name(io_port.name)
        pythonic_name = _make_name_unique_by_adding_index(pythonic_name, pythonic_name_to_original, '_')
        output_name_to_pythonic[io_port.name] = pythonic_name
        pythonic_name_to_original[pythonic_name] = io_port.name

        kubernetes_name = _sanitize_kubernetes_resource_name(io_port.name)
        kubernetes_name = _make_name_unique_by_adding_index(kubernetes_name, kubernetes_name_to_output_name, '-')
        output_name_to_kubernetes[io_port.name] = kubernetes_name
        kubernetes_name_to_output_name[kubernetes_name] = io_port.name

    container_spec = component_spec.implementation.docker_container
    container_image = container_spec.image

    file_inputs={}
    file_outputs_from_def={}
    if container_spec.file_outputs != None:
        file_outputs_from_def = {output_name_to_kubernetes[param]: path for param, path in container_spec.file_outputs.items()}

    def create_container_op_with_expanded_arguments(pythonic_input_argument_values):
        file_outputs = file_outputs_from_def.copy()
        
        def expand_command_part(arg): #input values with original names
            #(Union[str,List],Mapping[str, Any]) -> str
            if isinstance(arg, dict) or isinstance(arg, list):
                if isinstance(arg, list):
                    (func_name, func_argument) = (arg[0], arg[1:])
                    if len(func_argument) == 1:
                        func_argument = func_argument[0]
                elif isinstance(arg, dict):
                    if len(arg) != 1:
                        raise ValueError('Failed to parse argument dict: "{}"'.format(arg))
                    (func_name, func_argument) = list(arg.items())[0]
                else:
                    raise TypeError()
                func_name=func_name.lower()
                if func_name == 'value':
                    assert isinstance(func_argument, str)
                    port_name = func_argument
                    input_value = pythonic_input_argument_values[input_name_to_pythonic[port_name]]
                    return str(input_value)

                elif func_name == 'file':
                    assert isinstance(func_argument, str)
                    port_name = func_argument
                    input_filename = _generate_input_file_name(port_name)
                    input_key = input_name_to_kubernetes[port_name]
                    input_value = pythonic_input_argument_values[input_name_to_pythonic[port_name]]
                    file_inputs[input_key] = {'local_path': input_filename, 'data_source': input_value}
                    return input_filename

                elif func_name == 'output':
                    assert isinstance(func_argument, str)
                    port_name = func_argument
                    pythonic_port_name = output_name_to_pythonic[port_name]
                    if pythonic_port_name in pythonic_input_argument_values and pythonic_input_argument_values[pythonic_port_name] is not None:
                        output_filename = str(pythonic_input_argument_values[pythonic_port_name])
                    else:
                        output_filename = _generate_output_file_name(port_name)
                    #We need to pass the file mapping to file_outputs
                    output_key = output_name_to_kubernetes[port_name]
                    if output_key in file_outputs:
                        if file_outputs[output_key] != output_filename:
                            raise ValueError('Conflicting output files specified for port {}: {} and {}'.format(port_name, file_outputs[output_key], output_filename))
                    else:
                        file_outputs[output_key] = output_filename
                    
                    return output_filename

                elif func_name == 'concat':
                    assert isinstance(func_argument, list)
                    items_to_concatenate = func_argument
                    expanded_arguments = [expand_command_part(arg1) for arg1 in items_to_concatenate]
                    expanded_argument_strings = [str(arg2) for arg2 in expanded_arguments if arg2 is not None]
                    return ''.join(expanded_argument_strings)
                
                elif func_name == 'if':
                    if isinstance(func_argument, dict):
                        condition_node = func_argument['cond']
                        then_node = func_argument['then']
                        else_node = func_argument.get('else', None)
                    elif isinstance(func_argument, list):
                        assert len(func_argument) in [2, 3]
                        condition_node = func_argument[0]
                        then_node = func_argument[1]
                        else_node = func_argument[2] if len(func_argument) == 3 else None
                    else:
                        raise TypeError()
                    condition_result = bool(expand_command_part(condition_node))
                    result_node = then_node if condition_result else else_node
                    if isinstance(result_node, list):
                        expanded_result = [expand_command_part(arg1) for arg1 in result_node]
                    else:
                        expanded_result = expand_command_part(result_node)
                    return expanded_result

                elif func_name == 'ispresent':
                    assert isinstance(func_argument, str)
                    input_name = func_argument
                    pythonic_input_name = input_name_to_pythonic[input_name]
                    argument_is_present = pythonic_input_name in pythonic_input_argument_values
                    return str(argument_is_present)
            else:
                return arg
        
        expanded_command = []
        if container_spec.command != None:
            for part in container_spec.command:
                expanded_part = expand_command_part(part)
                if expanded_part is not None:
                    if isinstance(expanded_part, list):
                        expanded_command.extend(expanded_part)
                    else:
                        expanded_command.append(str(expanded_part))

        expanded_args = []
        if container_spec.arguments != None:
            for part in container_spec.arguments:
                expanded_part = expand_command_part(part)
                if expanded_part is not None:
                    if isinstance(expanded_part, list):
                        expanded_args.extend(expanded_part)
                    else:
                        expanded_args.append(str(expanded_part))

        #Working around Python's variable scoping. Do not write to variable from global scope as that makes the variable local.

        file_outputs_to_pass = file_outputs 
        if file_outputs_to_pass == {}:
            file_outputs_to_pass = None
        
        from . import _dsl_bridge
        return _dsl_bridge._task_object_factory(
            name=name,
            container_image=container_image,
            command=expanded_command,
            arguments=expanded_args,
            file_inputs=file_inputs,
            file_outputs=file_outputs_to_pass,
        )

    import inspect
    from . import _dynamic
    
    #Still allowing to set the output parameters, but make them optional and auto-generate if missing.
    input_parameters  = [_dynamic.KwParameter(input_name_to_pythonic[port.name], annotation=(_try_get_object_by_name(port.type) if port.type else inspect.Parameter.empty)) for port in inputs_list]
    output_parameters = [_dynamic.KwParameter(output_name_to_pythonic[port.name], annotation=('OutputFile[{}]'.format(port.type) if port.type else inspect.Parameter.empty), default=None) for port in outputs_list]

    factory_function_parameters = input_parameters + output_parameters
    
    return _dynamic.create_function_from_parameters(
        create_container_op_with_expanded_arguments,        
        factory_function_parameters,
        documentation=description,
        func_name=name,
        func_filename=component_filename
    )
