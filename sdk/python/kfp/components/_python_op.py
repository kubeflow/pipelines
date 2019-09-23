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
    'create_component_from_func',
    'func_to_container_op',
    'func_to_component_text',
    'default_base_image_or_builder',
    'get_default_base_image',
    'set_default_base_image',
    'InputPath',
    'InputTextFile',
    'InputBinaryFile',
    'OutputPath',
    'OutputTextFile',
    'OutputBinaryFile',
]

from ._yaml_utils import dump_yaml
from ._components import _create_task_factory_from_component_spec
from ._data_passing import serialize_value, type_name_to_deserializer, type_name_to_serializer, type_to_type_name
from ._naming import _make_name_unique_by_adding_index
from .structures import *

import inspect
from pathlib import Path
import typing
from typing import Callable, Generic, List, TypeVar, Union
from deprecated.sphinx import deprecated

T = TypeVar('T')


# InputPath(list) or InputPath('JsonObject')

class InputPath:
    '''When creating component from function, InputPath should be used as function parameter annotation to tell the system to pass the *data file path* to the function instead of passing the actual data.'''
    def __init__(self, type=None):
        self.type = type


class InputTextFile:
    '''When creating component from function, InputTextFile should be used as function parameter annotation to tell the system to pass the *text data stream* object (`io.TextIOWrapper`) to the function instead of passing the actual data.'''
    def __init__(self, type=None):
        self.type = type


class InputBinaryFile:
    '''When creating component from function, InputBinaryFile should be used as function parameter annotation to tell the system to pass the *binary data stream* object (`io.BytesIO`) to the function instead of passing the actual data.'''
    def __init__(self, type=None):
        self.type = type


#OutputFile[GcsPath[Gzipped[Text]]]


class OutputPath:
    '''When creating component from function, OutputPath should be used as function parameter annotation to tell the system that the function wants to output data by writing it into a file with the given path instead of returning the data from the function.'''
    def __init__(self, type=None):
        self.type = type


class OutputTextFile:
    '''When creating component from function, OutputTextFile should be used as function parameter annotation to tell the system that the function wants to output data by writing it into a given text file stream (`io.TextIOWrapper`) instead of returning the data from the function.'''
    def __init__(self, type=None):
        self.type = type


class OutputBinaryFile:
    '''When creating component from function, OutputBinaryFile should be used as function parameter annotation to tell the system that the function wants to output data by writing it into a given binary file stream (`io.BytesIO`) instead of returning the data from the function.'''
    def __init__(self, type=None):
        self.type = type


def _make_parent_dirs_and_return_path(file_path: str):
    import os
    os.makedirs(os.path.dirname(file_path), exist_ok=True)
    return file_path


def _parent_dirs_maker_that_returns_open_file(mode: str, encoding: str = None):
    def make_parent_dirs_and_return_path(file_path: str):
        import os
        os.makedirs(os.path.dirname(file_path), exist_ok=True)
        return open(file_path, mode=mode, encoding=encoding)
    return make_parent_dirs_and_return_path


#TODO: Replace this image name with another name once people decide what to replace it with.
default_base_image_or_builder='tensorflow/tensorflow:1.13.2-py3'

@deprecated(version='0.1.32', reason='Use the kfp.components.default_base_image_or_builder variable instead')
def get_default_base_image() -> Union[str, Callable[[], str]]:
    return default_base_image_or_builder


@deprecated(version='0.1.32', reason='Use the kfp.components.default_base_image_or_builder variable instead')
def set_default_base_image(image_or_factory: Union[str, Callable[[], str]]):
    '''set_default_base_image sets the name of the container image that will be used for component creation when base_image is not specified.
    Alternatively, the base image can also be set to a factory function that will be returning the image.
    '''
    global default_base_image_or_builder
    default_base_image_or_builder = image_or_factory


def _python_function_name_to_component_name(name):
    import re
    name_with_spaces = re.sub(' +', ' ', name.replace('_', ' ')).strip(' ')
    return name_with_spaces[0].upper() + name_with_spaces[1:]


