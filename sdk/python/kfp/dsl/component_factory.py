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
import base64
import dataclasses
import gzip
import inspect
import io
import itertools
import pathlib
import re
import tarfile
import textwrap
from typing import Any, Callable, Dict, List, Mapping, Optional, Tuple, Type, Union
import warnings

import docstring_parser
import kfp
from kfp import dsl
from kfp.dsl import container_component_artifact_channel
from kfp.dsl import container_component_class
from kfp.dsl import graph_component
from kfp.dsl import pipeline_config
from kfp.dsl import placeholders
from kfp.dsl import python_component
from kfp.dsl import structures
from kfp.dsl import task_final_status
from kfp.dsl.component_task_config import TaskConfigPassthrough
from kfp.dsl.task_config import TaskConfig
from kfp.dsl.types import artifact_types
from kfp.dsl.types import custom_artifact_types
from kfp.dsl.types import type_annotations
from kfp.dsl.types import type_utils

_DEFAULT_BASE_IMAGE = 'python:3.11'
SINGLE_OUTPUT_NAME = 'Output'


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
    pip_trusted_hosts: Optional[List[str]] = None
    use_venv: bool = False
    task_config_passthroughs: Optional[List[TaskConfigPassthrough]] = None


# A map from function_name to components.  This is always populated when a
# module containing KFP components is loaded. Primarily used by KFP CLI
# component builder to package components in a file into containers.
REGISTERED_MODULES = None


def _python_function_name_to_component_name(name):
    name_with_spaces = re.sub(' +', ' ', name.replace('_', ' ')).strip(' ')
    return name_with_spaces[0].upper() + name_with_spaces[1:]


def make_index_url_options(pip_index_urls: Optional[List[str]],
                           pip_trusted_hosts: Optional[List[str]]) -> str:
    """Generates index URL options for the pip install command based on the
    provided pip_index_urls and pip_trusted_hosts.

    Args:
        pip_index_urls (Optional[List[str]]): Optional list of pip index URLs.
        pip_trusted_hosts (Optional[List[str]]): Optional list of pip trusted hosts.

    Returns:
        str:
            - An empty string if pip_index_urls is empty or None.
            - '--index-url url ' if pip_index_urls contains 1 URL.
            - The above followed by '--extra-index-url url ' for each additional URL in pip_index_urls
            if pip_index_urls contains more than 1 URL.
            - If pip_trusted_hosts is None:
                - The above followed by '--trusted-host url ' for each URL in pip_index_urls.
            - If pip_trusted_hosts is an empty List.
                - No --trusted-host information will be added
            - If pip_trusted_hosts contains any URLs:
                - The above followed by '--trusted-host url ' for each URL in pip_trusted_hosts.
    Note:
        In case pip_index_urls is not empty, the returned string will contain a space at the end.
    """
    if not pip_index_urls:
        return ''

    index_url = pip_index_urls[0]
    extra_index_urls = pip_index_urls[1:]

    options = [f'--index-url {index_url}']
    options.extend(f'--extra-index-url {extra_index_url}'
                   for extra_index_url in extra_index_urls)

    if pip_trusted_hosts is None:
        options.extend([f'--trusted-host {index_url}'])
        options.extend(f'--trusted-host {extra_index_url}'
                       for extra_index_url in extra_index_urls)
    elif len(pip_trusted_hosts) > 0:
        options.extend(f'--trusted-host {trusted_host}'
                       for trusted_host in pip_trusted_hosts)

    return ' '.join(options) + ' '


def make_pip_install_command(
    install_parts: List[str],
    index_url_options: str,
) -> str:
    concat_package_list = ' '.join(
        [repr(str(package)) for package in install_parts])
    return f'python3 -m pip install --quiet --no-warn-script-location {index_url_options}{concat_package_list}'


_install_python_packages_script_template = '''
if ! [ -x "$(command -v pip)" ]; then
    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip
fi

PIP_DISABLE_PIP_VERSION_CHECK=1 {pip_install_commands} && "$0" "$@"
'''

