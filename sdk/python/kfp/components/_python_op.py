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

import inspect
from pathlib import Path
from typing import TypeVar, Generic, List

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


def _capture_function_code_using_cloudpickle(func, modules_to_capture: List[str] = None) -> str:
    import base64
    import sys
    import cloudpickle
    import pickle

    if modules_to_capture is None:
        modules_to_capture = [func.__module__]

    # Hack to force cloudpickle to capture the whole function instead of just referencing the code file. See https://github.com/cloudpipe/cloudpickle/blob/74d69d759185edaeeac7bdcb7015cfc0c652f204/cloudpickle/cloudpickle.py#L490
    old_modules = {}
    try: # Try is needed to restore the state if something goes wrong
        for module_name in modules_to_capture:
            if module_name in sys.modules:
                old_modules[module_name] = sys.modules.pop(module_name)
        func_pickle = base64.b64encode(cloudpickle.dumps(func, pickle.DEFAULT_PROTOCOL))
    finally:
        sys.modules.update(old_modules)

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
    import inspect	

    #Source code can include decorators line @python_op. Remove them
    (func_code_lines, _) = inspect.getsourcelines(func)
    while func_code_lines[0].lstrip().startswith('@'): #decorator
        del func_code_lines[0]

    #Function might be defined in some indented scope (e.g. in another function).
    #We need to handle this and properly dedent the function source code
    first_line = func_code_lines[0]
    indent = len(first_line) - len(first_line.lstrip())
    func_code_lines = [line[indent:] for line in func_code_lines]

    #TODO: Add support for copying the NamedTuple subclass declaration code
    #Adding NamedTuple import if needed
    if hasattr(inspect.signature(func).return_annotation, '_fields'): #NamedTuple
        func_code_lines.insert(0, '\n')
        func_code_lines.insert(0, 'from typing import NamedTuple\n')

    return ''.join(func_code_lines) #Lines retain their \n endings


def _extract_component_interface(func) -> ComponentSpec:
    single_output_name_const = 'Output'

    signature = inspect.signature(func)
    parameters = list(signature.parameters.values())
    inputs = []
    outputs = []

    def annotation_to_type_struct(annotation):
        if not annotation or annotation == inspect.Parameter.empty:
            return None
        if isinstance(annotation, type):
            return str(annotation.__name__)
        else:
            return str(annotation)

    for parameter in parameters:
        type_struct = annotation_to_type_struct(parameter.annotation)
        #TODO: Humanize the input/output names

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
    elif signature.return_annotation is not None and signature.return_annotation != inspect.Parameter.empty:
        type_struct = annotation_to_type_struct(signature.return_annotation)
        output_spec = OutputSpec(
            name=single_output_name_const,
            type=type_struct,
        )
        outputs.append(output_spec)

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
    )
    return component_spec


def _func_to_component_spec(func, extra_code='', base_image=_default_base_image, modules_to_capture: List[str] = None, use_code_pickling=False) -> ComponentSpec:
    '''Takes a self-contained python function and converts it to component

    Args:
        func: Required. The function to be converted
        base_image: Optional. Docker image to be used as a base image for the python component. Must have python 3.5+ installed. Default is tensorflow/tensorflow:1.11.0-py3
                    Note: The image can also be specified by decorating the function with the @python_component decorator. If different base images are explicitly specified in both places, an error is raised.
        extra_code: Optional. Python source code that gets placed before the function code. Can be used as workaround to define types used in function signature.
        modules_to_capture: Optional. List of module names that will be captured (instead of just referencing) during the dependency scan. By default the func.__module__ is captured.
        use_code_pickling: Specifies whether the function code should be captured using pickling as opposed to source code manipulation. Pickling has better support for capturing dependencies, but is sensitive to version mismatch between python in component creation environment and runtime image.
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

    component_spec = _extract_component_interface(func)

    arguments = []
    arguments.extend(InputValuePlaceholder(input.name) for input in component_spec.inputs)
    arguments.extend(OutputPathPlaceholder(output.name) for output in component_spec.outputs)

    if use_code_pickling:
        func_code = _capture_function_code_using_cloudpickle(func, modules_to_capture)
    else:
        func_code = _capture_function_code_using_source_copy(func)

    extra_output_names = [output.name for output in component_spec.outputs]
    extra_output_external_names = [name + '_file' for name in extra_output_names]

    from collections import OrderedDict
    parameter_to_type_name = OrderedDict((input.name, str(input.type)) for input in component_spec.inputs)

    arg_parse_code_lines = [
        'import argparse',
        '_parser = argparse.ArgumentParser(prog={prog_repr}, description={description_repr})'.format(
            prog_repr=repr(component_spec.name or ''),
            description_repr=repr(component_spec.description or ''),
        ),
    ]
    arguments = []
    for input in component_spec.inputs:
        param_flag = "--" + input.name.replace("_", "-")
        line = '_parser.add_argument("{param_flag}", dest="{param_var}", type={param_type}, required={is_required}, default={default_repr})'.format(
            param_flag=param_flag,
            param_var=input.name,
            param_type=(input.type if input.type in ['int', 'float', 'bool'] else 'str'),
            is_required=str(input.default is None), # TODO: Handle actual 'None' defaults!
            default_repr=repr(str(input.default)) if input.default is not None else None,
        )
        arg_parse_code_lines.append(line)
        arguments.append(param_flag)
        arguments.append(InputValuePlaceholder(input.name))

    if component_spec.outputs:
        param_flag="----output-paths"
        output_param_var="_output_paths"
        line = '_parser.add_argument("{param_flag}", dest="{param_var}", type=str, nargs={nargs})'.format(
            param_flag=param_flag,
            param_var=output_param_var,
            nargs=len(component_spec.outputs),
        )
        arg_parse_code_lines.append(line)
        arguments.append(param_flag)
        arguments.extend(OutputPathPlaceholder(output.name) for output in component_spec.outputs)

    arg_parse_code_lines.extend([
        '_parsed_args = vars(_parser.parse_args())',
    ])

    if component_spec.outputs:
        arg_parse_code_lines.extend([
            '_output_files = _parsed_args.pop("_output_paths")',
        ])

    full_source = \
'''\
{extra_code}

