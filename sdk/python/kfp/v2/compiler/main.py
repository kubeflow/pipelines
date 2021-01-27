# Copyright 2020 Google LLC
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

import argparse
import json
import os
import sys
from typing import Any, Callable, List, Mapping, Optional

import kfp.dsl as dsl
import kfp.v2.compiler as compiler


def parse_arguments() -> argparse.Namespace:
  """Parse command line arguments."""

  parser = argparse.ArgumentParser()
  parser.add_argument(
      '--py', type=str, required=True, help='local absolute path to a py file.')
  parser.add_argument(
      '--function',
      type=str,
      help='The name of the function to compile if there are multiple.')
  parser.add_argument(
      '--pipeline-root',
      type=str,
      required=True,
      help='The root of the pipeline outputs.')
  parser.add_argument(
      '--pipeline-parameters',
      type=json.loads,
      help='The pipeline parameters in JSON dict format.')
  parser.add_argument(
      '--namespace', type=str, help='The namespace for the pipeline function')
  parser.add_argument(
      '--output',
      type=str,
      required=True,
      help='local path to the output PipelineJob json file.')
  parser.add_argument(
      '--disable-type-check',
      action='store_true',
      help='disable the type check, default is enabled.')

  args = parser.parse_args()
  return args


def _compile_pipeline_function(pipeline_funcs: List[Callable],
                               function_name: Optional[str], pipeline_root: str,
                               pipeline_parameters: Optional[Mapping[str, Any]],
                               output_path: str, type_check: bool) -> None:
  """Compiles a pipeline function.

  Args:
    pipeline_funcs: A list of pipeline_functions.
    function_name: The name of the pipeline function to compile if there were
      multiple.
    pipeline_root: The root output directory for pipeline runtime.
    pipeline_parameters: The pipeline parameters as a dict of {name: value}.
    output_path: The output path of the compiled result.
    type_check: Whether to enable the type checking.
  """
  if len(pipeline_funcs) == 0:
    raise ValueError(
        'A function with @dsl.pipeline decorator is required in the py file.')

  if len(pipeline_root) == 0:
    raise ValueError(
        'A pipeline-root (e.g.: a GCS URI) is required. Please specify --pipeline-root.'
    )

  if len(pipeline_funcs) > 1 and not function_name:
    func_names = [x.__name__ for x in pipeline_funcs]
    raise ValueError(
        'There are multiple pipelines: %s. Please specify --function.' %
        func_names)

  if function_name:
    pipeline_func = next(
        (x for x in pipeline_funcs if x.__name__ == function_name), None)
    if not pipeline_func:
      raise ValueError('The function "%s" does not exist. '
                       'Did you forget @dsl.pipeline decoration?' %
                       function_name)
  else:
    pipeline_func = pipeline_funcs[0]

  compiler.Compiler().compile(
      pipeline_func=pipeline_func,
      pipeline_root=pipeline_root,
      pipeline_parameters=pipeline_parameters,
      output_path=output_path,
      type_check=type_check)


class PipelineCollectorContext():

  def __enter__(self):
    pipeline_funcs = []

    def add_pipeline(func: Callable) -> Callable:
      pipeline_funcs.append(func)
      return func

    self.old_handler = dsl._pipeline._pipeline_decorator_handler
    dsl._pipeline._pipeline_decorator_handler = add_pipeline
    return pipeline_funcs

  def __exit__(self, *args):
    dsl._pipeline._pipeline_decorator_handler = self.old_handler


def compile_pyfile(pyfile: str, function_name: Optional[str],
                   pipeline_root: str,
                   pipeline_parameters: Optional[Mapping[str, Any]],
                   output_path: str, type_check: bool) -> None:
  """Compiles a pipeline written in a .py file.

  Args:
    pyfile: The path to the .py file that contains the pipeline definition.
    function_name: The name of the pipeline function.
    pipeline_root: The root output directory for pipeline runtime.
    pipeline_parameters: The pipeline parameters as a dict of {name: value}.
    output_path: The output path of the compiled result.
    type_check: Whether to enable the type checking.
  """
  sys.path.insert(0, os.path.dirname(pyfile))
  try:
    filename = os.path.basename(pyfile)
    with PipelineCollectorContext() as pipeline_funcs:
      __import__(os.path.splitext(filename)[0])
    _compile_pipeline_function(
        pipeline_funcs=pipeline_funcs,
        function_name=function_name,
        pipeline_root=pipeline_root,
        pipeline_parameters=pipeline_parameters,
        output_path=output_path,
        type_check=type_check)
  finally:
    del sys.path[0]


def main():
  args = parse_arguments()
  if args.py is None:
    raise ValueError('The --py option must be specified.')
  compile_pyfile(
      pyfile=args.py,
      function_name=args.function,
      pipeline_root=args.pipeline_root,
      pipeline_parameters=args.pipeline_parameters,
      output_path=args.output,
      type_check=not args.disable_type_check,
  )
