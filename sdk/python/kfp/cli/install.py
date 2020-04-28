"""CLI interface for KFP installation tool."""

import json as json_library
import sys
from typing import Dict, Text
import click
from .install_cli import prerequest
from . install_cli import project_id_resolver
from .install_cli import core_installer
from .install_cli import cluster_resolver

@click.group()
def install():
  pass


@install.command()
@click.option(
    '--gcp-project-id',
    type=Text,
    help='Target project id. It will use environment default if not specified.')
@click.option(
    '--gcp-cluster-id',
    type=Text,
    help='Namespace to use for Kubernetes cluster.'
)
@click.option(
    '--gcp-cluster-zone',
    type=Text,
    help='Namespace to use for Kubernetes cluster.'
)
@click.option(
    '--gcp-create-cluster',
    type=click.Choice(['true', 'false']),
    help='Whether create cluster or use existing or ask interactive')
@click.option(
    '--namespace',
    type=Text,
    help='Namespace to use for Kubernetes cluster.'
)
@click.pass_context
def install(ctx, gcp_project_id, gcp_cluster_id, gcp_cluster_zone, gcp_create_cluster, namespace):
  """Kubeflow Pipelines CLI Installer"""

  # Show welcome messages
  prerequest.show_welcome_message()

  # Check whether required tools are installed
  prerequest.check_tools()

  # Check current user
  gcp_account = prerequest.check_gcp_account()

  # Resolve GCP Project ID
  gcp_project_id = project_id_resolver.resolve_project_id(gcp_project_id)

  # Resolve GCP Cluster
  gcp_cluster_id, gcp_cluster_zone = cluster_resolver.resolve_cluster(
      gcp_project_id, gcp_create_cluster, gcp_cluster_id, gcp_cluster_zone)

  # Resolve Namespace

  # Resolve GCS Default Bucket

  # Resolve GCS CloudSQL

  # Resolve GPU node pool

  # Resolve KFP
  core_installer.install()
