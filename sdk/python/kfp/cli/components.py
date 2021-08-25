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
import enum
import pathlib
import shutil
import subprocess
import tempfile
from typing import Any, List, Optional

_DOCKER_IS_PRESENT = True
try:
    import docker
except ImportError:
    _DOCKER_IS_PRESENT = False

import typer

import kfp
from kfp.v2.components import component_factory, utils

_REQUIREMENTS_TXT = 'requirements.txt'

_DOCKERFILE = 'Dockerfile'

_DOCKERFILE_TEMPLATE = '''
FROM {base_image}

WORKDIR {component_root_dir}
COPY requirements.txt requirements.txt
RUN pip install --no-cache-dir -r requirements.txt
{maybe_copy_kfp_package}
RUN pip install --no-cache-dir {kfp_package_path}
COPY . .
'''

_DOCKERIGNORE = '.dockerignore'

_DOCKERIGNORE_TEMPLATE = '''
component_metadata/
'''


class _Engine(str, enum.Enum):
    """Supported container build engines."""
    DOCKER = 'docker'
    KANIKO = 'kaniko'
    CLOUD_BUILD = 'cloudbuild'


app = typer.Typer()


def _info(message: Any):
    info = typer.style('INFO', fg=typer.colors.GREEN)
    typer.echo('{}: {}'.format(info, message))


def _warning(message: Any):
    info = typer.style('WARNING', fg=typer.colors.YELLOW)
    typer.echo('{}: {}'.format(info, message))


def _error(message: Any):
    info = typer.style('ERROR', fg=typer.colors.RED)
    typer.echo('{}: {}'.format(info, message))


