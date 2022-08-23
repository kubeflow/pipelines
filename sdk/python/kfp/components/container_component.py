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
"""Container-based component."""

from typing import Callable

from kfp.components import base_component
from kfp.components import structures


class ContainerComponent(base_component.BaseComponent):
    """Component defined via pre-built container.

    Attribute:
        pipeline_func: The function that becomes the implementation of this component.
    """

    def __init__(self, component_spec: structures.ComponentSpec,
                 pipeline_func: Callable) -> None:
        super().__init__(component_spec=component_spec)
        self.pipeline_func = pipeline_func

    def execute(self, **kwargs):
        # ContainerComponent`: Also inherits from `BaseComponent`.
        # As its name suggests, this class backs (custom) container components.
        # Its `execute()` method uses `docker run` for local component execution
        raise NotImplementedError
