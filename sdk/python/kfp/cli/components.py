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
import configparser
import contextlib
import enum
import pathlib
import shutil
import subprocess
import tempfile
from typing import Any, List, Optional
import uuid

_DOCKER_IS_PRESENT = True
try:
    import docker
except ImportError:
    _DOCKER_IS_PRESENT = False

_CLOUD_BUILD_IS_PRESENT = True
try:
    from google.cloud.devtools import cloudbuild_v1 as cloudbuild
except ImportError:
    _CLOUD_BUILD_IS_PRESENT = False

_CLOUD_STORAGE_IS_PRESENT = True
try:
    from google.cloud import storage
    from google.cloud import exceptions
except ImportError:
    _CLOUD_BUILD_IS_PRESENT = False

import typer

import kfp
from kfp.v2.components import component_factory, kfp_config, utils

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

# Location in which to write out shareable YAML for components.
_COMPONENT_METADATA_DIR = 'component_metadata'

_DOCKERIGNORE_TEMPLATE = '''
{}/
'''.format(_COMPONENT_METADATA_DIR)

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


class _Engine(str, enum.Enum):
    """Supported container build engines."""
    DOCKER = 'docker'
    CLOUD_BUILD = 'cloudbuild'
    # Uncomment when support for this lands.
    # KANIKO = 'kaniko'


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


def _build_image_with_docker(context_directory: pathlib.Path,
                             target_image: str,
                             push_image: bool = True):
    _info('Building image {} using Docker...'.format(
        typer.style(target_image, fg=typer.colors.YELLOW)))
    client = docker.from_env()

    docker_log_prefix = typer.style('Docker', fg=typer.colors.CYAN)

    try:
        context = str(context_directory)
        logs = client.api.build(
            path=context,
            dockerfile='Dockerfile',
            tag=target_image,
            decode=True,
        )
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

    if not push_image:
        return

    _info('Pushing image {}...'.format(
        typer.style(target_image, fg=typer.colors.YELLOW)))

    try:
        response = client.images.push(target_image, stream=True, decode=True)
        for log in response:
            status = log.get('status', '').rstrip('\n')
            layer = log.get('id', '')
            if status:
                _info('{}: {} {}'.format(docker_log_prefix, layer, status))
    except docker.errors.BuildError as e:
        _error('{}: {}'.format(docker_log_prefix, e))
        raise e


def _build_image_with_cloud_build(
        context_directory: pathlib.Path,
        target_image: str,
        project_id: str,
        staging_bucket_name: Optional[str] = None,
        staging_bucket_location: Optional[str] = None):
    _info('Building image {} using Google Cloud Build...'.format(
        typer.style(target_image, fg=typer.colors.YELLOW)))

    import google.auth
    credentials, _ = google.auth.default()

    if not staging_bucket_name:
        staging_bucket_name = '{}_cloudbuild'.format(project_id)

    storage_client = storage.Client(credentials=credentials)
    bucket = storage_client.bucket(bucket_name=staging_bucket_name)

    if not staging_bucket_location:
        # Default to US, which is also the default GCS client behaviour.
        staging_bucket_location = 'us'

    try:
        new_bucket = storage_client.create_bucket(
            bucket_or_name=bucket, location=staging_bucket_location)
        _info('Created staging bucket: {}, location={}'.format(
            typer.style(staging_bucket_name, fg=typer.colors.CYAN),
            typer.style(staging_bucket_location, fg=typer.colors.CYAN)))
    except exceptions.Conflict:
        _info('Using existing staging bucket: {}'.format(
            typer.style(staging_bucket_name, fg=typer.colors.CYAN)))
        new_bucket = storage_client.get_bucket(bucket)

    zipped_context_basename = uuid.uuid4().hex
    zipped_context = shutil.make_archive(zipped_context_basename, 'zip',
                                         context_directory)

    source_blob_name = f'source/{zipped_context_basename}.zip'
    source_blob = new_bucket.blob(source_blob_name)

    _info('Staging source files from {} => {}'.format(
        typer.style(zipped_context, fg=typer.colors.CYAN),
        typer.style(
            f'gs://{staging_bucket_name}/{source_blob_name}',
            fg=typer.colors.CYAN)))

    source_blob.upload_from_filename(f'{zipped_context_basename}.zip')
    _info('Source files uploaded to staging bucket.')

    build = cloudbuild.Build()
    build.steps = [{
        'name': 'gcr.io/cloud-builders/docker',
        'args': ['build', '-t', target_image, '.']
    }]
    build.images = [target_image]
    storage_source = cloudbuild.StorageSource(
        bucket=staging_bucket_name,
        object_=f'source/{zipped_context_basename}.zip')
    build_source = cloudbuild.Source()
    build_source.storage_source = storage_source
    build.source = build_source

    cloudbuild_client = cloudbuild.services.cloud_build.CloudBuildClient(
        credentials=credentials)
    operation = cloudbuild_client.create_build(
        project_id=project_id, build=build)

    _info('Waiting for CloudBuild....\n{}'.format(operation.metadata))

    try:
        result = operation.result()
    except Exception:
        _error('Building with Google Cloud Build failed.'
               ' See the logs for details.')
        raise typer.Exit(1)

    if result.status != cloudbuild.Build.Status.SUCCESS:
        _error('Building with Google Cloud Build failed with status {}.'
               ' See the logs for details:\n{}'.format(result.status.name,
                                                       result))
        raise typer.Exit(1)


