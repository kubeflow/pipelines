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
    'python_op',
    'func_to_container_op',
    'func_to_component_text',
]

from ._yaml_utils import dump_yaml
from ._components import _create_task_factory_from_component_spec
from ._structures import InputSpec, OutputSpec, ImplementationSpec, DockerContainerSpec, ComponentSpec

from pathlib import Path
from typing import TypeVar, Generic

T = TypeVar('T')

#OutputFile[GcsPath[Gzipped[Text]]]


class InputFile(Generic[T], str):
    pass


class OutputFile(Generic[T], str):
    pass

#TODO: Replace this image name with another name once people decide what to replace it with.
_default_base_image='tensorflow/tensorflow:1.11.0-py3'


def _python_function_name_to_component_name(name):
    import re
    return re.sub(' +', ' ', name.replace('_', ' ')).strip(' ').capitalize()


def _func_to_component_spec(func, extra_code='', base_image=_default_base_image) -> ComponentSpec:
    import inspect
    import re
    from collections import OrderedDict
    
    single_output_name_const = 'Output'
    single_output_pythonic_name_const = 'output'

    signature = inspect.signature(func)
    parameters = list(signature.parameters.values())
    parameter_to_type_name = OrderedDict()
    inputs = []
    outputs = []
    extra_output_names = []
    arguments = []

    def annotation_to_argument_kind_and_type_name(annotation):
        if not annotation or annotation == inspect.Parameter.empty:
            return ('value', None)
        if hasattr(annotation, '__origin__'): #Generic type
            type_name = annotation.__origin__.__name__
            type_args = annotation.__args__
            #if len(type_args) != 1:
            #    raise TypeError('Unsupported generic type {}'.format(type_name))
            inner_type = type_args[0]
            if type_name == InputFile.__name__:
                return ('file', inner_type.__name__)
            elif type_name == OutputFile.__name__:
                return ('output', inner_type.__name__)
        if isinstance(annotation, type):
            return ('value', annotation.__name__)
        else:
            #!!! It's important to preserve string anotations as strings. Annotations that are neither types nor strings are converted to strings.
            #Materializer adds double quotes to the types it does not recognize. - fix it to not quote strings.
            #We need two kind of strings: we can use any type name for component YAML, but for generated Python code we must use valid python type annotations.
            return ('value', "'" + str(annotation) + "'") 

    for parameter in parameters:
        annotation = parameter.annotation
        
        (argument_kind, parameter_type_name) = annotation_to_argument_kind_and_type_name(annotation)

        parameter_to_type_name[parameter.name] = parameter_type_name

        #TODO: Humanize the input/output names
        arguments.append({argument_kind: parameter.name})

        parameter_spec = OrderedDict([('name', parameter.name)])
        if parameter_type_name:
            parameter_spec['type'] = parameter_type_name
        if argument_kind == 'value' or argument_kind == 'file':
            inputs.append(parameter_spec)
        elif argument_kind == 'output':
            outputs.append(parameter_spec)
        else:
            #Cannot happen
            raise ValueError('Unrecognized argument kind {}.'.format(argument_kind))

    #Analyzing the return type annotations.
    return_ann = signature.return_annotation
    if hasattr(return_ann, '_fields'): #NamedTuple
        for field_name in return_ann._fields:
            output_spec = OrderedDict([('name', field_name)])
            if hasattr(return_ann, '_field_types'):
                output_type = return_ann._field_types.get(field_name, None)
                if isinstance(output_type, type):
                    output_type_name = output_type.__name__
                else:
                    output_type_name = str(output_type)
                
                if output_type:
                    output_spec['type'] = output_type_name
            outputs.append(output_spec)
            extra_output_names.append(field_name)
            arguments.append({'output': field_name})
    else:
        output_spec = OrderedDict([('name', single_output_name_const)])
        (_, output_type_name) = annotation_to_argument_kind_and_type_name(signature.return_annotation)
        if output_type_name:
            output_spec['type'] = output_type_name
        outputs.append(output_spec)
        extra_output_names.append(single_output_pythonic_name_const)
        arguments.append({'output': single_output_name_const})

    func_name=func.__name__

    #Source code can include decorators line @python_op. Remove them
    (func_code_lines, _) = inspect.getsourcelines(func) 
    while func_code_lines[0].lstrip().startswith('@'): #decorator
        del func_code_lines[0]
        
    #Function might be defined in some indented scope (e.g. in another function).
    #We need to handle this and properly dedent the function source code
    first_line = func_code_lines[0]
    indent = len(first_line) - len(first_line.lstrip())
    func_code_lines = [line[indent:] for line in func_code_lines]
    
    func_code = ''.join(func_code_lines) #Lines retain their \n endings

    extra_output_external_names = [name + '_file' for name in extra_output_names]

    input_args_parsing_code_lines =(
        "    '{arg_name}': {arg_type}(sys.argv[{arg_idx}]),".format(
            arg_name=name_type[0],
            arg_type=name_type[1] if name_type[1] in ['int', 'float', 'bool'] else 'str',
            arg_idx=idx + 1
        )
        for idx, name_type in enumerate(parameter_to_type_name.items())
    )

    output_files_parsing_code_lines = (
        '    sys.argv[{}],'.format(idx + len(parameter_to_type_name) + 1)
        for idx in range(len(extra_output_external_names))
    )

    full_source = \
