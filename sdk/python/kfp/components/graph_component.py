# Copyright 2022 The Kubeflow Authors
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
"""Pipeline as a component (aka graph component)."""

import inspect
from typing import Callable, List
import uuid

from kfp.compiler import compiler_utils
from kfp.compiler import pipeline_spec_builder as builder
from kfp.components import base_component
from kfp.components import pipeline_channel
from kfp.components import pipeline_context
from kfp.components import structures
from kfp.components import tasks_group
from kfp.pipeline_spec import pipeline_spec_pb2


class GraphComponent(base_component.BaseComponent):
    """A component definced via @dsl.pipeline decorator.

    Attribute:
        pipeline_func: The function that becomes the implementation of this component.
    """

    def __init__(
        self,
        component_spec: structures.ComponentSpec,
        pipeline_func: Callable,
        name: str,
    ):
        super().__init__(component_spec=component_spec)
        self.pipeline_func = pipeline_func
        self.name = name

        args_list = []
        signature = inspect.signature(pipeline_func)

        for arg_name in signature.parameters:
            arg_type = component_spec.inputs[arg_name].type
            args_list.append(
                pipeline_channel.create_pipeline_channel(
                    name=arg_name,
                    channel_type=arg_type,
                ))

        with pipeline_context.Pipeline(
                self.component_spec.name) as dsl_pipeline:
            pipeline_func(*args_list)

        if not dsl_pipeline.tasks:
            raise ValueError('Task is missing from pipeline.')

        component_inputs = self.component_spec.inputs or {}

        # Fill in the default values.
        args_list_with_defaults = [
            pipeline_channel.create_pipeline_channel(
                name=input_name,
                channel_type=input_spec.type,
                value=input_spec.default,
            ) for input_name, input_spec in component_inputs.items()
        ]

        # Making the pipeline group name unique to prevent name clashes with
        # templates
        pipeline_group = dsl_pipeline.groups[0]
        pipeline_group.name = uuid.uuid4().hex

        self.pipeline_spec, self.deployment_config = (
            builder.create_pipeline_spec_and_deployment_config(
                pipeline=dsl_pipeline, pipeline_args=args_list_with_defaults))

        self.component_spec.implementation.graph = self.pipeline_spec

    def execute(self, **kwargs):
        raise RuntimeError('Graph component has no local execution mode.')
