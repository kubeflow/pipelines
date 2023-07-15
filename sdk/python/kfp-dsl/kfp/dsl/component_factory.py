# Copyright 2021-2022 The Kubeflow Authors
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
import dataclasses
import inspect
import itertools
import pathlib
import re
import textwrap
from typing import Callable, List, Mapping, Optional, Tuple, Type, Union
import warnings

from kfp import dsl
from kfp.dsl import container_component_artifact_channel
from kfp.dsl import container_component_class
from kfp.dsl import graph_component
from kfp.dsl import placeholders
from kfp.dsl import python_component
from kfp.dsl import structures
from kfp.dsl import task_final_status
from kfp.dsl.types import artifact_types
from kfp.dsl.types import custom_artifact_types
from kfp.dsl.types import type_annotations
from kfp.dsl.types import type_utils

_DEFAULT_BASE_IMAGE = 'python:3.7'


@dataclasses.dataclass
class ComponentInfo():
    """A dataclass capturing registered components.

    This will likely be subsumed/augmented with BaseComponent.
    """
    name: str
    function_name: str
    func: Callable
    target_image: str
    module_path: pathlib.Path
    component_spec: structures.ComponentSpec
    output_component_file: Optional[str] = None
    base_image: str = _DEFAULT_BASE_IMAGE
    packages_to_install: Optional[List[str]] = None
    pip_index_urls: Optional[List[str]] = None


# A map from function_name to components.  This is always populated when a
# module containing KFP components is loaded. Primarily used by KFP CLI
# component builder to package components in a file into containers.
REGISTERED_MODULES = None


def _python_function_name_to_component_name(name):
    name_with_spaces = re.sub(' +', ' ', name.replace('_', ' ')).strip(' ')
    return name_with_spaces[0].upper() + name_with_spaces[1:]


def make_index_url_options(pip_index_urls: Optional[List[str]]) -> str:
    """Generates index url options for pip install command based on provided
    pip_index_urls.

    Args:
        pip_index_urls: Optional list of pip index urls

    Returns:
        - Empty string if pip_index_urls is empty/None.
        - '--index-url url --trusted-host url ' if pip_index_urls contains 1
        url
        - the above followed by '--extra-index-url url --trusted-host url '
        for
        each next url in pip_index_urls if pip_index_urls contains more than 1
        url

        Note: In case pip_index_urls is not empty, the returned string will
        contain space at the end.
    """
    if not pip_index_urls:
        return ''

    index_url = pip_index_urls[0]
    extra_index_urls = pip_index_urls[1:]

    options = [f'--index-url {index_url} --trusted-host {index_url}']
    options.extend(
        f'--extra-index-url {extra_index_url} --trusted-host {extra_index_url}'
        for extra_index_url in extra_index_urls)

    return ' '.join(options) + ' '


_install_python_packages_script_template = '''
if ! [ -x "$(command -v pip)" ]; then
    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip
fi

PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet \
    --no-warn-script-location {index_url_options}{concat_package_list} && "$0" "$@"
'''


def _get_packages_to_install_command(
        package_list: Optional[List[str]] = None,
        pip_index_urls: Optional[List[str]] = None) -> List[str]:

    if not package_list:
        return []

    concat_package_list = ' '.join(
        [repr(str(package)) for package in package_list])
    index_url_options = make_index_url_options(pip_index_urls)
    install_python_packages_script = _install_python_packages_script_template.format(
        index_url_options=index_url_options,
        concat_package_list=concat_package_list)
    return ['sh', '-c', install_python_packages_script]


def _get_kfp_dsl_requirement() -> str:
    import kfp
    return f'kfp-dsl=={kfp.__version__}'


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
            f'Failed to dedent and clean up the source of function "{func.__name__}". It is probably not properly indented.'
        )

    return '\n'.join(func_code_lines)


def _maybe_make_unique(name: str, names: List[str]):
    if name not in names:
        return name

    for i in range(2, 100):
        unique_name = f'{name}_{i}'
        if unique_name not in names:
            return unique_name

    raise RuntimeError(f'Too many arguments with the name {name}')


