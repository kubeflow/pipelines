# Copyright 2018 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
# in compliance with the License. You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software distributed under the License
# is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
# or implied. See the License for the specific language governing permissions and limitations under
# the License.


import mlp


def pipeline(func):
  """Decorator of pipeline functions.

  Usage:
  ```python
  @mlp.pipeline
  def my_pipeline(a: mlp.PipelineParam, b: mlp.PipelineParam):
    ...
  ```
  """
  Pipeline.add_pipeline_function(func)
  return func


class Pipeline():
  """A pipeline contains a list of operators.

  This class is not supposed to be used by pipeline authors since pipeline authors can use
  pipeline functions (decorated with @pipeline) to reference their pipelines. This class
  is useful for implementing a compiler. For example, the compiler can use the following
  to get the pipeline object and its ops:

  ```python
  with mlp.Pipeline() as p:
    pipeline_func(*args_list)

  traverse(p.ops)
  ```
  """

  # _default_pipeline is set when it (usually a compiler) runs "with mlp.Pipeline()"
  _default_pipeline = None

  # All pipeline functions with @pipeline decorator that are imported.
  _pipeline_functions = []

  @staticmethod
  def get_default_pipeline():
    """Get default pipeline. """
    return Pipeline._default_pipeline

  @staticmethod
  def get_pipeline_functions():
    """Get all imported pipeline functions (decorated with @mlp.pipeline)."""
    return Pipeline._pipeline_functions

  @staticmethod
  def add_pipeline_function(func):
    """Add a pipeline function (decorated with @mlp.pipeline)."""
    Pipeline._pipeline_functions.append(func)
  
  def __init__(self, name: str):
    """Create a new instance of Pipeline.

    Args:
      name: the name of the pipeline. Once deployed, the name will show up in Pipeline System UI.
    """
    self.name = name
    self.ops = {}
    self.groups = [mlp.OpsGroup('pipeline')]

  def __enter__(self):
    if Pipeline._default_pipeline:
      raise Exception('Nested pipelines are not allowed.')

    Pipeline._default_pipeline = self
    return self

  def __exit__(self, *args):
    Pipeline._default_pipeline = None
        
  def add_op(self, op: mlp.ContainerOp):
    """Add a new operator.

    Args:
      op: An operator of mlp.ContainerOp or its inherited type.
    Raises:
      ValueError if an op with the same name exists.
    """
    if op.name in self.ops:
      raise ValueError('Op with name "%s" exists already.' % op.name)

    self.ops[op.name] = op
    self.groups[-1].ops.append(op)

  def push_ops_group(self, group: mlp.OpsGroup):
    """Push an OpsGroup into the stack.

    Args:
      group: An OpsGroup. Typically it is one of ExitHandler, Branch, and Loop.
    """
    self.groups[-1].groups.append(group)
    self.groups.append(group)

  def pop_ops_group(self):
    """Remove the current OpsGroup from the stack."""
    del self.groups[-1]
