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


from . import _container_op
from . import _ops_group
import re
import sys


def pipeline(name, description):
  """Decorator of pipeline functions.

  Usage:
  ```python
  @pipeline(
    name='my awesome pipeline',
    description='Is it really awesome?'
  )
  def my_pipeline(a: PipelineParam, b: PipelineParam):
    ...
  ```
  """
  def _pipeline(func):
    Pipeline.add_pipeline(name, description, func)
    return func

  return _pipeline

def _make_kubernetes_name(name):
    return re.sub('-+', '-', re.sub('[^-0-9a-z]+', '-', name.lower())).lstrip('-').rstrip('-')

class Pipeline():
  """A pipeline contains a list of operators.

  This class is not supposed to be used by pipeline authors since pipeline authors can use
  pipeline functions (decorated with @pipeline) to reference their pipelines. This class
  is useful for implementing a compiler. For example, the compiler can use the following
  to get the pipeline object and its ops:

  ```python
  with Pipeline() as p:
    pipeline_func(*args_list)

  traverse(p.ops)
  ```
  """

  # _default_pipeline is set when it (usually a compiler) runs "with Pipeline()"
  _default_pipeline = None

  # All pipeline functions with @pipeline decorator that are imported.
  # Each key is a pipeline function. Each value is a (name, description).
  _pipeline_functions = {}

  @staticmethod
  def get_default_pipeline():
    """Get default pipeline. """
    return Pipeline._default_pipeline

  @staticmethod
  def get_pipeline_functions():
    """Get all imported pipeline functions (decorated with @pipeline)."""
    return Pipeline._pipeline_functions

  @staticmethod
  def add_pipeline(name, description, func):
    """Add a pipeline function (decorated with @pipeline)."""
    Pipeline._pipeline_functions[func] = (name, description)

  def __init__(self, name: str):
    """Create a new instance of Pipeline.

    Args:
      name: the name of the pipeline. Once deployed, the name will show up in Pipeline System UI.
    """
    self.name = name
    self.ops = {}
    # Add the root group.
    self.groups = [_ops_group.OpsGroup('pipeline', name=name)]
    self.group_id = 0

  def __enter__(self):
    if Pipeline._default_pipeline:
      raise Exception('Nested pipelines are not allowed.')

    Pipeline._default_pipeline = self
    return self

  def __exit__(self, *args):
    Pipeline._default_pipeline = None
        
  def add_op(self, op: _container_op.ContainerOp, define_only: bool):
    """Add a new operator.

    Args:
      op: An operator of ContainerOp or its inherited type.
    """

    kubernetes_name = _make_kubernetes_name(op.human_name)
    step_id = kubernetes_name
    #If there is an existing op with this name then generate a new name.
    if step_id in self.ops:
      for i in range(2, sys.maxsize**10):
        step_id = kubernetes_name + '-' + str(i)
        if step_id not in self.ops:
          break

    self.ops[step_id] = op
    if not define_only:
      self.groups[-1].ops.append(op)

    return step_id

  def push_ops_group(self, group: _ops_group.OpsGroup):
    """Push an OpsGroup into the stack.

    Args:
      group: An OpsGroup. Typically it is one of ExitHandler, Branch, and Loop.
    """
    self.groups[-1].groups.append(group)
    self.groups.append(group)

  def pop_ops_group(self):
    """Remove the current OpsGroup from the stack."""
    del self.groups[-1]

  def get_next_group_id(self):
    """Get next id for a new group. """

    self.group_id += 1
    return self.group_id