def extract_component_interface(
    func: Callable,
    containerized: bool = False,
    description: Optional[str] = None,
    name: Optional[str] = None,
) -> structures.ComponentSpec:
    single_output_name_const = 'Output'

    signature = inspect.signature(func)
    parameters = list(signature.parameters.values())

    original_docstring = inspect.getdoc(func)

    try:
        import docstring_parser
    except ImportError as e:
        raise ImportError(dsl._kfp_dsl_import_error_msg) from e

    parsed_docstring = docstring_parser.parse(original_docstring)

    inputs = {}
    outputs = {}

    input_names = set()
    output_names = set()
    for parameter in parameters:
        parameter_type = type_annotations.maybe_strip_optional_from_annotation(
            parameter.annotation)
        passing_style = None
        io_name = parameter.name
        is_artifact_list = False

        if type_annotations.is_Input_Output_artifact_annotation(parameter_type):
            # passing_style is either type_annotations.InputAnnotation or
            # type_annotations.OutputAnnotation.
            passing_style = type_annotations.get_io_artifact_annotation(
                parameter_type)

            # parameter_type is a type like typing_extensions.Annotated[kfp.dsl.types.artifact_types.Artifact, <class 'kfp.dsl.types.type_annotations.OutputAnnotation'>] OR typing_extensions.Annotated[typing.List[kfp.dsl.types.artifact_types.Artifact], <class 'kfp.dsl.types.type_annotations.OutputAnnotation'>]

            is_artifact_list = type_annotations.is_list_of_artifacts(
                parameter_type.__origin__)

            parameter_type = type_annotations.get_io_artifact_class(
                parameter_type)
            if not type_annotations.is_artifact_class(parameter_type):
                raise ValueError(
                    f'Input[T] and Output[T] are only supported when T is an artifact or list of artifacts. Found `{io_name} with type {parameter_type}`'
                )

            if parameter.default is not inspect.Parameter.empty:
                if passing_style in [
                        type_annotations.OutputAnnotation,
                        type_annotations.OutputPath,
                ]:
                    raise ValueError(
                        'Default values for Output artifacts are not supported.'
                    )
                elif parameter.default is not None:
                    raise ValueError(
                        f'Optional Input artifacts may only have default value None. Got: {parameter.default}.'
                    )

        elif isinstance(
                parameter_type,
            (type_annotations.InputPath, type_annotations.OutputPath)):
            passing_style = type(parameter_type)
            parameter_type = parameter_type.type
            if parameter.default is not inspect.Parameter.empty and not (
                    passing_style == type_annotations.InputPath and
                    parameter.default is None):
                raise ValueError(
                    'Path inputs only support default values of None. Default'
                    ' values for outputs are not supported.')

        type_struct = type_utils._annotation_to_type_struct(parameter_type)
        if type_struct is None:
            raise TypeError(
                f'Missing type annotation for argument: {parameter.name}')

        if passing_style in [
                type_annotations.OutputAnnotation, type_annotations.OutputPath
        ]:
            if io_name == single_output_name_const:
                raise ValueError(
                    f'"{single_output_name_const}" is an invalid parameter name.'
                )
            io_name = _maybe_make_unique(io_name, output_names)
            output_names.add(io_name)
            if type_annotations.is_artifact_class(parameter_type):
                schema_version = parameter_type.schema_version
                output_spec = structures.OutputSpec(
                    type=type_utils.create_bundled_artifact_type(
                        type_struct, schema_version),
                    is_artifact_list=is_artifact_list)
            else:
                output_spec = structures.OutputSpec(type=type_struct)
            outputs[io_name] = output_spec
        else:
            io_name = _maybe_make_unique(io_name, input_names)
            input_names.add(io_name)
            type_ = type_utils.create_bundled_artifact_type(
                type_struct, parameter_type.schema_version
            ) if type_annotations.is_artifact_class(
                parameter_type) else type_struct
            default = None if parameter.default == inspect.Parameter.empty or type_annotations.is_artifact_class(
                parameter_type) else parameter.default
            optional = parameter.default is not inspect.Parameter.empty or type_utils.is_task_final_status_type(
                type_struct)
            input_spec = structures.InputSpec(
                type=type_,
                default=default,
                optional=optional,
                is_artifact_list=is_artifact_list,
            )

            inputs[io_name] = input_spec

    #Analyzing the return type annotations.
    return_ann = signature.return_annotation
    if not containerized:
        if hasattr(return_ann, '_fields'):  #NamedTuple
            # Getting field type annotations.
            # __annotations__ does not exist in python 3.5 and earlier
            # _field_types does not exist in python 3.9 and later
            field_annotations = getattr(return_ann, '__annotations__',
                                        None) or getattr(
                                            return_ann, '_field_types', None)
            for field_name in return_ann._fields:
                output_name = _maybe_make_unique(field_name, output_names)
                output_names.add(output_name)
                type_var = field_annotations.get(field_name)
                if type_annotations.is_list_of_artifacts(type_var):
                    artifact_cls = type_var.__args__[0]
                    output_spec = structures.OutputSpec(
                        type=type_utils.create_bundled_artifact_type(
                            artifact_cls.schema_title,
                            artifact_cls.schema_version),
                        is_artifact_list=True)
                elif type_annotations.is_artifact_class(type_var):
                    output_spec = structures.OutputSpec(
                        type=type_utils.create_bundled_artifact_type(
                            type_var.schema_title, type_var.schema_version))
                else:
                    type_struct = type_utils._annotation_to_type_struct(
                        type_var)
                    output_spec = structures.OutputSpec(type=type_struct)
                outputs[output_name] = output_spec
        # Deprecated dict-based way of declaring multiple outputs. Was only used by
        # the @component decorator
        elif isinstance(return_ann, dict):
            warnings.warn(
                'The ability to specify multiple outputs using the dict syntax'
                ' has been deprecated. It will be removed soon after release'
                ' 0.1.32. Please use typing.NamedTuple to declare multiple'
                ' outputs.')
            for output_name, output_type_annotation in return_ann.items():
                output_type_struct = type_utils._annotation_to_type_struct(
                    output_type_annotation)
                output_spec = structures.OutputSpec(type=output_type_struct)
                outputs[name] = output_spec
        elif signature.return_annotation is not None and signature.return_annotation != inspect.Parameter.empty:
            output_name = _maybe_make_unique(single_output_name_const,
                                             output_names)
            # Fixes exotic, but possible collision:
            #   `def func(output_path: OutputPath()) -> str: ...`
            output_names.add(output_name)
            return_ann = signature.return_annotation
            if type_annotations.is_list_of_artifacts(return_ann):
                artifact_cls = return_ann.__args__[0]
                output_spec = structures.OutputSpec(
                    type=type_utils.create_bundled_artifact_type(
                        artifact_cls.schema_title, artifact_cls.schema_version),
                    is_artifact_list=True)
            elif type_annotations.is_artifact_class(return_ann):
                output_spec = structures.OutputSpec(
                    type=type_utils.create_bundled_artifact_type(
                        return_ann.schema_title, return_ann.schema_version),
                    is_artifact_list=False)
            else:
                type_struct = type_utils._annotation_to_type_struct(return_ann)
                output_spec = structures.OutputSpec(type=type_struct)

            outputs[output_name] = output_spec
    elif return_ann != inspect.Parameter.empty and return_ann != structures.ContainerSpec:
        raise TypeError(
            'Return annotation should be either ContainerSpec or omitted for container components.'
        )

    component_name = name or _python_function_name_to_component_name(
        func.__name__)

    def assign_descriptions(
        inputs_or_outputs: Mapping[str, Union[structures.InputSpec,
                                              structures.OutputSpec]],
        docstring_params: List[docstring_parser.DocstringParam],
    ) -> None:
        """Assigns descriptions to InputSpec or OutputSpec for each component
        input/output found in the parsed docstring parameters."""
        docstring_inputs = {param.arg_name: param for param in docstring_params}
        for name, spec in inputs_or_outputs.items():
            if name in docstring_inputs:
                spec.description = docstring_inputs[name].description

    def parse_docstring_with_return_as_args(
            docstring: Union[str,
                             None]) -> Union[docstring_parser.Docstring, None]:
        """Modifies docstring so that a return section can be treated as an
        args section, then parses the docstring."""
        if docstring is None:
            return None

        # Returns and Return are the only two keywords docstring_parser uses for returns
        # use newline to avoid replacements that aren't in the return section header
        return_keywords = ['Returns:\n', 'Returns\n', 'Return:\n', 'Return\n']
        for keyword in return_keywords:
            if keyword in docstring:
                modified_docstring = docstring.replace(keyword.strip(), 'Args:')
                return docstring_parser.parse(modified_docstring)

        return None

    assign_descriptions(inputs, parsed_docstring.params)

    modified_parsed_docstring = parse_docstring_with_return_as_args(
        original_docstring)
    if modified_parsed_docstring is not None:
        assign_descriptions(outputs, modified_parsed_docstring.params)

    description = get_pipeline_description(
        decorator_description=description,
        docstring=parsed_docstring,
    )

    return structures.ComponentSpec(
        name=component_name,
        description=description,
        inputs=inputs or None,
        outputs=outputs or None,
        implementation=structures.Implementation(),
    )


