"""CLI interface for KFP diagnose_me tool."""

import json as json_library
import sys
from typing import Dict, List, Text, Union

import click
from kfp.cli.diagnose_me import dev_env
from kfp.cli.diagnose_me import gcp
from kfp.cli.diagnose_me import kubernetes_cluster
from kfp.cli.diagnose_me import kubernetes_cluster as k8
from kfp.cli.diagnose_me import utility

ResultsType = Dict[Union[gcp.Commands, dev_env.Commands,
                         kubernetes_cluster.Commands], utility.ExecutorResponse]


@click.command()
@click.option(
    '-j',
    '--json',
    is_flag=True,
    help='Output in Json format, human readable format is set by default.')
@click.option(
    '-p',
    '--project-id',
    type=Text,
    help='Target project id. It will use environment default if not specified.')
@click.option(
    '-n',
    '--namespace',
    type=Text,
    help='Namespace to use for Kubernetes cluster.all-namespaces is used if not specified.'
)
@click.pass_context
def diagnose_me(ctx: click.Context, json: bool, project_id: str,
                namespace: str):
    """Runs KFP environment diagnostic."""
    # validate kubectl, gcloud , and gsutil exist
    local_env_gcloud_sdk = gcp.get_gcp_configuration(
        gcp.Commands.GET_GCLOUD_VERSION,
        project_id=project_id,
        human_readable=False)
    for app in ['Google Cloud SDK', 'gsutil', 'kubectl']:
        if app not in local_env_gcloud_sdk.json_output:
            raise RuntimeError(
                f'{app} is not installed, gcloud, gsutil and kubectl are required '
                + 'for this app to run. Please follow instructions at ' +
                'https://cloud.google.com/sdk/install to install the SDK.')

    click.echo('Collecting diagnostic information ...', file=sys.stderr)

    # default behaviour dump all configurations
    results: ResultsType = {
        gcp_command: gcp.get_gcp_configuration(
            gcp_command, project_id=project_id, human_readable=not json)
        for gcp_command in gcp.Commands
    }

    for k8_command in k8.Commands:
        results[k8_command] = k8.get_kubectl_configuration(
            k8_command, human_readable=not json)

    for dev_env_command in dev_env.Commands:
        results[dev_env_command] = dev_env.get_dev_env_configuration(
            dev_env_command, human_readable=not json)

    print_to_sdtout(results, not json)


def print_to_sdtout(results: ResultsType, human_readable: bool):
    """Viewer to print the ExecutorResponse results to stdout.

    Args:
      results: A dictionary with key:command names and val: Execution response
      human_readable: Print results in human readable format. If set to True
        command names will be printed as visual delimiters in new lines. If False
        results are printed as a dictionary with command as key.
    """

    output_dict = {}
    human_readable_result: List[str] = []
    for key, val in results.items():
        if val.has_error:
            output_dict[
                key.
                name] = f'Following error occurred during the diagnoses: {val.stderr}'
            continue

        output_dict[key.name] = val.json_output
        human_readable_result.extend(
            (f'================ {key.name} ===================',
             val.parsed_output))

    if human_readable:
        result = '\n'.join(human_readable_result)
    else:
        result = json_library.dumps(
            output_dict, sort_keys=True, indent=2, separators=(',', ': '))

    click.echo(result)
