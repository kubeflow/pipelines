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