def _get_command_and_args_for_lightweight_component(
        func: Callable) -> Tuple[List[str], List[str]]:
    imports_source = [
        'import kfp',
        'from kfp import dsl',
        'from kfp.dsl import *',
        'from typing import *',
    ] + custom_artifact_types.get_custom_artifact_type_import_statements(func)

    func_source = _get_function_source_definition(func)
    source = textwrap.dedent('''
        {imports_source}

        {func_source}\n''').format(
        imports_source='\n'.join(imports_source), func_source=func_source)
    command = [
        'sh',
        '-ec',
        textwrap.dedent('''\
                    program_path=$(mktemp -d)
                    printf "%s" "$0" > "$program_path/ephemeral_component.py"
                    python3 -m kfp.dsl.executor_main \
                        --component_module_path \
                        "$program_path/ephemeral_component.py" \
                        "$@"
                '''),
        source,
    ]

    args = [
        '--executor_input',
        placeholders.ExecutorInputPlaceholder(),
        '--function_to_execute',
        func.__name__,
    ]

    return command, args


def _get_command_and_args_for_containerized_component(
        function_name: str) -> Tuple[List[str], List[str]]:
    command = [
        'python3',
        '-m',
        'kfp.dsl.executor_main',
    ]

    args = [
        '--executor_input',
        placeholders.ExecutorInputPlaceholder()._to_string(),
        '--function_to_execute',
        function_name,
    ]
    return command, args