# Creates and activates a virtual environment in a temporary directory.
# The environment inherits the system site packages.
_use_venv_script_template = '''
export PIP_DISABLE_PIP_VERSION_CHECK=1
tmp=$(mktemp -d)
python3 -m venv "$tmp/venv" --system-site-packages
. "$tmp/venv/bin/activate"
'''


def _get_packages_to_install_command(
    kfp_package_path: Optional[str] = None,
    pip_index_urls: Optional[List[str]] = None,
    packages_to_install: Optional[List[str]] = None,
    install_kfp_package: bool = True,
    target_image: Optional[str] = None,
    pip_trusted_hosts: Optional[List[str]] = None,
    use_venv: bool = False,
) -> List[str]:
    packages_to_install = packages_to_install or []
    kfp_in_user_pkgs = any(pkg.startswith('kfp') for pkg in packages_to_install)
    # if the user doesn't say "don't install", they aren't building a
    # container component, and they haven't already specified a KFP dep
    # themselves, we install KFP for them
    inject_kfp_install = install_kfp_package and target_image is None and not kfp_in_user_pkgs
    if not inject_kfp_install and not packages_to_install:
        return []
    pip_install_strings = []
    index_url_options = make_index_url_options(pip_index_urls,
                                               pip_trusted_hosts)

    # Install packages before KFP. This allows us to
    # control where we source kfp-pipeline-spec.
    # This is particularly useful for development and
    # CI use-case when you want to install the spec
    # from source.
    if packages_to_install:
        user_packages_pip_install_command = make_pip_install_command(
            install_parts=packages_to_install,
            index_url_options=index_url_options,
        )
        pip_install_strings.append(user_packages_pip_install_command)
        if inject_kfp_install:
            pip_install_strings.append(' && ')

    if inject_kfp_install:
        if use_venv:
            pip_install_strings.append(_use_venv_script_template)
        if kfp_package_path:
            kfp_pip_install_command = make_pip_install_command(
                install_parts=[kfp_package_path],
                index_url_options=index_url_options,
            )
        else:
            kfp_pip_install_command = make_pip_install_command(
                install_parts=[
                    f'kfp=={kfp.__version__}',
                    '--no-deps',
                    'typing-extensions>=3.7.4,<5; python_version<"3.9"',
                ],
                index_url_options=index_url_options,
            )
        pip_install_strings.append(kfp_pip_install_command)

    return [
        'sh', '-c',
        _install_python_packages_script_template.format(
            pip_install_commands=' '.join(pip_install_strings))
    ]


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


def maybe_make_unique(name: str, names: List[str]):
    if name not in names:
        return name

    for i in range(2, 100):
        unique_name = f'{name}_{i}'
        if unique_name not in names:
            return unique_name

    raise RuntimeError(f'Too many arguments with the name {name}')


