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

import dataclasses
import enum
import json
from typing import Any, Dict, Mapping, Optional, Sequence, Union

from kfp.components import _components
from kfp.components import structures
import pydantic
import yaml


class InputSpec(pydantic.BaseModel):
    """Component input definitions.

    Attributes:
      type: The type of the input.
      default: Optional; the default value for the input.
    """
    # TODO(ji-yaqi): Add logic to cast default value into the specified type.
    type: str
    default: Optional[Union[str, int, float, bool, dict, list]] = None


class OutputSpec(pydantic.BaseModel):
    """Component output definitions.

    Attributes:
      type: The type of the output.
    """
    type: Union[str, int, float, bool, dict, list]


class BasePlaceholder(pydantic.BaseModel):
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


@dataclasses.dataclass
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


class ContainerSpec(pydantic.BaseModel):
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


@dataclasses.dataclass
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
    metadata: Optional[Mapping[str, Any]] = None


@dataclasses.dataclass
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


@dataclasses.dataclass
class DagSpec:
    """DAG(graph) implementation definition.

    Attributes:
      tasks: The tasks inside the DAG.
      outputs: Defines how the outputs of the dag are linked to the sub tasks.
    """
    tasks: Mapping[str, TaskSpec]
    # TODO(chensun): revisit if we need a DagOutputsSpec class.
    outputs: Mapping[str, Any]


class SchemaVersion(str, enum.Enum):
    V1 = 'v1'
    V2 = 'v2'