def create_component_from_func(
    func: Callable,
    base_image: Optional[str] = None,
    target_image: Optional[str] = None,
    packages_to_install: List[str] = None,
    pip_index_urls: Optional[List[str]] = None,
    output_component_file: Optional[str] = None,
    install_kfp_package: bool = True,
    kfp_package_path: Optional[str] = None,
) -> python_component.PythonComponent:
    """Implementation for the @component decorator.

    The decorator is defined under component_decorator.py. See the
    decorator for the canonical documentation for this function.
    """
    packages_to_install = packages_to_install or []

    if install_kfp_package and target_image is None:
        if kfp_package_path is None:
            kfp_package_path = _get_kfp_dsl_requirement()
        packages_to_install.append(kfp_package_path)

    packages_to_install_command = _get_packages_to_install_command(
        package_list=packages_to_install, pip_index_urls=pip_index_urls)

    command = []
    args = []
    if base_image is None:
        base_image = _DEFAULT_BASE_IMAGE

    component_image = base_image

    if target_image:
        component_image = target_image
        command, args = _get_command_and_args_for_containerized_component(
            function_name=func.__name__,)
    else:
        command, args = _get_command_and_args_for_lightweight_component(
            func=func)

    component_spec = extract_component_interface(func)
    component_spec.implementation = structures.Implementation(
        container=structures.ContainerSpecImplementation(
            image=component_image,
            command=packages_to_install_command + command,
            args=args,
        ))

    module_path = pathlib.Path(inspect.getsourcefile(func))
    module_path.resolve()

    component_name = _python_function_name_to_component_name(func.__name__)
    component_info = ComponentInfo(
        name=component_name,
        function_name=func.__name__,
        func=func,
        target_image=target_image,
        module_path=module_path,
        component_spec=component_spec,
        output_component_file=output_component_file,
        base_image=base_image,
        packages_to_install=packages_to_install,
        pip_index_urls=pip_index_urls)

    if REGISTERED_MODULES is not None:
        REGISTERED_MODULES[component_name] = component_info

    if output_component_file:
        component_spec.save_to_component_yaml(output_component_file)

    return python_component.PythonComponent(
        component_spec=component_spec, python_func=func)


