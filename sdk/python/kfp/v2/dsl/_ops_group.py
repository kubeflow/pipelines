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

from typing import Union

from kfp.dsl import _container_op
from kfp.dsl import _for_loop
from kfp.dsl import _ops_group
from kfp.dsl import _pipeline_param


class ExitHandler(_ops_group.OpsGroup):
  """Represents an exit handler that is invoked upon exiting a group of ops."""

  def __init__(self, exit_op: _container_op.ContainerOp):
    raise NotImplementedError(
        'dsl.ExitHandler is not yet supported in KFP v2 compiler.')


class Condition(_ops_group.OpsGroup):
  """Represents an condition group with a condition."""

  def __init__(self, condition, name=None):
    raise NotImplementedError(
        'dsl.Condition is not yet supported in KFP v2 compiler.')


class Graph(_ops_group.OpsGroup):
  """Graph DAG with inputs, recursive_inputs, and outputs."""

  def __init__(self, name):
    raise NotImplementedError('Graph is not yet supported in KFP v2 compiler.')


class ParallelFor(_ops_group.OpsGroup):
  """Represents a parallel for loop over a static set of items."""

  def __init__(self,
               loop_args: Union[_for_loop.ItemList,
                                _pipeline_param.PipelineParam],
               parallelism: int = None):
    raise NotImplementedError(
        'dsl.ParallelFor is not yet supported in KFP v2 compiler.')