def get_name_to_specs(
    signature: inspect.Signature,
    containerized: bool = False,
) -> Tuple[Dict[str, Any], Dict[str, Any]]:
    """Returns two dictionaries.

    The first is a mapping of input name to input annotation. The second
    is a mapping of output name to output annotation.
    """
    func_params = list(signature.parameters.values())

    name_to_input_specs = {}
    name_to_output_specs = {}

    ### handle function parameter annotations ###
    for func_param in func_params:
        name = func_param.name
        if name == SINGLE_OUTPUT_NAME:
            raise ValueError(
                f'"{SINGLE_OUTPUT_NAME}" is an invalid parameter name.')
        # Stripping Optional from Optional[<type>] is the only processing done
        # on annotations in this flow. Other than that, we extract the raw
        # annotation and process later.
        annotation = type_annotations.maybe_strip_optional_from_annotation(
            func_param.annotation)

        # no annotation
        if annotation == inspect._empty:
            raise TypeError(f'Missing type annotation for argument: {name}')

        # Exclude EmbeddedInput[...] from interface (runtime-only)
        elif type_annotations.is_embedded_input_annotation(annotation):
            continue
        # is Input[Artifact], Input[List[<Artifact>]], <param> (e.g., str), or InputPath(<param>)
        elif (type_annotations.is_artifact_wrapped_in_Input(annotation) or
              isinstance(
                  annotation,
                  type_annotations.InputPath,
              ) or type_utils.is_parameter_type(annotation)):
            name_to_input_specs[maybe_make_unique(
                name, list(name_to_input_specs))] = make_input_spec(
                    annotation, func_param)
        # is Artifact annotation (e.g., Artifact, Dataset, etc.)
        # or List[<Artifact>]
        elif type_annotations.issubclass_of_artifact(
                annotation) or type_annotations.is_list_of_artifacts(
                    annotation):
            if containerized:
                raise TypeError(
                    f"Container Components must wrap input and output artifact annotations with Input/Output type markers (Input[<artifact>] or Output[<artifact>]). Got function input '{name}' with annotation {annotation}."
                )
            name_to_input_specs[maybe_make_unique(
                name, list(name_to_input_specs))] = make_input_spec(
                    annotation, func_param)

        # is Output[Artifact] or OutputPath(<param>)
        elif type_annotations.is_artifact_wrapped_in_Output(
                annotation) or isinstance(annotation,
                                          type_annotations.OutputPath):
            name_to_output_specs[maybe_make_unique(
                name,
                list(name_to_output_specs))] = make_output_spec(annotation)

        # parameter type
        else:
            type_string, _ = type_utils.annotation_to_type_struct_and_literals(
                annotation)
            name_to_input_specs[maybe_make_unique(
                name, list(name_to_input_specs))] = make_input_spec(
                    type_string, func_param)

    ### handle return annotations ###
    return_ann = signature.return_annotation

    # validate container component returns
    if containerized:
        if return_ann not in [
                inspect.Parameter.empty,
                structures.ContainerSpec,
        ]:
            raise TypeError(
                'Return annotation should be either ContainerSpec or omitted for container components.'
            )
    # ignore omitted returns
    elif return_ann is None or return_ann == inspect.Parameter.empty:
        pass
    # is NamedTuple
    elif hasattr(return_ann, '_fields'):
        # Getting field type annotations.
        # __annotations__ does not exist in python 3.5 and earlier
        # _field_types does not exist in python 3.9 and later
        field_annotations = getattr(return_ann, '__annotations__',
                                    None) or getattr(return_ann, '_field_types')
        for name in return_ann._fields:
            annotation = field_annotations[name]
            if not type_annotations.is_list_of_artifacts(
                    annotation) and not type_annotations.is_artifact_class(
                        annotation):
                annotation = type_utils._annotation_to_type_struct(annotation)
            name_to_output_specs[maybe_make_unique(
                name,
                list(name_to_output_specs))] = make_output_spec(annotation)
    # is deprecated dict returns style
    elif isinstance(return_ann, dict):
        warnings.warn(
            'The ability to specify multiple outputs using the dict syntax'
            ' has been deprecated. It will be removed soon after release'
            ' 0.1.32. Please use typing.NamedTuple to declare multiple'
            ' outputs.', DeprecationWarning)
        for output_name, output_type_annotation in return_ann.items():
            output_type, _ = type_utils.annotation_to_type_struct_and_literals(
                output_type_annotation)
            name_to_output_specs[maybe_make_unique(
                output_name, list(name_to_output_specs))] = output_type
    # is the simple single return case (can be `-> <param>` or `-> Artifact`)
    # treated the same way, since processing is done in inner functions
    else:
        name_to_output_specs[maybe_make_unique(
            SINGLE_OUTPUT_NAME,
            list(name_to_output_specs))] = make_output_spec(return_ann)
    return name_to_input_specs, name_to_output_specs


