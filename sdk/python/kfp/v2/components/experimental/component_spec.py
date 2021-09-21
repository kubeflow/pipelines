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
"""Definitions for component spec."""

from dataclasses import dataclass
from typing import Any, Mapping, Optional, Sequence, Union

from kfp.components import _structures as v1_components
from kfp.components import _data_passing


@dataclass
class InputSpec:
  """Component input definitions.

  Attributes:
    name: The name of the input.
    type: The type of the input.
    default: Optional; the default value for the input.
  """
  name: str
  type: Any
  default: Optional[Any] = None


@dataclass
class OutputSpec:
  """Component output definitions.

  Attributes:
    name: The name of the output.
    type: The type of the output.
  """
  name: str
  type: Any


@dataclass
class BasePlaceholder:
  """Base class for placeholders that could appear in container cmd and args.

  Attributes:
    name: Referencing an input or an output from the component.
  """
  name: str


class InputValuePlaceholder(BasePlaceholder):
  pass


class InputPathPlaceholder(BasePlaceholder):
  pass


class InputUriPlaceholder(BasePlaceholder):
  pass


class OutputPathPlaceholder(BasePlaceholder):
  pass


class OutputUriPlaceholder(BasePlaceholder):
  pass


@dataclass
class ResourceSpec:
  """The resource requirements of a container execution.

  Attributes:
    cpu_limit: Optional; the limit of the number of vCPU cores.
    memory_limit: Optional; the memory limit in GB.
    accelerator_type: Optional; the type of accelerators attached to the
      container.
    accelerator_count: Optional; the number of accelerators attached.
  """
  cpu_limit: Optional[float] = None
  memory_limit: Optional[float] = None
  accelerator_type: Optional[str] = None
  accelerator_count: Optional[int] = None


@dataclass
class ContainerSpec:
  """Container implementation definition.

  Attributes:
    image: The container image.
    commands: Optional; the container entrypoint.
    arguments: Optional; the arguments to the container entrypoint.
    env: Optional; the environment variables to be passed to the container.
    resources: Optional; the specification on the resource requirements.
  """
  image: str
  commands: Optional[Sequence[Union[str, BasePlaceholder]]] = None
  arguments: Optional[Sequence[Union[str, BasePlaceholder]]] = None
  env: Optional[Mapping[str, Union[str, BasePlaceholder]]] = None
  resources: Optional[ResourceSpec] = None


@dataclass
class ImporterSpec:
  """ImporterSpec definition.

  Attributes:
    artifact_uri: The URI of the artifact.
    type_schema: The type of the artifact.
    reimport: Whether or not import an artifact regardless it has been imported
      before.
    metadata: Optional; the properties of the artifact.
  """
  artifact_uri: str
  type_schema: str
  reimport: bool
  metadata: Mapping[str, Any] = None


@dataclass
class TaskSpec:
  """The spec of a pipeline task.

  Attributes:
    name: The name of the task.
    inputs: The sources of task inputs. Constant values or PipelineParams.
    dependent_tasks: The list of upstream tasks.
    enable_caching: Whether or not to enable caching for the task.
    component_ref: The name of a component spec this task is based on.
    trigger_condition: Optional; an expression which will be evaluated into a
      boolean value. True to trigger the task to run.
    trigger_strategy: Optional; when the task will be ready to be triggered.
      Valid values include: "TRIGGER_STRATEGY_UNSPECIFIED",
        "ALL_UPSTREAM_TASKS_SUCCEEDED", and "ALL_UPSTREAM_TASKS_COMPLETED".
    iterator_items: Optional; the items to iterate on. A constant value or a
      PipelineParam.
    iterator_item_input: Optional; the name of the input which has the item from
      the [items][] collection.
  """
  name: str
  inputs: Mapping[str, Any]
  dependent_tasks: Sequence[str]
  enable_caching: bool
  component_ref: str
  trigger_condition: Optional[str] = None
  trigger_strategy: Optional[str] = None
  iterator_items: Optional[Any] = None
  iterator_item_input: Optional[str] = None


@dataclass
class DagSpec:
  """DAG(graph) implementation definition.

  Attributes:
    tasks: The tasks inside the DAG.
    outputs: Defines how the outputs of the dag are linked to the sub tasks.
  """
  tasks: Mapping[str, TaskSpec]
  # TODO(chensun): revisit if we need a DagOutputsSpec class.
  outputs: Mapping[str, Any]


@dataclass
class MetadataSpec:
  """Metadata for a component.

  Attributes:
    description: Optional; the description of the component.
    annotations: Optional; the annotations of the component as key-value pairs.
    labels: Optional; the labels of the component as key-value pairs.
  """
  description: Optional[str] = None
  annotations: Optional[Mapping[str, str]] = None
  labels: Optional[Mapping[str, str]] = None


