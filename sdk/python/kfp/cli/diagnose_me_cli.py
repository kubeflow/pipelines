# Lint as: python3
"""CLI interface for KFP diagnose_me tool."""

from typing import Text
import click
from .diagnose_me import dev_env
from .diagnose_me import gcp
from .diagnose_me import kubernetes_cluster as k8
from .diagnose_me import utility


@click.group()
def diagnose_me():
  """Prints diagnoses information for KFP environment."""
  pass


@diagnose_me.command()
@click.option(
    '--json',
    is_flag=True,
    help='Output in Json fromat, human readable format is set by default.')
@click.option(
    '--project-id',
    type=Text,
    help='Target project id. It will use environment default if not specified.')
@click.option(
    '--namespace',
    type=Text,
    help='Namespace to use for Kubernatees cluster. It will use environment default if not specified.'
)
@click.pass_context
def diagnose_me(ctx, json, project_id, namespace):
  """Runs environment diagnostic with specified parameters."""
  # validate kubectl, gcloud , and gsutil exist
  local_env_gcloud_sdk = gcp.get_gcp_configuration(
      gcp.Commands.GET_GCLOUD_VERSION,
      project_id=project_id,
      human_readable=False)
  for app in ['Google Cloud SDK', 'gsutil', 'kubectl']:
    if app not in local_env_gcloud_sdk.json_output:
      print(
          '%s is not installed, gcloud, gsutil and kubectl are required' % app,
          'for this app to run. Please follow instructions at',
          'https://cloud.google.com/sdk/install to install the SDK.')
      return

  # default behaviour dump all ocnfiguration
  for gcp_command in gcp.Commands:
    results = gcp.get_gcp_configuration(
        gcp_command, project_id=project_id, human_readable=not json)
    print_to_sdtout(results, gcp_command, not json)

  for k8_command in k8.Commands:
    results = k8.get_kubectl_configuration(
        k8_command, human_readable=not json)
    print_to_sdtout(results, k8_command, not json)

  for dev_env_command in dev_env.Commands:
    results = dev_env.get_dev_env_configuration(
        dev_env_command, human_readable=not json)
    print_to_sdtout(results, dev_env_command, not json)


def print_to_sdtout(results: utility.ExecutorResponse, command: gcp.Commands,
                    human_readable: bool):
  """Viewer to print the ExecutorResponse results to stdout.

  Args:
    results: Execution results to be printed out
    command: Command that was used for this execution
    human_readable: Print results in human readable format.
  """
  print('\n================', command.name, '===================\n')
  if results.has_error:
    print(' An error occurred during the executions with details as follows:',
          results.stderr)
    return
  if human_readable:
    print(results.parsed_output)
  else:
    print(results.json_output)
