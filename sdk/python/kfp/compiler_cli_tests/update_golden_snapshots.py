import os
import subprocess
import sys
from typing import List

import click
from click import testing
from kfp.cli import cli
from kfp.cli import cli_test

TEST_DATA_DIR = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "test_data")


def call_and_raise(cmd: str) -> None:
    runner = testing.CliRunner()
    runner.invoke(cli=cli.cli, args=cmd.split(' '), catch_exceptions=False)


@click.command()
@click.option(
    '--pipelines',
    is_flag=True,
    default=False,
    help='Update golden snapshots for pipelines.')
@click.option(
    '--components',
    is_flag=True,
    default=False,
    help='Update golden snapshots for components.')
def update_snapshots(pipelines: bool, components: bool):
    if pipelines:
        python_files = [
            f for f in os.listdir(TEST_DATA_DIR) if f.endswith('.py')
        ]

        for file in python_files:
            python_file = os.path.join(TEST_DATA_DIR, file)
            call_and_raise(f'python {python_file}')
    if components:
        for data in cli_test.COMPILE_COMPONENTS_TEST_CASES:
            print(data)
            file = data['file']
            component_name = data['component_name']
            python_file = os.path.join(TEST_DATA_DIR, file + '.py')
            yaml_file = os.path.join(TEST_DATA_DIR,
                                     f'{file}-{component_name}.yaml')
            call_and_raise(
                f'dsl compile --py {python_file} --function {component_name} --output {yaml_file}'
            )


if __name__ == '__main__':
    update_snapshots()
