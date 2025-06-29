import click
from kfp import client
from kfp.cli import output
from kfp.cli.utils import parsing


@click.group()
def experiment():
    """Manage experiment resources."""


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
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    experiment = client_obj.create_experiment(name, description=description)
    output.print_output(
        experiment,
        output.ModelType.EXPERIMENT,
        output_format,
    )


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
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    response = client_obj.list_experiments(
        page_token=page_token,
        page_size=max_size,
        sort_by=sort_by,
        filter=filter)
    output.print_output(
        response.experiments or [],
        output.ModelType.EXPERIMENT,
        output_format,
    )


@experiment.command()
@click.argument('experiment-id')
@click.pass_context
def get(ctx: click.Context, experiment_id: str):
    """Get information about an experiment."""
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    experiment = client_obj.get_experiment(experiment_id)
    output.print_output(
        experiment,
        output.ModelType.EXPERIMENT,
        output_format,
    )


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

    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    client_obj.delete_experiment(experiment_id)
    output.print_deleted_text('experiment', experiment_id, output_format)


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
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    if (experiment_id is None) == (experiment_name is None):
        raise ValueError(either_option_required)

    if not experiment_id:
        experiment = client_obj.get_experiment(experiment_name=experiment_name)
        experiment_id = experiment.experiment_id

    client_obj.archive_experiment(experiment_id=experiment_id)
    if experiment_id:
        experiment = client_obj.get_experiment(experiment_id=experiment_id)
    else:
        experiment = client_obj.get_experiment(experiment_name=experiment_name)
    output.print_output(
        experiment,
        output.ModelType.EXPERIMENT,
        output_format,
    )


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
    client_obj: client.Client = ctx.obj['client']
    output_format = ctx.obj['output']

    if (experiment_id is None) == (experiment_name is None):
        raise ValueError(either_option_required)

    if not experiment_id:
        experiment = client_obj.get_experiment(experiment_name=experiment_name)
        experiment_id = experiment.experiment_id

    client_obj.unarchive_experiment(experiment_id=experiment_id)
    if experiment_id:
        experiment = client_obj.get_experiment(experiment_id=experiment_id)
    else:
        experiment = client_obj.get_experiment(experiment_name=experiment_name)
    output.print_output(
        experiment,
        output.ModelType.EXPERIMENT,
        output_format,
    )
