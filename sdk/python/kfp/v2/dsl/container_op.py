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
"""IR-based ContainerOp."""

from typing import List, Text, Union

from kfp import dsl
from kfp.v2.proto import pipeline_spec_pb2

# type alias: either a string or a list of string
StringOrStringList = Union[str, List[str]]


class ContainerOp(dsl.ContainerOp):
  """V2 ContainerOp class.

  This class inherits an almost identical behavior as the previous ContainerOp
  class. The diffs are in two aspects:
  - The source of truth is migrating to the PipelineContainerSpec proto.
  - The implementation (and impact) of several APIs are different. For example,
    resource spec will be set in the pipeline IR proto instead of using k8s API.
  """

  def __init__(self, **kwargs):
    super(ContainerOp, self).__init__(**kwargs)

  # Override resource specification calls.
  def set_cpu_limit(self, cpu: Text) -> 'ContainerOp':
    """Sets the cpu provisioned for this task.
    
    Args:
      cpu: a string indicating the amount of vCPU required by this task.
        Please refer to dsl.ContainerOp._validate_cpu_string regarding
        its format.
    Returns:
      self return to allow chained call with other resource specification.
    """
    self._validate_cpu_string(cpu)

  def set_memory_limit(self, memory: Text) -> 'ContainerOp':
    """Sets the memory provisioned for this task.
    
    Args:
      memory: a string described the amount of memory required by this
        task. Please refer to dsl.ContainerOp._validate_size_string
        regarding its format.
    Returns:
      self return to allow chained call with other resource specification.
    """
    self._validate_size_string(memory)

  def add_node_selector_constraint(
      self, label_name: Text, value: Text
  ) -> 'ContainerOp':
    """Sets accelerator requirement for this task.
    
    This function is designed to enable users to specify accelerator using
    a similar DSL syntax as KFP V1. Under the hood, it will directly specify
    the accelerator required in the IR proto, instead of relying on the
    k8s node selector API.
    
    Args:
      label_name: only support 'cloud.google.com/gke-accelerator' now.
        value: name of the accelerator. For example, 'nvidia-tesla-k80', or
        'tpu-v3'.
    Returns:
      self return to allow chained call with other resource specification.
    """
    pass
