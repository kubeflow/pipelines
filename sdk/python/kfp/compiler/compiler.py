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

from typing import Any, Dict, Optional

from kfp.compiler import pipeline_spec_builder as builder
from kfp.dsl import base_component
from kfp.dsl.types import type_utils


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
        pipeline_func: base_component.BaseComponent,
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
            if not isinstance(pipeline_func, base_component.BaseComponent):
                raise ValueError(
                    'Unsupported pipeline_func type. Expected '
                    'subclass of `base_component.BaseComponent` or '
                    '`Callable` constructed with @dsl.pipeline '
                    f'decorator. Got: {type(pipeline_func)}')

            pipeline_spec = builder.modify_pipeline_spec_with_override(
                pipeline_spec=pipeline_func.pipeline_spec,
                pipeline_name=pipeline_name,
                pipeline_parameters=pipeline_parameters,
            )

            builder.write_pipeline_spec_to_file(
                pipeline_spec=pipeline_spec,
                pipeline_description=pipeline_func.description,
                platform_spec=pipeline_func.platform_spec,
                package_path=package_path,
            )
