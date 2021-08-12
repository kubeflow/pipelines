# Copyright 2021 The Kubeflow Authors
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
"""Pipeline task class and operations."""

from typing import Any, Mapping, Optional

from kfp.dsl import _component_bridge
from kfp.v2.components.experimental import component_spec as cspec


# TODO(chensun): return PipelineTask object instead of ContainerOp object.
def create_pipeline_task(
    component_spec: cspec.ComponentSpec,
    arguments: Mapping[str, Any],
) -> 'ContainerOp':
  return _component_bridge._create_container_op_from_component_and_arguments(
      component_spec=component_spec.to_v1_component_spec(),
      arguments=arguments,
  )


class PipelineTask:
  """Represents a pipeline task -- an instantiated component.

  Replaces `ContainerOp`. Holds operations available on a task object, such as
  `.after()`, `.set_memory_limit()`, `enable_caching()`, etc.

  Attributes:
    task_spec:
    component_spec:
    executor_spec:
  """

  def __init__(
      self,
      component_spec: cspec.ComponentSpec,
      arguments: Mapping[str, Any],
  ):
    # TODO(chensun): move logic from _component_bridge over here.
    raise NotImplementedError
