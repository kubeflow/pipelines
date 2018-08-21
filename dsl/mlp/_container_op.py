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
import re
from typing import Dict, List


class ContainerOp(object):
  """Represents an op implemented by a docker container image."""

  def __init__(self, name: str, image: str, command: str=None, arguments: str=None,
               file_inputs : Dict[mlp.PipelineParam, str]=None,
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

    if not mlp.Pipeline.get_default_pipeline():
      raise ValueError('Default pipeline not defined.')

    if not re.match(r'^[A-Za-z][A-Za-z0-9-]*$', name):
      raise ValueError('Only letters, numbers and "-" allowed in name. Must begin with letter.')

    self.name = name
    self.image = image
    self.command = command
    self.arguments = arguments
    self.is_exit_handler = is_exit_handler

    matches = []
    if arguments:
      for arg in arguments:
        match = re.findall(r'{{pipelineparam:op=([\w-]*);name=([\w-]+);value=(.*?)}}', str(arg))
        matches += match

    self.argument_inputs = [mlp.PipelineParam(x[1], x[0], x[2]) for x in matches]
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
      self.outputs = {name: mlp.PipelineParam(name, op_name=self.name)
          for name in file_outputs.keys()}

    self.output=None
    if len(self.outputs) == 1:
      self.output = list(self.outputs.values())[0]

    mlp.Pipeline.get_default_pipeline().add_op(self, is_exit_handler)

  def after(self, op):
    """Specify explicit dependency on another op."""
    self.dependent_op_names.append(op.name)