def _capture_function_code_using_cloudpickle(func, modules_to_capture: List[str] = None) -> str:
    import base64
    import sys
    import cloudpickle
    import pickle

    if modules_to_capture is None:
        modules_to_capture = [func.__module__]

    # Hack to force cloudpickle to capture the whole function instead of just referencing the code file. See https://github.com/cloudpipe/cloudpickle/blob/74d69d759185edaeeac7bdcb7015cfc0c652f204/cloudpickle/cloudpickle.py#L490
    old_modules = {}
    old_sig = getattr(func, '__signature__', None)
    try: # Try is needed to restore the state if something goes wrong
        for module_name in modules_to_capture:
            if module_name in sys.modules:
                old_modules[module_name] = sys.modules.pop(module_name)
        # Hack to prevent cloudpickle from trying to pickle generic types that might be present in the signature. See https://github.com/cloudpipe/cloudpickle/issues/196 
        # Currently the __signature__ is only set by Airflow components as a means to spoof/pass the function signature to _func_to_component_spec
        if hasattr(func, '__signature__'):
            del func.__signature__
        func_pickle = base64.b64encode(cloudpickle.dumps(func, pickle.DEFAULT_PROTOCOL))
    finally:
        sys.modules.update(old_modules)
        if old_sig:
            func.__signature__ = old_sig

    function_loading_code = '''\
import sys
try:
    import cloudpickle as _cloudpickle
except ImportError:
    import subprocess
    try:
        print("cloudpickle is not installed. Installing it globally", file=sys.stderr)
        subprocess.run([sys.executable, "-m", "pip", "install", "cloudpickle==1.1.1", "--quiet"], env={"PIP_DISABLE_PIP_VERSION_CHECK": "1"}, check=True)
        print("Installed cloudpickle globally", file=sys.stderr)
    except:
        print("Failed to install cloudpickle globally. Installing for the current user.", file=sys.stderr)
        subprocess.run([sys.executable, "-m", "pip", "install", "cloudpickle==1.1.1", "--user", "--quiet"], env={"PIP_DISABLE_PIP_VERSION_CHECK": "1"}, check=True)
        print("Installed cloudpickle for the current user", file=sys.stderr)
        # Enable loading from user-installed package directory. Python does not add it to sys.path if it was empty at start. Running pip does not refresh `sys.path`.
        import site
        sys.path.append(site.getusersitepackages())
    import cloudpickle as _cloudpickle
    print("cloudpickle loaded successfully after installing.", file=sys.stderr)
''' + '''
pickler_python_version = {pickler_python_version}
current_python_version = tuple(sys.version_info)
if (
    current_python_version[0] != pickler_python_version[0] or
    current_python_version[1] < pickler_python_version[1] or
    current_python_version[0] == 3 and ((pickler_python_version[1] < 6) != (current_python_version[1] < 6))
    ):
    raise RuntimeError("Incompatible python versions: " + str(current_python_version) + " instead of " + str(pickler_python_version))

if current_python_version != pickler_python_version:
    print("Warning!: Different python versions. The code may crash! Current environment python version: " + str(current_python_version) + ". Component code python version: " + str(pickler_python_version), file=sys.stderr)

import base64
import pickle

{func_name} = pickle.loads(base64.b64decode({func_pickle}))
'''.format(
        func_name=func.__name__,
        func_pickle=repr(func_pickle),
        pickler_python_version=repr(tuple(sys.version_info)),
    )

    return function_loading_code


def _capture_function_code_using_source_copy(func) -> str:	
    import textwrap

    func_code = inspect.getsource(func)

    #Function might be defined in some indented scope (e.g. in another function).
    #We need to handle this and properly dedent the function source code
    func_code = textwrap.dedent(func_code)
    func_code_lines = func_code.split('\n')

    # Removing possible decorators (can be multiline) until the function definition is found
    while func_code_lines and not func_code_lines[0].startswith('def '):
        del func_code_lines[0]

    if not func_code_lines:
        raise ValueError('Failed to dedent and clean up the source of function "{}". It is probably not properly indented.'.format(func.__name__))

    #TODO: Add support for copying the NamedTuple subclass declaration code
    #Adding NamedTuple import if needed
    if hasattr(inspect.signature(func).return_annotation, '_fields'): #NamedTuple
        func_code_lines.insert(0, '')
        func_code_lines.insert(0, 'from typing import NamedTuple')

    return '\n'.join(func_code_lines)


