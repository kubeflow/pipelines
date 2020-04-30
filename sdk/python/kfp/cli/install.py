"""CLI interface for KFP installation tool."""

import json as json_library
import sys
from typing import Dict, Text
import click
from .install_cli import prerequest, project_id_resolver, kfp_installer, cluster_resolver, gcs_resolver, kfp_input_resolver, cloud_sql_resolver, gpu_resolver

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
    help='Namespace to use for Kubernetes cluster.')
@click.option(
    '--gcp-cluster-zone',
    type=Text,
    help='Namespace to use for Kubernetes cluster.')
@click.option(
    '--gcs-default-bucket',
    type=Text,
    help='GCS default bucket. No "gs://" prefix')
@click.option(
    '--instance_name',
    type=Text,
    help='The instance name of the Kubeflow Pipelines installation')
@click.option(
    '--namespace',
    type=Text,
    help='Namespace to use for Kubernetes cluster.')
@click.option(
    '--enable_managed_storage',
    type=click.Choice(['true','false']),
    help='Whether use CloudSQL & GCS to store KFP system data.'
       + ' If false, it will use in-cluster volume while will be deleted in uninstallation.')
@click.option(
    '--cloud_sql_instance_name',
    type=Text,
    help='CloudSQL instance name formatted as projectId::region::name.')
@click.option(
    '--cloud_sql_username',
    type=Text,
    help='CloudSQL username.')
@click.option(
    '--cloud_sql_password',
    type=Text,
    help="CloudSQL password. Not suggested to pass password here.")
@click.pass_context
def install(ctx, gcp_project_id, gcp_cluster_id, gcp_cluster_zone,
            gcs_default_bucket,
            instance_name, namespace,
            enable_managed_storage,
            cloud_sql_instance_name, cloud_sql_username, cloud_sql_password):
  """Kubeflow Pipelines CLI Installer"""

  # Show welcome messages
  prerequest.show_welcome_message()

  # Check whether required tools are installed
  prerequest.check_tools()

  # Check whether already login, if not or expired, gcloud auth login
  prerequest.check_gcloud_auth_login()

  # Check current user
  gcp_account = prerequest.check_gcp_account()

  # Resolve GCP Project ID
  gcp_project_id = project_id_resolver.resolve_project_id(gcp_project_id)

  # Resolve GCP Cluster
  gcp_cluster_id, gcp_cluster_zone = cluster_resolver.resolve_cluster(
      gcp_project_id, gcp_cluster_id, gcp_cluster_zone)

  # Resolve GCS Default Bucket
  gcs_default_bucket = gcs_resolver.resolve_gcs_default_bucket(
      gcp_project_id, gcs_default_bucket)

  # Resolve AppName & Namespace inputs (don't create Namespace here)
  instance_name, namespace = kfp_input_resolver.resolve_kfp_input(instance_name, namespace)

  # Resolve GCS CloudSQL (only required when enable Managed Storage)
  result = cloud_sql_resolver.resolve_cloud_sql(enable_managed_storage,
      cloud_sql_instance_name, cloud_sql_username, cloud_sql_password)
  enable_managed_storage, cloud_sql_instance_name, cloud_sql_username, cloud_sql_password = result

  # Resolve GPU node pool (only required when enable GPU)
  gpu_resolver.resolve_gpu()

  # Resolve KFP
  kfp_installer.install(gcp_project_id, gcp_cluster_id, gcp_cluster_zone, gcs_default_bucket,
      instance_name, namespace,
      enable_managed_storage, cloud_sql_instance_name, cloud_sql_username, cloud_sql_password)
