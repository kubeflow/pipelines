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

from typing import Callable

from kfp.dsl import component_factory
from kfp.dsl import container_component_class


def container_component(
        func: Callable) -> container_component_class.ContainerComponent:
    """Decorator for container-based components in KFP v2.

    Args:
        func: The python function to create a component from. The function
            should have type annotations for all its arguments, indicating how
            it is intended to be used (e.g. as an input/output Artifact object,
            a plain parameter, or a path to a file).

    Example:
      ::

        from kfp.dsl import container_component, ContainerSpec, InputPath, OutputPath, Output

        @container_component
        def my_component(
            dataset_path: InputPath(Dataset),
            model: Output[Model],
            num_epochs: int,
            output_parameter: OutputPath(str),
        ):
            return ContainerSpec(
                image='gcr.io/my-image',
                command=['sh', 'my_component.sh'],
                args=[
                '--dataset_path', dataset_path,
                '--model_path', model.path,
                '--output_parameter_path', output_parameter,
            ]
        )
    """
    return component_factory.create_container_component_from_func(func)