def _extract_component_interface(func) -> ComponentSpec:
    single_output_name_const = 'Output'

    signature = inspect.signature(func)
    parameters = list(signature.parameters.values())
    inputs = []
    outputs = []

    def annotation_to_type_struct(annotation):
        if not annotation or annotation == inspect.Parameter.empty:
            return None
        if hasattr(annotation, 'to_dict'):
            annotation = annotation.to_dict()
        if isinstance(annotation, dict):
            return annotation
        if isinstance(annotation, type):
            if annotation in type_to_type_name:
                return type_to_type_name[annotation]
            type_name = str(annotation.__name__)
        elif hasattr(annotation, '__forward_arg__'): # Handling typing.ForwardRef('Type_name') (the name was _ForwardRef in python 3.5-3.6)
            type_name = str(annotation.__forward_arg__)
        else:
            type_name = str(annotation)

        if type_name in type_to_type_name: # type_to_type_name can also have type name keys
            type_name = type_to_type_name[type_name]
        return type_name

    input_names = set()
    output_names = set()
    for parameter in parameters:
        parameter_annotation = parameter.annotation
        passing_style = None
        io_name = parameter.name
        if isinstance(parameter_annotation, (InputPath, InputTextFile, InputBinaryFile, OutputPath, OutputTextFile, OutputBinaryFile)):
            passing_style = type(parameter_annotation)
            parameter_annotation = parameter_annotation.type
            if parameter.default is not inspect.Parameter.empty:
                raise ValueError('Default values for file inputs/outputs are not supported. If you need them for some reason, please create an issue and write about your usage scenario.')
            # Removing the "_path" and "_file" suffixes from the input/output names as the argument passed to the component needs to be the data itself, not local file path.
            # Problem: When accepting file inputs (outputs), the function inside the component receives file paths (or file streams), so it's natural to call the function parameter "something_file_path" (e.g. model_file_path or number_file_path).
            # But from the outside perspective, there are no files or paths - the actual data objects (or references to them) are passed in.
            # It looks very strange when argument passing code looks like this: `component(number_file_path=42)`. This looks like an error since 42 is not a path. It's not even a string.
            # It's much more natural to strip the names of file inputs and outputs of "_file" or "_path" suffixes. Then the argument passing code will look natural: "component(number=42)".
            if isinstance(parameter.annotation, (InputPath, OutputPath)) and io_name.endswith('_path'):
                io_name = io_name[0:-len('_path')]
            if io_name.endswith('_file'):
                io_name = io_name[0:-len('_file')]
        type_struct = annotation_to_type_struct(parameter_annotation)
        #TODO: Humanize the input/output names

        if isinstance(parameter.annotation, (OutputPath, OutputTextFile, OutputBinaryFile)):
            io_name = _make_name_unique_by_adding_index(io_name, output_names, '_')
            output_names.add(io_name)
            output_spec = OutputSpec(
                name=io_name,
                type=type_struct,
            )
            output_spec._passing_style = passing_style
            output_spec._parameter_name = parameter.name
            outputs.append(output_spec)
        else:
            io_name = _make_name_unique_by_adding_index(io_name, input_names, '_')
            input_names.add(io_name)
            input_spec = InputSpec(
                name=io_name,
                type=type_struct,
            )
            if parameter.default is not inspect.Parameter.empty:
                input_spec.optional = True
                if parameter.default is not None:
                    outer_type_name = list(type_struct.keys())[0] if isinstance(type_struct, dict) else type_struct
                    input_spec.default = serialize_value(parameter.default, outer_type_name)
            input_spec._passing_style = passing_style
            input_spec._parameter_name = parameter.name
            inputs.append(input_spec)

    #Analyzing the return type annotations.
    return_ann = signature.return_annotation
    if hasattr(return_ann, '_fields'): #NamedTuple
        for field_name in return_ann._fields:
            type_struct = None
            if hasattr(return_ann, '_field_types'):
                type_struct = annotation_to_type_struct(return_ann._field_types.get(field_name, None))

            output_name = _make_name_unique_by_adding_index(field_name, output_names, '_')
            output_names.add(output_name)
            output_spec = OutputSpec(
                name=output_name,
                type=type_struct,
            )
            output_spec._passing_style = None
            output_spec._return_tuple_field_name = field_name
            outputs.append(output_spec)
    # Deprecated dict-based way of declaring multiple outputs. Was only used by the @component decorator
    elif isinstance(return_ann, dict):
        import warnings
        warnings.warn(
            "The ability to specify multiple outputs using the dict syntax has been deprecated."
            "It will be removed soon after release 0.1.32."
            "Please use typing.NamedTuple to declare multiple outputs."
        )
        for output_name, output_type_annotation in return_ann.items():
            output_type_struct = annotation_to_type_struct(output_type_annotation)
            output_spec = OutputSpec(
                name=output_name,
                type=output_type_struct,
            )
            outputs.append(output_spec)
    elif signature.return_annotation is not None and signature.return_annotation != inspect.Parameter.empty:
        output_name = _make_name_unique_by_adding_index(single_output_name_const, output_names, '_') # Fixes exotic, but possible collision: `def func(output_path: OutputPath()) -> str: ...`
        output_names.add(output_name)
        type_struct = annotation_to_type_struct(signature.return_annotation)
        output_spec = OutputSpec(
            name=output_name,
            type=type_struct,
        )
        output_spec._passing_style = None
        outputs.append(output_spec)

    #Component name and description are derived from the function's name and docstribng, but can be overridden by @python_component function decorator
    #The decorator can set the _component_human_name and _component_description attributes. getattr is needed to prevent error when these attributes do not exist.
    component_name = getattr(func, '_component_human_name', None) or _python_function_name_to_component_name(func.__name__)
    description = getattr(func, '_component_description', None) or func.__doc__
    if description:
        description = description.strip()

    # TODO: Parse input/output descriptions from the function docstring. See:
    # https://github.com/rr-/docstring_parser
    # https://github.com/terrencepreilly/darglint/blob/master/darglint/parse.py

    component_spec = ComponentSpec(
        name=component_name,
        description=description,
        inputs=inputs if inputs else None,
        outputs=outputs if outputs else None,
    )
    return component_spec


