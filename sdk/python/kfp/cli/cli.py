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
from .._client import Client
from .run import run

@click.group()
@click.option('--endpoint', help='Endpoint of the KFP API service to connect.')
@click.option('--iap-client-id', help='Client ID for IAP protected endpoint.')
@click.option('-n', '--namespace', default='kubeflow', help='Kubernetes namespace to connect to the KFP API.')
@click.pass_context
def cli(ctx, endpoint, iap_client_id, namespace):
    """kfp is the command line interface to KFP service."""
    ctx.obj['client'] = Client(endpoint, iap_client_id, namespace)
    ctx.obj['namespace']= namespace

def main():
    cli.add_command(run)
    cli(obj={}, auto_envvar_prefix='KFP')