def canonicalize_annotation(annotation: Any):
    """Does cleaning on annotations that are common between input and output
    annotations."""
    if type_annotations.is_Input_Output_artifact_annotation(annotation):
        annotation = type_annotations.strip_Input_or_Output_marker(annotation)
    if isinstance(annotation,
                  (type_annotations.InputPath, type_annotations.OutputPath)):
        annotation = annotation.type
    return annotation


def make_input_output_spec_args(annotation: Any) -> Dict[str, Any]:
    """Gets a dict of kwargs shared between InputSpec and OutputSpec."""
    is_artifact_list = type_annotations.is_list_of_artifacts(annotation)
    if is_artifact_list:
        annotation = type_annotations.get_inner_type(annotation)

    literals = None

    if type_annotations.issubclass_of_artifact(annotation):
        typ = type_utils.create_bundled_artifact_type(annotation.schema_title,
                                                      annotation.schema_version)
    else:
        typ, literals = type_utils.annotation_to_type_struct_and_literals(
            annotation)
    return {
        'type': typ,
        'is_artifact_list': is_artifact_list,
        'literals': literals,
    }


def make_output_spec(annotation: Any) -> structures.OutputSpec:
    annotation = canonicalize_annotation(annotation)
    args = make_input_output_spec_args(annotation)
    args.pop('literals', None)
    return structures.OutputSpec(**args)


def make_input_spec(annotation: Any,
                    inspect_param: inspect.Parameter) -> structures.InputSpec:
    """Makes an InputSpec from a cleaned output annotation."""
    annotation = canonicalize_annotation(annotation)
    input_output_spec_args = make_input_output_spec_args(annotation)

    if (type_annotations.issubclass_of_artifact(annotation) or
            input_output_spec_args['is_artifact_list']
       ) and inspect_param.default not in {None, inspect._empty}:
        raise ValueError(
            f'Optional Input artifacts may only have default value None. Got: {inspect_param.default}.'
        )

    default = None if inspect_param.default == inspect.Parameter.empty or type_annotations.issubclass_of_artifact(
        annotation) else inspect_param.default

    optional = (
        inspect_param.default is not inspect.Parameter.empty or
        type_utils.is_task_final_status_type(
            getattr(inspect_param.annotation, '__name__', '')) or
        type_utils.is_task_config_type(
            getattr(inspect_param.annotation, '__name__', '')))
    return structures.InputSpec(
        **input_output_spec_args,
        default=default,
        optional=optional,
    )


def extract_component_interface(
    func: Callable,
    containerized: bool = False,
    description: Optional[str] = None,
    name: Optional[str] = None,
) -> structures.ComponentSpec:

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
                             None]) -> Optional[docstring_parser.Docstring]:
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

    signature = inspect.signature(func)
    name_to_input_spec, name_to_output_spec = get_name_to_specs(
        signature, containerized)
    original_docstring = inspect.getdoc(func)
    parsed_docstring = docstring_parser.parse(original_docstring)

    assign_descriptions(name_to_input_spec, parsed_docstring.params)

    modified_parsed_docstring = parse_docstring_with_return_as_args(
        original_docstring)
    if modified_parsed_docstring is not None:
        assign_descriptions(name_to_output_spec,
                            modified_parsed_docstring.params)

    description = get_pipeline_description(
        decorator_description=description,
        docstring=parsed_docstring,
    )

    component_name = name or _python_function_name_to_component_name(
        func.__name__)
    return structures.ComponentSpec(
        name=component_name,
        description=description,
        inputs=name_to_input_spec or None,
        outputs=name_to_output_spec or None,
        implementation=structures.Implementation(),
    )


EXECUTOR_MODULE = 'kfp.dsl.executor_main'
CONTAINERIZED_PYTHON_COMPONENT_COMMAND = [
    'python3',
    '-m',
    EXECUTOR_MODULE,
]


