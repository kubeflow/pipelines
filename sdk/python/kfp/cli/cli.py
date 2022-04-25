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

from itertools import chain

import click
from kfp.cli import component
from kfp.cli import diagnose_me_cli
from kfp.cli import experiment
from kfp.cli import pipeline
from kfp.cli import recurring_run
from kfp.cli import run
from kfp.cli.output import OutputFormat
from kfp.cli.utils import aliased_plurals_group
from kfp.client import Client

COMMANDS = {
    'client': {
        run.run, recurring_run.recurring_run, experiment.experiment,
        pipeline.pipeline
    },
    'no_client': {diagnose_me_cli.diagnose_me, component.component}
}


@click.group(
    cls=aliased_plurals_group.AliasedPluralsGroup,
    commands=list(chain.from_iterable(COMMANDS.values())))  # type: ignore
@click.option('--endpoint', help='Endpoint of the KFP API service to connect.')
@click.option('--iap-client-id', help='Client ID for IAP protected endpoint.')
@click.option(
    '-n',
    '--namespace',
    default='kubeflow',
    show_default=True,
    help='Kubernetes namespace to connect to the KFP API.')
@click.option(
    '--other-client-id',
    help='Client ID for IAP protected endpoint to obtain the refresh token.')
@click.option(
    '--other-client-secret',
    help='Client ID for IAP protected endpoint to obtain the refresh token.')
@click.option(
    '--output',
    type=click.Choice(list(map(lambda x: x.name, OutputFormat))),
    default=OutputFormat.table.name,
    show_default=True,
    help='The formatting style for command output.')
@click.pass_context
def cli(ctx: click.Context, endpoint: str, iap_client_id: str, namespace: str,
        other_client_id: str, other_client_secret: str, output: OutputFormat):
    """kfp is the command line interface to KFP service.

    Feature stage:
    [Alpha](https://github.com/kubeflow/pipelines/blob/07328e5094ac2981d3059314cc848fbb71437a76/docs/release/feature-stages.md#alpha)
    """
    client_commands = set(
        chain.from_iterable([
            (command.name, f'{command.name}s')
            for command in COMMANDS['client']  # type: ignore
        ]))

    if ctx.invoked_subcommand in client_commands:
        ctx.obj['client'] = Client(endpoint, iap_client_id, namespace,
                                   other_client_id, other_client_secret)
        ctx.obj['namespace'] = namespace
        ctx.obj['output'] = output
