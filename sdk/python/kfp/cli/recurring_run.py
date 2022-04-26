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

import json
from typing import Any, Dict, List, Optional

import click
import kfp_server_api
from kfp import client
from kfp.cli.output import OutputFormat
from kfp.cli.output import print_output
from kfp.cli.utils import parsing


@click.group()
def recurring_run():
    """Manage recurring run resources."""
    pass


either_option_required = 'Either --experiment-id or --experiment-name is required.'


@recurring_run.command()
@click.option(
    '--catchup/--no-catchup',
    type=bool,
    help=parsing.get_param_descr(client.Client.create_recurring_run,
                                 'no_catchup'),
)
@click.option(
    '--cron-expression',
    help=parsing.get_param_descr(client.Client.create_recurring_run,
                                 'cron_expression'))
@click.option(
    '--description',
    help=parsing.get_param_descr(client.Client.create_recurring_run,
                                 'description'))
@click.option(
    '--enable-caching/--disable-caching',
    type=bool,
    help=parsing.get_param_descr(client.Client.create_recurring_run,
                                 'enable_caching'),
)
@click.option(
    '--enabled/--disabled',
    type=bool,
    help=parsing.get_param_descr(client.Client.create_recurring_run, 'enabled'))
@click.option(
    '--end-time',
    help=parsing.get_param_descr(client.Client.create_recurring_run,
                                 'end_time'))
@click.option(
    '--experiment-id',
    help=parsing.get_param_descr(client.Client.create_recurring_run,
                                 'experiment_id') + ' ' + either_option_required
)
@click.option(
    '--experiment-name',
    help='The name of the experiment to create the recurring run under.' + ' ' +
    either_option_required)
@click.option(
    '--job-name',
    help=parsing.get_param_descr(client.Client.create_recurring_run,
                                 'job_name'))
@click.option(
    '--interval-second',
    help=parsing.get_param_descr(client.Client.create_recurring_run,
                                 'interval_second'))
@click.option(
    '--max-concurrency',
    type=int,
    help=parsing.get_param_descr(client.Client.create_recurring_run,
                                 'max_concurrency'),
)
@click.option(
    '--pipeline-id',
    help=parsing.get_param_descr(client.Client.create_recurring_run,
                                 'pipeline_id'),
)
@click.option(
    '--pipeline-package-path',
    help=parsing.get_param_descr(client.Client.create_recurring_run,
                                 'pipeline_package_path'))
@click.option(
    '--start-time',
    help=parsing.get_param_descr(client.Client.create_recurring_run,
                                 'start_time'))
@click.option('--version-id', help='The id of a pipeline version.')
@click.argument('args', nargs=-1)
@click.pass_context
def create(ctx: click.Context,
           job_name: str,
           experiment_id: Optional[str] = None,
           experiment_name: Optional[str] = None,
           catchup: Optional[bool] = None,
           cron_expression: Optional[str] = None,
           enabled: Optional[bool] = None,
           description: Optional[str] = None,
           enable_caching: Optional[bool] = None,
           end_time: Optional[str] = None,
           interval_second: Optional[int] = None,
           max_concurrency: Optional[int] = None,
           pipeline_package_path: Optional[str] = None,
           pipeline_id: Optional[str] = None,
           start_time: Optional[str] = None,
           version_id: Optional[str] = None,
           args: Optional[List[str]] = None):
    """Create a recurring run."""
    client = ctx.obj['client']
    output_format = ctx.obj['output']

    if (experiment_id is None) == (experiment_name is None):
        raise ValueError(either_option_required)
    if not experiment_id:
        experiment = client.create_experiment(experiment_name)
        experiment_id = experiment.id

    # Ensure we only split on the first equals char so the value can contain
    # equals signs too.
    split_args: List = [arg.split('=', 1) for arg in args or []]
    params: Dict[str, Any] = dict(split_args)
    recurring_run = client.create_recurring_run(
        cron_expression=cron_expression,
        description=description,
        enabled=enabled,
        enable_caching=enable_caching,
        end_time=end_time,
        experiment_id=experiment_id,
        interval_second=interval_second,
        job_name=job_name,
        max_concurrency=max_concurrency,
        no_catchup=not catchup,
        params=params,
        pipeline_package_path=pipeline_package_path,
        pipeline_id=pipeline_id,
        start_time=start_time,
        version_id=version_id)
    _display_recurring_run(recurring_run, output_format)