def _get_command_and_args_for_lightweight_component(
    func: Callable,
    additional_funcs: Optional[List[Callable]] = None,
    embedded_artifact_path: Optional[str] = None,
    _helper_source_template: Optional[str] = None,
) -> Tuple[List[str], List[str]]:
    imports_source = [
        'import kfp',
        'from kfp import dsl',
        'from kfp.dsl import *',
        'from typing import *',
    ] + custom_artifact_types.get_custom_artifact_type_import_statements(func)

    func_source = _get_function_source_definition(func)
    additional_funcs_source = ''
    if additional_funcs:
        additional_funcs_source = '\n\n' + '\n\n'.join(
            _get_function_source_definition(_f) for _f in additional_funcs)
    # If an embedded_artifact_path is provided, build the archive and generate
    # a shared extraction helper.
    helper_source = None
    if embedded_artifact_path:
        asset_path = pathlib.Path(embedded_artifact_path)
        if not asset_path.exists():
            raise ValueError(
                f'embedded_artifact_path does not exist: {embedded_artifact_path}'
            )

        buf = io.BytesIO()
        with tarfile.open(fileobj=buf, mode='w') as tar:
            if asset_path.is_dir():
                tar.add(asset_path, arcname='.')
            else:
                tar.add(asset_path, arcname=asset_path.name)

        archive_bytes = buf.getvalue()

        compressed_bytes = gzip.compress(archive_bytes)
        compressed_size_mb = len(compressed_bytes) / (1024 * 1024)
        if compressed_size_mb > 1:
            warnings.warn(
                f'Embedded artifact archive is large ({compressed_size_mb:.1f}MB compressed). '
                f'Consider moving large assets to a container image or object store.',
                UserWarning,
                stacklevel=4)

        archive_b64 = base64.b64encode(compressed_bytes).decode('ascii')

        if _helper_source_template:
            # Allow callers to inject a custom helper source via a template.
            # Use literal replacement to avoid str.format interpreting other braces
            # in the helper source (e.g., f-strings) as format fields.
            helper_source = _helper_source_template.replace(
                '{embedded_archive}', archive_b64)
        else:
            file_basename = asset_path.name if asset_path.is_file() else None
            helper_source = _generate_shared_extraction_helper(
                archive_b64, file_basename)

    helper_block = f'\n\n{helper_source}\n' if helper_source else ''
    source = textwrap.dedent('''
        {imports_source}{additional_funcs_source}{helper_block}

        {func_source}\n''').format(
        imports_source='\n'.join(imports_source),
        additional_funcs_source=additional_funcs_source,
        helper_block=helper_block,
        func_source=func_source)
    command = [
        'sh',
        '-ec',
        textwrap.dedent(f'''\
                    program_path=$(mktemp -d)

                    printf "%s" "$0" > "$program_path/ephemeral_component.py"
                    _KFP_RUNTIME=true python3 -m {EXECUTOR_MODULE} \
                        --component_module_path \
                        "$program_path/ephemeral_component.py" \
                        "$@"
                '''),
        source,
    ]

    args = [
        '--executor_input',
        dsl.PIPELINE_TASK_EXECUTOR_INPUT_PLACEHOLDER,
        '--function_to_execute',
        func.__name__,
    ]

    return command, args


