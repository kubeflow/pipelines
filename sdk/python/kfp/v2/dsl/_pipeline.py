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
"""Pipeline class and decorator function definition."""

import collections
from typing import Any, Callable

from kfp.v2.components import components
from kfp.v2.components import _naming
from kfp.dsl import _container_op
from kfp.dsl import _ops_group
from kfp.v2.dsl import component_bridge


# TODO: Pipeline is in fact an opsgroup, refactor the code.
class Pipeline():
  """A pipeline contains a list of operators.

  This class is not supposed to be used by pipeline authors since pipeline
  authors can use
  pipeline functions (decorated with @pipeline) to reference their pipelines.
  This class
  is useful for implementing a compiler. For example, the compiler can use the
  following
  to get the pipeline object and its ops:

  Example:
    ::

      with Pipeline() as p:
        pipeline_func(*args_list)

      traverse(p.ops)
  """

  # _default_pipeline is set when it (usually a compiler) runs "with Pipeline()"
  _default_pipeline = None

  @staticmethod
  def get_default_pipeline():
    """Get default pipeline."""
    return Pipeline._default_pipeline

  @staticmethod
  def add_pipeline(name, description, func):
    """Add a pipeline function with the specified name and description."""
    # Applying the @pipeline decorator to the pipeline function
    func = pipeline(name=name, description=description)(func)

  def __init__(self, name: str):
    """Create a new instance of Pipeline.

    Args:
      name: the name of the pipeline. Once deployed, the name will show up in
        Pipeline System UI.
    """
    self.name = name
    self.ops = collections.OrderedDict()
    # Add the root group.
    self.groups = [_ops_group.OpsGroup('pipeline', name=name)]
    self.group_id = 0
    self._metadata = None

  def __enter__(self):
    if Pipeline._default_pipeline:
      raise Exception('Nested pipelines are not allowed.')

    Pipeline._default_pipeline = self
    self._old_container_task_constructor = (
        components._container_task_constructor)
    components._container_task_constructor = (
        component_bridge.create_container_op_from_component_and_arguments)

    def register_op_and_generate_id(op):
      return self.add_op(op)

    self._old__register_op_handler = _container_op._register_op_handler
    _container_op._register_op_handler = register_op_and_generate_id
    return self

  def __exit__(self, *args):
    Pipeline._default_pipeline = None
    _container_op._register_op_handler = self._old__register_op_handler
    components._container_task_constructor = (
        self._old_container_task_constructor)

  def add_op(self, op: _container_op.BaseOp) -> str:
    """Add a new operator.

    Args:
      op: An operator of ContainerOp, ResourceOp or their inherited types.

    Returns:
      The name of the op.
    """
    # If there is an existing op with this name then generate a new name.
    op_name = _naming._make_name_unique_by_adding_index(op.human_name,
                                                        list(self.ops.keys()),
                                                        ' ')
    self.ops[op_name] = op
    return op_name