class _ComponentBuilder():
    """Helper class for building containerized v2 KFP components."""

    def __init__(self,
                 component_modules: List[pathlib.Path],
                 context_directory: pathlib.Path,
                 kfp_package_path: Optional[pathlib.Path] = None):
        """ComponentBuilder builds containerized components.

        Args:
            component_modules: Paths to one or more Python files containing
                one or more KFP v2 components.
            context_directory: The directory to use as the container build
                context.
            kfp_package_path: Path to a pip-installable location for KFP.
                This can either be pointing to KFP SDK root directory located in
                a local clone of the KFP repo, or a git+https location.
                If left empty, defaults to KFP on PyPi.
        """
        self._component_modules = component_modules
        self._context_directory = context_directory
        self._dockerfile = self._context_directory / _DOCKERFILE

        # This is only set if we need to install KFP from local copy.
        self._maybe_copy_kfp_package = ''

        if kfp_package_path is None:
            self._kfp_package_path = 'kfp=={}'.format(kfp.__version__)
        elif kfp_package_path.is_dir():
            _info('Building KFP package from local directory {}'.format(
                typer.style(str(kfp_package_path), fg=typer.colors.CYAN)))
            temp_dir = pathlib.Path(tempfile.mkdtemp())
            try:
                subprocess.run([
                    'python3',
                    kfp_package_path / 'setup.py',
                    'bdist_wheel',
                    '--dist-dir',
                    str(temp_dir),
                ], cwd=kfp_package_path)
                wheel_files = list(temp_dir.glob('*.whl'))
                if len(wheel_files) != 1:
                    _error('Failed to find built KFP wheel under {}'.format(
                        temp_dir))
                    raise typer.Exit(1)

                wheel_file = wheel_files[0]
                shutil.copy(wheel_file, self._context_directory)
                self._kfp_package_path = wheel_file.name
                self._maybe_copy_kfp_package = 'COPY {wheel_name} {wheel_name}'.format(
                    wheel_name=self._kfp_package_path)
            except subprocess.CalledProcessError as e:
                _error('Failed to build KFP wheel locally:\n{}'.format(e))
                raise typer.Exit(1)
            finally:
                _info('Cleaning up temporary directory {}'.format(temp_dir))
                shutil.rmtree(temp_dir)
        else:
            self._kfp_package_path = kfp_package_path

        _info('Building component using KFP package path: {}'.format(
            typer.style(str(self._kfp_package_path), fg=typer.colors.CYAN)))

        self._files = [
            file.name
            for file in self._context_directory.glob('*')
            if file.is_file()
        ]

        self._base_image = None
        self._target_image = None
        self._load_components()

    def _load_components(self):
        module_components: component_factory.ModuleComponents = {}
        for component_module in self._component_modules:
            module_name = component_module.name[:-len('.py')]
            module_directory = component_module.parent
            utils.load_module(module_name=module_name,
                              module_directory=module_directory)

            formatted_module_file = typer.style(str(self._component_modules),
                                                fg=typer.colors.CYAN)

            module_components.update(
                component_factory.REGISTERED_MODULE_COMPONENTS.get(
                    component_module, None))

            if module_components is None:
                _error('Failed to load components from module file {}'.format(
                    formatted_module_file))
                raise typer.Exit(1)

            if not module_components:
                _error('No KFP components found in file {}'.format(
                    formatted_module_file))
                raise typer.Exit(1)

            _info('Found {} components in file {}:'.format(
                len(module_components), formatted_module_file))
            for component in module_components.values():
                _info(component)

        base_images = set(
            [info.base_image for info in module_components.values()])
        target_images = set(
            [info.target_image for info in module_components.values()])

        if len(base_images) != 1:
            _error(
                'Found {} unique base_image values {}. Components'
                ' must specify the same base_image and target_image.'.
                format(len(base_images), base_images))
            raise typer.Exit(1)

        self._base_image = base_images.pop()
        if self._base_image is None:
            _error('Did not find a base_image specified in any of the'
                   ' components. A base_image must be specified in order to'
                   ' build the component.')
            raise typer.Exit(1)
        _info('Using base image: {}'.format(
            typer.style(self._base_image, fg=typer.colors.YELLOW)))

        if len(target_images) != 1:
            _error('Found {} unique target_image values {}. Components'
                   ' must specify the same base_image and'
                   ' target_image.'.format(len(target_images), target_images))
            raise typer.Exit(1)

        self._target_image = target_images.pop()
        if self._target_image is None:
            _error('Did not find a target_image specified in any of the'
                   ' components. A target_image must be specified in order'
                   ' to build the component.')
            raise typer.Exit(1)
        _info('Using target image: {}'.format(
            typer.style(self._target_image, fg=typer.colors.YELLOW)))

    def _maybe_write_file(self,
                          filename: str,
                          contents: str,
                          overwrite: bool = False):
        formatted_filename = typer.style(filename, fg=typer.colors.CYAN)
        if filename in self._files:
            _info('Found existing file {} under {}.'.format(
                formatted_filename, self._context_directory))
            if not overwrite:
                _info('Leaving this file untouched.')
                return
            else:
                _warning(
                    'Overwriting existing file {}'.format(formatted_filename))
        else:
            _warning('{} not found under {}. Creating one.'.format(
                formatted_filename, self._context_directory))

        filepath = self._context_directory / filename
        with open(filepath, 'w') as f:
            f.write('# Generated by KFP.\n{}'.format(contents))
        _info('Generated file {}.'.format(filepath))

    def maybe_generate_requirements_txt(self):
        self._maybe_write_file(_REQUIREMENTS_TXT, '')

    def maybe_generate_dockerignore(self):
        self._maybe_write_file(_DOCKERIGNORE, _DOCKERIGNORE_TEMPLATE)

    def maybe_generate_dockerfile(self, overwrite_dockerfile: bool = False):
        dockerfile_contents = _DOCKERFILE_TEMPLATE.format(
            base_image=self._base_image,
            maybe_copy_kfp_package=self._maybe_copy_kfp_package,
            component_root_dir=component_factory.COMPONENT_ROOT_DIR,
            kfp_package_path=self._kfp_package_path)

        self._maybe_write_file(_DOCKERFILE, dockerfile_contents,
                               overwrite_dockerfile)

    def build_image(self, skip_pushing_image: bool = False):
        _info('Building image {} using Docker...'.format(
            typer.style(self._target_image, fg=typer.colors.YELLOW)))
        client = docker.from_env()

        docker_log_prefix = typer.style('Docker', fg=typer.colors.CYAN)

        try:
            context = str(self._context_directory)
            _, logs = client.images.build(path=context,
                                          dockerfile='Dockerfile',
                                          tag=self._target_image)
            for log in logs:
                message = log.get('stream', '').rstrip('\n')
                if message:
                    _info('{}: {}'.format(docker_log_prefix, message))

        except docker.errors.BuildError as e:
            for log in e.build_log:
                message = log.get('message', '').rstrip('\n')
                if message:
                    _error('{}: {}'.format(docker_log_prefix, message))
            _error('{}: {}'.format(docker_log_prefix, e))
            raise typer.Exit(1)

        if skip_pushing_image:
            return

        _info('Pushing image {}...'.format(
            typer.style(self._target_image, fg=typer.colors.YELLOW)))

        try:
            response = client.images.push(self._target_image,
                                          stream=True,
                                          decode=True)
            for log in response:
                status = log.get('status', '').rstrip('\n')
                layer = log.get('id', '')
                if status:
                    _info('{}: {} {}'.format(docker_log_prefix, layer, status))
        except docker.errors.BuildError as e:
            _error('{}: {}'.format(docker_log_prefix, e))
            raise e

        _info('Built and pushed component container {}'.format(
            typer.style(self._target_image, fg=typer.colors.YELLOW)))