'''\
from typing import NamedTuple

{extra_code}

{func_code}

import sys
_args = {{
{input_args_parsing_code}
}}
_output_files = [
{output_files_parsing_code}
]

_outputs = {func_name}(**_args)

from collections.abc import Sequence
if not isinstance(_outputs, Sequence) or isinstance(_outputs, str):
    _outputs = [_outputs]

from pathlib import Path
for idx, filename in enumerate(_output_files):
    _output_path = Path(filename)
    _output_path.parent.mkdir(parents=True, exist_ok=True)
    _output_path.write_text(str(_outputs[idx]))
'''.format(
        func_name=func_name,
        func_code=func_code,
        extra_code=extra_code,
        input_args_parsing_code='\n'.join(input_args_parsing_code_lines),
        output_files_parsing_code='\n'.join(output_files_parsing_code_lines),
    )

    #Removing consecutive blank lines
    full_source = re.sub('\n\n\n+', '\n\n', full_source).strip('\n') + '\n'

    component_name = _python_function_name_to_component_name(func_name)
    description = func.__doc__.strip() + '\n' if func.__doc__ else None #Interesting: unlike ruamel.yaml, PyYaml cannot handle trailing spaces in the last line (' \n') and switches the style to double-quoted.

    component_spec = ComponentSpec(
        name=component_name,
        description=description,
        inputs=[InputSpec.from_struct(input) for input in inputs],
        outputs=[OutputSpec.from_struct(output) for output in outputs],
        implementation=ImplementationSpec(
            docker_container=DockerContainerSpec(
                image=base_image,
                command=['python3', '-c', full_source],
                arguments=arguments,
            )
        )
    )

    return component_spec


def _func_to_component_dict(func, extra_code='', base_image=_default_base_image):
    return _func_to_component_spec(func, extra_code, base_image).to_struct()


def func_to_component_text(func, extra_code='', base_image=_default_base_image):
    '''
    Converts a Python function to a component definition and returns its textual representation

    Function docstring is used as component description.
    Argument and return annotations are used as component input/output types.
    To declare a function with multiple return values, use the NamedTuple return annotation syntax:

        from typing import NamedTuple
        def add_multiply_two_numbers(a: float, b: float) -> NamedTuple('DummyName', [('sum', float), ('product', float)]):
            """Returns sum and product of two arguments"""
            return (a + b, a * b)

    Args:
        func: The python function to convert
        base_image: Optional. Specify a custom Docker containerimage to use in the component. For lightweight components, the image needs to have python and the fire package.
        extra_code: Optional. Extra code to add before the function code. May contain imports and other functions.
    
    Returns:
        Textual representation of a component definition
    '''
    component_dict = _func_to_component_dict(func, extra_code, base_image)
    return dump_yaml(component_dict)


