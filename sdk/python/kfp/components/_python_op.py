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
    'func_to_container_op',
    'func_to_component_text',
]

from ._yaml_utils import dump_yaml
from ._components import _create_task_factory_from_component_spec
from ._structures import *

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
    '''Takes a self-contained python function and converts it to component

    Args:
        func: Required. The function to be converted
        base_image: Optional. Docker image to be used as a base image for the python component. Must have python 3.5+ installed. Default is tensorflow/tensorflow:1.11.0-py3
                    Note: The image can also be specified by decorating the function with the @python_component decorator. If different base images are explicitly specified in both places, an error is raised.
        extra_code: Optional. Python source code that gets placed before the function code. Can be used as workaround to define types used in function signature.
    '''
    decorator_base_image = getattr(func, '_component_base_image', None)
    if decorator_base_image is not None:
        if base_image is not _default_base_image and decorator_base_image != base_image:
            raise ValueError('base_image ({}) conflicts with the decorator-specified base image metadata ({})'.format(base_image, decorator_base_image))
        else:
            base_image = decorator_base_image
    else:
        if base_image is None:
            raise ValueError('base_image cannot be None')

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

    def annotation_to_type_struct(annotation):
        if not annotation or annotation == inspect.Parameter.empty:
            return None
        if isinstance(annotation, type):
            return str(annotation.__name__)
        else:
            return str(annotation)

    for parameter in parameters:
        type_struct = annotation_to_type_struct(parameter.annotation)
        parameter_to_type_name[parameter.name] = str(type_struct)
        #TODO: Humanize the input/output names
        arguments.append(InputValuePlaceholder(parameter.name))

        input_spec = InputSpec(
            name=parameter.name,
            type=type_struct,
            default=str(parameter.default) if parameter.default is not inspect.Parameter.empty else None,
        )
        inputs.append(input_spec)

    #Analyzing the return type annotations.
    return_ann = signature.return_annotation
    if hasattr(return_ann, '_fields'): #NamedTuple
        for field_name in return_ann._fields:
            type_struct = None
            if hasattr(return_ann, '_field_types'):
                type_struct = annotation_to_type_struct(return_ann._field_types.get(field_name, None))

            output_spec = OutputSpec(
                name=field_name,
                type=type_struct,
            )
            outputs.append(output_spec)
            extra_output_names.append(field_name)
            arguments.append(OutputPathPlaceholder(field_name))
    elif signature.return_annotation is not None and signature.return_annotation != inspect.Parameter.empty:
        type_struct = annotation_to_type_struct(signature.return_annotation)
        output_spec = OutputSpec(
            name=single_output_name_const,
            type=type_struct,
        )
        outputs.append(output_spec)
        extra_output_names.append(single_output_pythonic_name_const)
        arguments.append(OutputPathPlaceholder(single_output_name_const))

    func_name=func.__name__

    #TODO: Add support for copying the NamedTuple subclass declaration code
    #Adding NamedTuple import if needed
    func_type_declarations_code = ""
    if hasattr(return_ann, '_fields'): #NamedTuple
        func_type_declarations_code = func_type_declarations_code + '\n' + 'from typing import NamedTuple'

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
{extra_code}

{func_type_declarations_code}

{func_code}

import sys
_args = {{
{input_args_parsing_code}
}}
_output_files = [
{output_files_parsing_code}
]

_outputs = {func_name}(**_args)

if not hasattr(_outputs, '__getitem__') or isinstance(_outputs, str):
    _outputs = [_outputs]

from pathlib import Path
for idx, filename in enumerate(_output_files):
    _output_path = Path(filename)
    _output_path.parent.mkdir(parents=True, exist_ok=True)
    _output_path.write_text(str(_outputs[idx]))
'''.format(
        func_name=func_name,
        func_code=func_code,
        func_type_declarations_code=func_type_declarations_code,
        extra_code=extra_code,
        input_args_parsing_code='\n'.join(input_args_parsing_code_lines),
        output_files_parsing_code='\n'.join(output_files_parsing_code_lines),
    )

    #Removing consecutive blank lines
    full_source = re.sub('\n\n\n+', '\n\n', full_source).strip('\n') + '\n'

    #Component name and description are derived from the function's name and docstribng, but can be overridden by @python_component function decorator
    #The decorator can set the _component_human_name and _component_description attributes. getattr is needed to prevent error when these attributes do not exist.
    component_name = getattr(func, '_component_human_name', None) or _python_function_name_to_component_name(func.__name__)
    description = getattr(func, '_component_description', None) or func.__doc__
    if description:
        description = description.strip() + '\n' #Interesting: unlike ruamel.yaml, PyYaml cannot handle trailing spaces in the last line (' \n') and switches the style to double-quoted.

    component_spec = ComponentSpec(
        name=component_name,
        description=description,
        inputs=inputs,
        outputs=outputs,
        implementation=ContainerImplementation(
            container=ContainerSpec(
                image=base_image,
                command=['python3', '-c', full_source],
                args=arguments,
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
        base_image: Optional. Specify a custom Docker container image to use in the component. For lightweight components, the image needs to have python 3.5+. Default is tensorflow/tensorflow:1.11.0-py3
                    Note: The image can also be specified by decorating the function with the @python_component decorator. If different base images are explicitly specified in both places, an error is raised.
        extra_code: Optional. Extra code to add before the function code. Can be used as workaround to define types used in function signature.
    
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
        base_image: Optional. Specify a custom Docker container image to use in the component. For lightweight components, the image needs to have python 3.5+. Default is tensorflow/tensorflow:1.11.0-py3
                    Note: The image can also be specified by decorating the function with the @python_component decorator. If different base images are explicitly specified in both places, an error is raised.
        extra_code: Optional. Extra code to add before the function code. Can be used as workaround to define types used in function signature.
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
        base_image: Optional. Specify a custom Docker container image to use in the component. For lightweight components, the image needs to have python 3.5+. Default is tensorflow/tensorflow:1.11.0-py3
                    Note: The image can also be specified by decorating the function with the @python_component decorator. If different base images are explicitly specified in both places, an error is raised.
        output_component_file: Optional. Write a component definition to a local file. Can be used for sharing.
        extra_code: Optional. Extra code to add before the function code. Can be used as workaround to define types used in function signature.

    Returns:
        A factory function with a strongly-typed signature taken from the python function.
        Once called with the required arguments, the factory constructs a pipeline task instance (ContainerOp) that can run the original function in a container.
    '''

    component_spec = _func_to_component_spec(func, extra_code, base_image)

    output_component_file = output_component_file or getattr(func, '_component_target_component_file', None)
    if output_component_file:
        component_dict = component_spec.to_struct()
        component_yaml = dump_yaml(component_dict)
        Path(output_component_file).write_text(component_yaml)
        #TODO: assert ComponentSpec.from_struct(load_yaml(output_component_file)) == component_spec

    return _create_task_factory_from_component_spec(component_spec)
