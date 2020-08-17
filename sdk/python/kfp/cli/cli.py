# Copyright 2018 Google LLC
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

import click
import logging
import sys
from .._client import Client
from .run import run
from .pipeline import pipeline
from .diagnose_me_cli import diagnose_me
from .experiment import experiment

@click.group()
@click.option('--endpoint', help='Endpoint of the KFP API service to connect.')
@click.option('--iap-client-id', help='Client ID for IAP protected endpoint.')
@click.option('-n', '--namespace', default='kubeflow', help='Kubernetes namespace to connect to the KFP API.')
@click.option('--other-client-id', help='Client ID for IAP protected endpoint to obtain the refresh token.')
@click.option('--other-client-secret', help='Client ID for IAP protected endpoint to obtain the refresh token.')
@click.pass_context
def cli(ctx, endpoint, iap_client_id, namespace, other_client_id, other_client_secret):
    """kfp is the command line interface to KFP service."""
    if ctx.invoked_subcommand == 'diagnose_me':
          # Do not create a client for diagnose_me
          return
    ctx.obj['client'] = Client(endpoint, iap_client_id, namespace, other_client_id, other_client_secret)
    ctx.obj['namespace']= namespace

def main():
    logging.basicConfig(format='%(message)s', level=logging.INFO)
    cli.add_command(run)
    cli.add_command(pipeline)
    cli.add_command(diagnose_me,'diagnose_me')
    cli.add_command(experiment)
    try:
        cli(obj={}, auto_envvar_prefix='KFP')
    except Exception as e:
        logging.error(e)
        sys.exit(1)