class _ComponentBuilder():
    """Helper class for building containerized v2 KFP components."""

    def __init__(
        self,
        context_directory: pathlib.Path,
        kfp_package_path: Optional[pathlib.Path] = None,
        component_filepattern: str = '**/*.py',
        build_engine: _Engine = _Engine.DOCKER,
        cloudbuild_project_id: Optional[str] = None,
        cloudbuild_staging_bucket: Optional[str] = None,
        cloudbuild_staging_location: Optional[str] = None,
    ):
        """ComponentBuilder builds containerized components.

        Args:
            context_directory: Directory containing one or more Python files
            with one or more KFP v2 components.
            kfp_package_path: Path to a pip-installable location for KFP.
                This can either be pointing to KFP SDK root directory located in
                a local clone of the KFP repo, or a git+https location.
                If left empty, defaults to KFP on PyPi.
        """
        self._context_directory = context_directory
        self._dockerfile = self._context_directory / _DOCKERFILE
        self._component_filepattern = component_filepattern
        self._components: List[
            component_factory.component_factory.ComponentInfo] = []
        self._build_engine = build_engine
        self._cloudbuild_project_id = cloudbuild_project_id
        self._cloudbuild_staging_bucket = cloudbuild_staging_bucket
        self._cloudbuild_staging_location = cloudbuild_staging_location

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
                ],
                               cwd=kfp_package_path)
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
        self._load_components()

    def _load_components(self):
        if not self._component_files:
            _error(
                'No component files found matching pattern `{}` in directory {}'
                .format(self._component_filepattern, self._context_directory))
            raise typer.Exit(1)

        for python_file in self._component_files:
            with _registered_modules() as component_modules:
                module_name = python_file.name[:-len('.py')]
                module_directory = python_file.parent
                utils.load_module(
                    module_name=module_name, module_directory=module_directory)

                formatted_module_file = typer.style(
                    str(python_file), fg=typer.colors.CYAN)
                if not component_modules:
                    _error('No KFP components found in file {}'.format(
                        formatted_module_file))
                    raise typer.Exit(1)

                _info('Found {} component(s) in file {}:'.format(
                    len(component_modules), formatted_module_file))
                for name, component in component_modules.items():
                    _info('{}: {}'.format(name, component))
                    self._components.append(component)

        base_images = set([info.base_image for info in self._components])
        target_images = set([info.target_image for info in self._components])

        if len(base_images) != 1:
            _error('Found {} unique base_image values {}. Components'
                   ' must specify the same base_image and target_image.'.format(
                       len(base_images), base_images))
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
        if filename in self._context_directory_files:
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

    def write_component_files(self):
        for component_info in self._components:
            filename = (
                component_info.output_component_file or
                component_info.function_name + '.yaml')
            container_filename = (
                self._context_directory / _COMPONENT_METADATA_DIR / filename)
            container_filename.parent.mkdir(exist_ok=True, parents=True)
            component_info.component_spec.save(container_filename)

    def generate_kfp_config(self):
        config = kfp_config.KFPConfig(config_directory=self._context_directory)
        for component_info in self._components:
            relative_path = component_info.module_path.relative_to(
                self._context_directory)
            config.add_component(
                function_name=component_info.function_name, path=relative_path)
        config.save()

    def maybe_generate_dockerfile(self, overwrite_dockerfile: bool = False):
        dockerfile_contents = _DOCKERFILE_TEMPLATE.format(
            base_image=self._base_image,
            maybe_copy_kfp_package=self._maybe_copy_kfp_package,
            component_root_dir=_COMPONENT_ROOT_DIR,
            kfp_package_path=self._kfp_package_path)

        self._maybe_write_file(_DOCKERFILE, dockerfile_contents,
                               overwrite_dockerfile)

    def build_image(self, push_image: bool = True):
        if self._build_engine == _Engine.DOCKER:
            _build_image_with_docker(
                context_directory=self._context_directory,
                target_image=self._target_image,
                push_image=push_image)
        elif self._build_engine == _Engine.CLOUD_BUILD:
            _build_image_with_cloud_build(
                context_directory=self._context_directory,
                target_image=self._target_image,
                project_id=self._cloudbuild_project_id,
                staging_bucket_name=self._cloudbuild_staging_bucket,
                staging_bucket_location=self._cloudbuild_staging_location)

        _info('Built and pushed component container {}'.format(
            typer.style(self._target_image, fg=typer.colors.YELLOW)))


