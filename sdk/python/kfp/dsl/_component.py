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

def python_component(name, description, base_image):
  """Decorator of component functions.

  Usage:
  ```python
  @dsl.python_component(
    name='my awesome component',
    description='Come, Let's play'
    base_image='tensorflow/tensorflow'
  )
  def my_component(a: str, b: int) -> str:
    ...
  ```
  """
  def _python_component(func):
    PythonComponent.add_python_component(name, description, base_image, func)
    return func

  return _python_component

class PythonComponent():
  """A pipeline contains a list of operators.

  This class is not supposed to be used by component authors since component authors can use
  component functions (decorated with @python_component) to reference their pipelines. This class
  is useful for implementing a compiler. For example, the compiler can use the following
  to get the PythonComponent object:
  """


  # All pipeline functions with @pipeline decorator that are imported.
  # Each key is a pipeline function. Each value is a dictionary of name, description, base_image.
  _component_functions = {}

  @staticmethod
  def add_python_component(name, description, base_image, func):
    """ Add a python component """
    PythonComponent._component_functions[func] = {
      'name': name,
      'description': description,
      'base_image': base_image
    }

  @staticmethod
  def get_python_component(func):
    """ Get a python component """
    return PythonComponent._component_functions.get(func, None)