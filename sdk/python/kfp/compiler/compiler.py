# Copyright 2021-2022 The Kubeflow Authors
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
"""KFP DSL compiler.

Implementation of KFP compiler that compiles KFP pipeline into Pipeline IR:
https://docs.google.com/document/d/1PUDuSQ8vmeKSBloli53mp7GIvzekaY7sggg6ywy35Dk/
"""

import inspect
from typing import Any, Callable, Dict, Mapping, Optional, Union
import uuid

from kfp.compiler import pipeline_spec_builder as builder
from kfp.components import base_component
from kfp.components import component_factory
from kfp.components import graph_component
from kfp.components import pipeline_channel
from kfp.components import pipeline_context
from kfp.components.types import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2


class Compiler:
    """Compiles pipelines composed using the KFP SDK DSL to a YAML pipeline
    definition.

    The pipeline definition is `PipelineSpec IR <https://github.com/kubeflow/pipelines/blob/2060e38c5591806d657d85b53eed2eef2e5de2ae/api/v2alpha1/pipeline_spec.proto#L50>`_, the protobuf message that defines a pipeline.

    Example:
      ::

        @dsl.pipeline(
          name='name',
        )
        def my_pipeline(a: int, b: str = 'default value'):
            ...

        kfp.compiler.Compiler().compile(
            pipeline_func=my_pipeline,
            package_path='path/to/pipeline.yaml',
            pipeline_parameters={'a': 1},
        )
    """

    def compile(
        self,
        pipeline_func: Union[Callable[..., Any], base_component.BaseComponent],
        package_path: str,
        pipeline_name: Optional[str] = None,
        pipeline_parameters: Optional[Dict[str, Any]] = None,
        type_check: bool = True,
    ) -> None:
        """Compiles the pipeline or component function into IR YAML.

        Args:
            pipeline_func: Pipeline function constructed with the ``@dsl.pipeline`` or component constructed with the ``@dsl.component`` decorator.
            package_path: Output YAML file path. For example, ``'~/my_pipeline.yaml'`` or ``'~/my_component.yaml'``.
            pipeline_name: Name of the pipeline.
            pipeline_parameters: Map of parameter names to argument values.
            type_check: Whether to enable type checking of component interfaces during compilation.
        """

        with type_utils.TypeCheckManager(enable=type_check):
            if isinstance(pipeline_func, graph_component.GraphComponent):
                pipeline_spec = self._create_pipeline(
                    pipeline_func=pipeline_func.pipeline_func,
                    pipeline_name=pipeline_name,
                    pipeline_parameters_override=pipeline_parameters,
                )

            elif isinstance(pipeline_func, base_component.BaseComponent):
                component_spec = builder.modify_component_spec_for_compile(
                    component_spec=pipeline_func.component_spec,
                    pipeline_name=pipeline_name,
                    pipeline_parameters_override=pipeline_parameters,
                )
                pipeline_spec = component_spec.to_pipeline_spec()
            else:
                raise ValueError(
                    'Unsupported pipeline_func type. Expected '
                    'subclass of `base_component.BaseComponent` or '
                    '`Callable` constructed with @dsl.pipeline '
                    f'decorator. Got: {type(pipeline_func)}')
            builder.write_pipeline_spec_to_file(
                pipeline_spec=pipeline_spec, package_path=package_path)

    def _create_pipeline(
        self,
        pipeline_func: Callable[..., Any],
        pipeline_name: Optional[str] = None,
        pipeline_parameters_override: Optional[Mapping[str, Any]] = None,
    ) -> pipeline_spec_pb2.PipelineSpec:
        """Creates a pipeline instance and constructs the pipeline spec from
        it.

        Args:
            pipeline_func: The pipeline function with @dsl.pipeline decorator.
            pipeline_name: Optional; the name of the pipeline.
            pipeline_parameters_override: Optional; the mapping from parameter
                names to values.

        Returns:
            A PipelineSpec proto representing the compiled pipeline.
        """

        # pipeline_func is a GraphComponent instance, retrieve its the original
        # pipeline function
        pipeline_func = getattr(pipeline_func, 'python_func', pipeline_func)

        # Create the arg list with no default values and call pipeline function.
        # Assign type information to the PipelineChannel
        pipeline_meta = component_factory.extract_component_interface(
            pipeline_func)
        pipeline_name = pipeline_name or pipeline_meta.name

        pipeline_root = getattr(pipeline_func, 'pipeline_root', None)

        args_list = []
        signature = inspect.signature(pipeline_func)

        for arg_name in signature.parameters:
            arg_type = pipeline_meta.inputs[arg_name].type
            args_list.append(
                pipeline_channel.create_pipeline_channel(
                    name=arg_name,
                    channel_type=arg_type,
                ))

        with pipeline_context.Pipeline(pipeline_name) as dsl_pipeline:
            pipeline_func(*args_list)

        if not dsl_pipeline.tasks:
            raise ValueError('Task is missing from pipeline.')

        pipeline_inputs = pipeline_meta.inputs or {}

        # Verify that pipeline_parameters_override contains only input names
        # that match the pipeline inputs definition.
        pipeline_parameters_override = pipeline_parameters_override or {}
        for input_name in pipeline_parameters_override:
            if input_name not in pipeline_inputs:
                raise ValueError(
                    'Pipeline parameter {} does not match any known '
                    'pipeline argument.'.format(input_name))

        # Fill in the default values.
        args_list_with_defaults = [
            pipeline_channel.create_pipeline_channel(
                name=input_name,
                channel_type=input_spec.type,
                value=pipeline_parameters_override.get(input_name) or
                input_spec.default,
            ) for input_name, input_spec in pipeline_inputs.items()
        ]

        # Making the pipeline group name unique to prevent name clashes with
        # templates
        pipeline_group = dsl_pipeline.groups[0]
        pipeline_group.name = uuid.uuid4().hex

        pipeline_spec, _ = builder.create_pipeline_spec_and_deployment_config(
            pipeline_args=args_list_with_defaults,
            pipeline=dsl_pipeline,
        )

        if pipeline_root:
            pipeline_spec.default_pipeline_root = pipeline_root

        return pipeline_spec
