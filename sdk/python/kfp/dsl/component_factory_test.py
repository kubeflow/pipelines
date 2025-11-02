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
from kfp.dsl import python_component
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


class TestKubeflowAutoDetection(unittest.TestCase):
    """Tests for AST-based kubeflow import detection and auto-installation."""

    def test_detect_kubeflow_imports_simple_import(self):
        """Test detection of 'import kubeflow'."""

        def component_with_kubeflow_import():
            return 'test'

        result = component_factory._detect_kubeflow_imports_in_function(
            component_with_kubeflow_import)
        self.assertTrue(result)

    def test_detect_kubeflow_imports_from_import(self):
        """Test detection of 'from kubeflow import X'."""

        def component_with_kubeflow_from_import():
            return 'test'

        result = component_factory._detect_kubeflow_imports_in_function(
            component_with_kubeflow_from_import)
        self.assertTrue(result)

    def test_detect_kubeflow_imports_submodule_import(self):
        """Test detection of 'import kubeflow.training'."""

        def component_with_kubeflow_submodule():
            return 'test'

        result = component_factory._detect_kubeflow_imports_in_function(
            component_with_kubeflow_submodule)
        self.assertTrue(result)

    def test_detect_kubeflow_imports_from_submodule(self):
        """Test detection of 'from kubeflow.training import X'."""

        def component_with_kubeflow_from_submodule():
            return 'test'

        result = component_factory._detect_kubeflow_imports_in_function(
            component_with_kubeflow_from_submodule)
        self.assertTrue(result)

    def test_detect_kubeflow_imports_star_import(self):
        """Test detection of 'from kubeflow.submodule import *'."""
        # Skip this test since star imports aren't allowed inside functions in Python
        # Our AST parser would detect it if it were valid syntax
        self.skipTest(
            'Star imports not allowed inside functions - syntax limitation')

    def test_detect_kubeflow_imports_no_imports(self):
        """Test no detection when kubeflow is not imported."""

        def component_without_kubeflow():
            return 'test'

        result = component_factory._detect_kubeflow_imports_in_function(
            component_without_kubeflow)
        self.assertFalse(result)

    def test_detect_kubeflow_imports_similar_name(self):
        """Test no false positives with similar package names."""

        def component_with_similar_name():
            return 'test'

        result = component_factory._detect_kubeflow_imports_in_function(
            component_with_similar_name)
        self.assertFalse(result)

    def test_detect_kubeflow_imports_handles_syntax_error(self):
        """Test graceful handling of syntax errors."""

        def malformed_function():
            pass

        import unittest.mock
        with unittest.mock.patch(
                'inspect.getsource', side_effect=SyntaxError('Invalid syntax')):
            result = component_factory._detect_kubeflow_imports_in_function(
                malformed_function)
            self.assertFalse(result)

    def test_parse_package_name_simple(self):
        """Test parsing simple package names."""
        result = component_factory._parse_package_name('kubeflow')
        self.assertEqual(result, 'kubeflow')

    def test_parse_package_name_with_version_operators(self):
        """Test parsing package names with version operators."""
        test_cases = [
            ('kubeflow==2.0.0', 'kubeflow'),
            ('kubeflow>=1.5.0', 'kubeflow'),
            ('kubeflow<=2.0', 'kubeflow'),
            ('kubeflow~=1.5', 'kubeflow'),
            ('kubeflow!=1.0', 'kubeflow'),
        ]

        for package_spec, expected in test_cases:
            with self.subTest(package_spec=package_spec):
                result = component_factory._parse_package_name(package_spec)
                self.assertEqual(result, expected)

    def test_parse_package_name_with_extras(self):
        """Test parsing package names with extras."""
        test_cases = [
            ('kubeflow[training]', 'kubeflow'),
            ('kubeflow[training,pipeline]', 'kubeflow'),
            ('kubeflow[training]==2.0.0', 'kubeflow'),
        ]

        for package_spec, expected in test_cases:
            with self.subTest(package_spec=package_spec):
                result = component_factory._parse_package_name(package_spec)
                self.assertEqual(result, expected)

    def test_parse_package_name_vcs_url(self):
        """Test parsing package names from VCS URLs."""
        test_cases = [
            ('git+https://github.com/kubeflow/sdk.git', 'kubeflow'),
            ('git+ssh://git@github.com/kubeflow/sdk.git', 'kubeflow'),
            ('git+https://github.com/some-org/my-package.git', 'my-package'),
        ]

        for package_spec, expected in test_cases:
            with self.subTest(package_spec=package_spec):
                result = component_factory._parse_package_name(package_spec)
                self.assertEqual(result, expected)

    def test_auto_installation_with_kubeflow_import(self):
        """Test that kubeflow is automatically added when detected."""

        def component_with_kubeflow():
            return 'test'

        command = component_factory._get_packages_to_install_command(
            func=component_with_kubeflow,
            packages_to_install=[],
            install_kubeflow_package=True)

        # Should contain kubeflow in the installation command
        command_str = ' '.join(command)
        self.assertIn('kubeflow', command_str)

    def test_auto_installation_disabled_by_flag(self):
        """Test that auto-installation can be disabled."""

        def component_with_kubeflow():
            return 'test'

        command = component_factory._get_packages_to_install_command(
            func=component_with_kubeflow,
            packages_to_install=[],
            install_kubeflow_package=False)

        # Should not contain kubeflow when disabled
        command_str = ' '.join(command)
        self.assertNotIn('kubeflow', command_str)

    def test_auto_installation_no_duplicate_when_already_specified(self):
        """Test that kubeflow is not duplicated if already in
        packages_to_install."""

        def component_with_kubeflow():
            return 'test'

        command = component_factory._get_packages_to_install_command(
            func=component_with_kubeflow,
            packages_to_install=['kubeflow==2.0.0'],
            install_kubeflow_package=True)

        # Should only appear once (the user-specified version)
        command_str = ' '.join(command)
        kubeflow_count = command_str.count('kubeflow')
        self.assertEqual(kubeflow_count, 1)
        self.assertIn('kubeflow==2.0.0', command_str)

    def test_auto_installation_respects_user_version(self):
        """Test that user-specified kubeflow version is respected."""

        def component_with_kubeflow():
            return 'test'

        command = component_factory._get_packages_to_install_command(
            func=component_with_kubeflow,
            packages_to_install=['kubeflow>=1.5.0', 'pandas'],
            install_kubeflow_package=True)

        command_str = ' '.join(command)
        # Should contain the user-specified version
        self.assertIn('kubeflow>=1.5.0', command_str)
        # Should not add a generic 'kubeflow'
        self.assertNotIn("'kubeflow'", command_str)

    def test_auto_installation_no_kubeflow_imports(self):
        """Test that kubeflow is not added when not imported."""

        def component_without_kubeflow():
            return 'test'

        command = component_factory._get_packages_to_install_command(
            func=component_without_kubeflow,
            packages_to_install=['pandas'],
            install_kubeflow_package=True)

        command_str = ' '.join(command)
        self.assertNotIn('kubeflow', command_str)

    def test_component_decorator_with_kubeflow_auto_install(self):
        """Test that @component decorator properly auto-installs kubeflow."""

        @dsl.component(base_image='python:3.9', install_kubeflow_package=True)
        def my_component() -> str:
            return 'test'

        # Check that the component was created successfully
        self.assertIsInstance(my_component, python_component.PythonComponent)

        # Check that kubeflow is in the installation command
        container_spec = my_component.component_spec.implementation.container
        command_str = ' '.join(container_spec.command)
        self.assertIn('kubeflow', command_str)

    def test_component_decorator_kubeflow_auto_install_disabled(self):
        """Test that @component decorator respects
        install_kubeflow_package=False."""

        @dsl.component(base_image='python:3.9', install_kubeflow_package=False)
        def my_component() -> str:
            return 'test'

        # Check that the component was created successfully
        self.assertIsInstance(my_component, python_component.PythonComponent)

        container_spec = my_component.component_spec.implementation.container
        command_str = ' '.join(container_spec.command)
        pip_install_has_kubeflow = "'kubeflow'" in command_str and 'pip install' in command_str
        self.assertFalse(pip_install_has_kubeflow,
                         'kubeflow should not be in pip install when disabled')

    def test_component_decorator_explicit_kubeflow_package(self):
        """Test that explicit kubeflow in packages_to_install is respected."""

        @dsl.component(
            base_image='python:3.9',
            packages_to_install=['kubeflow==2.0.0'],
            install_kubeflow_package=True)
        def my_component() -> str:
            return 'test'

        # Check that the component was created successfully
        self.assertIsInstance(my_component, python_component.PythonComponent)

        # Check that the explicit version is preserved
        container_spec = my_component.component_spec.implementation.container
        command_str = ' '.join(container_spec.command)
        self.assertIn('kubeflow==2.0.0', command_str)
        pip_kubeflow_count = command_str.count("'kubeflow==2.0.0'")
        pip_generic_kubeflow_count = command_str.count("'kubeflow'")
        self.assertEqual(pip_kubeflow_count, 1,
                         'Should have exactly one kubeflow==2.0.0')
        self.assertEqual(
            pip_generic_kubeflow_count, 0,
            'Should not have generic kubeflow when version specified')


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
                r"The default base_image used by the @dsl\.component decorator will switch from 'python:3\.11' to 'python:3\.12' on Oct 1, 2027\. To ensure your existing components work with versions of the KFP SDK released after that date, you should provide an explicit base_image argument and ensure your component works as intended on Python 3\.12\."
        ):

            @dsl.component
            def foo():
                pass


if __name__ == '__main__':
    unittest.main()