def _generate_shared_extraction_helper(embedded_archive_b64: str,
                                       file_basename: Optional[str] = None
                                      ) -> str:
    """Generate shared archive extraction helper source code.

    Produces Python source that extracts a tar.gz archive (provided as
    base64) into a temporary directory, then prepends that directory to
    sys.path. When a single file was embedded, sets
    __KFP_EMBEDDED_ASSET_FILE to the extracted file path; otherwise only
    __KFP_EMBEDDED_ASSET_DIR is set.
    """
    file_assignment = ''
    if file_basename:
        file_assignment = f"\n__KFP_EMBEDDED_ASSET_FILE = __kfp_os.path.join(__KFP_EMBEDDED_ASSET_DIR, '{file_basename}')\n"
    return f'''__KFP_EMBEDDED_ARCHIVE_B64 = '{embedded_archive_b64}'

import base64 as __kfp_b64
import io as __kfp_io
import os as __kfp_os
import sys as __kfp_sys
import tarfile as __kfp_tarfile
import tempfile as __kfp_tempfile

# Extract embedded archive at import time to ensure sys.path and globals are set
__kfp_tmpdir = __kfp_tempfile.TemporaryDirectory()
__KFP_EMBEDDED_ASSET_DIR = __kfp_tmpdir.name
try:
    __kfp_bytes = __kfp_b64.b64decode(__KFP_EMBEDDED_ARCHIVE_B64.encode('ascii'))
    with __kfp_tarfile.open(fileobj=__kfp_io.BytesIO(__kfp_bytes), mode='r:gz') as __kfp_tar:
        __kfp_tar.extractall(path=__KFP_EMBEDDED_ASSET_DIR)
except Exception as __kfp_e:
    raise RuntimeError(f'Failed to extract embedded archive: {{__kfp_e}}')

# Always prepend the extracted directory to sys.path for import resolution
if __KFP_EMBEDDED_ASSET_DIR not in __kfp_sys.path:
    __kfp_sys.path.insert(0, __KFP_EMBEDDED_ASSET_DIR)
{file_assignment}
'''


def create_notebook_component_from_func(
    func: Callable,
    *,
    notebook_path: str,
    base_image: Optional[str] = None,
    packages_to_install: Optional[List[str]] = None,
    pip_index_urls: Optional[List[str]] = None,
    output_component_file: Optional[str] = None,
    install_kfp_package: bool = True,
    kfp_package_path: Optional[str] = None,
    pip_trusted_hosts: Optional[List[str]] = None,
    use_venv: bool = False,
    task_config_passthroughs: Optional[List[TaskConfigPassthrough]] = None,
) -> python_component.PythonComponent:
    """Builds a notebook-based component by delegating to
    create_component_from_func.

    Validates the notebook path and constructs a helper source template
    that executes the embedded notebook at runtime. The common
    component-building logic is reused by passing the template and the
    notebook path as an embedded artifact.
    """

    # Resolve default packages for notebooks when not specified
    if packages_to_install is None:
        packages_to_install = [
            'nbclient>=0.10,<1', 'ipykernel>=6,<7', 'jupyter_client>=7,<9'
        ]

    # Validate notebook path and determine relpath
    nb_path = pathlib.Path(notebook_path)
    if not nb_path.exists():
        raise ValueError(f'Notebook path does not exist: {notebook_path}')

    if nb_path.is_dir():
        # Ignore .ipynb_checkpoints directories
        ipynbs = [
            p for p in nb_path.rglob('*.ipynb')
            if p.is_file() and '.ipynb_checkpoints' not in p.parts
        ]
        if len(ipynbs) == 0:
            raise ValueError(
                f'No .ipynb files found in directory: {notebook_path}')
        if len(ipynbs) > 1:
            raise ValueError(
                f'Multiple .ipynb files found in directory: {notebook_path}. Exactly one is required.'
            )
        nb_file = ipynbs[0]
        notebook_relpath = nb_file.relative_to(nb_path).as_posix()
    else:
        if nb_path.suffix.lower() != '.ipynb':
            raise ValueError(
                f'Invalid notebook_path (expected .ipynb file): {notebook_path}'
            )
        notebook_relpath = nb_path.name

    # Build the helper source template with a placeholder for the embedded archive
    from kfp.dsl.templates.notebook_executor import get_notebook_executor_source
    helper_template = get_notebook_executor_source('{embedded_archive}',
                                                   notebook_relpath)

    # Delegate to the common component creation using the embedded artifact path
    return create_component_from_func(
        func=func,
        base_image=base_image,
        target_image=None,
        embedded_artifact_path=notebook_path,
        _helper_source_template=helper_template,
        packages_to_install=packages_to_install or [],
        pip_index_urls=pip_index_urls,
        output_component_file=output_component_file,
        install_kfp_package=install_kfp_package,
        kfp_package_path=kfp_package_path,
        pip_trusted_hosts=pip_trusted_hosts,
        use_venv=use_venv,
        additional_funcs=None,
        task_config_passthroughs=task_config_passthroughs,
    )


