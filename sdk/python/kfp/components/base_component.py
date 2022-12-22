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
"""Base class for KFP components."""

import abc
from typing import List

from kfp.components import pipeline_task
from kfp.components import structures
from kfp.components.types import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2


class BaseComponent(abc.ABC):
    """Base class for a component.

    **Note:** ``BaseComponent`` is not intended to be used to construct components directly. Use ``@kfp.dsl.component`` or ``kfp.components.load_component_from_*()`` instead.

    Attributes:
      name: Name of the component.
      component_spec: Component definition.
    """

    def __init__(self, component_spec: structures.ComponentSpec):
        """Init function for BaseComponent.

        Args:
          component_spec: The component definition.
        """
        self.component_spec = component_spec
        self.name = component_spec.name
        self.description = component_spec.description or None

        # Arguments typed as PipelineTaskFinalStatus are special arguments that
        # do not count as user inputs. Instead, they are reserved to for the
        # (backend) system to pass a value.
        self._component_inputs = {
            input_name for input_name, input_spec in (
                self.component_spec.inputs or {}).items()
            if not type_utils.is_task_final_status_type(input_spec.type)
        }

    def __call__(self, *args, **kwargs) -> pipeline_task.PipelineTask:
        """Creates a PipelineTask object.

        The arguments are generated on the fly based on component input
        definitions.
        """
        task_inputs = {}

        if args:
            raise TypeError(
                'Components must be instantiated using keyword arguments. Positional '
                f'parameters are not allowed (found {len(args)} such parameters for '
                f'component "{self.name}").')

        for k, v in kwargs.items():
            if k not in self._component_inputs:
                raise TypeError(
                    f'{self.name}() got an unexpected keyword argument "{k}".')
            task_inputs[k] = v

        # Skip optional inputs and arguments typed as PipelineTaskFinalStatus.
        missing_arguments = [
            arg for arg in self.required_inputs if arg not in kwargs
        ]
        if missing_arguments:
            argument_or_arguments = 'argument' if len(
                missing_arguments) == 1 else 'arguments'
            arguments = ', '.join(
                arg_name.replace('-', '_') for arg_name in missing_arguments)

            raise TypeError(
                f'{self.name}() missing {len(missing_arguments)} required '
                f'{argument_or_arguments}: {arguments}.')

        return pipeline_task.PipelineTask(
            component_spec=self.component_spec,
            args=task_inputs,
        )

    @property
    def pipeline_spec(self) -> pipeline_spec_pb2.PipelineSpec:
        """Returns the pipeline spec of the component."""
        with BlockPipelineTaskRegistration():
            return self.component_spec.to_pipeline_spec()

    @abc.abstractmethod
    def execute(self, **kwargs):
        """Executes the component locally if implemented by the inheriting
        subclass."""

    @property
    def required_inputs(self) -> List[str]:
        return [
            input_name for input_name, input_spec in (
                self.component_spec.inputs or {}).items()
            if not input_spec.optional
        ]


class BlockPipelineTaskRegistration:
    """Temporarily stop registering tasks to the default pipeline.

    Handles special, uncommon functions that decorate and mutate a
    component, possibly by using the component's .pipeline_spec
    attribute. This is exhibited in the version of
    google_cloud_pipeline_components compatible with KFP SDK v2.
    """

    # TODO: this handles the special case of a compiled component (when compiled inside a pipeline), which should not have any concept of a default pipeline. Perhaps there is a way to unify component/pipeline compilation concepts to remove this workaround?

    def __enter__(self):
        self.task_handler, pipeline_task.PipelineTask._register_task_handler = pipeline_task.PipelineTask._register_task_handler, pipeline_task._register_task_handler

    def __exit__(self, *args):
        pipeline_task.PipelineTask._register_task_handler = self.task_handler
