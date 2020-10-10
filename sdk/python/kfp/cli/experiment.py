import click
import logging
from tabulate import tabulate


@click.group()
def experiment():
    """Manage experiment resources"""
    pass


@experiment.command()
@click.option(
    '-d',
    '--description',
    help="Description of the experiment."
)
@click.argument("name")
@click.pass_context
def create(ctx, description, name):
    """Create an experiment"""
    client = ctx.obj["client"]

    response = client.create_experiment(name, description=description)
    logging.info("Experiment {} has been submitted\n".format(name))
    _display_experiment(response)


@experiment.command()
@click.option(
    '-m',
    '--max-size',
    default=100,
    help="Max size of the listed experiments."
)
@click.pass_context
def list(ctx, max_size):
    """List experiments"""
    client = ctx.obj['client']

    response = client.experiments.list_experiment(
        page_size=max_size,
        sort_by="created_at desc"
    )
    if response.experiments:
        _display_experiments(response.experiments)
    else:
        logging.info("No experiments found")


@experiment.command()
@click.argument("experiment-id")
@click.pass_context
def get(ctx, experiment_id):
    """Get detailed information about an experiment"""
    client = ctx.obj["client"]

    response = client.get_experiment(experiment_id)
    _display_experiment(response)


@experiment.command()
@click.argument("experiment-id")
@click.pass_context
def delete(ctx, experiment_id):
    """Delete an experiment"""

    confirmation = "Caution. The RunDetails page could have an issue" \
                   " when it renders a run that has no experiment." \
                   " Do you want to continue?"
    if not click.confirm(confirmation):
        return

    client = ctx.obj["client"]

    client.experiments.delete_experiment(id=experiment_id)
    print("{} is deleted.".format(experiment_id))


def _display_experiments(experiments):
    headers = ["Experiment ID", "Name", "Created at"]
    data = [[
        exp.id,
        exp.name,
        exp.created_at.isoformat()
    ] for exp in experiments]
    print(tabulate(data, headers=headers, tablefmt="grid"))


def _display_experiment(exp):
    print(tabulate([], headers=["Experiment Details"]))
    table = [
        ["ID", exp.id],
        ["Name", exp.name],
        ["Description", exp.description],
        ["Created at", exp.created_at.isoformat()],
    ]
    print(tabulate(table, tablefmt="plain"))