def func_to_component_file(func, output_component_file, base_image=_default_base_image, extra_code='') -> None:
    '''
    Converts a Python function to a component definition and writes it to a file

    Function docstring is used as component description.
    Argument and return annotations are used as component input/output types.
    To declare a function with multiple return values, use the NamedTuple return annotation syntax:

        from typing import NamedTuple
        def add_multiply_two_numbers(a: float, b: float) -> NamedTuple('DummyName', [('sum', float), ('product', float)]):
            """Returns sum and product of two arguments"""
            return (a + b, a * b)

    Args:
        func: The python function to convert
        output_component_file: Write a component definition to a local file. Can be used for sharing.
        base_image: Optional. Specify a custom Docker containerimage to use in the component. For lightweight components, the image needs to have python and the fire package.
        extra_code: Optional. Extra code to add before the function code. May contain imports and other functions.
    '''

    component_yaml = func_to_component_text(func, extra_code, base_image)
    
    Path(output_component_file).write_text(component_yaml)


def func_to_container_op(func, output_component_file=None, base_image=_default_base_image, extra_code=''):
    '''
    Converts a Python function to a component and returns a task (ContainerOp) factory

    Function docstring is used as component description.
    Argument and return annotations are used as component input/output types.
    To declare a function with multiple return values, use the NamedTuple return annotation syntax:

        from typing import NamedTuple
        def add_multiply_two_numbers(a: float, b: float) -> NamedTuple('DummyName', [('sum', float), ('product', float)]):
            """Returns sum and product of two arguments"""
            return (a + b, a * b)

    Args:
        func: The python function to convert
        base_image: Optional. Specify a custom Docker containerimage to use in the component. For lightweight components, the image needs to have python and the fire package.
        output_component_file: Optional. Write a component definition to a local file. Can be used for sharing.
        extra_code: Optional. Extra code to add before the function code. May contain imports and other functions.

    Returns:
        A factory function with a strongly-typed signature taken from the python function.
        Once called with the required arguments, the factory constructs a pipeline task instance (ContainerOp) that can run the original function in a container.
    '''

    component_spec = _func_to_component_spec(func, extra_code, base_image)

    if output_component_file:
        component_dict = component_spec.to_struct()
        component_yaml = dump_yaml(component_dict)
        Path(output_component_file).write_text(component_yaml)
        #TODO: assert ComponentSpec.from_struct(load_yaml(output_component_file)) == component_spec

    return _create_task_factory_from_component_spec(component_spec)


def python_op(func=None, base_image=_default_base_image, output_component_file=None, extra_code=''):
    '''
    Decorator that replaces a Python function with an equivalent task (ContainerOp) factory

    Function docstring is used as component description.
    Argument and return annotations are used as component input/output types.
    To declare a function with multiple return values, use the NamedTuple return annotation syntax:

        from typing import NamedTuple
        @python_op(base_image='tensorflow/tensorflow:1.11.0-py3')
        def add_multiply_two_numbers_op(a: float, b: float) -> NamedTuple('DummyName', [('sum', float), ('product', float)]):
            """Returns sum and product of two arguments"""
            return (a + b, a * b)

    Args:
        func: The python function to convert
        base_image: Optional. Specify a custom Docker containerimage to use in the component. For lightweight components, the image needs to have python and the fire package.
        output_component_file: Optional. Write a component definition to a local file. Can be used for sharing.
        extra_code: Optional. Extra code to add before the function code. May contain imports and other functions.

    Returns:
        A factory function with a strongly-typed signature taken from the python function.
        Once called with the required arguments, the factory constructs a pipeline task instance (ContainerOp) that can run the original function in a container.
    '''

    if func:
        return func_to_container_op(func, output_component_file, base_image, extra_code)
    else:
        return lambda f: func_to_container_op(f, output_component_file, base_image, extra_code)
