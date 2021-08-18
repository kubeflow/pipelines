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
import inspect
import itertools
import re
import textwrap
from typing import Callable, Dict, List, Mapping, Optional, TypeVar
import warnings

import docstring_parser

from kfp import components as v1_components
from kfp.components import _components, _data_passing, structures, type_annotation_utils
from kfp.v2.components.types import artifact_types, type_annotations

_DEFAULT_BASE_IMAGE = 'python:3.7'


def _python_function_name_to_component_name(name):
    name_with_spaces = re.sub(' +', ' ', name.replace('_', ' ')).strip(' ')
    return name_with_spaces[0].upper() + name_with_spaces[1:]


def _get_packages_to_install_command(
        package_list: Optional[List[str]] = None) -> List[str]:
    result = []
    if package_list is not None:
        install_pip_command = 'python3 -m ensurepip'
        install_packages_command = (
            'PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet \
                --no-warn-script-location {}'                                             ).format(' '.join(
                [repr(str(package)) for package in package_list]))
        result = [
            'sh', '-c',
            '({install_pip} || {install_pip} --user) &&'
            ' ({install_packages} || {install_packages} --user) && "$0" "$@"'.
            format(install_pip=install_pip_command,
                   install_packages=install_packages_command)
        ]
    return result


def _get_default_kfp_package_path() -> str:
    import kfp
    return 'kfp=={}'.format(kfp.__version__)


def _get_function_source_definition(func: Callable) -> str:
    func_code = inspect.getsource(func)

    # Function might be defined in some indented scope (e.g. in another
    # function). We need to handle this and properly dedent the function source
    # code
    func_code = textwrap.dedent(func_code)
    func_code_lines = func_code.split('\n')

    # Removing possible decorators (can be multiline) until the function
    # definition is found
    func_code_lines = itertools.dropwhile(lambda x: not x.startswith('def'),
                                          func_code_lines)

    if not func_code_lines:
        raise ValueError(
            'Failed to dedent and clean up the source of function "{}". '
            'It is probably not properly indented.'.format(func.__name__))

    return '\n'.join(func_code_lines)


def _annotation_to_type_struct(annotation):
    if not annotation or annotation == inspect.Parameter.empty:
        return None
    if hasattr(annotation, 'to_dict'):
        annotation = annotation.to_dict()
    if isinstance(annotation, dict):
        return annotation
    if isinstance(annotation, type):
        type_struct = _data_passing.get_canonical_type_struct_for_type(
            annotation)
        if type_struct:
            return type_struct
        type_name = str(annotation.__name__)
    elif hasattr(
            annotation, '__forward_arg__'
    ):  # Handling typing.ForwardRef('Type_name') (the name was _ForwardRef in python 3.5-3.6)
        type_name = str(annotation.__forward_arg__)
    else:
        type_name = str(annotation)

    # It's also possible to get the converter by type name
    type_struct = _data_passing.get_canonical_type_struct_for_type(type_name)
    if type_struct:
        return type_struct
    return type_name


def _maybe_make_unique(name: str, names: List[str]):
    if name not in names:
        return name

    for i in range(2, 100):
        unique_name = '{}_{}'.format(name, i)
        if unique_name not in names:
            return unique_name

    raise RuntimeError('Too many arguments with the name {}'.format(name))


# TODO(KFPv2): Replace with v2 ComponentSpec.
def _func_to_component_spec(
        func: Callable,
        base_image: Optional[str] = None,
        packages_to_install: Optional[List[str]] = None,
        install_kfp_package: bool = True,
        kfp_package_path: Optional[str] = None) -> structures.ComponentSpec:
    decorator_base_image = getattr(func, '_component_base_image', None)
    if decorator_base_image is not None:
        if base_image is not None and decorator_base_image != base_image:
            raise ValueError(
                'base_image ({}) conflicts with the decorator-specified base image metadata ({})'
                .format(base_image, decorator_base_image))
        else:
            base_image = decorator_base_image
    else:
        if base_image is None:
            base_image = _DEFAULT_BASE_IMAGE
            if isinstance(base_image, Callable):
                base_image = base_image()

    imports_source = [
        "from kfp.v2.dsl import *",
        "from typing import *",
    ]

    func_source = _get_function_source_definition(func)

    source = textwrap.dedent("""
        {imports_source}

        {func_source}\n""").format(imports_source='\n'.join(imports_source),
                                   func_source=func_source)

    packages_to_install = packages_to_install or []
    if install_kfp_package:
        if kfp_package_path is None:
            kfp_package_path = _get_default_kfp_package_path()
        packages_to_install.append(kfp_package_path)

    packages_to_install_command = _get_packages_to_install_command(
        package_list=packages_to_install)

    from kfp.components._structures import ExecutorInputPlaceholder
    component_spec = extract_component_interface(func)

    component_spec.implementation = structures.ContainerImplementation(
        container=structures.ContainerSpec(image=base_image,
                                           command=packages_to_install_command +
                                           [
                                               'sh',
                                               '-ec',
                                               textwrap.dedent('''\
                    program_path=$(mktemp -d)
                    printf "%s" "$0" > "$program_path/ephemeral_component.py"
                    python3 -m kfp.v2.components.executor_main \
                        --component_module_path \
                        "$program_path/ephemeral_component.py" \
                        "$@"
                '''),
                                               source,
                                           ],
                                           args=[
                                               "--executor_input",
                                               ExecutorInputPlaceholder(),
                                               "--function_to_execute",
                                               func.__name__,
                                           ]))
    return component_spec


