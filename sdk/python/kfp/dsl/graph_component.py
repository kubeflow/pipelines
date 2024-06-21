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
from typing import Callable, Optional
import uuid

from kfp.compiler import pipeline_spec_builder as builder
from kfp.dsl import base_component
from kfp.dsl import pipeline_channel
from kfp.dsl import pipeline_config
from kfp.dsl import pipeline_context
from kfp.dsl import structures
from kfp.pipeline_spec import pipeline_spec_pb2


class GraphComponent(base_component.BaseComponent):
    """A component defined via @dsl.pipeline decorator.

    Attribute:
        pipeline_func: The function that becomes the implementation of this component.
    """

    def __init__(
        self,
        component_spec: structures.ComponentSpec,
        pipeline_func: Callable,
        display_name: Optional[str] = None,
        pipeline_config: pipeline_config.PipelineConfig = None,
    ):
        super().__init__(component_spec=component_spec)
        self.pipeline_func = pipeline_func
        self.pipeline_config = pipeline_config

        args_list = []
        signature = inspect.signature(pipeline_func)

        for arg_name in signature.parameters:
            input_spec = component_spec.inputs[arg_name]
            args_list.append(
                pipeline_channel.create_pipeline_channel(
                    name=arg_name,
                    channel_type=input_spec.type,
                    is_artifact_list=input_spec.is_artifact_list,
                ))

        with pipeline_context.Pipeline(
                self.component_spec.name) as dsl_pipeline:
            pipeline_outputs = pipeline_func(*args_list)

        if not dsl_pipeline.tasks:
            raise ValueError('Task is missing from pipeline.')

        # Making the pipeline group name unique to prevent name clashes with
        # templates
        pipeline_group = dsl_pipeline.groups[0]
        pipeline_group.name = uuid.uuid4().hex

        pipeline_spec, platform_spec = builder.create_pipeline_spec(
            pipeline=dsl_pipeline,
            component_spec=self.component_spec,
            pipeline_outputs=pipeline_outputs,
            pipeline_config=pipeline_config,
        )

        pipeline_root = getattr(pipeline_func, 'pipeline_root', None)
        if pipeline_root is not None:
            pipeline_spec.default_pipeline_root = pipeline_root
        if display_name is not None:
            pipeline_spec.pipeline_info.display_name = display_name
        if component_spec.description is not None:
            pipeline_spec.pipeline_info.description = component_spec.description

        self.component_spec.implementation.graph = pipeline_spec
        self.component_spec.platform_spec = platform_spec

    @property
    def pipeline_spec(self) -> pipeline_spec_pb2.PipelineSpec:
        """Returns the pipeline spec of the component."""
        return self.component_spec.implementation.graph

    def execute(self, **kwargs):
        raise RuntimeError('Graph component has no local execution mode.')
