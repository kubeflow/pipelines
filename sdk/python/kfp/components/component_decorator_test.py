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

from kfp.components import python_component
from kfp.components.component_decorator import component


def func(text: str) -> str:
    """Function for lightweight component used throughout tests below."""
    return text


class TestComponentDecorator(unittest.TestCase):

    def test_as_decorator_syntactic_sugar_no_args(self):

        @component
        def hello_world(text: str) -> str:
            """Hello world component."""
            return text

        self.assertIsInstance(hello_world, python_component.PythonComponent)

    def test_as_decorator_syntactic_sugar_some_args(self):

        @component(base_image='python:3.9')
        def hello_world(text: str) -> str:
            """Hello world component."""
            return text

        self.assertIsInstance(hello_world, python_component.PythonComponent)

    def test_no_args(self):

        comp = component(func)
        self.assertIsInstance(comp, python_component.PythonComponent)

    def test_some_args(self):

        comp = component(func, base_image='python:3.9')
        self.assertIsInstance(comp, python_component.PythonComponent)

    def test_packages_to_install(self):

        comp = component(func, packages_to_install=['numpy', 'tensorflow'])
        self.assertIsInstance(comp, python_component.PythonComponent)

        concat_command = ' '.join(
            comp.component_spec.implementation.container.command)
        self.assertTrue('numpy' in concat_command and
                        'tensorflow' in concat_command)

    def test_packages_to_install_with_custom_index_url(self):
        comp = component(
            func,
            packages_to_install=['numpy', 'tensorflow'],
            pip_index_urls=['https://pypi.org/simple'])
        self.assertIsInstance(comp, python_component.PythonComponent)

        concat_command = ' '.join(
            comp.component_spec.implementation.container.command)
        self.assertTrue('numpy' in concat_command and
                        'tensorflow' in concat_command)
        self.assertTrue('https://pypi.org/simple' in concat_command)