def _get_command_and_args_for_containerized_component(
        function_name: str) -> Tuple[List[str], List[str]]:

    args = [
        '--executor_input',
        dsl.PIPELINE_TASK_EXECUTOR_INPUT_PLACEHOLDER,
        '--function_to_execute',
        function_name,
    ]
    return CONTAINERIZED_PYTHON_COMPONENT_COMMAND, args


def create_component_from_func(
    func: Callable,
    base_image: Optional[str] = None,
    target_image: Optional[str] = None,
    embedded_artifact_path: Optional[str] = None,
    packages_to_install: List[str] = None,
    pip_index_urls: Optional[List[str]] = None,
    output_component_file: Optional[str] = None,
    install_kfp_package: bool = True,
    kfp_package_path: Optional[str] = None,
    pip_trusted_hosts: Optional[List[str]] = None,
    use_venv: bool = False,
    additional_funcs: Optional[List[Callable]] = None,
    task_config_passthroughs: Optional[List[TaskConfigPassthrough]] = None,
    _helper_source_template: Optional[str] = None,
) -> python_component.PythonComponent:
    """Implementation for the @component decorator.

    The decorator is defined under component_decorator.py. See the
    decorator for the canonical documentation for this function.
    """

    packages_to_install_command = _get_packages_to_install_command(
        install_kfp_package=install_kfp_package,
        target_image=target_image,
        kfp_package_path=kfp_package_path,
        packages_to_install=packages_to_install,
        pip_index_urls=pip_index_urls,
        pip_trusted_hosts=pip_trusted_hosts,
        use_venv=use_venv,
    )

    command = []
    args = []
    if base_image is None:
        base_image = _DEFAULT_BASE_IMAGE
        warnings.warn(
            ("The default base_image used by the @dsl.component decorator will switch from 'python:3.11' to 'python:3.12' on Oct 1, 2027. To ensure your existing components work with versions of the KFP SDK released after that date, you should provide an explicit base_image argument and ensure your component works as intended on Python 3.12."
            ),
            FutureWarning,
            stacklevel=3,
        )

    component_image = base_image

    if target_image:
        component_image = target_image
        command, args = _get_command_and_args_for_containerized_component(
            function_name=func.__name__,)
    else:
        command, args = _get_command_and_args_for_lightweight_component(
            func=func,
            additional_funcs=additional_funcs,
            embedded_artifact_path=embedded_artifact_path,
            _helper_source_template=_helper_source_template,
        )

    component_spec = extract_component_interface(func)
    # Attach task_config_passthroughs to the ComponentSpec structure if provided.
    if task_config_passthroughs:
        component_spec.task_config_passthroughs = task_config_passthroughs
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
        pip_index_urls=pip_index_urls,
        pip_trusted_hosts=pip_trusted_hosts,
        task_config_passthroughs=task_config_passthroughs)

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
    if type_annotations.is_artifact_wrapped_in_Input(annotation):

        if type_annotations.is_list_of_artifacts(annotation.__origin__):
            return placeholders.InputListOfArtifactsPlaceholder(name)
        else:
            return container_component_artifact_channel.ContainerComponentArtifactChannel(
                io_type='input', var_name=name)

    elif type_annotations.is_artifact_wrapped_in_Output(annotation):

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
        elif annotation == TaskConfig:
            # Treat as STRUCT for IR and use reserved input name at runtime.
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
    pipeline_config: pipeline_config.PipelineConfig = None,
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
        pipeline_config=pipeline_config,
    )


def get_pipeline_description(
    decorator_description: Union[str, None],
    docstring: docstring_parser.Docstring,
) -> Optional[str]:
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
