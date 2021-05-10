# Copyright 2021 The Kubeflow Authors
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

import functools
from typing import Callable, Optional, List

from kfp import components


def component(func: Optional[Callable] = None,
              *,
              base_image: Optional[str] = None,
              packages_to_install: List[str] = None,
              output_component_file: Optional[str] = None):
  """Decorator for Python-function based components in v2.

  Example usage:

  from kfp.v2 import dsl
  @dsl.component
  def my_function_one(input: str, output: Output[Model]):
    ...

  @dsl.component(
    base_image='python:3.9',
    output_component_file='my_function.yaml'
  )
  def my_function_two(input: Input[Mode])):
    ...

  @dsl.pipeline(pipeline_root='...',
                name='my-pipeline')
  def pipeline():
    my_function_one_task = my_function_one(input=...)
    my_function_two_task = my_function_two(input=my_function_one_task.outputs..
  """
  if func is None:
    return functools.partial(component,
                             base_image=base_image,
                             packages_to_install=packages_to_install,
                             output_component_file=output_component_file)

  return components.create_component_from_func_v2(
      func,
      base_image=base_image,
      packages_to_install=packages_to_install,
      output_component_file=output_component_file)
