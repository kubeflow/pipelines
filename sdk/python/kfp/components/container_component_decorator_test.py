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

import unittest

from kfp.components import container_component
from kfp.components import structures
from kfp.components import container_component_decorator 


class TestContainerComponentDecorator(unittest.TestCase):

    def test_func_with_no_arg(self):

        @container_component_decorator.container_component
        def hello_world() -> None:
            """Hello world component."""
            return structures.ContainerSpec(
                image='python3.7',
                command=['echo', 'hello world'],
                args=[],
            )

        self.assertIsInstance(hello_world, container_component.ContainerComponent)
        self.assertIsNone(hello_world.component_spec.inputs)
