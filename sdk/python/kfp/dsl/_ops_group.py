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
from . import _pipeline


class OpsGroup(object):
  """Represents a logical group of ops and group of OpsGroups.

  This class is the base class for groups of ops, such as ops sharing an exit handler,
  a condition branch, or a loop. This class is not supposed to be used by pipeline authors.
  It is useful for implementing a compiler.
  """

  def __init__(self, group_type: str, name: str=None):
    """Create a new instance of OpsGroup.
    Args:
      group_type: usually one of 'exit_handler', 'condition', and 'loop'.
    """
    self.type = group_type
    self.ops = list()
    self.groups = list()
    self.name = name

  def __enter__(self):
    if not _pipeline.Pipeline.get_default_pipeline():
      raise ValueError('Default pipeline not defined.')

    if not self.name:
      self.name = (self.type + '-' +
          str(_pipeline.Pipeline.get_default_pipeline().get_next_group_id()))
    self.name = self.name.replace('_', '-')

    _pipeline.Pipeline.get_default_pipeline().push_ops_group(self)
    return self

  def __exit__(self, *args):
    _pipeline.Pipeline.get_default_pipeline().pop_ops_group()


class ExitHandler(OpsGroup):
  """Represents an exit handler that is invoked upon exiting a group of ops.

  Example usage:
  ```python
  exit_op = ContainerOp(...)
  with ExitHandler(exit_op):
    op1 = ContainerOp(...)
    op2 = ContainerOp(...)
  ```
  """

  def __init__(self, exit_op: _container_op.ContainerOp):
    """Create a new instance of ExitHandler.
    Args:
      exit_op: an operator invoked at exiting a group of ops.

    Raises:
      ValueError is the exit_op is invalid.
    """
    super(ExitHandler, self).__init__('exit_handler')
    if exit_op.dependent_op_names:
      raise ValueError('exit_op cannot depend on any other ops.')

    self.exit_op = exit_op


class Condition(OpsGroup):
  """Represents an condition group with a condition.

  Example usage:
  ```python
  with Condition(param1=='pizza'):
    op1 = ContainerOp(...)
    op2 = ContainerOp(...)
  ```
  Note: Only equal operator "==" is supported.
  """

  def __init__(self, condition):
    """Create a new instance of ExitHandler.
    Args:
      exit_op: an operator invoked at exiting a group of ops.

    Raises:
      ValueError is the exit_op is invalid.
    """
    super(Condition, self).__init__('condition')
    self.condition = condition
