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

import re
from typing import List
import unittest

from kfp import dsl
from kfp.dsl import component_factory
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl import structures
from kfp.dsl.component_decorator import component
from kfp.dsl.types.artifact_types import Artifact
from kfp.dsl.types.artifact_types import Model
from kfp.dsl.types.type_annotations import OutputPath


def strip_kfp_version(command: List[str]) -> List[str]:
    return [
        re.sub(r"'kfp==(\d+).(\d+).(\d+)(-[a-z]+.\d+)?'", 'kfp', c)
        for c in command
    ]


class TestGetPackagesToInstallCommand(unittest.TestCase):

    def test_with_no_user_packages_to_install(self):
        packages_to_install = []

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install)
        self.assertEqual(
            strip_kfp_version(command),
            strip_kfp_version([
                'sh', '-c',
                '\nif ! [ -x "$(command -v pip)" ]; then\n    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'kfp==2.1.2\' \'--no-deps\' \'typing-extensions>=3.7.4,<5; python_version<"3.9"\' && "$0" "$@"\n'
            ]))

    def test_with_no_user_packages_to_install_and_install_kfp_false(self):
        packages_to_install = []

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install,
            install_kfp_package=False,
        )
        self.assertEqual(command, [])

    def test_with_no_user_packages_to_install_and_kfp_package_path(self):
        packages_to_install = []

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install,
            kfp_package_path='git+https://github.com/kubeflow/pipelines.git@master#subdirectory=sdk/python'
        )

        self.assertEqual(
            strip_kfp_version(command),
            strip_kfp_version([
                'sh', '-c',
                '\nif ! [ -x "$(command -v pip)" ]; then\n    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'git+https://github.com/kubeflow/pipelines.git@master#subdirectory=sdk/python\' && "$0" "$@"\n'
            ]))

    def test_with_no_user_packages_to_install_and_kfp_package_path_and_install_kfp_false(
            self):
        packages_to_install = []

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install,
            kfp_package_path='git+https://github.com/kubeflow/pipelines.git@master#subdirectory=sdk/python',
            install_kfp_package=False,
        )
        self.assertEqual(command, [])

    def test_with_user_packages_to_install_and_kfp_package_path_and_install_kfp_false(
            self):
        packages_to_install = ['sklearn']

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install,
            kfp_package_path='git+https://github.com/kubeflow/pipelines.git@master#subdirectory=sdk/python',
            install_kfp_package=False,
        )

        self.assertEqual(
            strip_kfp_version(command),
            strip_kfp_version([
                'sh', '-c',
                '\nif ! [ -x "$(command -v pip)" ]; then\n    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'sklearn\' && "$0" "$@"\n'
            ]))

    def test_with_no_user_packages_to_install_and_kfp_package_path_and_target_image(
            self):
        packages_to_install = []

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install,
            target_image='gcr.io/my-kfp-image',
            kfp_package_path='./sdk/python')

        self.assertEqual(command, [])

    def test_with_no_user_packages_to_install_and_kfp_package_path_and_target_image_and_install_kfp_false(
            self):
        packages_to_install = []

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install,
            target_image='gcr.io/my-kfp-image',
            kfp_package_path='./sdk/python',
            install_kfp_package=False)

        self.assertEqual(command, [])

    def test_with_user_packages_to_install_and_no_pip_index_url(self):
        packages_to_install = ['package1', 'package2']

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install)

        self.assertEqual(
            strip_kfp_version(command),
            strip_kfp_version([
                'sh', '-c', '\n'
                'if ! [ -x "$(command -v pip)" ]; then\n'
                '    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install '
                'python3-pip\n'
                'fi\n'
                '\n'
                'PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet '
                "--no-warn-script-location 'package1' 'package2'  &&  python3 -m pip install "
                "--quiet --no-warn-script-location kfp '--no-deps' "
                '\'typing-extensions>=3.7.4,<5; python_version<"3.9"\' && "$0" "$@"\n'
            ]))

    def test_with_packages_to_install_with_pip_index_url(self):
        packages_to_install = ['package1', 'package2']
        pip_index_urls = ['https://myurl.org/simple']

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install,
            pip_index_urls=pip_index_urls,
        )

        self.assertEqual(
            strip_kfp_version(command),
            strip_kfp_version([
                'sh', '-c', '\n'
                'if ! [ -x "$(command -v pip)" ]; then\n'
                '    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install '
                'python3-pip\n'
                'fi\n'
                '\n'
                'PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet '
                '--no-warn-script-location --index-url https://myurl.org/simple '
                "--trusted-host https://myurl.org/simple 'package1' 'package2'  &&  python3 "
                '-m pip install --quiet --no-warn-script-location --index-url '
                'https://myurl.org/simple --trusted-host https://myurl.org/simple kfp '
                '\'--no-deps\' \'typing-extensions>=3.7.4,<5; python_version<"3.9"\' && "$0" '
                '"$@"\n'
            ]))

    def test_with_packages_to_install_with_pip_index_url_and_trusted_host(self):
        packages_to_install = ['package1', 'package2']
        pip_index_urls = ['https://myurl.org/simple']
        pip_trusted_hosts = ['myurl.org']

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install,
            pip_index_urls=pip_index_urls,
            pip_trusted_hosts=pip_trusted_hosts,
        )

        self.assertEqual(
            strip_kfp_version(command),
            strip_kfp_version([
                'sh', '-c', '\n'
                'if ! [ -x "$(command -v pip)" ]; then\n'
                '    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install '
                'python3-pip\n'
                'fi\n'
                '\n'
                'PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet '
                '--no-warn-script-location --index-url https://myurl.org/simple '
                "--trusted-host myurl.org 'package1' 'package2'  &&  python3 -m pip install "
                '--quiet --no-warn-script-location --index-url https://myurl.org/simple '
                "--trusted-host myurl.org kfp '--no-deps' 'typing-extensions>=3.7.4,<5; "
                'python_version<"3.9"\' && "$0" "$@"\n'
            ]))

    def test_with_packages_to_install_with_pip_index_url_and_empty_trusted_host(
            self):
        packages_to_install = ['package1', 'package2']
        pip_index_urls = ['https://myurl.org/simple']

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install,
            pip_index_urls=pip_index_urls,
            pip_trusted_hosts=[],
        )

        self.assertEqual(
            strip_kfp_version(command),
            strip_kfp_version([
                'sh', '-c', '\n'
                'if ! [ -x "$(command -v pip)" ]; then\n'
                '    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install '
                'python3-pip\n'
                'fi\n'
                '\n'
                'PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet '
                "--no-warn-script-location --index-url https://myurl.org/simple 'package1' "
                "'package2'  &&  python3 -m pip install --quiet --no-warn-script-location "
                "--index-url https://myurl.org/simple kfp '--no-deps' "
                '\'typing-extensions>=3.7.4,<5; python_version<"3.9"\' && "$0" "$@"\n'
            ]))


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


