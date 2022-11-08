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

from typing import List
import unittest

from kfp.components import component_factory
from kfp.components import structures
from kfp.components.component_decorator import component
from kfp.components.types.type_annotations import OutputPath


class TestGetPackagesToInstallCommand(unittest.TestCase):

    def test_with_no_packages_to_install(self):
        packages_to_install = []

        command = component_factory._get_packages_to_install_command(
            packages_to_install)
        self.assertEqual(command, [])

    def test_with_packages_to_install_and_no_pip_index_url(self):
        packages_to_install = ['package1', 'package2']

        command = component_factory._get_packages_to_install_command(
            packages_to_install)
        concat_command = ' '.join(command)
        for package in packages_to_install:
            self.assertTrue(package in concat_command)

    def test_with_packages_to_install_with_pip_index_url(self):
        packages_to_install = ['package1', 'package2']
        pip_index_urls = ['https://myurl.org/simple']

        command = component_factory._get_packages_to_install_command(
            packages_to_install, pip_index_urls)
        concat_command = ' '.join(command)
        for package in packages_to_install + pip_index_urls:
            self.assertTrue(package in concat_command)


class TestInvalidParameterName(unittest.TestCase):

    def test_output_named_Output(self):

        with self.assertRaisesRegex(ValueError,
                                    r'"Output" is an invalid parameter name.'):

            @component
            def comp(Output: OutputPath(str)):
                pass

    def test_output_named_Output_with_string_output(self):

        with self.assertRaisesRegex(ValueError,
                                    r'"Output" is an invalid parameter name.'):

            @component
            def comp(Output: OutputPath(str), text: str) -> str:
                pass


from kfp.components.types.artifact_types import Artifact
from kfp.components.types.artifact_types import Model
from kfp.dsl import Input
from kfp.dsl import Output


class TestExtractComponentInterfaceListofArtifacts(unittest.TestCase):

    def test_python_component_fn(self):

        def comp(i: Input[List[Model]], o: Output[List[Artifact]]):
            ...

        component_spec = component_factory.extract_component_interface(comp)
        self.assertEqual(component_spec.name, 'comp')
        self.assertEqual(component_spec.description, None)
        self.assertEqual(
            component_spec.inputs, {
                'i':
                    structures.InputSpec(
                        type='system.Model@0.0.1',
                        default=None,
                        is_artifact_list=True)
            })
        self.assertEqual(
            component_spec.outputs, {
                'o':
                    structures.OutputSpec(
                        type='system.Artifact@0.0.1', is_artifact_list=True)
            })

    def test_custom_container_component_fn(self):

        def comp(i: Input[List[Artifact]], o: Output[List[Model]]):
            ...

        component_spec = component_factory.extract_component_interface(
            comp, containerized=True)
        self.assertEqual(component_spec.name, 'comp')
        self.assertEqual(component_spec.description, None)
        self.assertEqual(
            component_spec.inputs, {
                'i':
                    structures.InputSpec(
                        type='system.Artifact@0.0.1',
                        default=None,
                        is_artifact_list=True)
            })
        self.assertEqual(
            component_spec.outputs, {
                'o':
                    structures.OutputSpec(
                        type='system.Model@0.0.1', is_artifact_list=True)
            })

    def test_pipeline_fn(self):

        def comp(i: Input[List[Model]]) -> List[Artifact]:
            ...

        component_spec = component_factory.extract_component_interface(comp)
        self.assertEqual(component_spec.name, 'comp')
        self.assertEqual(component_spec.description, None)
        self.assertEqual(
            component_spec.inputs, {
                'i':
                    structures.InputSpec(
                        type='system.Model@0.0.1',
                        default=None,
                        is_artifact_list=True)
            })
        self.assertEqual(
            component_spec.outputs, {
                'Output':
                    structures.OutputSpec(
                        type='system.Artifact@0.0.1', is_artifact_list=True)
            })

    def test_pipeline_with_named_tuple_fn(self):
        from typing import NamedTuple

        def comp(
            i: Input[List[Model]]
        ) -> NamedTuple('outputs', [('list_of_artifacts', List[Artifact]),
                                    ('single_artifact', Model)]):
            ...

        component_spec = component_factory.extract_component_interface(comp)
        self.assertEqual(component_spec.name, 'comp')
        self.assertEqual(component_spec.description, None)
        self.assertEqual(
            component_spec.inputs, {
                'i':
                    structures.InputSpec(
                        type='system.Model@0.0.1',
                        default=None,
                        is_artifact_list=True)
            })
        self.assertEqual(
            component_spec.outputs, {
                'list_of_artifacts':
                    structures.OutputSpec(
                        type='system.Artifact@0.0.1', is_artifact_list=True),
                'single_artifact':
                    structures.OutputSpec(
                        type='system.Model@0.0.1', is_artifact_list=False)
            })


if __name__ == '__main__':
    unittest.main()
