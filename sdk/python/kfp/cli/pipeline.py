# Copyright 2019-2022 The Kubeflow Authors
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

from typing import Optional

import click
from kfp import client
from kfp.cli import output
from kfp.cli.utils import deprecated_alias_group
from kfp.cli.utils import parsing


@click.group(
    cls=deprecated_alias_group.deprecated_alias_group_factory({
        'upload': 'create',
        'upload-version': 'create-version'
    }))
def pipeline():
    """Manage pipeline resources."""


@pipeline.command()
@click.option(
    '-p',
    '--pipeline-name',
    help=parsing.get_param_descr(client.Client.upload_pipeline,
                                 'pipeline_name'))
@click.option(
    '-d',
    '--description',
    help=parsing.get_param_descr(client.Client.upload_pipeline, 'description'))
@click.argument('package-file')
@click.pass_context
def create(ctx: click.Context,
           pipeline_name: str,
           package_file: str,
           description: str = None):
    """Upload a pipeline."""
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    pipeline = client_obj.upload_pipeline(package_file, pipeline_name,
                                          description)
    output.print_output(
        pipeline,
        output.ModelType.PIPELINE,
        output_format,
    )


either_option_required = 'Either --pipeline-id or --pipeline-name is required.'


@pipeline.command()
@click.argument('package-file', type=click.Path(exists=True, dir_okay=False))
@click.option(
    '-v',
    '--pipeline-version',
    help=parsing.get_param_descr(client.Client.upload_pipeline_version,
                                 'pipeline_version_name'),
    required=True,
)
@click.option(
    '-p',
    '--pipeline-id',
    required=False,
    help=parsing.get_param_descr(client.Client.upload_pipeline_version,
                                 'pipeline_id') + ' ' + either_option_required)
@click.option(
    '-n',
    '--pipeline-name',
    required=False,
    help=parsing.get_param_descr(client.Client.upload_pipeline_version,
                                 'pipeline_name') + ' ' + either_option_required
)
@click.option(
    '-d',
    '--description',
    help=parsing.get_param_descr(client.Client.upload_pipeline_version,
                                 'description'))
@click.pass_context
def create_version(ctx: click.Context,
                   package_file: str,
                   pipeline_version: str,
                   pipeline_id: Optional[str] = None,
                   pipeline_name: Optional[str] = None,
                   description: Optional[str] = None):
    """Upload a version of a pipeline."""
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']
    if bool(pipeline_id) == bool(pipeline_name):
        raise ValueError(either_option_required)
    if pipeline_name is not None:
        pipeline_id = client_obj.get_pipeline_id(name=pipeline_name)
        if pipeline_id is None:
            raise ValueError(
                f"Can't find a pipeline with name: {pipeline_name}")
    version = client_obj.upload_pipeline_version(
        pipeline_package_path=package_file,
        pipeline_version_name=pipeline_version,
        pipeline_id=pipeline_id,
        description=description)
    output.print_output(
        version,
        output.ModelType.PIPELINE,
        output_format,
    )


@pipeline.command()
@click.option(
    '--page-token',
    default='',
    help=parsing.get_param_descr(client.Client.list_pipelines, 'page_token'))
@click.option(
    '-m',
    '--max-size',
    default=100,
    help=parsing.get_param_descr(client.Client.list_pipelines, 'page_size'))
@click.option(
    '--sort-by',
    default='created_at desc',
    help=parsing.get_param_descr(client.Client.list_pipelines, 'sort_by'))
@click.option(
    '--filter',
    help=parsing.get_param_descr(client.Client.list_pipelines, 'filter'))
@click.pass_context
def list(ctx: click.Context, page_token: str, max_size: int, sort_by: str,
         filter: str):
    """List pipelines."""
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    response = client_obj.list_pipelines(
        page_token=page_token,
        page_size=max_size,
        sort_by=sort_by,
        filter=filter)
    output.print_output(
        response.pipelines or [],
        output.ModelType.PIPELINE,
        output_format,
    )


@pipeline.command()
@click.argument('pipeline-id')
@click.option(
    '--page-token',
    default='',
    help=parsing.get_param_descr(client.Client.list_pipeline_versions,
                                 'page_token'))
@click.option(
    '-m',
    '--max-size',
    default=100,
    help=parsing.get_param_descr(client.Client.list_pipeline_versions,
                                 'page_size'))
@click.option(
    '--sort-by',
    default='created_at desc',
    help=parsing.get_param_descr(client.Client.list_pipeline_versions,
                                 'sort_by'))
@click.option(
    '--filter',
    help=parsing.get_param_descr(client.Client.list_pipeline_versions,
                                 'filter'))
@click.pass_context
def list_versions(ctx: click.Context, pipeline_id: str, page_token: str,
                  max_size: int, sort_by: str, filter: str):
    """List versions of a pipeline."""
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    response = client_obj.list_pipeline_versions(
        pipeline_id,
        page_token=page_token,
        page_size=max_size,
        sort_by=sort_by,
        filter=filter)
    output.print_output(
        response.pipeline_versions or [],
        output.ModelType.PIPELINE_VERSION,
        output_format,
    )


@pipeline.command()
@click.argument('pipeline-id')
@click.argument('version-id')
@click.pass_context
def delete_version(ctx: click.Context, pipeline_id: str, version_id: str):
    """Delete a version of a pipeline."""
    confirmation = f'Are you sure you want to delete pipeline version {version_id}?'
    if not click.confirm(confirmation):
        return

    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    client_obj.delete_pipeline_version(
        pipeline_id=pipeline_id, pipeline_version_id=version_id)
    output.print_deleted_text('pipeline version', version_id, output_format)


@pipeline.command()
@click.argument('pipeline-id')
@click.pass_context
def get(ctx: click.Context, pipeline_id: str):
    """Get information about a pipeline."""
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    pipeline = client_obj.get_pipeline(pipeline_id)
    output.print_output(
        pipeline,
        output.ModelType.PIPELINE,
        output_format,
    )


@pipeline.command()
@click.argument('pipeline-id')
@click.argument('version-id')
@click.pass_context
def get_version(ctx: click.Context, pipeline_id: str, version_id: str):
    """Get information about a version of a pipeline."""
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    version = client_obj.get_pipeline_version(
        pipeline_id=pipeline_id, pipeline_version_id=version_id)
    output.print_output(
        version,
        output.ModelType.PIPELINE,
        output_format,
    )


@pipeline.command()
@click.argument('pipeline-id')
@click.pass_context
def delete(ctx: click.Context, pipeline_id: str):
    """Delete a pipeline."""
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    confirmation = f'Are you sure you want to delete pipeline {pipeline_id}?'
    if not click.confirm(confirmation):
        return

    client_obj.delete_pipeline(pipeline_id)
    output.print_deleted_text('pipeline', pipeline_id, output_format)