def _func_to_component_spec(func, extra_code='', base_image : str = None, packages_to_install: List[str] = None, modules_to_capture: List[str] = None, use_code_pickling=False) -> ComponentSpec:
    '''Takes a self-contained python function and converts it to component

    Args:
        func: Required. The function to be converted
        base_image: Optional. Docker image to be used as a base image for the python component. Must have python 3.5+ installed. Default is tensorflow/tensorflow:1.11.0-py3
                    Note: The image can also be specified by decorating the function with the @python_component decorator. If different base images are explicitly specified in both places, an error is raised.
        extra_code: Optional. Python source code that gets placed before the function code. Can be used as workaround to define types used in function signature.
        packages_to_install: Optional. List of [versioned] python packages to pip install before executing the user function.
        modules_to_capture: Optional. List of module names that will be captured (instead of just referencing) during the dependency scan. By default the func.__module__ is captured.
        use_code_pickling: Specifies whether the function code should be captured using pickling as opposed to source code manipulation. Pickling has better support for capturing dependencies, but is sensitive to version mismatch between python in component creation environment and runtime image.
    '''
    decorator_base_image = getattr(func, '_component_base_image', None)
    if decorator_base_image is not None:
        if base_image is not None and decorator_base_image != base_image:
            raise ValueError('base_image ({}) conflicts with the decorator-specified base image metadata ({})'.format(base_image, decorator_base_image))
        else:
            base_image = decorator_base_image
    else:
        if base_image is None:
            base_image = default_base_image_or_builder
            if isinstance(base_image, Callable):
                base_image = base_image()

    packages_to_install = packages_to_install or []

    component_spec = _extract_component_interface(func)

    component_inputs = component_spec.inputs or []
    component_outputs = component_spec.outputs or []

    arguments = []
    arguments.extend(InputValuePlaceholder(input.name) for input in component_inputs)
    arguments.extend(OutputPathPlaceholder(output.name) for output in component_outputs)

    if use_code_pickling:
        func_code = _capture_function_code_using_cloudpickle(func, modules_to_capture)
        # pip startup is quite slow. TODO: Remove the special cloudpickle installation code in favor of the the following line once a way to speed up pip startup is discovered.
        #packages_to_install.append('cloudpickle==1.1.1')
    else:
        func_code = _capture_function_code_using_source_copy(func)

    definitions = set()
    def get_deserializer_and_register_definitions(type_name):
        if type_name in type_name_to_deserializer:
            (deserializer_code_str, definition_str) = type_name_to_deserializer[type_name]
            if definition_str:
                definitions.add(definition_str)
            return deserializer_code_str
        return 'str'

    pre_func_definitions = set()
    def get_argparse_type_for_input_file(passing_style):
        if passing_style is None:
            return None
        pre_func_definitions.add(inspect.getsource(passing_style))

        if passing_style is InputPath:
            return 'str'
        elif passing_style is InputTextFile:
            return "argparse.FileType('rt')"
        elif passing_style is InputBinaryFile:
            return "argparse.FileType('rb')"
        # For Output* we cannot use the build-in argparse.FileType objects since they do not create parent directories.
        elif passing_style is OutputPath:
            # ~= return 'str'
            pre_func_definitions.add(inspect.getsource(_make_parent_dirs_and_return_path))
            return _make_parent_dirs_and_return_path.__name__
        elif passing_style is OutputTextFile:
            # ~= return "argparse.FileType('wt')"
            pre_func_definitions.add(inspect.getsource(_parent_dirs_maker_that_returns_open_file))
            return _parent_dirs_maker_that_returns_open_file.__name__ + "('wt')"
        elif passing_style is OutputBinaryFile:
            # ~= return "argparse.FileType('wb')"
            pre_func_definitions.add(inspect.getsource(_parent_dirs_maker_that_returns_open_file))
            return _parent_dirs_maker_that_returns_open_file.__name__ + "('wb')"
        raise NotImplementedError('Unexpected data passing style: "{}".'.format(str(passing_style)))

    def get_serializer_and_register_definitions(type_name) -> str:
        if type_name in type_name_to_serializer:
            serializer_func = type_name_to_serializer[type_name]
            # If serializer is not part of the standard python library, then include its code in the generated program
            if hasattr(serializer_func, '__module__') and not _module_is_builtin_or_standard(serializer_func.__module__):
                import inspect
                serializer_code_str = inspect.getsource(serializer_func)
                definitions.add(serializer_code_str)
            return serializer_func.__name__
        return 'str'

    arg_parse_code_lines = [
        'import argparse',
        '_parser = argparse.ArgumentParser(prog={prog_repr}, description={description_repr})'.format(
            prog_repr=repr(component_spec.name or ''),
            description_repr=repr(component_spec.description or ''),
        ),
    ]
    outputs_passed_through_func_return_tuple = [output for output in component_outputs if output._passing_style is None]
    file_outputs_passed_using_func_parameters = [output for output in component_outputs if output._passing_style is not None]
    arguments = []
    for input in component_inputs + file_outputs_passed_using_func_parameters:
        param_flag = "--" + input.name.replace("_", "-")
        is_required = isinstance(input, OutputSpec) or not input.optional
        line = '_parser.add_argument("{param_flag}", dest="{param_var}", type={param_type}, required={is_required}, default=argparse.SUPPRESS)'.format(
            param_flag=param_flag,
            param_var=input._parameter_name, # Not input.name, since the inputs could have been renamed
            param_type=get_argparse_type_for_input_file(input._passing_style) or get_deserializer_and_register_definitions(input.type),
            is_required=str(is_required),
        )
        arg_parse_code_lines.append(line)

        if input._passing_style in [InputPath, InputTextFile, InputBinaryFile]:
            arguments_for_input = [param_flag, InputPathPlaceholder(input.name)]
        elif input._passing_style in [OutputPath, OutputTextFile, OutputBinaryFile]:
            arguments_for_input = [param_flag, OutputPathPlaceholder(input.name)]
        else:
            arguments_for_input = [param_flag, InputValuePlaceholder(input.name)]

        if is_required:
            arguments.extend(arguments_for_input)
        else:
            arguments.append(
                IfPlaceholder(
                    IfPlaceholderStructure(
                        condition=IsPresentPlaceholder(input.name),
                        then_value=arguments_for_input,
                    )
                )
            )

    if outputs_passed_through_func_return_tuple:
        param_flag="----output-paths"
        output_param_var="_output_paths"
        line = '_parser.add_argument("{param_flag}", dest="{param_var}", type=str, nargs={nargs})'.format(
            param_flag=param_flag,
            param_var=output_param_var,
            nargs=len(outputs_passed_through_func_return_tuple),
        )
        arg_parse_code_lines.append(line)
        arguments.append(param_flag)
        arguments.extend(OutputPathPlaceholder(output.name) for output in outputs_passed_through_func_return_tuple)

    output_serialization_expression_strings = []
    for output in outputs_passed_through_func_return_tuple:
        serializer_call_str = get_serializer_and_register_definitions(output.type)
        output_serialization_expression_strings.append(serializer_call_str)

    pre_func_code = '\n'.join(list(pre_func_definitions))

    arg_parse_code_lines = list(definitions) + arg_parse_code_lines

    arg_parse_code_lines.extend([
        '_parsed_args = vars(_parser.parse_args())',
        '_output_files = _parsed_args.pop("_output_paths", [])',
    ])

    output_serialization_code = ''.join('    {},\n'.format(s) for s in output_serialization_expression_strings)

    full_source = \
