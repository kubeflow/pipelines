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

from typing import Optional, Union

from kfp.dsl import _container_op
from kfp.dsl import _for_loop
from kfp.dsl import _pipeline_param
from kfp.v2.dsl import _pipeline


class OpsGroup(object):
  """Represents a logical group of ops and group of OpsGroups.

  This class is the base class for groups of ops, such as ops sharing an exit
  handler,
  a condition branch, or a loop. This class is not supposed to be used by
  pipeline authors.
  It is useful for implementing a compiler.
  """

  def __init__(self,
               group_type: str,
               name: Optional[str] = None,
               parallelism: Optional[int] = None):
    """Create a new instance of OpsGroup.

    Args:
      group_type (str): one of 'pipeline', 'exit_handler', 'condition',
        'for_loop', and 'graph'.
      name (str): name of the opsgroup
      parallelism (int): parallelism for the sub-DAG:s
    """
    #TODO: declare the group_type to be strongly typed
    self.type = group_type
    self.ops = list()
    self.groups = list()
    self.name = name
    self.dependencies = []
    self.parallelism = parallelism
    # recursive_ref points to the opsgroups with the same name if exists.
    self.recursive_ref = None

    self.loop_args = None

  @staticmethod
  def _get_matching_opsgroup_already_in_pipeline(group_type, name):
    """Retrieves the opsgroup when the pipeline already contains it.

    the opsgroup might be already in the pipeline in case of recursive calls.

    Args:
      group_type (str): one of 'pipeline', 'exit_handler', 'condition', and
        'graph'.
      name (str): the name before conversion.
    """
    if not _pipeline.Pipeline.get_default_pipeline():
      raise ValueError('Default pipeline not defined.')
    if name is None:
      return None
    name_pattern = '^' + (group_type + '-' + name + '-').replace('_',
                                                                 '-') + '[\d]+$'
    for ops_group_already_in_pipeline in _pipeline.Pipeline.get_default_pipeline(
    ).groups:
      import re
      if ops_group_already_in_pipeline.type == group_type \
              and re.match(name_pattern ,ops_group_already_in_pipeline.name):
        return ops_group_already_in_pipeline
    return None

  def _make_name_unique(self):
    """Generate a unique opsgroup name in the pipeline"""
    if not _pipeline.Pipeline.get_default_pipeline():
      raise ValueError('Default pipeline not defined.')

    self.name = (
        self.type + '-' + ('' if self.name is None else self.name + '-') +
        str(_pipeline.Pipeline.get_default_pipeline().get_next_group_id()))
    self.name = self.name.replace('_', '-')

  def __enter__(self):
    if not _pipeline.Pipeline.get_default_pipeline():
      raise ValueError('Default pipeline not defined.')

    self.recursive_ref = self._get_matching_opsgroup_already_in_pipeline(
        self.type, self.name)
    if not self.recursive_ref:
      self._make_name_unique()

    _pipeline.Pipeline.get_default_pipeline().push_ops_group(self)
    return self

  def __exit__(self, *args):
    _pipeline.Pipeline.get_default_pipeline().pop_ops_group()

  def after(self, *ops):
    """Specify explicit dependency on other ops."""
    for op in ops:
      self.dependencies.append(op)
    return self

  def remove_op_recursive(self, op):
    if self.ops and op in self.ops:
      self.ops.remove(op)
    for sub_group in self.groups or []:
      sub_group.remove_op_recursive(op)


class SubGraph(OpsGroup):
  TYPE_NAME = 'subgraph'

  def __init__(self, parallelism: int):
    raise NotImplementedError('Graph is not yet supported in KFP v2 compiler.')


class Condition(OpsGroup):
  """Represents an condition group with a condition.

  Args:
    condition (ConditionOperator): the condition.
    name (str): name of the condition
  Example: ::
      with Condition(param1=='pizza', '[param1 is pizza]'): op1 =
        ContainerOp(...) op2 = ContainerOp(...)
  """

  def __init__(self,
               condition: _pipeline_param.ConditionOperator,
               name: Optional[str] = None):
    super(Condition, self).__init__('condition', name)
    self.condition = condition


class Graph(OpsGroup):
  """Graph DAG with inputs, recursive_inputs, and outputs.

  This is not used directly by the users but auto generated when the
  graph_component decoration exists

  Args:
    name: Name of the graph.
  """

  def __init__(self, name):
    raise NotImplementedError('Graph is not yet supported in KFP v2 compiler.')


class ExitHandler(OpsGroup):
  """Represents an exit handler that is invoked upon exiting a group of ops."""

  def __init__(self, exit_op: _container_op.ContainerOp):
    raise NotImplementedError(
        'dsl.ExitHandler is not yet supported in KFP v2 compiler.')


class ParallelFor(OpsGroup):
  """Represents a parallel for loop over a static set of items."""

  def __init__(self,
               loop_args: Union[_for_loop.ItemList,
                                _pipeline_param.PipelineParam],
               parallelism: int = None):
    raise NotImplementedError(
        'dsl.ParallelFor is not yet supported in KFP v2 compiler.')
