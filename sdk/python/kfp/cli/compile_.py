# Copyright 2020-2022 The Kubeflow Authors
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
"""KFP SDK compiler CLI tool."""

import json
import logging
import os
import sys
import types
from typing import Callable, Dict, Optional

import click
from kfp import compiler
from kfp.components import base_component
from kfp.components import graph_component


def is_pipeline_func(func: Callable) -> bool:
    """Checks if a function is a pipeline function.

    Args:
        func (Callable): The function to check.

    Returns:
        bool: True if the function is a pipeline function.
    """
    return isinstance(func, graph_component.GraphComponent)


def is_component_func(func: Callable) -> bool:
    """Checks if a function is a component function.

    Args:
        func (Callable): The function to check.

    Returns:
        bool: True if the function is a component function.
    """
    return not is_pipeline_func(func) and isinstance(
        func, base_component.BaseComponent)


def collect_pipeline_or_component_from_module(
        target_module: types.ModuleType) -> base_component.BaseComponent:
    pipelines = []
    components = []
    module_attrs = dir(target_module)
    for attr in module_attrs:
        obj = getattr(target_module, attr)
        if is_pipeline_func(obj):
            pipelines.append(obj)
        elif is_component_func(obj):
            components.append(obj)

    if len(pipelines) == 1:
        return pipelines[0]
    elif not pipelines and len(components) == 1:
        return components[0]
    else:
        raise ValueError(
            f'Expected one pipeline or one component in module {target_module}. Got {len(pipelines)} pipeline(s): {[p.name for p in pipelines]} and {len(components)} component(s): {[c.name for c in components]}. Please specify which pipeline or component to compile using --function.'
        )


def collect_pipeline_or_component_func(
        python_file: str,
        function_name: Optional[str]) -> base_component.BaseComponent:
    sys.path.insert(0, os.path.dirname(python_file))
    try:
        filename = os.path.basename(python_file)
        module_name = os.path.splitext(filename)[0]
        if function_name is None:
            return collect_pipeline_or_component_from_module(
                target_module=__import__(module_name))

        module = __import__(module_name, fromlist=[function_name])
        if not hasattr(module, function_name):
            raise ValueError(
                f'Pipeline function or component "{function_name}" not found in module {filename}.'
            )

        return getattr(module, function_name)

    finally:
        del sys.path[0]


def parse_parameters(parameters: Optional[str]) -> Dict:
    try:
        return json.loads(parameters) if parameters is not None else {}
    except json.JSONDecodeError as e:
        logging.error(
            f'Failed to parse --pipeline-parameters argument: {parameters}')
        raise e


@click.command(name='compile')
@click.option(
    '--py',
    type=click.Path(exists=True, dir_okay=False),
    required=True,
    help='Local absolute path to a py file.')
@click.option(
    '--output',
    type=click.Path(exists=False, dir_okay=False),
    required=True,
    help='Path to write the compiled result.')
@click.option(
    '--function',
    'function_name',
    type=str,
    default=None,
    help='The name of the pipeline or component to compile if there are multiple.'
)
@click.option(
    '--pipeline-parameters',
    type=str,
    default=None,
    help='The pipeline or component input parameters in JSON dict format.')
@click.option(
    '--disable-type-check',
    is_flag=True,
    default=False,
    help='Whether to disable type checking.')
def compile_(
    py: str,
    output: str,
    function_name: Optional[str] = None,
    pipeline_parameters: Optional[str] = None,
    disable_type_check: bool = False,
) -> None:
    """Compiles a pipeline or component written in a .py file."""
    pipeline_func = collect_pipeline_or_component_func(
        python_file=py, function_name=function_name)
    parsed_parameters = parse_parameters(parameters=pipeline_parameters)
    package_path = os.path.join(os.getcwd(), output)
    compiler.Compiler().compile(
        pipeline_func=pipeline_func,
        pipeline_parameters=parsed_parameters,
        package_path=package_path,
        type_check=not disable_type_check)

    click.echo(package_path)


def main():
    logging.basicConfig(format='%(message)s', level=logging.INFO)
    try:
        compile_.help = '(Deprecated. Please use `kfp dsl compile` instead.)\n\n' + compile_.help

        click.echo(
            '`dsl-compile` is deprecated. Please use `kfp dsl compile` instead.',
            err=True)

        compile_(obj={}, auto_envvar_prefix='KFP')
    except Exception as e:
        click.echo(str(e), err=True)
        sys.exit(1)