'''\
{pre_func_code}

{extra_code}

{func_code}

{arg_parse_code}

_outputs = {func_name}(**_parsed_args)

if not hasattr(_outputs, '__getitem__') or isinstance(_outputs, str):
    _outputs = [_outputs]

_output_serializers = [
{output_serialization_code}
]

import os
for idx, output_file in enumerate(_output_files):
    try:
        os.makedirs(os.path.dirname(output_file))
    except OSError:
        pass
    with open(output_file, 'w') as f:
        f.write(_output_serializers[idx](_outputs[idx]))
'''.format(
        func_name=func.__name__,
        func_code=func_code,
        pre_func_code=pre_func_code,
        extra_code=extra_code,
        arg_parse_code='\n'.join(arg_parse_code_lines),
        output_serialization_code=output_serialization_code,
    )

    #Removing consecutive blank lines
    import re
    full_source = re.sub('\n\n\n+', '\n\n', full_source).strip('\n') + '\n'

    package_preinstallation_command = []
    if packages_to_install:
        package_install_command_line = 'PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location {}'.format(' '.join([repr(str(package)) for package in packages_to_install]))
        package_preinstallation_command = ['sh', '-c', '({pip_install} || {pip_install} --user) && "$0" "$@"'.format(pip_install=package_install_command_line)]

    component_spec.implementation=ContainerImplementation(
        container=ContainerSpec(
            image=base_image,
            command=package_preinstallation_command + ['python3', '-u', '-c', full_source],
            args=arguments,
        )
    )

    return component_spec


