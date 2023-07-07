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

import contextlib
import logging
import pathlib
import shutil
import subprocess
import sys
import tempfile
from typing import List, Optional
import warnings

import click

_DOCKER_IS_PRESENT = True
try:
    import docker
except ImportError:
    _DOCKER_IS_PRESENT = False

import kfp as kfp
from kfp.dsl import component_factory
from kfp.dsl import kfp_config
from kfp.dsl import utils

_REQUIREMENTS_TXT = 'runtime-requirements.txt'

_DOCKERFILE = 'Dockerfile'

# TODO: merge kfp_package_path into runtime-requirements.txt, once we have
# kfp_runtime package that is dependency-free.
_DOCKERFILE_TEMPLATE = '''
FROM {base_image}

WORKDIR {component_root_dir}
COPY {requirements_file} {requirements_file}
RUN pip install {index_urls}--no-cache-dir -r {requirements_file}
{maybe_copy_kfp_package}
RUN pip install {index_urls}--no-cache-dir {kfp_package_path}
COPY . .
'''

_DOCKERIGNORE = '.dockerignore'

# Location in which to write out shareable YAML for components.
_COMPONENT_METADATA_DIR = 'component_metadata'

_DOCKERIGNORE_TEMPLATE = f'''
{_COMPONENT_METADATA_DIR}/
'''

# Location at which v2 Python function-based components will stored
# in containerized components.
_COMPONENT_ROOT_DIR = pathlib.Path('/usr/local/src/kfp/components')


@contextlib.contextmanager
def _registered_modules():
    registered_modules = {}
    component_factory.REGISTERED_MODULES = registered_modules
    try:
        yield registered_modules
    finally:
        component_factory.REGISTERED_MODULES = None