class TestExtractComponentInterfaceListofArtifacts(unittest.TestCase):

    def test_python_component_input(self):

        def comp(i: Input[List[Model]]):
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

    def test_custom_container_component_input(self):

        def comp(i: Input[List[Artifact]]):
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

    def test_pipeline_input(self):

        def comp(i: Input[List[Model]]):
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


class TestArtifactStringInInputpathOutputpath(unittest.TestCase):

    def test_unknown(self):

        @dsl.component
        def comp(
                i: dsl.InputPath('MyCustomType'),
                o: dsl.OutputPath('MyCustomType'),
        ):
            ...

        self.assertEqual(comp.component_spec.outputs['o'].type,
                         'system.Artifact@0.0.1')
        self.assertFalse(comp.component_spec.outputs['o'].is_artifact_list)
        self.assertEqual(comp.component_spec.inputs['i'].type,
                         'system.Artifact@0.0.1')
        self.assertFalse(comp.component_spec.inputs['i'].is_artifact_list)

    def test_known_v1_back_compat(self):

        @dsl.component
        def comp(
                i: dsl.InputPath('Dataset'),
                o: dsl.OutputPath('Dataset'),
        ):
            ...

        self.assertEqual(comp.component_spec.outputs['o'].type,
                         'system.Dataset@0.0.1')
        self.assertFalse(comp.component_spec.outputs['o'].is_artifact_list)
        self.assertEqual(comp.component_spec.inputs['i'].type,
                         'system.Dataset@0.0.1')
        self.assertFalse(comp.component_spec.inputs['i'].is_artifact_list)


class TestOutputListsOfArtifactsTemporarilyBlocked(unittest.TestCase):

    def test_python_component(self):
        with self.assertRaisesRegex(
                ValueError,
                r"Output lists of artifacts are only supported for pipelines\. Got output list of artifacts for output parameter 'output_list' of component 'comp'\."
        ):

            @dsl.component
            def comp(output_list: Output[List[Artifact]]):
                ...

    def test_container_component(self):
        with self.assertRaisesRegex(
                ValueError,
                r"Output lists of artifacts are only supported for pipelines\. Got output list of artifacts for output parameter 'output_list' of component 'comp'\."
        ):

            @dsl.container_component
            def comp(output_list: Output[List[Artifact]]):
                return dsl.ContainerSpec(image='alpine')


class TestPythonEOLWarning(unittest.TestCase):

    def test_default_base_image(self):

        with self.assertWarnsRegex(
                FutureWarning,
                r"The default base_image used by the @dsl\.component decorator will switch from 'python:3\.9' to 'python:3\.10' on Oct 1, 2025\. To ensure your existing components work with versions of the KFP SDK released after that date, you should provide an explicit base_image argument and ensure your component works as intended on Python 3\.10\."
        ):

            @dsl.component
            def foo():
                pass


if __name__ == '__main__':
    unittest.main()