def _func_to_component_dict(func, extra_code='', base_image: str = None, packages_to_install: List[str] = None, modules_to_capture: List[str] = None, use_code_pickling=False):
    return _func_to_component_spec(
        func=func,
        extra_code=extra_code,
        base_image=base_image,
        packages_to_install=packages_to_install,
        modules_to_capture=modules_to_capture,
        use_code_pickling=use_code_pickling,
    ).to_dict()


def func_to_component_text(func, extra_code='', base_image: str = None, packages_to_install: List[str] = None, modules_to_capture: List[str] = None, use_code_pickling=False):
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
        base_image: Optional. Specify a custom Docker container image to use in the component. For lightweight components, the image needs to have python 3.5+. Default is tensorflow/tensorflow:1.13.2-py3
        extra_code: Optional. Extra code to add before the function code. Can be used as workaround to define types used in function signature.
        packages_to_install: Optional. List of [versioned] python packages to pip install before executing the user function.
        modules_to_capture: Optional. List of module names that will be captured (instead of just referencing) during the dependency scan. By default the func.__module__ is captured. The actual algorithm: Starting with the initial function, start traversing dependencies. If the dependecy.__module__ is in the modules_to_capture list then it's captured and it's dependencies are traversed. Otherwise the dependency is only referenced instead of capturing and its dependencies are not traversed.
        use_code_pickling: Specifies whether the function code should be captured using pickling as opposed to source code manipulation. Pickling has better support for capturing dependencies, but is sensitive to version mismatch between python in component creation environment and runtime image.
    
    Returns:
        Textual representation of a component definition
    '''
    component_dict = _func_to_component_dict(
        func=func,
        extra_code=extra_code,
        base_image=base_image,
        packages_to_install=packages_to_install,
        modules_to_capture=modules_to_capture,
        use_code_pickling=use_code_pickling,
    )
    return dump_yaml(component_dict)


def func_to_component_file(func, output_component_file, base_image: str = None, extra_code='', packages_to_install: List[str] = None, modules_to_capture: List[str] = None, use_code_pickling=False) -> None:
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
        base_image: Optional. Specify a custom Docker container image to use in the component. For lightweight components, the image needs to have python 3.5+. Default is tensorflow/tensorflow:1.13.2-py3
        extra_code: Optional. Extra code to add before the function code. Can be used as workaround to define types used in function signature.
        packages_to_install: Optional. List of [versioned] python packages to pip install before executing the user function.
        modules_to_capture: Optional. List of module names that will be captured (instead of just referencing) during the dependency scan. By default the func.__module__ is captured. The actual algorithm: Starting with the initial function, start traversing dependencies. If the dependecy.__module__ is in the modules_to_capture list then it's captured and it's dependencies are traversed. Otherwise the dependency is only referenced instead of capturing and its dependencies are not traversed.
        use_code_pickling: Specifies whether the function code should be captured using pickling as opposed to source code manipulation. Pickling has better support for capturing dependencies, but is sensitive to version mismatch between python in component creation environment and runtime image.
    '''

    component_yaml = func_to_component_text(
        func=func,
        extra_code=extra_code,
        base_image=base_image,
        packages_to_install=packages_to_install,
        modules_to_capture=modules_to_capture,
        use_code_pickling=use_code_pickling,
    )
    
    Path(output_component_file).write_text(component_yaml)


