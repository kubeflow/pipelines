"""CLI interface for KFP installation tool."""

import json as json_library
import sys
from typing import Dict, Text
import click
from .install_cli import prerequest
from . install_cli import project_id_resolver
from .install_cli import core_installer

@click.group()
def install():
  pass


@install.command()
@click.option(
    '--project-id',
    type=Text,
    help='Target project id. It will use environment default if not specified.')
@click.option(
    '--namespace',
    type=Text,
    help='Namespace to use for Kubernetes cluster.'
)
@click.pass_context
def install(ctx, project_id, namespace):
  """Kubeflow Pipelines CLI Installer"""

  # Show welcome messages
  prerequest.show_welcome_message()

  # Check whether required tools are installed
  prerequest.check_tools()

  # Check current user
  gcp_account = prerequest.check_gcp_account()

  # Resolve GCP Project ID
  resolved_project_id = project_id_resolver.resolve_project_id(project_id)

  # Resolve GCP Cluster

  # Resolve Namespace

  # Resolve GCS Default Bucket

  # Resolve GCS CloudSQL

  # Resolve GPU node pool

  core_installer.install()