def extract_component_interface(func: Callable) -> structures.ComponentSpec:
    single_output_name_const = 'Output'

    signature = inspect.signature(func)
    parameters = list(signature.parameters.values())

    parsed_docstring = docstring_parser.parse(inspect.getdoc(func))
    doc_dict = {p.arg_name: p.description for p in parsed_docstring.params}

    inputs = []
    outputs = []

    input_names = set()
    output_names = set()
    for parameter in parameters:
        parameter_type = type_annotation_utils.maybe_strip_optional_from_annotation(
            parameter.annotation)
        passing_style = None
        io_name = parameter.name

        if type_annotations.is_artifact_annotation(parameter_type):
            # passing_style is either type_annotations.InputAnnotation or
            # type_annotations.OutputAnnotation.
            passing_style = type_annotations.get_io_artifact_annotation(
                parameter_type)

            # parameter_type is type_annotations.Artifact or one of its subclasses.
            parameter_type = type_annotations.get_io_artifact_class(
                parameter_type)
            if not issubclass(parameter_type, artifact_types.Artifact):
                raise ValueError(
                    'Input[T] and Output[T] are only supported when T is a '
                    'subclass of Artifact. Found `{} with type {}`'.format(
                        io_name, parameter_type))

            if parameter.default is not inspect.Parameter.empty:
                raise ValueError(
                    'Default values for Input/Output artifacts are not supported.'
                )
        elif isinstance(parameter_type,
                        (v1_components.InputPath, v1_components.OutputPath)):
            raise TypeError(
                'In v2 components, please import the Python function'
                ' annotations `InputPath` and `OutputPath` from'
                ' package `kfp.v2.dsl` instead of `kfp.dsl`.')
        elif isinstance(
                parameter_type,
            (type_annotations.InputPath, type_annotations.OutputPath)):
            passing_style = type(parameter_type)
            parameter_type = parameter_type.type
            if parameter.default is not inspect.Parameter.empty and not (
                    passing_style == type_annotations.InputPath and
                    parameter.default is None):
                raise ValueError(
                    'Path inputs only support default values of None. Default values for outputs are not supported.'
                )

        type_struct = _annotation_to_type_struct(parameter_type)

        if passing_style in [
                type_annotations.OutputAnnotation, type_annotations.OutputPath
        ]:
            io_name = _maybe_make_unique(io_name, output_names)
            output_names.add(io_name)
            output_spec = structures.OutputSpec(name=io_name,
                                                type=type_struct,
                                                description=doc_dict.get(
                                                    parameter.name))
            output_spec._passing_style = passing_style
            output_spec._parameter_name = parameter.name
            outputs.append(output_spec)
        else:
            io_name = _maybe_make_unique(io_name, input_names)
            input_names.add(io_name)
            input_spec = structures.InputSpec(name=io_name,
                                              type=type_struct,
                                              description=doc_dict.get(
                                                  parameter.name))
            if parameter.default is not inspect.Parameter.empty:
                input_spec.optional = True
                if parameter.default is not None:
                    outer_type_name = list(type_struct.keys())[0] if isinstance(
                        type_struct, dict) else type_struct
                    try:
                        input_spec.default = _data_passing.serialize_value(
                            parameter.default, outer_type_name)
                    except Exception as ex:
                        warnings.warn(
                            'Could not serialize the default value of the parameter "{}". {}'
                            .format(parameter.name, ex))
            input_spec._passing_style = passing_style
            input_spec._parameter_name = parameter.name
            inputs.append(input_spec)

    #Analyzing the return type annotations.
    return_ann = signature.return_annotation
    if hasattr(return_ann, '_fields'):  #NamedTuple
        # Getting field type annotations.
        # __annotations__ does not exist in python 3.5 and earlier
        # _field_types does not exist in python 3.9 and later
        field_annotations = getattr(return_ann,
                                    '__annotations__', None) or getattr(
                                        return_ann, '_field_types', None)
        for field_name in return_ann._fields:
            type_struct = None
            if field_annotations:
                type_struct = _annotation_to_type_struct(
                    field_annotations.get(field_name, None))

            output_name = _maybe_make_unique(field_name, output_names)
            output_names.add(output_name)
            output_spec = structures.OutputSpec(
                name=output_name,
                type=type_struct,
            )
            output_spec._passing_style = None
            output_spec._return_tuple_field_name = field_name
            outputs.append(output_spec)
    # Deprecated dict-based way of declaring multiple outputs. Was only used by the @component decorator
    elif isinstance(return_ann, dict):
        warnings.warn(
            "The ability to specify multiple outputs using the dict syntax has been deprecated."
            "It will be removed soon after release 0.1.32."
            "Please use typing.NamedTuple to declare multiple outputs.")
        for output_name, output_type_annotation in return_ann.items():
            output_type_struct = _annotation_to_type_struct(
                output_type_annotation)
            output_spec = structures.OutputSpec(
                name=output_name,
                type=output_type_struct,
            )
            outputs.append(output_spec)
    elif signature.return_annotation is not None and signature.return_annotation != inspect.Parameter.empty:
        output_name = _maybe_make_unique(single_output_name_const, output_names)
        # Fixes exotic, but possible collision: `def func(output_path: OutputPath()) -> str: ...`
        output_names.add(output_name)
        type_struct = _annotation_to_type_struct(signature.return_annotation)
        output_spec = structures.OutputSpec(
            name=output_name,
            type=type_struct,
        )
        output_spec._passing_style = None
        outputs.append(output_spec)

    # Component name and description are derived from the function's name and docstring.
    # The name can be overridden by setting setting func.__name__ attribute (of the legacy func._component_human_name attribute).
    # The description can be overridden by setting the func.__doc__ attribute (or the legacy func._component_description attribute).
    component_name = getattr(func, '_component_human_name',
                             None) or _python_function_name_to_component_name(
                                 func.__name__)
    description = getattr(func, '_component_description',
                          None) or parsed_docstring.short_description
    if description:
        description = description.strip()

    component_spec = structures.ComponentSpec(
        name=component_name,
        description=description,
        inputs=inputs if inputs else None,
        outputs=outputs if outputs else None,
    )
    return component_spec