@app.callback()
def components():
    """Builds shareable, containerized components."""
    pass


_ENGINE_HELP_DOC = '''
Engine to use when building the component's container. Supported options:

'docker': Use Docker when building the container. Requires Docker daemon to be
running on the machine.

'cloudbuild': Use Google Cloud Build to build the container. Ensure that the
Cloud Build API has been enabled for the project specified using
--cloudbuild-project-id.
'''


@app.command()
def build(components_directory: pathlib.Path = typer.Argument(
    ...,
    help="Path to a directory containing one or more Python"
    " files with KFP v2 components. The container will be built"
    " with this directory as the context."),
          component_filepattern: str = typer.Option(
              '**/*.py',
              help="Filepattern to use when searching for KFP components. The"
              " default searches all Python files in the specified directory."),
          engine: _Engine = typer.Option(_Engine.DOCKER, help=_ENGINE_HELP_DOC),
          kfp_package_path: Optional[pathlib.Path] = typer.Option(
              None, help="A pip-installable path to the KFP package."),
          overwrite_dockerfile: bool = typer.Option(
              False,
              help="Set this to true to always generate a Dockerfile"
              " as part of the build process"),
          push_image: bool = typer.Option(
              True, help="Push the built image to its remote repository."),
          cloudbuild_project_id: Optional[str] = typer.Option(
              None,
              help="Google Cloud project id to use if building with"
              " CloudBuild"),
          cloudbuild_staging_bucket: Optional[str] = typer.Option(
              None,
              help="Google Cloud bucket name to use if building with"
              " CloudBuild. E.g. `my-staging-bucket`"),
          cloudbuild_staging_location: Optional[str] = typer.Option(
              None,
              help="Google Cloud Storage location to use if building with"
              " CloudBuild and the staging bucket needs to be created first."
              " Defaults to location `us`.")):
    """
    Builds containers for KFP v2 Python-based components.
    """
    components_directory = components_directory.resolve()
    if not components_directory.is_dir():
        _error('{} does not seem to be a valid directory.'.format(
            components_directory))
        raise typer.Exit(1)

    if engine not in [_Engine.DOCKER, _Engine.CLOUD_BUILD]:
        _error('Currently, only `docker` and `cloudbuild` options are'
               ' supported for --engine.')
        raise typer.Exit(1)

    package_error_message = (
        'The `{package}` Python package was not found in the current'
        ' environment. Please run `pip install {package}` to install it.'
        ' Optionally, you can also  install KFP with all of its'
        ' optional dependencies by running `pip install kfp[all]`.')

    if engine == _Engine.DOCKER:
        if not _DOCKER_IS_PRESENT:
            _error(package_error_message.format(package='docker'))
            raise typer.Exit(1)
    elif engine == _Engine.CLOUD_BUILD:
        if not _CLOUD_BUILD_IS_PRESENT:
            _error(package_error_message.format(package='google-cloud-build'))
            raise typer.Exit(1)
        if not _CLOUD_STORAGE_IS_PRESENT:
            _error(package_error_message.format(package='google-cloud-storage'))
            raise typer.Exit(1)
        if cloudbuild_project_id is None:
            _error('Must specify --cloudbuild-project-id to use with'
                   ' Google Cloud Build. ')
            raise typer.Exit(1)
        if cloudbuild_staging_bucket and cloudbuild_staging_bucket.startswith(
                'gs://'):
            cloudbuild_staging_bucket = cloudbuild_staging_bucket[len('gs://'):]

    builder = _ComponentBuilder(
        context_directory=components_directory,
        kfp_package_path=kfp_package_path,
        component_filepattern=component_filepattern,
        build_engine=engine,
        cloudbuild_project_id=cloudbuild_project_id,
        cloudbuild_staging_bucket=cloudbuild_staging_bucket,
        cloudbuild_staging_location=cloudbuild_staging_location,
    )
    builder.write_component_files()
    builder.generate_kfp_config()

    builder.maybe_generate_requirements_txt()
    builder.maybe_generate_dockerignore()
    builder.maybe_generate_dockerfile(overwrite_dockerfile=overwrite_dockerfile)
    builder.build_image(push_image=push_image)


if __name__ == '__main__':
    app()
