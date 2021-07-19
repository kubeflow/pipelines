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
"""Adaptors to support legacy data passing.

The adaptor offers backward compatiblity for pipelines utilize legacy data
passing. Passing parameters to artifacts or vice versa could result unexpected
runtime errors. It is highly recommended to not rely on the adaptors. We may
remove the adaptors in the later releases.
"""
from typing import Type, Union

from kfp import dsl
from kfp.components import _naming
from kfp.dsl import dsl_utils
from kfp.dsl import io_types
from kfp.dsl import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2 as pb

_PARAMETER_TYPES = Union[str, int, float, bool, dict, list]

# Global index to generate unique and stable adaptor name.
adaptor_op_index = 0


class BaseAdaptor:
  """Base class of legacy data passing adaptor.

  Attributes:
    INPUT_KEY: The name of the component input.
    OUTPUT_KEY: The name of the component output.
    task_spec: The IR PipelineTaskSpec of the component.
    component_spec: The IR ComponentSpec of the component.
    container_spec: The IR PipelineContainerSpec of the component.
  """
  INPUT_KEY = 'input'
  OUTPUT_KEY = 'output'
  _IMAGE = 'alpine:3.14'

  def __init__(self, base_name: str):
    """Inits partially the task_spec.

    The task input spec will be assigned during the components connecting phase
    (in `kfp.dsl._component_bridage`).

    Args:
      base_name: The base name of the adaptor op. Is will be suffixed with a
        unique index to make the component name unique within a pipeline.
    """
    global adaptor_op_index
    self._name = f'{base_name}-{adaptor_op_index}'
    adaptor_op_index += 1

    self.task_spec = pb.PipelineTaskSpec()
    self.task_spec.task_info.name = dsl_utils.sanitize_task_name(self._name)
    self.task_spec.component_ref.name = dsl_utils.sanitize_component_name(
        self._name)


class ParameterToArtifactAdaptor(BaseAdaptor):
  """The adaptor to convert a parameter into an artifact."""

  def __init__(
      self,
      input_type: _PARAMETER_TYPES,
      output_type: Union[str, Type[io_types.Artifact]],
  ):
    """Inits the component_spec, the container_spec, and part of the task_spec.

    Args:
      input_type: The type of the component input. Expects a parameter type.
      output_type: The type of the component output. Expects an artifact type.
    """
    super().__init__('legacy-p2a-adaptor')

    self.component_spec = pb.ComponentSpec()
    self.component_spec.input_definitions.parameters[
        self.INPUT_KEY].type = type_utils.get_parameter_type(input_type)
    self.component_spec.output_definitions.artifacts[
        self.OUTPUT_KEY].artifact_type.CopyFrom(
            type_utils.get_artifact_type_schema(output_type))
    self.component_spec.executor_label = dsl_utils.sanitize_executor_label(
        self._name)

    self.container_spec = pb.PipelineDeploymentConfig.PipelineContainerSpec(
        image=self._IMAGE,
        command=[
            'sh',
            '-c',
            '-ex',
            'mkdir -p "$(dirname "$1")"; echo "$0" > "$1"',
            dsl_utils.input_parameter_placeholder(self.INPUT_KEY),
            dsl_utils.output_artifact_path_placeholder(self.OUTPUT_KEY),
        ],
    )


class ArtifactToParameterAdaptor(BaseAdaptor):
  """The adaptor to convert an artifact into an parameter."""

  def __init__(
      self,
      input_type: Union[str, Type[io_types.Artifact]],
      output_type: _PARAMETER_TYPES,
  ):
    """Inits the component_spec, the container_spec, and part of the task_spec.

    Args:
      input_type: The type of the component input. Expects an artifact type.
      output_type: The type of the component output. Expects a parameter type.
    """
    super().__init__('legacy-a2p-adaptor')

    self.component_spec = pb.ComponentSpec()

    self.component_spec.input_definitions.artifacts[
        self.INPUT_KEY].artifact_type.CopyFrom(
            type_utils.get_artifact_type_schema(input_type))
    self.component_spec.output_definitions.parameters[
        self.OUTPUT_KEY].type = type_utils.get_parameter_type(output_type)
    self.component_spec.executor_label = dsl_utils.sanitize_executor_label(
        self._name)

    self.container_spec = pb.PipelineDeploymentConfig.PipelineContainerSpec(
        image=self._IMAGE,
        command=[
            'sh',
            '-c',
            '-ex',
            'mkdir -p "$(dirname "$1")"; cp "$0" "$1"',
            dsl_utils.input_artifact_path_placeholder(self.INPUT_KEY),
            dsl_utils.output_parameter_path_placeholder(self.OUTPUT_KEY),
        ],
    )
