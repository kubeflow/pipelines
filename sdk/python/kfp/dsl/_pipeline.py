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
from ._component_bridge import _create_container_op_from_component_and_arguments
from ..components import _components
from ..components._naming import _make_name_unique_by_adding_index
import sys


# This handler is called whenever the @pipeline decorator is applied.
# It can be used by command-line DSL compiler to inject code that runs for every pipeline definition.
_pipeline_decorator_handler = None


def pipeline(name : str = None, description : str = None):
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
    if name:
      func._component_human_name = name
    if description:
      func._component_description = description

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
    self.timeout = 0
    self.ttl_seconds_after_finished = -1
    self.artifact_location = None
    self.op_transformers = []

  def set_image_pull_secrets(self, image_pull_secrets):
    """Configures the pipeline level imagepullsecret

    Args:
      image_pull_secrets: a list of Kubernetes V1LocalObjectReference
      For detailed description, check Kubernetes V1LocalObjectReference definition
      https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1LocalObjectReference.md
    """
    self.image_pull_secrets = image_pull_secrets
    return self

  def set_timeout(self, seconds: int):
    """Configures the pipeline level timeout

    Args:
      seconds: number of seconds for timeout
    """
    self.timeout = seconds
    return self

  def set_ttl_seconds_after_finished(self, seconds: int):
    """Configures the ttl after the pipeline has finished.

    Args:
      seconds: number of seconds for the workflow to be garbage collected after it is finished.
    """
    self.ttl_seconds_after_finished = seconds
    return self

  def set_artifact_location(self, artifact_location):
    """Configures the pipeline level artifact location.

    Example::

      from kfp.dsl import ArtifactLocation, get_pipeline_conf, pipeline
      from kubernetes.client.models import V1SecretKeySelector


      @pipeline(name='foo', description='hello world')
      def foo_pipeline(tag: str, pull_image_policy: str):
        '''A demo pipeline'''
        # create artifact location object
        artifact_location = ArtifactLocation.s3(
                              bucket="foo",
                              endpoint="minio-service:9000",
                              insecure=True,
                              access_key_secret=V1SecretKeySelector(name="minio", key="accesskey"),
                              secret_key_secret=V1SecretKeySelector(name="minio", key="secretkey"))
        # config pipeline level artifact location
        conf = get_pipeline_conf().set_artifact_location(artifact_location)

        # rest of codes
        ...

    Args:
      artifact_location: V1alpha1ArtifactLocation object
      For detailed description, check Argo V1alpha1ArtifactLocation definition
      https://github.com/e2fyi/argo-models/blob/release-2.2/argo/models/v1alpha1_artifact_location.py
      https://github.com/argoproj/argo/blob/release-2.2/api/openapi-spec/swagger.json
    """
    self.artifact_location = artifact_location
    return self

  def add_op_transformer(self, transformer):
    """Configures the op_transformers which will be applied to all ops in the pipeline.

    Args:
      transformer: a function that takes a ContainOp as input and returns a ContainerOp
    """
    self.op_transformers.append(transformer)


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
    # Add the root group.
    self.groups = [_ops_group.OpsGroup('pipeline', name=name)]
    self.group_id = 0
    self.conf = PipelineConf()
    self._metadata = None

  def __enter__(self):
    if Pipeline._default_pipeline:
      raise Exception('Nested pipelines are not allowed.')

    Pipeline._default_pipeline = self
    self._old_container_task_constructor = _components._container_task_constructor
    _components._container_task_constructor = _create_container_op_from_component_and_arguments

    def register_op_and_generate_id(op):
      return self.add_op(op, op.is_exit_handler)

    self._old__register_op_handler = _container_op._register_op_handler
    _container_op._register_op_handler = register_op_and_generate_id
    return self

  def __exit__(self, *args):
    Pipeline._default_pipeline = None
    _container_op._register_op_handler = self._old__register_op_handler
    _components._container_task_constructor = self._old_container_task_constructor

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

  def remove_op_from_groups(self, op):
    for group in self.groups:
      group.remove_op_recursive(op)

  def get_next_group_id(self):
    """Get next id for a new group. """

    self.group_id += 1
    return self.group_id

  def _set_metadata(self, metadata):
    '''_set_metadata passes the containerop the metadata information
    Args:
      metadata (ComponentMeta): component metadata
    '''
    self._metadata = metadata


