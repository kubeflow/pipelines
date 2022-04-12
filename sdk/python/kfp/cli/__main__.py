import logging
import sys

import click
from kfp.cli import cli
from kfp.cli import components
from kfp.cli import diagnose_me_cli
from kfp.cli import experiment
from kfp.cli import pipeline
from kfp.cli import recurring_run
from kfp.cli import run


def main():
    logging.basicConfig(format='%(message)s', level=logging.INFO)
    cli.cli.add_command(run.run)
    cli.cli.add_command(recurring_run.recurring_run)
    cli.cli.add_command(pipeline.pipeline)
    cli.cli.add_command(diagnose_me_cli.diagnose_me)
    cli.cli.add_command(experiment.experiment)
    cli.cli.add_command(components.components)
    try:
        cli.cli(obj={}, auto_envvar_prefix='KFP')
    except Exception as e:
        click.echo(str(e), err=True)
        sys.exit(1)