class ComponentSpec(pydantic.BaseModel):
    """The definition of a component.

    Attributes:
      name: The name of the component.
      implementation: The implementation of the component. Either an executor
        (container, importer) or a DAG consists of other components.
      inputs: Optional; the input definitions of the component.
      outputs: Optional; the output definitions of the component.
      description: Optional; the description of the component.
      annotations: Optional; the annotations of the component as key-value pairs.
      labels: Optional; the labels of the component as key-value pairs.
      schema_version: Internal field for tracking component version.
    """

    name: str
    description: Optional[str] = None
    implementation: Union[ContainerSpec, ImporterSpec, DagSpec]
    inputs: Optional[Dict[str, InputSpec]] = None
    outputs: Optional[Dict[str, OutputSpec]] = None
    description: Optional[str] = None
    annotations: Optional[Mapping[str, str]] = None
    labels: Optional[Mapping[str, str]] = None
    schema_version: SchemaVersion = SchemaVersion.V2

    def _validate_placeholders(
        self,
        implementation: Union[ContainerSpec, ImporterSpec, DagSpec],
    ) -> Union[ContainerSpec, ImporterSpec, DagSpec]:
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
        output_names = [
            output_spec.name for output_spec in self.output_specs or []
        ]

        for arg in [
                *(implementation.commands or []),
                *(implementation.arguments or [])
        ]:
            if isinstance(arg, (InputValuePlaceholder, InputPathPlaceholder,
                                InputUriPlaceholder)):
                if arg.name not in input_names:
                    raise ValueError(
                        f'Argument "{arg}" references non-existing input.')
            elif isinstance(arg, (OutputPathPlaceholder, OutputUriPlaceholder)):
                if arg.name not in output_names:
                    raise ValueError(
                        f'Argument "{arg}" references non-existing output.')
            # TODO(chensun): revisit if we need to support IfPlaceholder,
            # ConcatPlaceholder, etc. in the new format.
            elif not isinstance(arg, str):
                raise TypeError(f'Unexpected argument "{arg}".')
        return implementation

    @classmethod
    def from_v1_component_spec(
            cls,
            v1_component_spec: structures.ComponentSpec) -> 'ComponentSpec':
        """Converts V1 ComponentSpec to V2 ComponentSpec.

        Args:
          v1_component_spec: The V1 ComponentSpec.

        Returns:
          Component spec in the form of V2 ComponentSpec.

        Raises:
          ValueError: If implementation is not found.
          TypeError: if any argument is neither a str nor Dict.
        """
        component_dict = v1_component_spec.to_dict()
        if component_dict.get('implementation') is None:
            raise ValueError('Implementation field not found')
        if 'container' not in component_dict.get('implementation'):
            raise NotImplementedError

        def _transform_arg(
                arg: Union[str, Dict[str, str]]) -> Union[str, BasePlaceholder]:
            if isinstance(arg, str):
                return arg
            elif 'inputValue' in arg:
                return InputValuePlaceholder(name=arg['inputValue'])
            elif 'inputPath' in arg:
                return InputPathPlaceholder(name=arg['inputPath'])
            elif 'inputUri' in arg:
                return InputUriPlaceholder(name=arg['inputUri'])
            elif 'outputPath' in arg:
                return OutputPathPlaceholder(name=arg['outputPath'])
            elif 'outputUri' in arg:
                return OutputUriPlaceholder(name=arg['outputUri'])
            else:
                raise ValueError(
                    f'Unexpected command/argument type: "{arg}" of type "{type(arg)}".'
                )

        implementation = component_dict['implementation']['container']
        implementation['commands'] = [
            _transform_arg(command)
            for command in implementation.pop('command', [])
        ]
        implementation['arguments'] = [
            _transform_arg(command)
            for command in implementation.pop('args', [])
        ]
        implementation['env'] = {
            key: _transform_arg(command)
            for key, command in implementation.pop('env', {}).items()
        }
        container_spec = ContainerSpec.parse_obj(implementation)

        return ComponentSpec(
            name=component_dict.get('name', 'name'),
            description=component_dict.get('description'),
            implementation=container_spec,
            inputs={
                spec['name']: InputSpec(
                    type=spec.get('type', 'Artifact'),
                    default=spec.get('default', None))
                for spec in component_dict.get('inputs', [])
            },
            outputs={
                spec['name']: OutputSpec(type=spec.get('type', 'String'))
                for spec in component_dict.get('outputs', [])
            },
            schema_version=SchemaVersion.V1)

    def to_v1_component_spec(self) -> structures.ComponentSpec:
        """Converts to v1 ComponentSpec.

        Returns:
          Component spec in the form of V1 ComponentSpec.

        Needed until downstream accept new ComponentSpec.
        """
        if isinstance(self.implementation, DagSpec):
            raise NotImplementedError

        def _transform_arg(arg: Union[str, BasePlaceholder]) -> Any:
            if isinstance(arg, str):
                return arg
            elif isinstance(arg, InputValuePlaceholder):
                return structures.InputValuePlaceholder(arg.name)
            elif isinstance(arg, InputPathPlaceholder):
                return structures.InputPathPlaceholder(arg.name)
            elif isinstance(arg, InputUriPlaceholder):
                return structures.InputUriPlaceholder(arg.name)
            elif isinstance(arg, OutputPathPlaceholder):
                return structures.OutputPathPlaceholder(arg.name)
            elif isinstance(arg, OutputUriPlaceholder):
                return structures.OutputUriPlaceholder(arg.name)
            else:
                # TODO(chensun): transform additional placeholders: if, concat, etc.?
                raise ValueError(
                    f'Unexpected command/argument type: "{arg}" of type "{type(arg)}".'
                )

        return structures.ComponentSpec(
            name=self.name,
            inputs=[
                structures.InputSpec(
                    name=name,
                    type=input_spec.type,
                    default=input_spec.default,
                ) for name, input_spec in self.inputs.items()
            ],
            outputs=[
                structures.OutputSpec(
                    name=name,
                    type=output_spec.type,
                ) for name, output_spec in self.outputs.items()
            ],
            implementation=structures.ContainerImplementation(
                container=structures.ContainerSpec(
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
        """Loads V1 or V2 component yaml into ComponentSpec.

        Args:
          component_yaml: the component yaml in string format.

        Returns:
          Component spec in the form of V2 ComponentSpec.
        """

        json_component = yaml.safe_load(component_yaml)

        if 'schema_version' in json_component and json_component[
                'schema_version'] == SchemaVersion.V2:
            return ComponentSpec.parse_obj(json_component)

        v1_component = _components._load_component_spec_from_component_text(
            component_yaml)
        return cls.from_v1_component_spec(v1_component)

    def save_to_component_yaml(self, output_file: str) -> None:
        """Saves ComponentSpec into yaml file.

        Args:
          output_file: File path to store the component yaml.
        """
        with open(output_file, 'a') as output_file:
            json_component = self.json(exclude_none=True)
            yaml_file = yaml.safe_dump(json.loads(json_component))
            output_file.write(yaml_file)