def create_component_from_func(func: Callable,
                               base_image: Optional[str] = None,
                               packages_to_install: List[str] = None,
                               output_component_file: Optional[str] = None,
                               install_kfp_package: bool = True,
                               kfp_package_path: Optional[str] = None):
    """Converts a Python function to a v2 lightweight component.

    A lightweight component is a self-contained Python function that includes
    all necessary imports and dependencies.

    Args:
        func: The python function to create a component from. The function
            should have type annotations for all its arguments, indicating how
            it is intended to be used (e.g. as an input/output Artifact object,
            a plain parameter, or a path to a file).
        base_image: The image to use when executing |func|. It should
            contain a default Python interpreter that is compatible with KFP.
        packages_to_install: A list of optional packages to install before
            executing |func|.
        install_kfp_package: Specifies if we should add a KFP Python package to
            |packages_to_install|. Lightweight Python functions always require
            an installation of KFP in |base_image| to work. If you specify
            a |base_image| that already contains KFP, you can set this to False.
        kfp_package_path: Specifies the location from which to install KFP. By
            default, this will try to install from PyPi using the same version
            as that used when this component was created. KFP developers can
            choose to override this to point to a Github pull request or
            other pip-compatible location when testing changes to lightweight
            Python functions.

    Returns:
        A component task factory that can be used in pipeline definitions.
    """
    component_spec = _func_to_component_spec(
        func=func,
        base_image=base_image,
        packages_to_install=packages_to_install,
        install_kfp_package=install_kfp_package,
        kfp_package_path=kfp_package_path)
    if output_component_file:
        component_spec.save(output_component_file)

    # TODO(KFPv2): Replace with v2 BaseComponent.
    return _components._create_task_factory_from_component_spec(component_spec)