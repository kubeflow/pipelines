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

from typing import Any, Dict, List, Optional
import json
import click
from kfp.cli.output import print_output, OutputFormat


@click.group()
def recurring_run():
    """Manage recurring-run resources"""
    pass


@recurring_run.command()
@click.option('--catchup/--no-catchup', help='Whether the recurring run should catch up if behind schedule.', type=bool)
@click.option('--cron-expression', help='A cron expression representing a set of times, using 6 space-separated fields, e.g. "0 0 9 ? * 2-6".')
@click.option('--description', help='The description of the recurring run.')
@click.option('--enable-caching/--disable-caching', help='Optional. Whether or not to enable caching for the run.', type=bool)
@click.option('--enabled/--disabled', help='A bool indicating whether the recurring run is enabled or disabled.', type=bool)
@click.option('--end-time', help='The RFC3339 time string of the time when to end the job.')
@click.option('--experiment-id', help='The ID of the experiment to create the recurring run under.')
@click.option('--experiment-name', help='The name of the experiment to create the recurring run under.')
@click.option('--job-name', help='The name of the recurring run.')
@click.option('--interval-second', help='Integer indicating the seconds between two recurring runs in for a periodic schedule.')
@click.option('--max-concurrency', help='Integer indicating how many jobs can be run in parallel.', type=int)
@click.option('--pipeline-id', help='The ID of the pipeline to use to create the recurring run.')
@click.option('--pipeline-package-path', help='Local path of the pipeline package(the filename should end with one of the following .tar.gz, .tgz, .zip, .yaml, .yml).')
@click.option('--start-time', help='The RFC3339 time string of the time when to start the job.')
@click.option('--version-id', help='The id of a pipeline version.')
@click.argument("args", nargs=-1)
@click.pass_context
def create(
    ctx: click.Context,
    experiment_id: str,
    experiment_name: str,
    job_name: str,
    catchup: Optional[bool] = None,
    cron_expression: Optional[str] = None,
    enabled: Optional[bool] = None,
    description: Optional[str] = None,
    enable_caching: Optional[bool] = None,
    end_time: Optional[str] = None,
    interval_second: Optional[int] = None,
    max_concurrency: Optional[int] = None,
    params: Optional[dict] = None,
    pipeline_package_path: Optional[str] = None,
    pipeline_id: Optional[str] = None,
    start_time: Optional[str] = None,
    version_id: Optional[str] = None,
    args: Optional[List[str]] = None
):
    """Create a recurring run."""
    client = ctx.obj["client"]
    output_format = ctx.obj['output']

    # Ensure we only split on the first equals char so the value can contain
    # equals signs too.
    split_args: List = [arg.split("=", 1) for arg in args or []]
    params: Dict[str, Any] = dict(split_args)
    recurring_run = client.create_recurring_run(
        cron_expression=cron_expression,
        description=description,
        enabled=enabled,
        enable_caching=enable_caching,
        end_time=end_time,
        experiment_id=experiment_id,
        experiment_name=experiment_name,
        interval_second=interval_second,
        job_name=job_name,
        max_concurrency=max_concurrency,
        no_catchup=not catchup,
        params=params,
        pipeline_package_path=pipeline_package_path,
        pipeline_id=pipeline_id,
        start_time=start_time,
        version_id=version_id
    )

    _display_recurring_run(recurring_run, output_format)


@recurring_run.command()
@click.option('-e', '--experiment-id', help='Parent experiment ID of listed recurring runs.')
@click.option(
    '-m',
    '--max-size',
    default=100,
    help="Max size of the listed recurring runs."
)
@click.pass_context
def list(ctx, max_size: int, experiment_id: str):
    """List recurring runs"""
    client = ctx.obj['client']
    output_format = ctx.obj['output']

    response = client.list_recurring_runs(
        page_size=max_size,
        sort_by="created_at desc",
        experiment_id=experiment_id
    )
    if response.jobs:
        _display_recurring_runs(response.jobs, output_format)
    else:
        if output_format == OutputFormat.json.name:
            msg = json.dumps([])
        else:
            msg = "No recurring runs found"
        click.echo(msg)


@recurring_run.command()
@click.option("--job-id", help="id of the recurring_run.")
@click.option("--job-name", help="name of the recurring_run.")
@click.pass_context
def get(ctx, job_id: str, job_name: str):
    """Get detailed information about an experiment"""
    client = ctx.obj["client"]
    output_format = ctx.obj["output"]

    response = client.get_recurring_run(job_id, job_name)
    _display_recurring_run(response, output_format)


@recurring_run.command()
@click.argument("job-id")
@click.pass_context
def delete(ctx: click.Context, job_id: str):
    """Delete a recurring run"""
    client = ctx.obj["client"]
    client.delete_recurring_run(job_id)


def _display_recurring_runs(recurring_runs, output_format):
    headers = ["Recurring Run ID", "Name"]
    data = [[rr.id, rr.name] for rr in recurring_runs]
    print_output(data, headers, output_format, table_format="grid")


def _display_recurring_run(recurring_run, output_format):
    table = [
        ["Recurring Run ID", recurring_run.id],
        ["Name", recurring_run.name],
    ]
    if output_format == OutputFormat.table.name:
        print_output([], ["Recurring Run Details"], output_format)
        print_output(table, [], output_format, table_format="plain")
    elif output_format == OutputFormat.json.name:
        print_output(dict(table), [], output_format)
