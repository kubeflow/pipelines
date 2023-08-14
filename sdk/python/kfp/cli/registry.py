# Copyright 2023 The Kubeflow Authors
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

from typing import List, Tuple

import click
from kfp.cli.utils import parsing
from kfp.registry import RegistryClient


@click.group()
@click.option('--host', help=parsing.get_param_descr(RegistryClient, 'host'))
@click.option(
    '--config-file',
    help=parsing.get_param_descr(RegistryClient, 'config_file'))
@click.option(
    '--auth-file', help=parsing.get_param_descr(RegistryClient, 'auth_file'))
@click.pass_context
def registry(ctx: click.Context, host: str, config_file: str, auth_file: str):
    """Interact with a pipeline/component registry."""
    ctx.obj['client'] = RegistryClient(
        host, config_file=config_file, auth_file=auth_file)


@registry.command()
@click.argument('file_name', type=click.Path(exists=True))
@click.option(
    '--tag',
    '-t',
    'tags',
    multiple=True,
    help=parsing.get_param_descr(RegistryClient.upload_pipeline, 'tags'))
@click.option(
    '--extra-headers',
    '-e',
    type=(str, str),
    multiple=True,
    help=parsing.get_param_descr(RegistryClient.upload_pipeline,
                                 'extra_headers'))
@click.pass_context
def upload_pipeline(ctx: click.Context,
                    file_name: str,
                    tags: List[str] = None,
                    extra_headers: List[Tuple[str, str]] = None):
    """Uploads the pipeline."""

    if tags:
        tags = list(tags)

    if extra_headers:
        extra_headers = dict(extra_headers)

    client_obj: RegistryClient = ctx.obj['client']
    package_name, version = client_obj.upload_pipeline(
        file_name, tags=tags, extra_headers=extra_headers)
    click.echo(f'Pipeline uploaded to {package_name}, version {version}')


@registry.command()
@click.argument(
    'package_name',
    type=str,
)
@click.option(
    '--version',
    '-v',
    help=parsing.get_param_descr(RegistryClient.download_pipeline, 'version'))
@click.option(
    '--tag',
    '-t',
    help=parsing.get_param_descr(RegistryClient.download_pipeline, 'tag'))
@click.option(
    '--output',
    '-o',
    'file_name',
    help=parsing.get_param_descr(RegistryClient.download_pipeline, 'file_name'))
@click.pass_context
def download_pipeline(ctx: click.Context,
                      package_name: str,
                      version: str = None,
                      tag: str = None,
                      file_name: str = None):
    """Downloads a pipeline.

    Either version or tag must be specified.
    """

    client_obj: RegistryClient = ctx.obj['client']
    result = client_obj.download_pipeline(
        package_name, version=version, tag=tag, file_name=file_name)
    click.echo(f'Pipeline downloaded to {result}')


@registry.command()
@click.argument(
    'package_name',
    type=str,
)
@click.pass_context
def get_package(ctx: click.Context, package_name: str):
    """Gets package metadata."""

    client_obj: RegistryClient = ctx.obj['client']
    result = client_obj.get_package(package_name)
    click.echo(result)


@registry.command()
@click.pass_context
def list_packages(ctx: click.Context):
    """Lists packages."""

    client_obj: RegistryClient = ctx.obj['client']
    result = client_obj.list_packages()
    click.echo(result)


@registry.command()
@click.argument(
    'package_name',
    type=str,
)
@click.pass_context
def delete_package(ctx: click.Context, package_name: str):
    """Deletes a package."""

    client_obj: RegistryClient = ctx.obj['client']
    client_obj.delete_package(package_name)


@registry.command()
@click.argument(
    'package_name',
    type=str,
)
@click.argument(
    'version',
    type=str,
)
@click.pass_context
def get_version(ctx: click.Context, package_name: str, version: str):
    """Gets package version metadata."""

    client_obj: RegistryClient = ctx.obj['client']
    result = client_obj.get_version(package_name, version)
    click.echo(result)


@registry.command()
@click.argument(
    'package_name',
    type=str,
)
@click.pass_context
def list_versions(ctx: click.Context, package_name: str):
    """Lists package versions."""

    client_obj: RegistryClient = ctx.obj['client']
    result = client_obj.list_versions(package_name)
    click.echo(result)


@registry.command()
@click.argument(
    'package_name',
    type=str,
)
@click.argument(
    'version',
    type=str,
)
@click.pass_context
def delete_version(ctx: click.Context, package_name: str, version: str):
    """Deletes package version."""

    client_obj: RegistryClient = ctx.obj['client']
    client_obj.delete_version(package_name, version)


@registry.command()
@click.argument(
    'package_name',
    type=str,
)
@click.argument(
    'version',
    type=str,
)
@click.argument(
    'tag',
    type=str,
)
@click.pass_context
def create_tag(ctx: click.Context, package_name: str, version: str, tag: str):
    """Creates a tag on a package version."""

    client_obj: RegistryClient = ctx.obj['client']
    result = client_obj.create_tag(package_name, version=version, tag=tag)
    click.echo(
        f'Tag {tag} created for package {package_name} version {version}')
    click.echo(result)


@registry.command()
@click.argument(
    'package_name',
    type=str,
)
@click.argument(
    'tag',
    type=str,
)
@click.pass_context
def get_tag(ctx: click.Context, package_name: str, tag: str):
    """Gets tag metadata."""

    client_obj: RegistryClient = ctx.obj['client']
    result = client_obj.get_tag(package_name, tag)
    click.echo(result)


@registry.command()
@click.argument(
    'package_name',
    type=str,
)
@click.argument(
    'version',
    type=str,
)
@click.argument(
    'tag',
    type=str,
)
@click.pass_context
def update_tag(ctx: click.Context, package_name: str, version: str, tag: str):
    """Updates a tag to another package version."""

    client_obj: RegistryClient = ctx.obj['client']
    result = client_obj.update_tag(package_name, version=version, tag=tag)
    click.echo(
        f'Tag {tag} updated for package {package_name}, new version {version}')
    click.echo(result)


@registry.command()
@click.argument(
    'package_name',
    type=str,
)
@click.pass_context
def list_tags(ctx: click.Context, package_name: str):
    """Lists package tags."""

    client_obj: RegistryClient = ctx.obj['client']
    result = client_obj.list_tags(package_name)
    click.echo(result)


@registry.command()
@click.argument(
    'package_name',
    type=str,
)
@click.argument(
    'tag',
    type=str,
)
@click.pass_context
def delete_tag(ctx: click.Context, package_name: str, tag: str):
    """Deletes package tag."""

    client_obj: RegistryClient = ctx.obj['client']
    result = client_obj.delete_tag(package_name, tag)
    click.echo(f'Tag {tag} deleted.')