{func_code}

{arg_parse_code}

_outputs = {func_name}(**_parsed_args)

if not hasattr(_outputs, '__getitem__') or isinstance(_outputs, str):
    _outputs = [_outputs]

from pathlib import Path
for idx, filename in enumerate(_output_files):
    _output_path = Path(filename)
    _output_path.parent.mkdir(parents=True, exist_ok=True)
    _output_path.write_text(str(_outputs[idx]))
'''.format(
        func_name=func.__name__,
        func_code=func_code,
        extra_code=extra_code,
        arg_parse_code='\n'.join(arg_parse_code_lines),
    )

    #Removing consecutive blank lines
    import re
    full_source = re.sub('\n\n\n+', '\n\n', full_source).strip('\n') + '\n'

    component_spec.implementation=ContainerImplementation(
        container=ContainerSpec(
            image=base_image,
            command=['python3', '-u', '-c', full_source],
            args=arguments,
        )
    )

    return component_spec


def _func_to_component_dict(func, extra_code='', base_image=_default_base_image, modules_to_capture: List[str] = None, use_code_pickling=False):
    return _func_to_component_spec(func, extra_code, base_image, modules_to_capture, use_code_pickling).to_dict()


def func_to_component_text(func, extra_code='', base_image=_default_base_image, modules_to_capture: List[str] = None, use_code_pickling=False):
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
        modules_to_capture: Optional. List of module names that will be captured (instead of just referencing) during the dependency scan. By default the func.__module__ is captured. The actual algorithm: Starting with the initial function, start traversing dependencies. If the dependecy.__module__ is in the modules_to_capture list then it's captured and it's dependencies are traversed. Otherwise the dependency is only referenced instead of capturing and its dependencies are not traversed.
        use_code_pickling: Specifies whether the function code should be captured using pickling as opposed to source code manipulation. Pickling has better support for capturing dependencies, but is sensitive to version mismatch between python in component creation environment and runtime image.
    
    Returns:
        Textual representation of a component definition
    '''
    component_dict = _func_to_component_dict(func, extra_code, base_image, modules_to_capture, use_code_pickling)
    return dump_yaml(component_dict)


def func_to_component_file(func, output_component_file, base_image=_default_base_image, extra_code='', modules_to_capture: List[str] = None, use_code_pickling=False) -> None:
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
        modules_to_capture: Optional. List of module names that will be captured (instead of just referencing) during the dependency scan. By default the func.__module__ is captured. The actual algorithm: Starting with the initial function, start traversing dependencies. If the dependecy.__module__ is in the modules_to_capture list then it's captured and it's dependencies are traversed. Otherwise the dependency is only referenced instead of capturing and its dependencies are not traversed.
        use_code_pickling: Specifies whether the function code should be captured using pickling as opposed to source code manipulation. Pickling has better support for capturing dependencies, but is sensitive to version mismatch between python in component creation environment and runtime image.
    '''

    component_yaml = func_to_component_text(func, extra_code, base_image, modules_to_capture, use_code_pickling)
    
    Path(output_component_file).write_text(component_yaml)


def func_to_container_op(func, output_component_file=None, base_image=_default_base_image, extra_code='', modules_to_capture: List[str] = None, use_code_pickling=False):
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
        modules_to_capture: Optional. List of module names that will be captured (instead of just referencing) during the dependency scan. By default the func.__module__ is captured. The actual algorithm: Starting with the initial function, start traversing dependencies. If the dependecy.__module__ is in the modules_to_capture list then it's captured and it's dependencies are traversed. Otherwise the dependency is only referenced instead of capturing and its dependencies are not traversed.
        use_code_pickling: Specifies whether the function code should be captured using pickling as opposed to source code manipulation. Pickling has better support for capturing dependencies, but is sensitive to version mismatch between python in component creation environment and runtime image.

    Returns:
        A factory function with a strongly-typed signature taken from the python function.
        Once called with the required arguments, the factory constructs a pipeline task instance (ContainerOp) that can run the original function in a container.
    '''

    component_spec = _func_to_component_spec(func, extra_code, base_image, modules_to_capture, use_code_pickling)

    output_component_file = output_component_file or getattr(func, '_component_target_component_file', None)
    if output_component_file:
        component_dict = component_spec.to_dict()
        component_yaml = dump_yaml(component_dict)
        Path(output_component_file).write_text(component_yaml)
        #TODO: assert ComponentSpec.from_dict(load_yaml(output_component_file)) == component_spec

    return _create_task_factory_from_component_spec(component_spec)
