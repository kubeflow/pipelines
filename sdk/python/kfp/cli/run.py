# Copyright 2018-2022 The Kubeflow Authors
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
import sys
import time
from typing import List

import click
from kfp import client
from kfp.cli import output
from kfp.cli.utils import deprecated_alias_group
from kfp.cli.utils import parsing


@click.group(
    cls=deprecated_alias_group.deprecated_alias_group_factory(
        {'submit': 'create'}))
def run():
    """Manage run resources."""


@run.command()
@click.option(
    '-e',
    '--experiment-id',
    help=parsing.get_param_descr(client.Client.list_runs, 'experiment_id'))
@click.option(
    '--page-token',
    default='',
    help=parsing.get_param_descr(client.Client.list_runs, 'page_token'))
@click.option(
    '-m',
    '--max-size',
    default=100,
    help=parsing.get_param_descr(client.Client.list_runs, 'page_size'))
@click.option(
    '--sort-by',
    default='created_at desc',
    help=parsing.get_param_descr(client.Client.list_runs, 'sort_by'))
@click.option(
    '--filter', help=parsing.get_param_descr(client.Client.list_runs, 'filter'))
@click.pass_context
def list(ctx: click.Context, experiment_id: str, page_token: str, max_size: int,
         sort_by: str, filter: str):
    """List pipeline runs."""
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']
    response = client_obj.list_runs(
        experiment_id=experiment_id,
        page_token=page_token,
        page_size=max_size,
        sort_by=sort_by,
        filter=filter)
    output.print_output(
        response.runs or [],
        output.ModelType.RUN,
        output_format,
    )


@run.command()
@click.option(
    '-e',
    '--experiment-name',
    required=True,
    help='Experiment name of the run.')
@click.option(
    '-r',
    '--run-name',
    help=parsing.get_param_descr(client.Client.run_pipeline, 'job_name'))
@click.option(
    '-f',
    '--package-file',
    type=click.Path(exists=True, dir_okay=False),
    help=parsing.get_param_descr(client.Client.run_pipeline,
                                 'pipeline_package_path'))
@click.option(
    '-p',
    '--pipeline-id',
    help=parsing.get_param_descr(client.Client.run_pipeline, 'pipeline_id'))
@click.option('-n', '--pipeline-name', help='Name of the pipeline template.')
@click.option(
    '-w',
    '--watch',
    is_flag=True,
    default=False,
    help='Watch the run status until it finishes.')
@click.option(
    '-v',
    '--version',
    help=parsing.get_param_descr(client.Client.run_pipeline, 'version_id'))
@click.option(
    '-t',
    '--timeout',
    default=0,
    help='Wait for a run to complete until timeout in seconds.',
    type=int)
@click.argument('args', nargs=-1)
@click.pass_context
def create(ctx: click.Context, experiment_name: str, run_name: str,
           package_file: str, pipeline_id: str, pipeline_name: str, watch: bool,
           timeout: int, version: str, args: List[str]):
    """Submit a pipeline run."""
    client_obj: client.Client = ctx.obj['client']
    namespace = ctx.obj['namespace']
    output_format = ctx.obj['output']
    if not run_name:
        run_name = experiment_name

    if not pipeline_id and pipeline_name:
        pipeline_id = client_obj.get_pipeline_id(name=pipeline_name)

    if not package_file and not pipeline_id and not version:
        click.echo(
            'You must provide one of [package_file, pipeline_id, version].',
            err=True)
        sys.exit(1)

    arg_dict = dict(arg.split('=', maxsplit=1) for arg in args)

    experiment = client_obj.create_experiment(experiment_name)
    run = client_obj.run_pipeline(
        experiment_id=experiment.experiment_id,
        job_name=run_name,
        pipeline_package_path=package_file,
        params=arg_dict,
        pipeline_id=pipeline_id,
        version_id=version)
    if timeout > 0:
        run_detail = client_obj.wait_for_run_completion(run.run_id, timeout)
        output.print_output(
            run_detail,
            output.ModelType.RUN,
            output_format,
        )
    else:
        display_run(client_obj, run.run_id, watch, output_format)


@run.command()
@click.option(
    '-w',
    '--watch',
    is_flag=True,
    default=False,
    help='Watch the run status until it finishes.')
@click.option(
    '-d',
    '--detail',
    is_flag=True,
    default=False,
    help='Get detailed information of the run in json format.')
@click.argument('run-id')
@click.pass_context
def get(ctx: click.Context, watch: bool, detail: bool, run_id: str):
    """Get information about a pipeline run."""
    client_obj: client.Client = ctx.obj['client']
    namespace = ctx.obj['namespace']
    output_format = ctx.obj['output']
    if detail:
        output_format = 'json'
        click.echo(
            'The --detail/-d flag is deprecated. Please use --output=json instead.',
            err=True)
    display_run(client_obj, run_id, watch, output_format)


@run.command()
@click.argument('run-id')
@click.pass_context
def archive(ctx: click.Context, run_id: str):
    """Archive a pipeline run."""
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    client_obj.archive_run(run_id=run_id)
    run = client_obj.get_run(run_id=run_id)
    output.print_output(
        run,
        output.ModelType.RUN,
        output_format,
    )


@run.command()
@click.argument('run-id')
@click.pass_context
def unarchive(ctx: click.Context, run_id: str):
    """Unarchive a pipeline run."""
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']
    client_obj.unarchive_run(run_id=run_id)
    run = client_obj.get_run(run_id=run_id)
    output.print_output(
        run,
        output.ModelType.RUN,
        output_format,
    )


@run.command()
@click.argument('run-id')
@click.pass_context
def delete(ctx: click.Context, run_id: str):
    """Delete a pipeline run."""

    confirmation = f'Are you sure you want to delete run {run_id}?'
    if not click.confirm(confirmation):
        return

    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    client_obj.delete_run(run_id=run_id)
    output.print_deleted_text('run', run_id, output_format)


def display_run(client: client.Client, run_id: str, watch: bool,
                output_format: str):
    run = client.get_run(run_id)

    output.print_output(
        run,
        output.ModelType.RUN,
        output_format,
    )
    if not watch:
        return
    while True:
        time.sleep(1)
        run_detail = client.get_run(run_id)
        run = run_detail
        if run_detail.state in [
                'SUCCEEDED', 'SKIPPED', 'FAILED', 'CANCELED', 'PAUSED'
        ]:
            click.echo(f'Run is finished with state {run_detail.state}.')
            return
