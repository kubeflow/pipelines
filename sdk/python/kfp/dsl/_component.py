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

def python_component(name, description=None, base_image=None, target_component_file: str = None):
  """Decorator for Python component functions.
  This decorator adds the metadata to the function object itself.

  Args:
      name: Human-readable name of the component
      description: Optional. Description of the component
      base_image: Optional. Docker container image to use as the base of the component. Needs to have Python 3.5+ installed.
      target_component_file: Optional. Local file to store the component definition. The file can then be used for sharing.

  Returns:
      The same function (with some metadata fields set).

  Usage:
  ```python
  @dsl.python_component(
    name='my awesome component',
    description='Come, Let's play',
    base_image='tensorflow/tensorflow:1.11.0-py3',
  )
  def my_component(a: str, b: int) -> str:
    ...
  ```
  """
  def _python_component(func):
    func._component_human_name = name
    if description:
      func._component_description = description
    if base_image:
      func._component_base_image = base_image
    if target_component_file:
      func._component_target_component_file = target_component_file
    return func

  return _python_component
