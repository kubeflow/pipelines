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
"""Python function-based component."""

from typing import Callable

from kfp import components
from kfp.components import structures


class PythonComponent(components.BaseComponent):
    """A component defined via Python function.

    **Note:** ``PythonComponent`` is not intended to be used to construct components directly. Use ``@kfp.dsl.component`` instead.

    Args:
        component_spec: Component definition.
        python_func: Python function that becomes the implementation of this component.
    """

    def __init__(
        self,
        component_spec: structures.ComponentSpec,
        python_func: Callable,
    ):
        super().__init__(component_spec=component_spec)
        self.python_func = python_func

    def execute(self, **kwargs):
        """Executes the Python function that defines the component."""
        return self.python_func(**kwargs)