def make_input_for_parameterized_container_component_function(
    name: str, annotation: Union[Type[List[artifact_types.Artifact]],
                                 Type[artifact_types.Artifact]]
) -> Union[placeholders.Placeholder, container_component_artifact_channel
           .ContainerComponentArtifactChannel]:
    if type_annotations.is_input_artifact(annotation):

        if type_annotations.is_list_of_artifacts(annotation.__origin__):
            return placeholders.InputListOfArtifactsPlaceholder(name)
        else:
            return container_component_artifact_channel.ContainerComponentArtifactChannel(
                io_type='input', var_name=name)

    elif type_annotations.is_output_artifact(annotation):

        if type_annotations.is_list_of_artifacts(annotation.__origin__):
            return placeholders.OutputListOfArtifactsPlaceholder(name)
        else:
            return container_component_artifact_channel.ContainerComponentArtifactChannel(
                io_type='output', var_name=name)

    elif isinstance(
            annotation,
        (type_annotations.OutputAnnotation, type_annotations.OutputPath)):
        return placeholders.OutputParameterPlaceholder(name)

    else:
        placeholder = placeholders.InputValuePlaceholder(name)
        # small hack to encode the runtime value's type for a custom json.dumps function
        if (annotation == task_final_status.PipelineTaskFinalStatus or
                type_utils.is_task_final_status_type(annotation)):
            placeholder._ir_type = 'STRUCT'
        else:
            placeholder._ir_type = type_utils.get_parameter_type_name(
                annotation)
        return placeholder


def create_container_component_from_func(
        func: Callable) -> container_component_class.ContainerComponent:
    """Implementation for the @container_component decorator.

    The decorator is defined under container_component_decorator.py. See
    the decorator for the canonical documentation for this function.
    """

    component_spec = extract_component_interface(func, containerized=True)
    signature = inspect.signature(func)
    parameters = list(signature.parameters.values())
    arg_list = []
    for parameter in parameters:
        parameter_type = type_annotations.maybe_strip_optional_from_annotation(
            parameter.annotation)
        arg_list.append(
            make_input_for_parameterized_container_component_function(
                parameter.name, parameter_type))

    container_spec = func(*arg_list)
    container_spec_implementation = structures.ContainerSpecImplementation.from_container_spec(
        container_spec)
    component_spec.implementation = structures.Implementation(
        container_spec_implementation)
    component_spec._validate_placeholders()
    return container_component_class.ContainerComponent(component_spec, func)


def create_graph_component_from_func(
    func: Callable,
    name: Optional[str] = None,
    description: Optional[str] = None,
    display_name: Optional[str] = None,
) -> graph_component.GraphComponent:
    """Implementation for the @pipeline decorator.

    The decorator is defined under pipeline_context.py. See the
    decorator for the canonical documentation for this function.
    """

    component_spec = extract_component_interface(
        func,
        description=description,
        name=name,
    )
    return graph_component.GraphComponent(
        component_spec=component_spec,
        pipeline_func=func,
        display_name=display_name,
    )


def get_pipeline_description(
    decorator_description: Union[str, None],
    docstring: 'docstring_parser.Docstring',
) -> Union[str, None]:
    """Obtains the correct pipeline description from the pipeline decorator's
    description argument and the parsed docstring.

    Gives precedence to the decorator argument.
    """
    if decorator_description:
        return decorator_description

    short_description = docstring.short_description
    long_description = docstring.long_description
    docstring_description = short_description + '\n' + long_description if (
        short_description and long_description) else short_description
    return docstring_description.strip() if docstring_description else None
