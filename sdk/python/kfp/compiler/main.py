# Copyright 2018 Google LLC
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


import argparse
import kfp.dsl as dsl
import kfp.compiler
import os
import shutil
import subprocess
import sys
import tempfile


def parse_arguments():
  """Parse command line arguments."""

  parser = argparse.ArgumentParser()
  parser.add_argument('--py',
                      type=str,
                      help='local absolute path to a py file.')
  parser.add_argument('--package',
                      type=str,
                      help='local path to a pip installable python package file.')
  parser.add_argument('--function',
                      type=str,
                      help='The name of the function to compile if there are multiple.')
  parser.add_argument('--namespace',
                      type=str,
                      help='The namespace for the pipeline function')
  parser.add_argument('--output',
                      type=str,
                      required=True,
                      help='local path to the output workflow yaml file.')

  args = parser.parse_args()
  return args


def _compile_pipeline_function(function_name, output_path):

  pipeline_funcs = list(dsl.Pipeline.get_pipeline_functions().keys())
  if len(pipeline_funcs) == 0:
    raise ValueError('A function with @dsl.pipeline decorator is required in the py file.')

  if len(pipeline_funcs) > 1 and not function_name:
    func_names = [x.__name__ for x in pipeline_funcs]
    raise ValueError('There are multiple pipelines: %s. Please specify --function.' % func_names)

  if function_name:
    pipeline_func = next((x for x in pipeline_funcs if x.__name__ == function_name), None)
    if not pipeline_func:
      raise ValueError('The function "%s" does not exist. '
                       'Did you forget @dsl.pipeline decoration?' % function_name)
  else:
    pipeline_func = pipeline_funcs[0]

  kfp.compiler.Compiler().compile(pipeline_func, output_path)


def compile_package(package_path, namespace, function_name, output_path):
  tmpdir = tempfile.mkdtemp()
  sys.path.insert(0, tmpdir)
  try:
    subprocess.check_call(['python3', '-m', 'pip', 'install', package_path, '-t', tmpdir])
    __import__(namespace)
    _compile_pipeline_function(function_name, output_path)
  finally:
    del sys.path[0]
    shutil.rmtree(tmpdir)


def compile_pyfile(pyfile, function_name, output_path):
  sys.path.insert(0, os.path.dirname(pyfile))
  try:
    filename = os.path.basename(pyfile)
    __import__(os.path.splitext(filename)[0])
    _compile_pipeline_function(function_name, output_path)
  finally:
    del sys.path[0]


def main():
  args = parse_arguments()
  if ((args.py is None and args.package is None) or
      (args.py is not None and args.package is not None)):
    raise ValueError('Either --py or --package is needed but not both.')
  if args.py:
    compile_pyfile(args.py, args.function, args.output)
  else:
    if args.namespace is None:
      raise ValueError('--namespace is required for compiling packages.')
    compile_package(args.package, args.namespace, args.function, args.output)
  
