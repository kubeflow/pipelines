#!/usr/bin/env python3
# Copyright 2019 Google LLC
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
import kfp
from kfp import compiler
from kfp import dsl


def process_args():
  """Define arguments and assign default values to the ones that are not set.

  Returns:
    parsed_args: The parsed namespace with defaults assigned to the flags.
  """

  parser = argparse.ArgumentParser(
      description='Basic sample to demonstrate component build.')
  parser.add_argument(
      '--output_path',
      default=None,
      required=True,
      help='GCS bucket path to store temporary assets from build process')
  parser.add_argument(
      '--target_image',
      default='component_build:latest',
      help='Name:tag for the image that will be built by component build')
  parser.add_argument(
      '--container_repo',
      default=None,
      required=True,
      help='Repo path ex. gcr.oi/projecID for output of component build.')
  parsed_args, _ = parser.parse_known_args()
  return parsed_args


args = process_args()


def add(a: float, b: float) -> float:
  """Calculates sum of two arguments."""
  print('Adding two values %s and %s' % (a, b))
  return a + b


# Using the build component to create a new docker image with add func code.
add_op = compiler.build_python_component(
    component_func=add,
    staging_gcs_path=args.output_path,
    # You can add additional dependencies to the image.
    dependency=[kfp.compiler.VersionedDependency(
        name='google-api-python-client', version='1.7.0')],
    base_image='python:alpine3.6',
    target_image= ''.join([
        args.container_repo,
        '/',
        args.target_image
        ]))


@dsl.pipeline(
    name='Component build pipeline',
    description='A simple pipeline to demonstrate component build.'
)
def component_build_pipeline(var1=1.0, var2=2.0):
  add_task = add_op(var1, var2)


kfp.compiler.Compiler().compile(component_build_pipeline, __file__ + '.zip')
