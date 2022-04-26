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

import json
from typing import Any, Dict, List, Optional, Union

import click
import kfp_server_api
from kfp import client
from kfp.cli.output import OutputFormat
from kfp.cli.output import print_output
from kfp.cli.utils import parsing


@click.group()
def pipeline():
    """Manage pipeline resources."""
    pass


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
def upload(ctx: click.Context,
           pipeline_name: str,
           package_file: str,
           description: str = None):
    """Upload a KFP pipeline."""
    client = ctx.obj['client']
    output_format = ctx.obj['output']
    if not pipeline_name:
        pipeline_name = package_file.split('.')[0]

    pipeline = client.upload_pipeline(package_file, pipeline_name, description)
    _display_pipeline(pipeline, output_format)
    click.echo('Pipeline uploaded successfully.')


either_option_required = 'Either --pipeline-id or --pipeline-name is required.'


@pipeline.command()
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
    '-v',
    '--pipeline-version',
    help=parsing.get_param_descr(client.Client.upload_pipeline_version,
                                 'pipeline_version_name'),
    required=True,
)
@click.argument('package-file')
@click.pass_context
def upload_version(ctx: click.Context,
                   package_file: str,
                   pipeline_version: str,
                   pipeline_id: Optional[str] = None,
                   pipeline_name: Optional[str] = None):
    """Upload a version of the KFP pipeline."""
    client = ctx.obj['client']
    output_format = ctx.obj['output']
    if bool(pipeline_id) == bool(pipeline_name):
        raise ValueError(either_option_required)
    if pipeline_name is not None:
        pipeline_id = client.get_pipeline_id(name=pipeline_name)
        if pipeline_id is None:
            raise ValueError(
                f"Can't find a pipeline with name: {pipeline_name}")
    # TODO: this is broken
    version = client.pipeline_uploads.upload_pipeline_version(
        package_file, name=pipeline_version, pipelineid=pipeline_id)
    _display_pipeline_version(version, output_format)


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
    """List uploaded KFP pipelines."""
    client = ctx.obj['client']
    output_format = ctx.obj['output']

    response = client.list_pipelines(
        page_token=page_token,
        page_size=max_size,
        sort_by=sort_by,
        filter=filter)
    if response.pipelines:
        _print_pipelines(response.pipelines, output_format)
    else:
        if output_format == OutputFormat.json.name:
            msg = json.dumps([])
        else:
            msg = 'No pipelines found'
        click.echo(msg)


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
    """List versions of an uploaded KFP pipeline."""
    client = ctx.obj['client']
    output_format = ctx.obj['output']

    response = client.list_pipeline_versions(
        pipeline_id,
        page_token=page_token,
        page_size=max_size,
        sort_by=sort_by,
        filter=filter)
    if response.versions:
        _print_pipeline_versions(response.versions, output_format)
    else:
        if output_format == OutputFormat.json.name:
            msg = json.dumps([])
        else:
            msg = 'No pipeline or version found'
        click.echo(msg)


@pipeline.command()
@click.argument('version-id')
@click.pass_context
def delete_version(ctx: click.Context, version_id: str):
    """Delete pipeline version.

    Args:
      version_id: id of the pipeline version.

    Returns:
      Object. If the method is called asynchronously, returns the request thread.

    Throws:
      Exception if pipeline version is not found.
    """
    confirmation = f'Are you sure you want to delete pipeline version {version_id}?'
    if not click.confirm(confirmation):
        return

    client = ctx.obj['client']
    res = client.delete_pipeline_version(version_id)
    click.echo(f'Pipeline version {version_id} deleted.')
    return res


@pipeline.command()
@click.argument('pipeline-id')
@click.pass_context
def get(ctx: click.Context, pipeline_id: str):
    """Get detailed information about an uploaded KFP pipeline."""
    client = ctx.obj['client']
    output_format = ctx.obj['output']

    pipeline = client.get_pipeline(pipeline_id)
    _display_pipeline(pipeline, output_format)


@pipeline.command()
@click.argument('version-id')
@click.pass_context
def get_version(ctx: click.Context, version_id: str):
    """Get detailed information about an uploaded KFP pipeline version."""
    client = ctx.obj['client']
    output_format = ctx.obj['output']

    version = client.get_pipeline_version(version_id=version_id)
    _display_pipeline_version(version, output_format)


@pipeline.command()
@click.argument('pipeline-id')
@click.pass_context
def delete(ctx: click.Context, pipeline_id: str):
    """Delete an uploaded KFP pipeline."""
    client = ctx.obj['client']

    confirmation = f'Are you sure you want to delete pipeline {pipeline_id}?'
    if not click.confirm(confirmation):
        return

    client.delete_pipeline(pipeline_id)
    click.echo(f'{pipeline_id} deleted.')


def _print_pipelines(pipelines: List[kfp_server_api.ApiPipeline],
                     output_format: OutputFormat):
    headers = ['Pipeline ID', 'Name', 'Uploaded at']
    data = [[pipeline.id, pipeline.name,
             pipeline.created_at.isoformat()] for pipeline in pipelines]
    print_output(data, headers, output_format, table_format='grid')


def _print_pipeline_versions(versions: List[kfp_server_api.ApiPipelineVersion],
                             output_format: OutputFormat):
    headers = ['Version ID', 'Version name', 'Uploaded at', 'Pipeline ID']
    data = [[
        version.id, version.name,
        version.created_at.isoformat(),
        next(rr
             for rr in version.resource_references
             if rr.key.type == kfp_server_api.ApiResourceType.PIPELINE).key.id
    ]
            for version in versions]
    print_output(data, headers, output_format, table_format='grid')


def _display_pipeline(pipeline: kfp_server_api.ApiPipeline,
                      output_format: OutputFormat):
    # Pipeline information
    table = [['Pipeline ID', pipeline.id], ['Name', pipeline.name],
             ['Description', pipeline.description],
             ['Uploaded at', pipeline.created_at.isoformat()],
             ['Version ID', pipeline.default_version.id]]

    # Pipeline parameter details
    headers = ['Parameter Name', 'Default Value']
    data = []
    if pipeline.parameters is not None:
        data = [[param.name, param.value] for param in pipeline.parameters]

    if output_format == OutputFormat.table.name:
        print_output([], ['Pipeline Details'], output_format)
        print_output(table, [], output_format, table_format='plain')
        print_output(data, headers, output_format, table_format='grid')
    elif output_format == OutputFormat.json.name:
        OutputType = Dict[str, Union[Dict[str, str], List[Dict[str, Any]]]]
        output: OutputType = {'Pipeline Details': dict(table)}
        params = [dict(zip(headers, item)) for item in data]
        output['Pipeline Parameters'] = params
        print_output(output, [], output_format)


def _display_pipeline_version(version: kfp_server_api.ApiPipelineVersion,
                              output_format: OutputFormat):
    pipeline_id = next(
        rr for rr in version.resource_references
        if rr.key.type == kfp_server_api.ApiResourceType.PIPELINE).key.id
    table = [['Pipeline ID', pipeline_id], ['Version name', version.name],
             ['Uploaded at', version.created_at.isoformat()],
             ['Version ID', version.id]]

    if output_format == OutputFormat.table.name:
        print_output([], ['Pipeline Version Details'], output_format)
        print_output(table, [], output_format, table_format='plain')
    elif output_format == OutputFormat.json.name:
        print_output(dict(table), [], output_format)
