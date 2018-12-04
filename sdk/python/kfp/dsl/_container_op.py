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


from . import _pipeline
from . import _pipeline_param
import re
from typing import Dict


class ContainerOp(object):
  """Represents an op implemented by a docker container image."""

  def __init__(self, name: str, image: str, command: str=None, arguments: str=None,
               file_inputs : Dict[_pipeline_param.PipelineParam, str]=None,
               file_outputs : Dict[str, str]=None, is_exit_handler=False):
    """Create a new instance of ContainerOp.

    Args:
      name: the name of the op. Has to be unique within a pipeline.
      image: the container image name, such as 'python:3.5-jessie'
      command: the command to run in the container.
          If None, uses default CMD in defined in container.
      arguments: the arguments of the command. The command can include "%s" and supply
          a PipelineParam as the string replacement. For example, ('echo %s' % input_param).
          At container run time the argument will be 'echo param_value'.
      file_inputs: Maps PipelineParams to local file paths. At pipeline run time,
          the value of a PipelineParam is saved to its corresponding local file. It is
          not implemented yet.
      file_outputs: Maps output labels to local file paths. At pipeline run time,
          the value of a PipelineParam is saved to its corresponding local file. It's
          one way for outside world to receive outputs of the container.
      is_exit_handler: Whether it is used as an exit handler.
    """

    if not _pipeline.Pipeline.get_default_pipeline():
      raise ValueError('Default pipeline not defined.')

    self.human_name = name
    self.name = _pipeline.Pipeline.get_default_pipeline().add_op(self, is_exit_handler)
    self.image = image
    self.command = command
    self.arguments = arguments
    self.is_exit_handler = is_exit_handler
    self.resource_limits = {}
    self.resource_requests = {}
    self.node_selector = {}
    self.volumes = []
    self.volume_mounts = []
    self.env_variables = []

    matches = []
    if arguments:
      for arg in arguments:
        match = re.findall(r'{{pipelineparam:op=([\w-]*);name=([\w-]+);value=(.*?)}}', str(arg))
        matches += match

    self.argument_inputs = [_pipeline_param.PipelineParam(x[1], x[0], x[2])
                            for x in list(set(matches))]
    self.file_inputs = file_inputs
    self.file_outputs = file_outputs
    self.dependent_op_names = []

    self.inputs = []
    if self.argument_inputs:
      self.inputs += self.argument_inputs

    if file_inputs:
      self.inputs += list(file_inputs.keys())

    self.outputs = {}
    if file_outputs:
      self.outputs = {name: _pipeline_param.PipelineParam(name, op_name=self.name)
          for name in file_outputs.keys()}

    self.output=None
    if len(self.outputs) == 1:
      self.output = list(self.outputs.values())[0]

  def apply(self, mod_func):
    """Applies a modifier function to self. The function should return the passed object.
    This is needed to chain "extention methods" to this class.

    Example:
      from kfp.gcp import use_gcp_secret
      task = (
        train_op(...)
          .set_memory_request('1GB')
          .apply(use_gcp_secret('user-gcp-sa'))
          .set_memory_limit('2GB')
      )
    """
    return mod_func(self)

  def after(self, op):
    """Specify explicit dependency on another op."""
    self.dependent_op_names.append(op.name)
    return self

  def _validate_memory_string(self, memory_string):
    """Validate a given string is valid for memory request or limit."""

    if re.match(r'^[0-9]+(E|Ei|P|Pi|T|Ti|G|Gi|M|Mi|K|Ki){0,1}$', memory_string) is None:
      raise ValueError('Invalid memory string. Should be an integer, or integer followed '
                       'by one of "E|Ei|P|Pi|T|Ti|G|Gi|M|Mi|K|Ki"')

  def _validate_cpu_string(self, cpu_string):
    "Validate a given string is valid for cpu request or limit."

    if re.match(r'^[0-9]+m$', cpu_string) is not None:
      return

    try:
      float(cpu_string)
    except ValueError:
      raise ValueError('Invalid cpu string. Should be float or integer, or integer followed '
                       'by "m".')

  def _validate_gpu_string(self, gpu_string):
    "Validate a given string is valid for gpu limit."

    try:
      gpu_value = int(gpu_string)
    except ValueError:
      raise ValueError('Invalid gpu string. Should be integer.')

    if gpu_value <= 0:
      raise ValueError('gpu must be positive integer.')

  def add_resource_limit(self, resource_name, value):
    """Add the resource limit of the container.

    Args:
      resource_name: The name of the resource. It can be cpu, memory, etc.
      value: The string value of the limit.
    """

    self.resource_limits[resource_name] = value
    return self

  def add_resource_request(self, resource_name, value):
    """Add the resource request of the container.

    Args:
      resource_name: The name of the resource. It can be cpu, memory, etc.
      value: The string value of the request.
    """

    self.resource_requests[resource_name] = value
    return self

  def set_memory_request(self, memory):
    """Set memory request (minimum) for this operator.

    Args:
      memory: a string which can be a number or a number followed by one of
              "E", "P", "T", "G", "M", "K".
    """

    self._validate_memory_string(memory)
    return self.add_resource_request("memory", memory)

  def set_memory_limit(self, memory):
    """Set memory limit (maximum) for this operator.

    Args:
      memory: a string which can be a number or a number followed by one of
              "E", "P", "T", "G", "M", "K".
    """
    self._validate_memory_string(memory)
    return self.add_resource_limit("memory", memory)

  def set_cpu_request(self, cpu):
    """Set cpu request (minimum) for this operator.

    Args:
      cpu: A string which can be a number or a number followed by "m", which means 1/1000.
    """

    self._validate_cpu_string(cpu)
    return self.add_resource_request("cpu", cpu)

  def set_cpu_limit(self, cpu):
    """Set cpu limit (maximum) for this operator.

    Args:
      cpu: A string which can be a number or a number followed by "m", which means 1/1000.
    """

    self._validate_cpu_string(cpu)
    return self.add_resource_limit("cpu", cpu)

  def set_gpu_limit(self, gpu, vendor = "nvidia"):
    """Set gpu limit for the operator. This function add '<vendor>.com/gpu' into resource limit. 
    Note that there is no need to add GPU request. GPUs are only supposed to be specified in 
    the limits section. See https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus/.

    Args:
      gpu: A string which must be a positive number.
      vendor: Optional. A string which is the vendor of the requested gpu. The supported values 
        are: 'nvidia' (default), and 'amd'. 
    """

    self._validate_gpu_string(gpu)
    if vendor != 'nvidia' and vendor != 'amd':
      raise ValueError('vendor can only be nvidia or amd.')

    return self.add_resource_limit("%s.com/gpu" % vendor, gpu)


  def add_volume(self, volume):
    """Add K8s volume to the container

    Args:
      volume: Kubernetes volumes
      For detailed spec, check volume definition
      https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_volume.py
    """

    self.volumes.append(volume)
    return self

  def add_volume_mount(self, volume_mount):
    """Add volume to the container

    Args:
      volume_mount: Kubernetes volume mount
      For detailed spec, check volume mount definition
      https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_volume_mount.py
    """

    self.volume_mounts.append(volume_mount)
    return self

  def add_env_variable(self, env_variable):
    """Add environment variable to the container.

    Args:
      env_variable: Kubernetes environment variable
      For detailed spec, check environment variable definition
      https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_env_var.py
    """

    self.env_variables.append(env_variable)
    return self

  def add_node_selector_constraint(self, label_name, value):
    """Add a constraint for nodeSelector. Each constraint is a key-value pair label. For the 
    container to be eligible to run on a node, the node must have each of the constraints appeared
    as labels.

    Args:
      label_name: The name of the constraint label.
      value: The value of the constraint label.
    """

    self.node_selector[label_name] = value
    return self

  def __repr__(self):
      return str({self.__class__.__name__: self.__dict__})
