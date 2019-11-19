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
from ._data_passing import serialize_value, type_name_to_type
from kfp.dsl import PipelineParam
from kfp.dsl.types import verify_type_compatibility

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
    component_ref = ComponentReference(url=url)
    return _load_component_from_yaml_or_zip_bytes(resp.content, url, component_ref)


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


def _load_component_from_yaml_or_zip_bytes(bytes, component_filename=None, component_ref: ComponentReference = None):
    import io
    component_stream = io.BytesIO(bytes)
    return _load_component_from_yaml_or_zip_stream(component_stream, component_filename, component_ref)


def _load_component_from_yaml_or_zip_stream(stream, component_filename=None, component_ref: ComponentReference = None):
    '''Loads component from a stream and creates a task factory function.
    The stream can be YAML or a zip file with a component.yaml file inside.
    '''
    import zipfile
    stream.seek(0)
    if zipfile.is_zipfile(stream):
        stream.seek(0)
        with zipfile.ZipFile(stream) as zip_obj:
            with zip_obj.open(_COMPONENT_FILE_NAME_IN_ARCHIVE) as component_stream:
                return _create_task_factory_from_component_text(component_stream, component_filename, component_ref)
    else:
        stream.seek(0)
        return _create_task_factory_from_component_text(stream, component_filename, component_ref)


def _create_task_factory_from_component_text(text_or_file, component_filename=None, component_ref: ComponentReference = None):
    component_dict = load_yaml(text_or_file)
    return _create_task_factory_from_component_dict(component_dict, component_filename, component_ref)


def _create_task_factory_from_component_dict(component_dict, component_filename=None, component_ref: ComponentReference = None):
    component_spec = ComponentSpec.from_dict(component_dict)
    return _create_task_factory_from_component_spec(component_spec, component_filename, component_ref)


_inputs_dir = '/tmp/inputs'
_outputs_dir = '/tmp/outputs'
_single_io_file_name = 'data'


def _generate_input_file_name(port_name):
    return _inputs_dir + '/' + _sanitize_file_name(port_name) + '/' + _single_io_file_name


def _generate_output_file_name(port_name):
    return _outputs_dir + '/' + _sanitize_file_name(port_name) + '/' + _single_io_file_name


#Holds the transformation functions that are called each time TaskSpec instance is created from a component. If there are multiple handlers, the last one is used.
_created_task_transformation_handler = []


#TODO: Move to the dsl.Pipeline context class
from . import _dsl_bridge
_created_task_transformation_handler.append(_dsl_bridge.create_container_op_from_task)


class _DefaultValue:
    def __init__(self, value):
        self.value = value

    def __repr__(self):
        return repr(self.value)


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
        component_ref = ComponentReference(spec=component_spec, url=component_filename)
    else:
        component_ref.spec = component_spec

    def create_task_from_component_and_arguments(pythonic_arguments):
        arguments = {}
        # Not checking for missing or extra arguments since the dynamic factory function checks that
        for argument_name, argument_value in pythonic_arguments.items():
            if isinstance(argument_value, _DefaultValue): # Skipping passing arguments for optional values that have not been overridden.
                continue
            input_name = pythonic_name_to_input_name[argument_name]
            input_type = component_spec._inputs_dict[input_name].type

            if isinstance(argument_value, (GraphInputArgument, TaskOutputArgument, PipelineParam)):
                # argument_value is a reference 

                if isinstance(argument_value, PipelineParam):
                    reference_type = argument_value.param_type
                    argument_value = str(argument_value)
                elif isinstance(argument_value, TaskOutputArgument):
                    reference_type = argument_value.task_output.type
                else:
                    reference_type = None

                verify_type_compatibility(reference_type, input_type, 'Incompatible argument passed to the input "{}" of component "{}": '.format(input_name, component_spec.name))

                arguments[input_name] = argument_value
            else:
                # argument_value is a constant value
                serialized_argument_value = serialize_value(argument_value, input_type)
                arguments[input_name] = serialized_argument_value

        task = TaskSpec(
            component_ref=component_ref,
            arguments=arguments,
        )
        task._init_outputs()
        
        if isinstance(component_spec.implementation, GraphImplementation):
            return _resolve_graph_task(task, component_spec)

        if _created_task_transformation_handler:
            task = _created_task_transformation_handler[-1](task)
        return task

    import inspect
    from . import _dynamic

    #Reordering the inputs since in Python optional parameters must come after required parameters
    reordered_input_list = [input for input in inputs_list if input.default is None and not input.optional] + [input for input in inputs_list if not (input.default is None and not input.optional)]

    def component_default_to_func_default(component_default: str, is_optional: bool):
        if is_optional:
            return _DefaultValue(component_default)
        if component_default is not None:
            return component_default
        return inspect.Parameter.empty

    input_parameters = [
        _dynamic.KwParameter(
            input_name_to_pythonic[port.name],
            annotation=(type_name_to_type.get(str(port.type), str(port.type)) if port.type else inspect.Parameter.empty),
            default=component_default_to_func_default(port.default, port.optional),
        )
        for port in reordered_input_list
    ]
    factory_function_parameters = input_parameters #Outputs are no longer part of the task factory function signature. The paths are always generated by the system.
    
    task_factory = _dynamic.create_function_from_parameters(
        create_task_from_component_and_arguments,        
        factory_function_parameters,
        documentation='\n'.join(func_docstring_lines),
        func_name=name,
        func_filename=component_filename if (component_filename and (component_filename.endswith('.yaml') or component_filename.endswith('.yml'))) else None,
    )
    task_factory.component_spec = component_spec
    return task_factory