class ComponentSpec:
  """The definition of a component.

  Attributes:
    name: The name of the component.
    implementation: The implementation of the component. Either a container a
      DAG consists of other components.
    input_specs: Optional; the input definitions of the component.
    output_specs: Optional; the output definitions of the component.
    metadata: Optional; the metadata of the component, including description,
      annotations, and labels.
  """

  def __init__(
      self,
      name: str,
      implementation: Union[ContainerSpec, ImporterSpec, DagSpec],
      input_specs: Optional[Sequence[InputSpec]] = None,
      output_specs: Optional[Sequence[OutputSpec]] = None,
      metadata: Optional[MetadataSpec] = None,
  ):
    """Init function for ComponentSpec.

    Args:
      name: The name of the component.
      implementation: The implementation of the component. Either an executor
        (container, importer) or a DAG consists of other components.
      input_specs: Optional; the input definitions of the component.
      output_specs: Optional; the output definitions of the component.
      metadata: Optional; the metadata of the component, including description,
        annotations, and labels.
    """
    self.name = name
    self.input_specs = self._validate_input_output_uniqueness(input_specs)
    self.output_specs = self._validate_input_output_uniqueness(output_specs)
    self.implementation = self._validate_placeholders(implementation)
    self.metadata = metadata

  def _validate_input_output_uniqueness(
      self,
      input_or_output_specs: Optional[Union[Sequence[InputSpec],
                                            Sequence[OutputSpec]]],
  ) -> Optional[Union[Sequence[InputSpec], Sequence[OutputSpec]]]:
    """Validates no duplicate input/output names.

    Args:
      input_or_output_specs: List of InputSpec or OutputSpec.

    Returns:
      The original input or output specs if no validation error.

    Raises:
      ValueError: if non-unique input or output name found.
    """
    names = set()
    for spec in input_or_output_specs or []:
      if spec.name in names:
        input_or_output = 'input' if isinstance(spec, InputSpec) else 'output'
        raise ValueError(f'Non-unique {input_or_output} name "{spec.name}".')
      names.add(spec.name)
    return input_or_output_specs

  def _validate_placeholders(
      self,
      implementation: Union[ContainerSpec, ImporterSpec, DagSpec],
  ) -> Union[ContainerSpec, DagSpec]:
    """Validates placeholders reference existing input/output names.

    Args:
      implementation: The component implementation spec.

    Returns:
      The original component implementation spec if no validation error.

    Raises:
      ValueError: if any placeholder references a non-existing input or output.
      TypeError: if any argument is neither a str nor a placeholder instance.
    """
    if not isinstance(implementation, ContainerSpec):
      return implementation

    input_names = [input_spec.name for input_spec in self.input_specs or []]
    output_names = [output_spec.name for output_spec in self.output_specs or []]

    for arg in [
        *(implementation.commands or []), *(implementation.arguments or [])
    ]:
      if isinstance(
          arg,
          (InputValuePlaceholder, InputPathPlaceholder, InputUriPlaceholder)):
        if arg.name not in input_names:
          raise ValueError(f'Argument "{arg}" references non-existing input.')
      elif isinstance(arg, (OutputPathPlaceholder, OutputUriPlaceholder)):
        if arg.name not in output_names:
          raise ValueError(f'Argument "{arg}" references non-existing output.')
      # TODO(chensun): revisit if we need to support IfPlaceholder,
      # ConcatPlaceholder, etc. in the new format.
      elif not isinstance(arg, str):
        raise TypeError(f'Unexpected argument "{arg}".')
    return implementation

  @classmethod
  def from_v1_component_spec(
      cls, v1_component_spec: v1_components.ComponentSpec) -> 'ComponentSpec':
    raise NotImplementedError

  def to_v1_component_spec(self) -> v1_components.ComponentSpec:
    """Convert to v1 ComponentSpec.

    Needed until downstream accept new ComponentSpec."""
    if isinstance(self.implementation, DagSpec):
      raise NotImplementedError

    def _transform_arg(arg: Union[str, BasePlaceholder]) -> Any:
      if isinstance(arg, str):
        return arg
      elif isinstance(arg, InputValuePlaceholder):
        return v1_components.InputValuePlaceholder(arg.name)
      elif isinstance(arg, InputPathPlaceholder):
        return v1_components.InputPathPlaceholder(arg.name)
      elif isinstance(arg, InputUriPlaceholder):
        return v1_components.InputUriPlaceholder(arg.name)
      elif isinstance(arg, OutputPathPlaceholder):
        return v1_components.OutputPathPlaceholder(arg.name)
      elif isinstance(arg, OutputUriPlaceholder):
        return v1_components.OutputUriPlaceholder(arg.name)
      else:
        # TODO(chensun): transform additional placeholders: if, concat, etc.?
        raise ValueError(
            f'Unexpected command/argument type: "{arg}" of type "{type(arg)}".')

    return v1_components.ComponentSpec(
        name=self.name,
        inputs=[
            v1_components.InputSpec(
                name=input_spec.name,
                type=_data_passing.get_canonical_type_struct_for_type(
                    input_spec.type),
                default=input_spec.default,
            ) for input_spec in self.input_specs
        ],
        outputs=[
            v1_components.OutputSpec(
                name=output_spec.name,
                type=_data_passing.get_canonical_type_struct_for_type(
                    output_spec.type),
            ) for output_spec in self.output_specs
        ],
        implementation=v1_components.ContainerImplementation(
            container=v1_components.ContainerSpec(
                image=self.implementation.image,
                command=[
                    _transform_arg(cmd)
                    for cmd in self.implementation.commands or []
                ],
                args=[
                    _transform_arg(arg)
                    for arg in self.implementation.arguments or []
                ],
                env={
                    name: _transform_arg(value)
                    for name, value in self.implementation.env or {}
                },
            )),
    )

  @classmethod
  def load_from_component_yaml(cls, component_yaml: str) -> 'ComponentSpec':
    raise NotImplementedError

  def save_to_component_yaml(self, output_file: str) -> None:
    raise NotImplementedError