@recurring_run.command()
@click.option(
    '-e',
    '--experiment-id',
    help=parsing.get_param_descr(client.Client.list_recurring_runs,
                                 'experiment_id'))
@click.option(
    '--page-token',
    default='',
    help=parsing.get_param_descr(client.Client.list_recurring_runs,
                                 'page_token'))
@click.option(
    '-m',
    '--max-size',
    default=100,
    help=parsing.get_param_descr(client.Client.list_recurring_runs,
                                 'page_size'))
@click.option(
    '--sort-by',
    default='created_at desc',
    help=parsing.get_param_descr(client.Client.list_recurring_runs, 'sort_by'))
@click.option(
    '--filter',
    help=parsing.get_param_descr(client.Client.list_recurring_runs, 'filter'))
@click.pass_context
def list(ctx: click.Context, experiment_id: str, page_token: str, max_size: int,
         sort_by: str, filter: str):
    """List recurring runs."""
    client = ctx.obj['client']
    output_format = ctx.obj['output']

    response = client.list_recurring_runs(
        experiment_id=experiment_id,
        page_token=page_token,
        page_size=max_size,
        sort_by=sort_by,
        filter=filter)
    if response.jobs:
        _display_recurring_runs(response.jobs, output_format)
    else:
        if output_format == OutputFormat.json.name:
            msg = json.dumps([])
        else:
            msg = 'No recurring runs found'
        click.echo(msg)


@recurring_run.command()
@click.argument('job-id')
@click.pass_context
def get(ctx: click.Context, job_id: str):
    """Get information about a recurring run."""
    client = ctx.obj['client']
    output_format = ctx.obj['output']

    response = client.get_recurring_run(job_id)
    _display_recurring_run(response, output_format)


@recurring_run.command()
@click.argument('job-id')
@click.pass_context
def delete(ctx: click.Context, job_id: str):
    """Delete a recurring run."""
    client = ctx.obj['client']
    confirmation = f'Are you sure you want to delete job {job_id}?'
    if not click.confirm(confirmation):
        return
    client.delete_job(job_id)
    click.echo(f'Deleted job {job_id}.')


@recurring_run.command()
@click.argument('job-id')
@click.pass_context
def enable(ctx: click.Context, job_id: str):
    """Enable a recurring run."""
    client = ctx.obj['client']
    client.enable_job(job_id=job_id)
    click.echo(f'Enabled job {job_id}.')


@recurring_run.command()
@click.argument('job-id')
@click.pass_context
def disable(ctx: click.Context, job_id: str):
    """Disable a recurring run."""
    client = ctx.obj['client']
    client.disable_job(job_id=job_id)
    click.echo(f'Disabled job {job_id}.')


def _display_recurring_runs(recurring_runs: List[kfp_server_api.ApiJob],
                            output_format: OutputFormat):
    headers = ['Recurring Run ID', 'Name']
    data = [[rr.id, rr.name] for rr in recurring_runs]
    print_output(data, headers, output_format, table_format='grid')


def _display_recurring_run(recurring_run: kfp_server_api.ApiJob,
                           output_format: OutputFormat):
    table = [
        ['Recurring Run ID', recurring_run.id],
        ['Name', recurring_run.name],
    ]
    if output_format == OutputFormat.table.name:
        print_output([], ['Recurring Run Details'], output_format)
        print_output(table, [], output_format, table_format='plain')
    elif output_format == OutputFormat.json.name:
        print_output(dict(table), [], output_format)
