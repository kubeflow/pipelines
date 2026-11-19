# Copyright 2020-2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Compile the KFP Python DSL to native Argo resources."""

import json
import logging
import os
import sys
import types
from typing import Callable, Dict, Optional

import click

from kfp import compiler
from kfp.dsl import base_component, graph_component
from kfp.dsl.pipeline_context import Pipeline


def is_pipeline_func(func: Callable) -> bool:
    return isinstance(func, graph_component.GraphComponent)


def is_component_func(func: Callable) -> bool:
    return not is_pipeline_func(func) and isinstance(
        func, base_component.BaseComponent
    )


def collect_pipeline_or_component_from_module(
    target_module: types.ModuleType,
) -> base_component.BaseComponent:
    pipelines, components = [], []
    for attr in dir(target_module):
        obj = getattr(target_module, attr)
        if is_pipeline_func(obj):
            pipelines.append(obj)
        elif is_component_func(obj):
            components.append(obj)
    if len(pipelines) == 1:
        return pipelines[0]
    if not pipelines and len(components) == 1:
        return components[0]
    raise ValueError(
        f'Expected one pipeline or one component in module {target_module}. Got {len(pipelines)} pipeline(s): {[p.name for p in pipelines]} and {len(components)} component(s): {[c.name for c in components]}. Please specify which pipeline or component to compile using --function.'
    )


def collect_pipeline_or_component_func(
    python_file: str, function_name: Optional[str]
) -> base_component.BaseComponent:
    sys.path.insert(0, os.path.dirname(python_file))
    try:
        module_name = os.path.splitext(os.path.basename(python_file))[0]
        if function_name is None:
            return collect_pipeline_or_component_from_module(
                __import__(module_name)
            )
        module = __import__(module_name, fromlist=[function_name])
        if not hasattr(module, function_name):
            raise ValueError(
                f'Pipeline function or component "{function_name}" not found in module {os.path.basename(python_file)}.'
            )
        return getattr(module, function_name)
    finally:
        del sys.path[0]


def parse_parameters(parameters: Optional[str]) -> Dict:
    value = json.loads(parameters) if parameters is not None else {}
    if not isinstance(value, dict):
        raise ValueError(
            'Parameters must be a JSON object, not a list or scalar.'
        )
    return value


@click.command(name='compile')
@click.option(
    '--py',
    type=click.Path(exists=True, dir_okay=False),
    required=True,
    help='Trusted pipeline Python source (module-level code executes).',
)
@click.option(
    '--output',
    type=click.Path(dir_okay=False),
    required=True,
    help='Destination Argo YAML file.',
)
@click.option(
    '--function',
    'function_name',
    default=None,
    help='Pipeline/component function when the module has several.',
)
@click.option(
    '--pipeline-parameters', default=None, help='Input values as a JSON object.'
)
@click.option('--pipeline-name', default=None, help='Logical pipeline name.')
@click.option(
    '--pipeline-display-name', default=None, help='Human-readable display name.'
)
@click.option(
    '--namespace', default=None, help='Namespace for the native resource.'
)
@click.option(
    '--resource-kind',
    type=click.Choice(['WorkflowTemplate', 'Workflow', 'CronWorkflow']),
    default='WorkflowTemplate',
    show_default=True,
)
@click.option(
    '--schedule',
    default=None,
    help='Cron expression; required only for CronWorkflow, which starts suspended.',
)
@click.option(
    '--artifact-repository', default='s3://argo-artifacts', show_default=True
)
@click.option('--service-account', default='argo-workflow', show_default=True)
@click.option('--cache-config-map', default='kfp-cache', show_default=True)
@click.option(
    '--image-digests',
    default=None,
    help='JSON map of local image names to verified sha256 build identities.',
)
@click.option(
    '--disable-caching',
    is_flag=True,
    help='Disable all memoization, including explicit per-task settings. Useful for offline previews.',
)
@click.option(
    '--disable-execution-caching-by-default',
    is_flag=True,
    envvar='KFP_DISABLE_EXECUTION_CACHING_BY_DEFAULT',
    help='Disable the DSL caching default while loading this source.',
)
def compile_(
    py: str,
    output: str,
    function_name: Optional[str] = None,
    pipeline_parameters: Optional[str] = None,
    pipeline_name: Optional[str] = None,
    pipeline_display_name: Optional[str] = None,
    namespace: Optional[str] = None,
    resource_kind: str = 'WorkflowTemplate',
    schedule: Optional[str] = None,
    artifact_repository: str = 's3://argo-artifacts',
    service_account: str = 'argo-workflow',
    cache_config_map: str = 'kfp-cache',
    image_digests: Optional[str] = None,
    disable_caching: bool = False,
    disable_execution_caching_by_default: bool = False,
) -> None:
    """Compile Python to native Argo YAML; no KFP IR or local-runner
    fallback."""
    original = Pipeline.get_execution_caching_default()
    try:
        Pipeline._execution_caching_default = (
            not disable_execution_caching_by_default
        )
        pipeline = collect_pipeline_or_component_func(py, function_name)
        compiler.Compiler(
            artifact_repository=artifact_repository,
            service_account_name=service_account,
            cache_config_map=cache_config_map,
            image_digests=parse_parameters(image_digests),
            disable_caching=disable_caching,
        ).compile(
            pipeline,
            os.path.abspath(output),
            pipeline_name=pipeline_name,
            pipeline_display_name=pipeline_display_name,
            pipeline_parameters=parse_parameters(pipeline_parameters),
            resource_kind=resource_kind,
            schedule=schedule,
            namespace=namespace,
        )
        click.echo(
            f'Pipeline code was successfully compiled with the output saved to {os.path.abspath(output)}'
        )
    except (ValueError, TypeError) as error:
        raise click.ClickException(str(error)) from error
    finally:
        Pipeline._execution_caching_default = original


def main():
    logging.basicConfig(format='%(message)s', level=logging.INFO)
    click.echo(
        '`dsl-compile` is deprecated. Please use `kfp dsl compile` instead.',
        err=True,
    )
    compile_(obj={}, auto_envvar_prefix='KFP')
