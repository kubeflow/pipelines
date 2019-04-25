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
from collections import OrderedDict
from ._naming import _sanitize_file_name, _sanitize_python_function_name, generate_unique_name_conversion_table
from ._yaml_utils import load_yaml
from ._structures import ComponentSpec
from ._structures import *
from kfp.dsl import PipelineParam
from kfp.dsl.types import InconsistentTypeException, check_types
import kfp

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

    #Handling Google Cloud Storage URIs
    if url.startswith('gs://'):
        #Replacing the gs:// URI with https:// URI (works for public objects)
        url = 'https://storage.googleapis.com/' + url[len('gs://'):]

    import requests
    resp = requests.get(url)
    resp.raise_for_status()
    return _load_component_from_yaml_or_zip_bytes(resp.content, url)


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
    with open(filename, 'rb') as component_stream:
        return _load_component_from_yaml_or_zip_stream(component_stream, filename)


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


_COMPONENT_FILE_NAME_IN_ARCHIVE = 'component.yaml'


def _load_component_from_yaml_or_zip_bytes(bytes, component_filename=None):
    import io
    component_stream = io.BytesIO(bytes)
    return _load_component_from_yaml_or_zip_stream(component_stream, component_filename)


def _load_component_from_yaml_or_zip_stream(stream, component_filename=None):
    '''Loads component from a stream and creates a task factory function.
    The stream can be YAML or a zip file with a component.yaml file inside.
    '''
    import zipfile
    stream.seek(0)
    if zipfile.is_zipfile(stream):
        stream.seek(0)
        with zipfile.ZipFile(stream) as zip_obj:
            with zip_obj.open(_COMPONENT_FILE_NAME_IN_ARCHIVE) as component_stream:
                return _create_task_factory_from_component_text(component_stream, component_filename)
    else:
        stream.seek(0)
        return _create_task_factory_from_component_text(stream, component_filename)


def _create_task_factory_from_component_text(text_or_file, component_filename=None):
    component_dict = load_yaml(text_or_file)
    return _create_task_factory_from_component_dict(component_dict, component_filename)


def _create_task_factory_from_component_dict(component_dict, component_filename=None):
    component_spec = ComponentSpec.from_struct(component_dict)
    return _create_task_factory_from_component_spec(component_spec, component_filename)


_inputs_dir = '/inputs'
_outputs_dir = '/outputs'
_single_io_file_name = 'data'


def _generate_input_file_name(port_name):
    return _inputs_dir + '/' + _sanitize_file_name(port_name) + '/' + _single_io_file_name


def _generate_output_file_name(port_name):
    return _outputs_dir + '/' + _sanitize_file_name(port_name) + '/' + _single_io_file_name


def _try_get_object_by_name(obj_name):
    '''Locates any Python object (type, module, function, global variable) by name'''
    try:
        ##Might be heavy since locate searches all Python modules
        #from pydoc import locate
        #return locate(obj_name) or obj_name
        import builtins
        return builtins.__dict__.get(obj_name, obj_name)
    except:
        pass
    return obj_name



#Holds the transformation functions that are called each time TaskSpec instance is created from a component. If there are multiple handlers, the last one is used.
_created_task_transformation_handler = []


#TODO: Move to the dsl.Pipeline context class
from . import _dsl_bridge
_created_task_transformation_handler.append(_dsl_bridge.create_container_op_from_task)


#TODO: Refactor the function to make it shorter
def _create_task_factory_from_component_spec(component_spec:ComponentSpec, component_filename=None, component_ref: ComponentReference = None):
    name = component_spec.name or _default_component_name

    func_docstring_lines = []
    if component_spec.name:
        func_docstring_lines.append(component_spec.name)
    if component_spec.description:
        func_docstring_lines.append(component_spec.description)
    
    inputs_list = component_spec.inputs or [] #List[InputSpec]
    input_names = [input.name for input in inputs_list]

    #Creating the name translation tables : Original <-> Pythonic 
    input_name_to_pythonic = generate_unique_name_conversion_table(input_names, _sanitize_python_function_name)
    pythonic_name_to_input_name = {v: k for k, v in input_name_to_pythonic.items()}

    if component_ref is None:
        component_ref = ComponentReference(name=component_spec.name or component_filename or _default_component_name)
    component_ref._component_spec = component_spec

    def create_task_from_component_and_arguments(pythonic_arguments):
        #Converting the argument names and not passing None arguments
        valid_argument_types = (str, int, float, bool, GraphInputArgument, TaskOutputArgument, PipelineParam) #Hack for passed PipelineParams. TODO: Remove the hack once they're no longer passed here.
        arguments = {
            pythonic_name_to_input_name[k]: (v if isinstance(v, valid_argument_types) else str(v))
            for k, v in pythonic_arguments.items()
            if v is not None
        }
        for key in arguments:
            if isinstance(arguments[key], PipelineParam):
                if kfp.TYPE_CHECK:
                    for input_spec in component_spec.inputs:
                        if input_spec.name == key:
                            if arguments[key].param_type is not None and not check_types(arguments[key].param_type.to_dict_or_str(), '' if input_spec.type is None else input_spec.type):
                                raise InconsistentTypeException('Component "' + name + '" is expecting ' + key + ' to be type(' + str(input_spec.type) + '), but the passed argument is type(' + arguments[key].param_type.serialize() + ')')
                arguments[key] = str(arguments[key])

        task = TaskSpec(
            component_ref=component_ref,
            arguments=arguments,
        )
        if _created_task_transformation_handler:
            task = _created_task_transformation_handler[-1](task)
        return task

    import inspect
    from . import _dynamic

    #Reordering the inputs since in Python optional parameters must come after required parameters
    reordered_input_list = [input for input in inputs_list if input.default is None and not input.optional] + [input for input in inputs_list if not (input.default is None and not input.optional)]
    input_parameters  = [_dynamic.KwParameter(input_name_to_pythonic[port.name], annotation=(_try_get_object_by_name(str(port.type)) if port.type else inspect.Parameter.empty), default=port.default if port.default is not None else (None if port.optional else inspect.Parameter.empty)) for port in reordered_input_list]
    factory_function_parameters = input_parameters #Outputs are no longer part of the task factory function signature. The paths are always generated by the system.
    
    return _dynamic.create_function_from_parameters(
        create_task_from_component_and_arguments,        
        factory_function_parameters,
        documentation='\n'.join(func_docstring_lines),
        func_name=name,
        func_filename=component_filename
    )