def _resolve_graph_task(graph_task: TaskSpec, graph_component_spec: ComponentSpec) -> TaskSpec:
    from ..components import ComponentStore
    component_store = ComponentStore.default_store

    graph = graph_component_spec.implementation.graph

    graph_input_arguments = {input.name: input.default for input in graph_component_spec.inputs if input.default is not None}
    graph_input_arguments.update(graph_task.arguments)

    outputs_of_tasks = {}
    def resolve_argument(argument):
        if isinstance(argument, (str, int, float, bool)):
            return argument
        elif isinstance(argument, GraphInputArgument):
            return graph_input_arguments[argument.graph_input.input_name]
        elif isinstance(argument, TaskOutputArgument):
            upstream_task_output_ref = argument.task_output
            upstream_task_outputs = outputs_of_tasks[upstream_task_output_ref.task_id]
            upstream_task_output = upstream_task_outputs[upstream_task_output_ref.output_name]
            return upstream_task_output
        else:
            raise TypeError('Argument for input has unexpected type "{}".'.format(type(argument)))

    for task_id, task_spec in graph._toposorted_tasks.items(): # Cannot use graph.tasks here since they might be listed not in dependency order. Especially on python <3.6 where the dicts do not preserve ordering
        task_factory = component_store._load_component_from_ref(task_spec.component_ref)
        # TODO: Handle the case when optional graph component input is passed to optional task component input
        task_arguments = {input_name: resolve_argument(argument) for input_name, argument in task_spec.arguments.items()}
        task_component_spec = task_factory.component_spec

        input_name_to_pythonic = generate_unique_name_conversion_table([input.name for input in task_component_spec.inputs or []], _sanitize_python_function_name)
        output_name_to_pythonic = generate_unique_name_conversion_table([output.name for output in task_component_spec.outputs or []], _sanitize_python_function_name)
        pythonic_output_name_to_original = {pythonic_name: original_name for original_name, pythonic_name in output_name_to_pythonic.items()}
        pythonic_task_arguments = {input_name_to_pythonic[input_name]: argument for input_name, argument in task_arguments.items()}

        task_obj = task_factory(**pythonic_task_arguments)
        task_outputs_with_pythonic_names = task_obj.outputs
        task_outputs_with_original_names = {pythonic_output_name_to_original[pythonic_output_name]: output_value for pythonic_output_name, output_value in task_outputs_with_pythonic_names.items()}
        outputs_of_tasks[task_id] = task_outputs_with_original_names

    resolved_graph_outputs = OrderedDict([(output_name, resolve_argument(argument)) for output_name, argument in graph.output_values.items()])

    # For resolved graph component tasks task.outputs point to the actual tasks that originally produced the output that is later returned from the graph
    graph_task.output_references = graph_task.outputs
    graph_task.outputs = resolved_graph_outputs
    
    return graph_task
