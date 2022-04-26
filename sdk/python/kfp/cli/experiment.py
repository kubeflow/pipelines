import json
from typing import List

import click
import kfp_server_api
from kfp import client
from kfp.cli.output import OutputFormat
from kfp.cli.output import print_output
from kfp.cli.utils import parsing
from kfp_server_api.models.api_experiment import ApiExperiment


@click.group()
def experiment():
    """Manage experiment resources."""
    pass


@experiment.command()
@click.option(
    '-d',
    '--description',
    help=parsing.get_param_descr(client.Client.create_experiment,
                                 'description'))
@click.argument('name')
@click.pass_context
def create(ctx: click.Context, description: str, name: str):
    """Create an experiment."""
    client = ctx.obj['client']
    output_format = ctx.obj['output']

    response = client.create_experiment(name, description=description)
    _display_experiment(response, output_format)
    click.echo('Experiment created.')


@experiment.command()
@click.option(
    '--page-token',
    default='',
    help=parsing.get_param_descr(client.Client.list_experiments, 'page_token'))
@click.option(
    '-m',
    '--max-size',
    default=100,
    help=parsing.get_param_descr(client.Client.list_experiments, 'page_size'))
@click.option(
    '--sort-by',
    default='created_at desc',
    help=parsing.get_param_descr(client.Client.list_experiments, 'sort_by'))
@click.option(
    '--filter',
    help=parsing.get_param_descr(client.Client.list_experiments, 'filter'))
@click.pass_context
def list(ctx: click.Context, page_token: str, max_size: int, sort_by: str,
         filter: str):
    """List experiments."""
    client = ctx.obj['client']
    output_format = ctx.obj['output']

    response = client.list_experiments(
        page_token=page_token,
        page_size=max_size,
        sort_by=sort_by,
        filter=filter)
    if response.experiments:
        _display_experiments(response.experiments, output_format)
    else:
        if output_format == OutputFormat.json.name:
            msg = json.dumps([])
        else:
            msg = 'No experiments found'
        click.echo(msg)


@experiment.command()
@click.argument('experiment-id')
@click.pass_context
def get(ctx: click.Context, experiment_id: str):
    """Get information about an experiment."""
    client = ctx.obj['client']
    output_format = ctx.obj['output']

    response = client.get_experiment(experiment_id)
    _display_experiment(response, output_format)


@experiment.command()
@click.argument('experiment-id')
@click.pass_context
def delete(ctx: click.Context, experiment_id: str):
    """Delete an experiment."""

    confirmation = 'Caution. The RunDetails page could have an issue' \
                   ' when it renders a run that has no experiment.' \
                   ' Do you want to continue?'
    if not click.confirm(confirmation):
        return

    client = ctx.obj['client']

    client.delete_experiment(experiment_id)
    click.echo(f'Experiment {experiment_id} deleted.')


def _display_experiments(experiments: List[ApiExperiment],
                         output_format: OutputFormat):
    headers = ['Experiment ID', 'Name', 'Created at']
    data = [
        [exp.id, exp.name, exp.created_at.isoformat()] for exp in experiments
    ]
    print_output(data, headers, output_format, table_format='grid')


def _display_experiment(exp: kfp_server_api.ApiExperiment,
                        output_format: OutputFormat):
    table = [
        ['ID', exp.id],
        ['Name', exp.name],
        ['Description', exp.description],
        ['Created at', exp.created_at.isoformat()],
    ]
    if output_format == OutputFormat.table.name:
        print_output([], ['Experiment Details'], output_format)
        print_output(table, [], output_format, table_format='plain')
    elif output_format == OutputFormat.json.name:
        print_output(dict(table), [], output_format)


either_option_required = 'Either --experiment-id or --experiment-name is required.'


@experiment.command()
@click.option(
    '--experiment-id',
    default=None,
    help=parsing.get_param_descr(client.Client.archive_experiment,
                                 'experiment_id') + ' ' + either_option_required
)
@click.option(
    '--experiment-name',
    default=None,
    help='Name of the experiment.' + ' ' + either_option_required)
@click.pass_context
def archive(ctx: click.Context, experiment_id: str, experiment_name: str):
    """Archive an experiment."""
    client = ctx.obj['client']

    if (experiment_id is None) == (experiment_name is None):
        raise ValueError(
            'Either --experiment-id or --experiment-name is required.')

    if not experiment_id:
        experiment = client.get_experiment(experiment_name=experiment_name)
        experiment_id = experiment.id

    client.archive_experiment(experiment_id=experiment_id)
    click.echo(f'Experiment {experiment_id} archived.')


@experiment.command()
@click.option(
    '--experiment-id',
    default=None,
    help=parsing.get_param_descr(client.Client.unarchive_experiment,
                                 'experiment_id') + ' ' + either_option_required
)
@click.option(
    '--experiment-name',
    default=None,
    help='Name of the experiment.' + ' ' + either_option_required)
@click.pass_context
def unarchive(ctx: click.Context, experiment_id: str, experiment_name: str):
    """Unarchive an experiment."""
    client = ctx.obj['client']

    if (experiment_id is None) == (experiment_name is None):
        raise ValueError(
            'Either --expriment-id or --experiment-name is required.')

    if not experiment_id:
        experiment = client.get_experiment(experiment_name=experiment_name)
        experiment_id = experiment.id

    client.archive_experiment(experiment_id=experiment_id)
    click.echo(f'Experiment {experiment_id} unarchived.')
