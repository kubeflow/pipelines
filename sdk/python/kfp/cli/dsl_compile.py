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
from typing import Any, Callable, List, Optional

import click
from kfp import compiler
from kfp.components import pipeline_context


def collect_pipeline_from_module(
        target_module: types.ModuleType) -> List[Callable[..., Any]]:
    pipelines = []
    module_attrs = dir(target_module)
    for attr in module_attrs:
        obj = getattr(target_module, attr)
        if pipeline_context.Pipeline.is_pipeline_func(obj):
            pipelines.append(obj)
    if len(pipelines) == 1:
        return pipelines[0]
    else:
        raise ValueError(
            f'Expect one pipeline function in module {target_module}, got {len(pipelines)}: {pipelines}. Please specify the pipeline function name with --function.'
        )


@click.command()
@click.option(
    '--py', type=str, required=True, help='Local absolute path to a py file.')
@click.option(
    '--output',
    type=str,
    required=True,
    help='Local path to the output PipelineJob JSON file.')
@click.option(
    '--function',
    'function_name',
    type=str,
    default=None,
    help='The name of the function to compile if there are multiple.')
@click.option(
    '--pipeline-parameters',
    type=str,
    default=None,
    help='The pipeline parameters in JSON dict format.')
@click.option(
    '--disable-type-check',
    is_flag=True,
    default=False,
    help='Disable the type check. Default: type check is not disabled.')
def dsl_compile(
    py: str,
    output: str,
    function_name: Optional[str] = None,
    pipeline_parameters: str = None,
    disable_type_check: bool = True,
) -> None:
    """Compiles a pipeline written in a .py file."""
    sys.path.insert(0, os.path.dirname(py))
    try:
        filename = os.path.basename(py)
        module_name = os.path.splitext(filename)[0]
        if function_name:
            module = __import__(module_name, fromlist=[function_name])
            if not hasattr(module, function_name):
                raise ValueError(
                    f'Pipeline function or component "{function_name}" not found in module {filename}.'
                )
            pipeline_func = getattr(module, function_name)
        else:
            pipeline_func = collect_pipeline_from_module(
                target_module=__import__(module_name))

        try:
            parsed_parameters = json.loads(
                pipeline_parameters) if pipeline_parameters is not None else {}
        except json.JSONDecodeError as e:
            logging.error(
                f"Failed to parse --pipeline-parameters argument: {pipeline_parameters}"
            )
            raise e
        compiler.Compiler().compile(
            pipeline_func=pipeline_func,
            pipeline_parameters=parsed_parameters,
            package_path=output,
            type_check=not disable_type_check)
    finally:
        del sys.path[0]


def main():
    logging.basicConfig(format='%(message)s', level=logging.INFO)
    try:
        dsl_compile(obj={}, auto_envvar_prefix='KFP')
    except Exception as e:
        click.echo(str(e), err=True)
        sys.exit(1)
