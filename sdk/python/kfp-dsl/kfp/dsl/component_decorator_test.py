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

import os
import tempfile
from typing import Dict, List, NamedTuple
import unittest

from kfp.dsl import python_component
from kfp.dsl import structures
from kfp.dsl.component_decorator import component


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

        @component
        def comp(text: str) -> str:
            return text

        self.assertIsInstance(comp, python_component.PythonComponent)

    def test_some_args(self):

        @component(base_image='python:3.9')
        def comp(text: str) -> str:
            return text

        self.assertIsInstance(comp, python_component.PythonComponent)

    def test_packages_to_install(self):

        @component(packages_to_install=['numpy', 'tensorflow'])
        def comp(text: str) -> str:
            return text

        self.assertIsInstance(comp, python_component.PythonComponent)

        concat_command = ' '.join(
            comp.component_spec.implementation.container.command)
        self.assertTrue('numpy' in concat_command and
                        'tensorflow' in concat_command)

    def test_packages_to_install_with_custom_index_url(self):

        @component(
            packages_to_install=['numpy', 'tensorflow'],
            pip_index_urls=['https://pypi.org/simple'])
        def comp(text: str) -> str:
            return text

        self.assertIsInstance(comp, python_component.PythonComponent)

        concat_command = ' '.join(
            comp.component_spec.implementation.container.command)
        self.assertTrue('numpy' in concat_command and
                        'tensorflow' in concat_command)
        self.assertTrue('https://pypi.org/simple' in concat_command)

    def test_output_component_file_parameter(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            filepath = os.path.join(tmpdir, 'my_component.yaml')
            with self.assertWarnsRegex(
                    DeprecationWarning,
                    r'output_component_file parameter is deprecated and will eventually be removed\.'
            ):

                @component(output_component_file=filepath)
                def comp(text: str) -> str:
                    return text

            self.assertIsInstance(comp, python_component.PythonComponent)
            self.assertTrue(os.path.exists(filepath))
            with open(filepath, 'r') as f:
                yaml_text = f.read()

        component_spec = structures.ComponentSpec.from_yaml_documents(yaml_text)
        self.assertEqual(component_spec.name, comp.component_spec.name)

    def test_output_named_tuple_with_dict(self):

        @component
        def comp(
                text: str) -> NamedTuple('outputs', [('data', Dict[str, str])]):
            outputs = NamedTuple('outputs', [('data', Dict[str, str])])
            return outputs(data={text: text})

        # TODO: ideally should be the canonical type string, rather than the specific annotation as string, but both work
        self.assertEqual(comp.component_spec.outputs['data'].type,
                         'typing.Dict[str, str]')

    def test_output_dict(self):

        @component
        def comp(text: str) -> Dict[str, str]:
            return {text: text}

        # TODO: ideally should be the canonical type string, rather than the specific annotation as string, but both work
        self.assertEqual(comp.component_spec.outputs['Output'].type,
                         'typing.Dict[str, str]')

    def test_output_named_tuple_with_list(self):

        @component
        def comp(text: str) -> NamedTuple('outputs', [('data', List[str])]):
            outputs = NamedTuple('outputs', [('data', List[str])])
            return outputs(data={text: text})

        # TODO: ideally should be the canonical type string, rather than the specific annotation as string, but both work
        self.assertEqual(comp.component_spec.outputs['data'].type,
                         'typing.List[str]')

    def test_output_list(self):

        @component
        def comp(text: str) -> List[str]:
            return {text: text}

        # TODO: ideally should be the canonical type string, rather than the specific annotation as string, but both work
        self.assertEqual(comp.component_spec.outputs['Output'].type,
                         'typing.List[str]')
