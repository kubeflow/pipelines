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
from typing import Callable
import uuid

from kfp.compiler import pipeline_spec_builder as builder
from kfp.components import base_component
from kfp.components import pipeline_channel
from kfp.components import pipeline_context
from kfp.components import structures
from kfp.components import utils
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
        name: str,
    ):
        super().__init__(component_spec=component_spec)
        self.pipeline_func = pipeline_func
        self.name = name

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
        )

        pipeline_root = getattr(pipeline_func, 'pipeline_root', None)
        if pipeline_root is not None:
            pipeline_spec.default_pipeline_root = pipeline_root

        pipeline_spec = self.dedupe_pipeline_spec(pipeline_spec)

        self.component_spec.implementation.graph = pipeline_spec
        self.component_spec.platform_spec = platform_spec

    @property
    def pipeline_spec(self) -> pipeline_spec_pb2.PipelineSpec:
        """Returns the pipeline spec of the component."""
        return self.component_spec.implementation.graph

    def execute(self, **kwargs):
        raise RuntimeError('Graph component has no local execution mode.')

    def dedupe_pipeline_spec(
        self, pipeline_spec: pipeline_spec_pb2.PipelineSpec
    ) -> pipeline_spec_pb2.PipelineSpec:
        clone_mapping = {}
        components_with_clones = {}

        # Collect the collection of dedupable components
        for component_name, component_spec in pipeline_spec.components.items():
            if component_spec.executor_label and component_name not in components_with_clones:
                original_components_executor_spec = pipeline_spec.deployment_spec.fields[
                    'executors'].struct_value.fields[
                        component_spec.executor_label]
                for executor_name, executor_spec in pipeline_spec.deployment_spec.fields[
                        'executors'].struct_value.fields.items():
                    corresponding_component_name = 'comp' + executor_name[4:]
                    if executor_name != component_spec.executor_label and executor_spec == original_components_executor_spec and corresponding_component_name not in components_with_clones and pipeline_spec.components[
                            corresponding_component_name].executor_label:
                        clone_mapping[component_name] = clone_mapping.get(
                            component_name, [])
                        clone_mapping[component_name].append(
                            corresponding_component_name)
                        components_with_clones[corresponding_component_name] = [
                            executor_name, pipeline_spec
                            .components[corresponding_component_name],
                            executor_spec
                        ]
                        components_with_clones[component_name] = [
                            component_spec.executor_label, component_spec,
                            original_components_executor_spec
                        ]

        # Process the pipeline spec
        for component in components_with_clones.keys():
            corresponding_executor_name = components_with_clones[component][0]
            del pipeline_spec.components[component]
            del pipeline_spec.deployment_spec.fields[
                'executors'].struct_value.fields[corresponding_executor_name]

        for component, clone_components in clone_mapping.items():
            clones = clone_components + [component]

            corresponding_executor_name, component_spec, executor_spec = components_with_clones[
                component]

            last_delimiter = component.rfind('-')
            if len(component) > last_delimiter + 1 and component[
                    last_delimiter + 1:].isnumeric():
                component = component[:-2]
                component = utils.make_name_unique_by_adding_index(
                    name=component,
                    collection=pipeline_spec.components.keys(),
                    delimiter='-')

            last_delimiter = corresponding_executor_name.rfind('-')
            if len(corresponding_executor_name
                  ) > last_delimiter + 1 and corresponding_executor_name[
                      last_delimiter + 1:].isnumeric():
                corresponding_executor_name = corresponding_executor_name[:-2]
                corresponding_executor_name = utils.make_name_unique_by_adding_index(
                    name=corresponding_executor_name,
                    collection=pipeline_spec.deployment_spec.fields['executors']
                    .struct_value.fields.keys(),
                    delimiter='-')

            pipeline_spec.components[component].CopyFrom(component_spec)
            pipeline_spec.components[
                component].executor_label = corresponding_executor_name
            pipeline_spec.deployment_spec.fields[
                'executors'].struct_value.fields[
                    corresponding_executor_name].CopyFrom(executor_spec)

            for _, task_spec in pipeline_spec.root.dag.tasks.items():
                if task_spec.component_ref.name in clones:
                    task_spec.component_ref.name = component

            # for inner task group calling on components
            for _, component_spec in pipeline_spec.components.items():
                if component_spec.dag:
                    for __, task_spec in component_spec.dag.tasks.items():
                        if task_spec.component_ref.name in clones:
                            task_spec.component_ref.name = component

        # clean up other component spec names
        if components_with_clones:
            changed_names = {}
            for component, component_spec in pipeline_spec.components.items():
                if component not in components_with_clones and component_spec.executor_label:
                    last_delimiter = component.rfind('-')
                    if len(component) > last_delimiter + 1 and component[
                            last_delimiter + 1:].isnumeric():
                        initial = component
                        component = component[:-2]
                        component = utils.make_name_unique_by_adding_index(
                            name=component,
                            collection=pipeline_spec.components.keys(),
                            delimiter='-')

                        executor_spec = pipeline_spec.deployment_spec.fields[
                            'executors'].struct_value.fields[
                                component_spec.executor_label]
                        del pipeline_spec.deployment_spec.fields[
                            'executors'].struct_value.fields[
                                component_spec.executor_label]

                        corresponding_executor_name = component_spec.executor_label
                        corresponding_executor_name = corresponding_executor_name[:
                                                                                  -2]
                        corresponding_executor_name = utils.make_name_unique_by_adding_index(
                            name=corresponding_executor_name,
                            collection=pipeline_spec.deployment_spec
                            .fields['executors'].struct_value.fields.keys(),
                            delimiter='-')

                        component_spec.executor_label = corresponding_executor_name
                        pipeline_spec.deployment_spec.fields[
                            'executors'].struct_value.fields[
                                corresponding_executor_name].CopyFrom(
                                    executor_spec)

                        changed_names[initial] = [component, component_spec]

                        for _, task_spec in pipeline_spec.root.dag.tasks.items(
                        ):
                            if task_spec.component_ref.name == initial:
                                task_spec.component_ref.name = component

            for initial, new_details in changed_names.items():
                del pipeline_spec.components[initial]
                component_name, component_spec = new_details
                pipeline_spec.components[component_name].CopyFrom(
                    component_spec)

        return pipeline_spec