@app.callback()
def components():
    """Builds shareable, containerized components."""
    pass



@app.command()
def build(component_modules: List[pathlib.Path] = typer.Argument(
    ...,
    help='Path to one or more Python files containing KFP v2 components.'
    ' All files should be under the same parent directory. The container will'
    ' be built with this parent directory as the context.'),
          engine: _Engine = typer.Option(
              _Engine.DOCKER,
              help="Engine to use to build the component's container."),
          kfp_package_path: Optional[pathlib.Path] = typer.Option(
              None, help="A pip-installable path to the KFP package."),
          overwrite_dockerfile: bool = typer.Option(
              False,
              help="Set this to true to always generate a Dockerfile"
              " as part of the build process")):
    """
    Builds containers for KFP v2 Python-based components.
    """
    resolved_component_modules = []
    module_parents = set()
    for module in component_modules:
        component_module = module.resolve()
        if not component_module.is_file():
            _error(
                '{} does not seem to be a valid file.'.format(component_module))
            raise typer.Exit(1)

        if not str(component_module).endswith('.py'):
            _error('{} does not have the `.py` suffix.'
                ' Is this a Python file?'.format(component_module))
            raise typer.Exit(1)

        module_parents.add(component_module.parent)
        resolved_component_modules.append(component_module)

    # For now, all modules must be under the same folder.
    # TODO: Allow different parent folders, under the same Docker context.
    if len(module_parents) > 1:
        _error('All component modules must be located under the same folder.'
               ' Found {} parent folders: {}'.format(len(module_parents),
                                                     module_parents))
        raise typer.Exit(1)

    if engine != _Engine.DOCKER:
        _error('Currently, only `docker` is supported for --engine.')
        raise typer.Exit(1)

    if engine == _Engine.DOCKER:
        if not _DOCKER_IS_PRESENT:
            _error(
                'The `docker` Python package was not found in the current'
                ' environment. Please run `pip install docker` to install it.'
                ' Optionally, you can also  install KFP with all of its optional'
                ' dependencies by running `pip install kfp[all]`.')
            raise typer.Exit(1)

    builder = _ComponentBuilder(component_modules=resolved_component_modules,
                                context_directory=pathlib.Path(
                                    module_parents.pop()),
                                kfp_package_path=kfp_package_path)

    builder.maybe_generate_requirements_txt()
    builder.maybe_generate_dockerignore()
    builder.maybe_generate_dockerfile(overwrite_dockerfile=overwrite_dockerfile)
    builder.build_image()


if __name__ == '__main__':
    app()