class ComponentBuilder():
    """Helper class for building containerized v2 KFP components."""

    def __init__(
        self,
        context_directory: pathlib.Path,
        kfp_package_path: Optional[pathlib.Path] = None,
        component_filepattern: str = '**/*.py',
    ):
        """ComponentBuilder builds containerized components.

        Args:
            context_directory: Directory containing one or more Python files
            with one or more KFP v2 components.
            kfp_package_path: Path to a pip-installable location for KFP.
                This can either be pointing to KFP SDK root directory located in
                a local clone of the KFP repo, or a git+https location.
                If left empty, defaults to KFP on PyPI.
        """
        self._context_directory = context_directory
        self._dockerfile = self._context_directory / _DOCKERFILE
        self._component_filepattern = component_filepattern
        self._components: List[
            component_factory.component_factory.ComponentInfo] = []

        # This is only set if we need to install KFP from local copy.
        self._maybe_copy_kfp_package = ''

        if kfp_package_path is None:
            self._kfp_package_path = f'kfp=={kfp.__version__}'
        elif kfp_package_path.is_dir():
            logging.info(
                f'Building KFP package from local directory {kfp_package_path}')
            temp_dir = pathlib.Path(tempfile.mkdtemp())
            try:
                subprocess.run([
                    'python3',
                    kfp_package_path / 'setup.py',
                    'bdist_wheel',
                    '--dist-dir',
                    str(temp_dir),
                ],
                               cwd=kfp_package_path)
                wheel_files = list(temp_dir.glob('*.whl'))
                if len(wheel_files) != 1:
                    logging.error(
                        f'Failed to find built KFP wheel under {temp_dir}')
                    raise sys.exit(1)

                wheel_file = wheel_files[0]
                shutil.copy(wheel_file, self._context_directory)
                self._kfp_package_path = wheel_file.name
                self._maybe_copy_kfp_package = 'COPY {wheel_name} {wheel_name}'.format(
                    wheel_name=self._kfp_package_path)
            except subprocess.CalledProcessError as e:
                logging.error(f'Failed to build KFP wheel locally:\n{e}')
                raise sys.exit(1)
            finally:
                logging.info(f'Cleaning up temporary directory {temp_dir}')
                shutil.rmtree(temp_dir)
        else:
            self._kfp_package_path = kfp_package_path

        logging.info(
            f'Building component using KFP package path: {str(self._kfp_package_path)}'
        )

        self._context_directory_files = [
            file.name
            for file in self._context_directory.glob('*')
            if file.is_file()
        ]

        self._component_files = [
            file for file in self._context_directory.glob(
                self._component_filepattern) if file.is_file()
        ]

        self._base_image = None
        self._target_image = None
        self._pip_index_urls = None
        self._load_components()

    def _load_components(self):
        if not self._component_files:
            logging.error(
                f'No component files found matching pattern `{self._component_filepattern}` in directory {self._context_directory}'
            )
            raise sys.exit(1)

        for python_file in self._component_files:
            with _registered_modules() as component_modules:
                module_name = python_file.name[:-len('.py')]
                module_directory = python_file.parent
                utils.load_module(
                    module_name=module_name, module_directory=module_directory)

                python_file = str(python_file)

                logging.info(
                    f'Found {len(component_modules)} component(s) in file {python_file}:'
                )
                for name, component in component_modules.items():
                    logging.info(f'{name}: {component}')
                    self._components.append(component)

        base_images = {info.base_image for info in self._components}
        target_images = {info.target_image for info in self._components}

        if len(base_images) != 1:
            logging.error(
                f'Found {len(base_images)} unique base_image values {base_images}. Components must specify the same base_image and target_image.'
            )
            raise sys.exit(1)

        self._base_image = base_images.pop()
        if self._base_image is None:
            logging.error(
                'Did not find a base_image specified in any of the'
                ' components. A base_image must be specified in order to'
                ' build the component.')
            raise sys.exit(1)
        logging.info(f'Using base image: {self._base_image}')

        if len(target_images) != 1:
            logging.error(
                f'Found {len(target_images)} unique target_image values {target_images}. Components must specify the same base_image and target_image.'
            )
            raise sys.exit(1)

        self._target_image = target_images.pop()
        if self._target_image is None:
            logging.error(
                'Did not find a target_image specified in any of the'
                ' components. A target_image must be specified in order'
                ' to build the component.')
            raise sys.exit(1)
        logging.info(f'Using target image: {self._target_image}')

        pip_index_urls = []
        for comp in self._components:
            if comp.pip_index_urls is not None:
                pip_index_urls.extend(comp.pip_index_urls)
        if pip_index_urls:
            self._pip_index_urls = list(dict.fromkeys(pip_index_urls))

    def _maybe_write_file(self,
                          filename: str,
                          contents: str,
                          overwrite: bool = False):
        if filename in self._context_directory_files:
            logging.info(
                f'Found existing file {filename} under {self._context_directory}.'
            )
            if not overwrite:
                logging.info('Leaving this file untouched.')
                return
            else:
                logging.warning(f'Overwriting existing file {filename}')
        else:
            logging.warning(
                f'{filename} not found under {self._context_directory}. Creating one.'
            )

        filepath = self._context_directory / filename
        with open(filepath, 'w') as f:
            f.write(f'# Generated by KFP.\n{contents}')
        logging.info(f'Generated file {filepath}.')

    def generate_requirements_txt(self):
        dependencies = set()
        for component in self._components:
            for package in component.packages_to_install or []:
                dependencies.add(package)
        self._maybe_write_file(
            filename=_REQUIREMENTS_TXT,
            contents='\n'.join(sorted(dependencies)),
            overwrite=True)

    def maybe_generate_dockerignore(self):
        self._maybe_write_file(_DOCKERIGNORE, _DOCKERIGNORE_TEMPLATE)

    def write_component_files(self):
        for component_info in self._components:
            filename = component_info.output_component_file or f'{component_info.function_name}.yaml'

            container_filename = (
                self._context_directory / _COMPONENT_METADATA_DIR / filename)
            container_filename.parent.mkdir(exist_ok=True, parents=True)
            component_info.component_spec.save_to_component_yaml(
                str(container_filename))

    def generate_kfp_config(self):
        config = kfp_config.KFPConfig(config_directory=self._context_directory)
        for component_info in self._components:
            relative_path = component_info.module_path.relative_to(
                self._context_directory)
            config.add_component(
                function_name=component_info.function_name, path=relative_path)
        config.save()

    def maybe_generate_dockerfile(self, overwrite_dockerfile: bool = False):
        index_urls_options = component_factory.make_index_url_options(
            self._pip_index_urls)
        dockerfile_contents = _DOCKERFILE_TEMPLATE.format(
            base_image=self._base_image,
            maybe_copy_kfp_package=self._maybe_copy_kfp_package,
            component_root_dir=_COMPONENT_ROOT_DIR,
            kfp_package_path=self._kfp_package_path,
            requirements_file=_REQUIREMENTS_TXT,
            index_urls=index_urls_options,
        )

        self._maybe_write_file(_DOCKERFILE, dockerfile_contents,
                               overwrite_dockerfile)

    def build_image(self, platform: str, push_image: bool):
        logging.info(f'Building image {self._target_image} using Docker...')
        client = docker.from_env()

        docker_log_prefix = 'Docker'

        try:
            context = str(self._context_directory)
            logs = client.api.build(
                path=context,
                dockerfile='Dockerfile',
                tag=self._target_image,
                decode=True,
                platform=platform,
            )
            for log in logs:
                message = log.get('stream', '').rstrip('\n')
                if message:
                    logging.info(f'{docker_log_prefix}: {message}')

        except docker.errors.BuildError as e:
            for log in e.build_log:
                message = log.get('message', '').rstrip('\n')
                if message:
                    logging.error(f'{docker_log_prefix}: {message}')
            logging.error(f'{docker_log_prefix}: {e}')
            raise sys.exit(1)

        if not push_image:
            return

        logging.info(f'Pushing image {self._target_image}...')

        try:
            response = client.images.push(
                self._target_image, stream=True, decode=True)
            for log in response:
                status = log.get('status', '').rstrip('\n')
                error = log.get('error', '').rstrip('\n')
                layer = log.get('id', '')
                if status:
                    logging.info(f'{docker_log_prefix}: {layer} {status}')
                if error:
                    logging.error(f'{docker_log_prefix} ERROR: {error}')
        except docker.errors.BuildError as e:
            logging.error(f'{docker_log_prefix}: {e}')
            raise e

        logging.info(
            f'Built and pushed component container {self._target_image}')


