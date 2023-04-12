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

from typing import Any, Dict, List, Optional

import click
from kfp import client
from kfp.cli import output
from kfp.cli.utils import parsing


@click.group()
def recurring_run():
    """Manage recurring run resources."""


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
    default=None,
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
    default=1,
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
    type=click.Path(exists=True, dir_okay=False),
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
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    if enable_caching is not None:
        click.echo(
            '--enable-caching and --disable-caching options are not yet supported.'
        )
        enable_caching = None

    if (interval_second is None) == (cron_expression is None):
        raise ValueError(
            'Either of --interval-second or --cron-expression options is required.'
        )

    if (experiment_id is None) == (experiment_name is None):
        raise ValueError(either_option_required)
    if not experiment_id:
        experiment = client_obj.create_experiment(experiment_name)
        experiment_id = experiment.experiment_id

    # Ensure we only split on the first equals char so the value can contain
    # equals signs too.
    split_args: List = [arg.split('=', 1) for arg in args or []]
    params: Dict[str, Any] = dict(split_args)
    recurring_run = client_obj.create_recurring_run(
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
    output.print_output(
        recurring_run,
        output.ModelType.RECURRING_RUN,
        output_format,
    )


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
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    response = client_obj.list_recurring_runs(
        experiment_id=experiment_id,
        page_token=page_token,
        page_size=max_size,
        sort_by=sort_by,
        filter=filter)
    output.print_output(
        response.recurring_runs or [],
        output.ModelType.RECURRING_RUN,
        output_format,
    )


@recurring_run.command()
@click.argument('recurring-run-id')
@click.pass_context
def get(ctx: click.Context, recurring_run_id: str):
    """Get information about a recurring run."""
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    recurring_run = client_obj.get_recurring_run(recurring_run_id)
    output.print_output(
        recurring_run,
        output.ModelType.RECURRING_RUN,
        output_format,
    )


@recurring_run.command()
@click.argument('recurring-run-id')
@click.pass_context
def delete(ctx: click.Context, recurring_run_id: str):
    """Delete a recurring run."""
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']
    confirmation = f'Are you sure you want to delete job {recurring_run_id}?'
    if not click.confirm(confirmation):
        return
    client_obj.delete_recurring_run(recurring_run_id)
    output.print_deleted_text('recurring_run', recurring_run_id, output_format)


@recurring_run.command()
@click.argument('recurring-run-id')
@click.pass_context
def enable(ctx: click.Context, recurring_run_id: str):
    """Enable a recurring run."""
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    client_obj.enable_recurring_run(recurring_run_id=recurring_run_id)
    # TODO: add wait option, since enable takes time to complete
    recurring_run = client_obj.get_recurring_run(
        recurring_run_id=recurring_run_id)
    output.print_output(
        recurring_run,
        output.ModelType.RECURRING_RUN,
        output_format,
    )


@recurring_run.command()
@click.argument('recurring-run-id')
@click.pass_context
def disable(ctx: click.Context, recurring_run_id: str):
    """Disable a recurring run."""
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    client_obj.disable_recurring_run(recurring_run_id=recurring_run_id)
    # TODO: add wait option, since disable takes time to complete
    recurring_run = client_obj.get_recurring_run(
        recurring_run_id=recurring_run_id)
    output.print_output(
        recurring_run,
        output.ModelType.RECURRING_RUN,
        output_format,
    )
