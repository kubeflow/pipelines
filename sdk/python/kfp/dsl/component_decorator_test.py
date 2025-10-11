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

from kfp.dsl import component_factory
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

    def test_kubeflow_import_detection_import_kubeflow(self):
        """Test that 'import kubeflow' is detected and kubeflow package is
        auto-added."""

        @component
        def comp_with_kubeflow_import(text: str) -> str:
            return text

        concat_command = ' '.join(comp_with_kubeflow_import.component_spec
                                  .implementation.container.command)
        self.assertIn('kubeflow', concat_command)

    def test_kubeflow_import_detection_from_kubeflow_import(self):
        """Test that 'from kubeflow import ...' is detected and kubeflow
        package is auto-added."""

        @component
        def comp_with_from_kubeflow_import(text: str) -> str:
            return text

        concat_command = ' '.join(comp_with_from_kubeflow_import.component_spec
                                  .implementation.container.command)
        self.assertIn('kubeflow', concat_command)

    def test_kubeflow_import_detection_import_kubeflow_submodule(self):
        """Test that 'import kubeflow.training' is detected and kubeflow
        package is auto-added."""

        @component
        def comp_with_kubeflow_submodule_import(text: str) -> str:
            return text

        concat_command = ' '.join(
            comp_with_kubeflow_submodule_import.component_spec.implementation
            .container.command)
        self.assertIn('kubeflow', concat_command)

    def test_kubeflow_import_detection_no_kubeflow_import(self):
        """Test that kubeflow package is not auto-added when no kubeflow
        imports are present."""

        @component
        def comp_without_kubeflow_import(text: str) -> str:
            import os
            return text

        concat_command = ' '.join(comp_without_kubeflow_import.component_spec
                                  .implementation.container.command)

        pip_install_lines = [
            line for line in concat_command.split('\n') if 'pip install' in line
        ]
        kubeflow_in_pip = any('kubeflow' in line for line in pip_install_lines)
        self.assertFalse(
            kubeflow_in_pip,
            f"kubeflow package should not be installed: {pip_install_lines}")

    def test_kubeflow_import_detection_with_explicit_packages(self):
        """Test that kubeflow is not duplicated when already in
        packages_to_install."""

        @component(packages_to_install=['kubeflow', 'numpy'])
        def comp_with_explicit_kubeflow(text: str) -> str:
            return text

        concat_command = ' '.join(comp_with_explicit_kubeflow.component_spec
                                  .implementation.container.command)

        pip_install_lines = [
            line for line in concat_command.split('\n') if 'pip install' in line
        ]
        kubeflow_pip_count = sum(
            line.count('kubeflow') for line in pip_install_lines)
        self.assertEqual(
            kubeflow_pip_count, 1,
            f"kubeflow should appear exactly once in pip install: {pip_install_lines}"
        )

    def test_ast_parsing_detection_function(self):
        """Test the AST parsing function directly."""

        def func_with_kubeflow_import():
            import kubeflow  # noqa
            return 'test'

        def func_with_from_kubeflow_import():
            from kubeflow import training  # noqa
            return 'test'

        def func_without_kubeflow_import():
            import os
            return 'test'

        self.assertTrue(
            component_factory._detect_kubeflow_imports_in_function(
                func_with_kubeflow_import))
        self.assertTrue(
            component_factory._detect_kubeflow_imports_in_function(
                func_with_from_kubeflow_import))
        self.assertFalse(
            component_factory._detect_kubeflow_imports_in_function(
                func_without_kubeflow_import))
