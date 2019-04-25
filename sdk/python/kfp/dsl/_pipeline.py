# Copyright 2018-2019 Google LLC
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
from . import _resource_op
from . import _ops_group
from ..components._naming import _make_name_unique_by_adding_index
import sys


# This handler is called whenever the @pipeline decorator is applied.
# It can be used by command-line DSL compiler to inject code that runs for every pipeline definition.
_pipeline_decorator_handler = None


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
    func._pipeline_name = name
    func._pipeline_description = description

    if _pipeline_decorator_handler:
      return _pipeline_decorator_handler(func) or func
    else:
      return func 

  return _pipeline

class PipelineConf():
  """PipelineConf contains pipeline level settings
  """
  def __init__(self):
    self.image_pull_secrets = []

  def set_image_pull_secrets(self, image_pull_secrets):
    """ configure the pipeline level imagepullsecret

    Args:
      image_pull_secrets: a list of Kubernetes V1LocalObjectReference
      For detailed description, check Kubernetes V1LocalObjectReference definition
      https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1LocalObjectReference.md
    """
    self.image_pull_secrets = image_pull_secrets

def get_pipeline_conf():
  """Configure the pipeline level setting to the current pipeline
    Note: call the function inside the user defined pipeline function.
  """
  return Pipeline.get_default_pipeline().conf

#TODO: Pipeline is in fact an opsgroup, refactor the code.
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

  @staticmethod
  def get_default_pipeline():
    """Get default pipeline. """
    return Pipeline._default_pipeline

  @staticmethod
  def add_pipeline(name, description, func):
    """Add a pipeline function with the specified name and description."""
    # Applying the @pipeline decorator to the pipeline function
    func = pipeline(name=name, description=description)(func)

  def __init__(self, name: str):
    """Create a new instance of Pipeline.

    Args:
      name: the name of the pipeline. Once deployed, the name will show up in Pipeline System UI.
    """
    self.name = name
    self.ops = {}
    self.cops = {}
    self.rops = {}
    # Add the root group.
    self.groups = [_ops_group.OpsGroup('pipeline', name=name)]
    self.group_id = 0
    self.conf = PipelineConf()
    self._metadata = None

  def __enter__(self):
    if Pipeline._default_pipeline:
      raise Exception('Nested pipelines are not allowed.')

    Pipeline._default_pipeline = self

    def register_op_and_generate_id(op):
      return self.add_op(op, op.is_exit_handler)

    self._old__register_op_handler = _container_op._register_op_handler
    _container_op._register_op_handler = register_op_and_generate_id
    return self

  def __exit__(self, *args):
    Pipeline._default_pipeline = None
    _container_op._register_op_handler = self._old__register_op_handler

  def add_op(self, op: _container_op.BaseOp, define_only: bool):
    """Add a new operator.

    Args:
      op: An operator of ContainerOp, ResourceOp or their inherited types.

    Returns
      op_name: a unique op name.
    """
    #If there is an existing op with this name then generate a new name.
    op_name = _make_name_unique_by_adding_index(op.human_name, list(self.ops.keys()), ' ')

    self.ops[op_name] = op
    if isinstance(op, _container_op.ContainerOp):
      self.cops[op_name] = op
    elif isinstance(op, _resource_op.ResourceOp):
      self.rops[op_name] = op
    if not define_only:
      self.groups[-1].ops.append(op)

    return op_name

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

  def _set_metadata(self, metadata):
    '''_set_metadata passes the containerop the metadata information
    Args:
      metadata (ComponentMeta): component metadata
    '''
    if not isinstance(metadata, PipelineMeta):
      raise ValueError('_set_medata is expecting PipelineMeta.')
    self._metadata = metadata