def func_to_container_op(func, output_component_file=None, base_image: str = None, extra_code='', packages_to_install: List[str] = None, modules_to_capture: List[str] = None, use_code_pickling=False):
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
        base_image: Optional. Specify a custom Docker container image to use in the component. For lightweight components, the image needs to have python 3.5+. Default is tensorflow/tensorflow:1.13.2-py3
        output_component_file: Optional. Write a component definition to a local file. Can be used for sharing.
        extra_code: Optional. Extra code to add before the function code. Can be used as workaround to define types used in function signature.
        packages_to_install: Optional. List of [versioned] python packages to pip install before executing the user function.
        modules_to_capture: Optional. List of module names that will be captured (instead of just referencing) during the dependency scan. By default the func.__module__ is captured. The actual algorithm: Starting with the initial function, start traversing dependencies. If the dependecy.__module__ is in the modules_to_capture list then it's captured and it's dependencies are traversed. Otherwise the dependency is only referenced instead of capturing and its dependencies are not traversed.
        use_code_pickling: Specifies whether the function code should be captured using pickling as opposed to source code manipulation. Pickling has better support for capturing dependencies, but is sensitive to version mismatch between python in component creation environment and runtime image.

    Returns:
        A factory function with a strongly-typed signature taken from the python function.
        Once called with the required arguments, the factory constructs a pipeline task instance (ContainerOp) that can run the original function in a container.
    '''

    component_spec = _func_to_component_spec(
        func=func,
        extra_code=extra_code,
        base_image=base_image,
        packages_to_install=packages_to_install,
        modules_to_capture=modules_to_capture,
        use_code_pickling=use_code_pickling,
    )

    output_component_file = output_component_file or getattr(func, '_component_target_component_file', None)
    if output_component_file:
        component_spec.save(output_component_file)
        #TODO: assert ComponentSpec.from_dict(load_yaml(output_component_file)) == component_spec

    return _create_task_factory_from_component_spec(component_spec)


def create_component_from_func(
    func: Callable,
    output_component_file: str=None,
    base_image: str = None,
    packages_to_install: List[str] = None,
):
    '''
    Converts a Python function to a component and returns a task factory (a function that accepts arguments and returns a task object).

    Function name and docstring are used as component name and description.
    Argument and return annotations are used as component input/output types.
    Example::

        def add(a: float, b: float) -> float:
            """Returns sum of two arguments"""
            return a + b

        # add_op is a task factory function that creates a task object when given arguments
        add_op = create_component_from_func(
            func=add,
            base_image='python:3.7', # Optional
            output_component_file='add.component.yaml', # Optional
            packages_to_install=['pandas==0.24'], # Optional
        )

        # The component spec can be accessed through the .component_spec attribute:
        add_op.component_spec.save('add.component.yaml')

        # The component function can be called with arguments to create a task:
        add_task = add_op(1, 3)

        # The resulting task has output references, corresponding to the component outputs.
        # When the function only has a single anonymous return value, the output name is "Output":
        sum_output_ref = add_task.outputs['Output']

        # These task output references can be passed to other component functions, constructing a computation graph:
        task2 = add_op(sum_output_ref, 5)


    `create_component_from_func` function can also be used as decorator::

        @create_component_from_func
        def add_op(a: float, b: float) -> float:
            """Returns sum of two arguments"""
            return a + b

    To declare a function with multiple return values, use the NamedTuple return annotation syntax::

        from typing import NamedTuple

        def add_multiply_two_numbers(a: float, b: float) -> NamedTuple('Outputs', [('sum', float), ('product', float)]):
            """Returns sum and product of two arguments"""
            return (a + b, a * b)

        add_multiply_op = create_component_from_func(add_multiply_two_numbers)

        # The component function can be called with arguments to create a task:
        add_multiply_task = add_multiply_op(1, 3)

        # The resulting task has output references, corresponding to the component outputs:
        sum_output_ref = add_multiply_task.outputs['sum']

        # These task output references can be passed to other component functions, constructing a computation graph:
        task2 = add_multiply_op(sum_output_ref, 5)


    Bigger data should be read from files and written to files.
    Use the `InputPath` parameter annotation to tell the system that the function wants to consume the corresponding input data as a file. The system will download the data, write it to a local file and then pass the **path** of that file to the function.
    Use the `OutputPath` parameter annotation to tell the system that the function wants to produce the corresponding output data as a file. The system will prepare and pass the **path** of a file where the function should write the output data. After the function exits, the system will upload the data to the storage system so that it can be passed to downstream components.
    You can specify the type of the consumed/produced data by specifying the type argument to `InputPath` and `OutputPath`. The type can be a python type or an arbitrary type name string. `OutputPath('CatBoostModel')` means that the function states that the data it has written to a file has type 'CatBoostModel'. `InputPath('CatBoostModel')` means that the function states that it expect the data it reads from a file to have type 'CatBoostModel'. When the pipeline author connects inputs to outputs the system checks whether the types match.
    Every kind of data can be consumed as a file input. Conversely, bigger data should not be consumed by value as all value inputs pass through the command line.

    Example of a component function declaring file input and output::

        def catboost_train_classifier(
            training_data_path: InputPath('CSV'),            # Path to input data file of type "CSV"
            trained_model_path: OutputPath('CatBoostModel'), # Path to output data file of type "CatBoostModel"
            number_of_trees: int = 100,                      # Small output of type "Integer"
        ) -> NamedTuple('Outputs', [
            ('Accuracy', float),  # Small output of type "Float"
            ('Precision', float), # Small output of type "Float"
            ('JobUri', 'URI'),    # Small output of type "URI"
        ]):
            """Trains CatBoost classification model"""
            ...

            return (accuracy, precision, recall)


    Args:
        func: The python function to convert
        base_image: Optional. Specify a custom Docker container image to use in the component. For lightweight components, the image needs to have python 3.5+. Default is the python image corresponding to the current python environment.
        output_component_file: Optional. Write a component definition to a local file. The produced component file can be loaded back by calling `load_component_from_file` or `load_component_from_uri`.
        packages_to_install: Optional. List of [versioned] python packages to pip install before executing the user function.

    Returns:
        A factory function with a strongly-typed signature taken from the python function.
        Once called with the required arguments, the factory constructs a task instance that can run the original function in a container.
    '''

    component_spec = _func_to_component_spec(
        func=func,
        base_image=base_image,
        packages_to_install=packages_to_install,
    )

    if output_component_file:
        component_spec.save(output_component_file)

    return _create_task_factory_from_component_spec(component_spec)


def _module_is_builtin_or_standard(module_name: str) -> bool:
    import sys
    if module_name in sys.builtin_module_names:
        return True
    import distutils.sysconfig as sysconfig
    import os
    std_lib_dir = sysconfig.get_python_lib(standard_lib=True)
    module_name_parts = module_name.split('.')
    expected_module_path = os.path.join(std_lib_dir, *module_name_parts)
    return os.path.exists(expected_module_path) or os.path.exists(expected_module_path + '.py')
