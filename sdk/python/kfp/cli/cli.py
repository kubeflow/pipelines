# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Authoring CLI. Submission and management belong to Kubernetes and Argo."""

import click

import kfp
from kfp.cli import component, dsl, visualize


@click.group(context_settings={'help_option_names': ['-h', '--help']})
@click.version_option(version=kfp.__version__)
def cli():
    """Compile, build and preview native Argo pipelines."""


cli.add_command(component.component)
cli.add_command(dsl.dsl)
cli.add_command(visualize.visualize)