@click.group()
def component():
    """Builds shareable, containerized components."""


@component.command()
@click.argument(
    'components_directory', type=click.Path(exists=True, file_okay=False))
@click.option(
    '--component-filepattern',
    type=str,
    default='**/*.py',
    help='Filepattern to use when searching for KFP components. The'
    ' default searches all Python files in the specified directory.')
@click.option(
    '--engine',
    type=str,
    default='docker',
    help="[Deprecated] Engine to use to build the component's container.",
    hidden=True)
@click.option(
    '--kfp-package-path',
    type=click.Path(exists=True),
    default=None,
    help='A pip-installable path to the KFP package.')
@click.option(
    '--overwrite-dockerfile',
    type=bool,
    is_flag=True,
    default=False,
    help='Set this to true to always generate a Dockerfile'
    ' as part of the build process')
@click.option(
    '--build-image/--no-build-image',
    type=bool,
    is_flag=True,
    default=True,
    help='Build the container image.')
@click.option(
    '--platform',
    type=str,
    default='linux/amd64',
    help='The platform to build the container image for.')
@click.option(
    '--push-image/--no-push-image',
    type=bool,
    is_flag=True,
    default=True,
    help='Push the built image to its remote repository.')
def build(components_directory: str, component_filepattern: str, engine: str,
          kfp_package_path: Optional[str], overwrite_dockerfile: bool,
          build_image: bool, platform: str, push_image: bool):
    """Builds containers for KFP v2 Python-based components."""

    if build_image and engine != 'docker':
        warnings.warn(
            'The --engine option is deprecated and does not need to be passed. Only Docker engine is supported and will be used by default.',
            DeprecationWarning,
            stacklevel=2)
        sys.exit(1)

    components_directory = pathlib.Path(components_directory)
    components_directory = components_directory.resolve()

    if not components_directory.is_dir():
        logging.error(
            f'{components_directory} does not seem to be a valid directory.')
        raise sys.exit(1)

    if build_image and not _DOCKER_IS_PRESENT:
        logging.error(
            'The `docker` Python package was not found in the current'
            ' environment. Please run `pip install docker` to install it.'
            ' Optionally, you can also  install KFP with all of its'
            ' optional dependencies by running `pip install kfp[all]`.'
            ' Alternatively, you can skip the container image build using'
            ' the --no-build-image flag.')
        raise sys.exit(1)

    kfp_package_path = pathlib.Path(
        kfp_package_path) if kfp_package_path is not None else None
    builder = ComponentBuilder(
        context_directory=components_directory,
        kfp_package_path=kfp_package_path,
        component_filepattern=component_filepattern,
    )
    builder.write_component_files()
    builder.generate_kfp_config()

    builder.generate_requirements_txt()
    builder.maybe_generate_dockerignore()
    builder.maybe_generate_dockerfile(overwrite_dockerfile=overwrite_dockerfile)
    if build_image:
        builder.build_image(platform=platform, push_image=push_image)
