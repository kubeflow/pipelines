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

import ast
import re
import textwrap
from typing import List
import unittest

from kfp import dsl
from kfp.dsl import component_factory
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl import python_component
from kfp.dsl import structures
from kfp.dsl.component_decorator import component
from kfp.dsl.types import type_utils
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


class TestBuiltinGenericParameterAnnotations(unittest.TestCase):

    def test_builtin_list_parameter_annotations(self):

        @dsl.component
        def comp(nums: list[int]) -> list[int]:
            return nums

        spec = comp.component_spec
        self.assertEqual(
            type_utils.get_canonical_name_for_outer_generic(
                spec.inputs['nums'].type), 'List')
        self.assertFalse(spec.inputs['nums'].is_artifact_list)
        self.assertEqual(
            type_utils.get_canonical_name_for_outer_generic(
                spec.outputs['Output'].type), 'List')
        self.assertFalse(spec.outputs['Output'].is_artifact_list)

    def test_builtin_dict_parameter_annotations(self):

        @dsl.component
        def comp(mapping: dict[str, int]) -> dict[str, int]:
            return mapping

        spec = comp.component_spec
        self.assertEqual(
            type_utils.get_canonical_name_for_outer_generic(
                spec.inputs['mapping'].type), 'Dict')
        self.assertFalse(spec.inputs['mapping'].is_artifact_list)
        self.assertEqual(
            type_utils.get_canonical_name_for_outer_generic(
                spec.outputs['Output'].type), 'Dict')
        self.assertFalse(spec.outputs['Output'].is_artifact_list)


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

    def test_detect_kubeflow_imports_patterns(self):
        """Test detection of various kubeflow import patterns."""
        test_cases = [
            ('import kubeflow', 'simple import'),
            ('import kubeflow.training', 'submodule import'),
            ('from kubeflow import training', 'from import'),
            ('from kubeflow.training import TorchJobClient',
             'from submodule import'),
        ]

        for import_statement, description in test_cases:
            with self.subTest(pattern=description):
                source = textwrap.dedent(f'''
                    def component():
                        {import_statement}
                        return 'test'
                ''')

                tree = ast.parse(source)
                has_kubeflow = False
                for node in ast.walk(tree):
                    if isinstance(node, ast.Import):
                        for alias in node.names:
                            if alias.name == 'kubeflow' or alias.name.startswith(
                                    'kubeflow.'):
                                has_kubeflow = True
                                break
                    elif isinstance(node, ast.ImportFrom):
                        if node.module and (
                                node.module == 'kubeflow' or
                                node.module.startswith('kubeflow.')):
                            has_kubeflow = True
                            break

                self.assertTrue(has_kubeflow,
                                f'Failed to detect: {import_statement}')

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

    def test_is_kubeflow_package_simple(self):
        """Test detection of simple kubeflow package."""
        self.assertTrue(component_factory._is_kubeflow_package('kubeflow'))

    def test_is_kubeflow_package_with_version_operators(self):
        """Test detection of kubeflow with version operators."""
        test_cases = [
            'kubeflow==2.0.0',
            'kubeflow>=1.5.0',
            'kubeflow<=2.0',
            'kubeflow~=1.5',
            'kubeflow!=1.0',
        ]

        for package_spec in test_cases:
            with self.subTest(package_spec=package_spec):
                self.assertTrue(
                    component_factory._is_kubeflow_package(package_spec),
                    f'{package_spec} should be detected as kubeflow')

    def test_is_kubeflow_package_with_extras(self):
        """Test detection of kubeflow with extras."""
        test_cases = [
            'kubeflow[training]',
            'kubeflow[training,pipeline]',
            'kubeflow[training]==2.0.0',
        ]

        for package_spec in test_cases:
            with self.subTest(package_spec=package_spec):
                self.assertTrue(
                    component_factory._is_kubeflow_package(package_spec),
                    f'{package_spec} should be detected as kubeflow')

    def test_is_kubeflow_package_vcs_url(self):
        """Test detection of kubeflow from VCS URLs."""
        test_cases = [
            'git+https://github.com/kubeflow/sdk.git',
            'git+ssh://git@github.com/kubeflow/sdk.git',
            'git+https://github.com/kubeflow/training.git@main',
        ]

        for package_spec in test_cases:
            with self.subTest(package_spec=package_spec):
                self.assertTrue(
                    component_factory._is_kubeflow_package(package_spec),
                    f'{package_spec} should be detected as kubeflow')

    def test_is_kubeflow_package_negative_cases(self):
        """Test that non-kubeflow packages are not detected."""
        test_cases = [
            'pandas',
            'kubeflow-server',  # Must be exact match
            'my-kubeflow',  # Must start with kubeflow
            'numpy>=1.0',
            'git+https://github.com/org/my-package.git',
        ]

        for package_spec in test_cases:
            with self.subTest(package_spec=package_spec):
                self.assertFalse(
                    component_factory._is_kubeflow_package(package_spec),
                    f'{package_spec} should NOT be detected as kubeflow')

    def test_auto_installation_with_kubeflow_import(self):
        """Test that kubeflow is automatically added when detected."""

        # Use a mock function that simulates having kubeflow imports
        def mock_func():
            pass

        # Mock the detection function to return True
        import unittest.mock
        with unittest.mock.patch(
                'kfp.dsl.component_factory._detect_kubeflow_imports_in_function',
                return_value=True):
            command = component_factory._get_packages_to_install_command(
                func=mock_func,
                packages_to_install=[],
                install_kubeflow_package=component_factory
                .KubeflowPackageInstallMode.AUTO)

            # Should contain kubeflow in the installation command
            command_str = ' '.join(command)
            self.assertIn('kubeflow', command_str)

    def test_auto_installation_disabled_by_flag(self):
        """Test that auto-installation can be disabled."""

        def mock_func():
            pass

        import unittest.mock
        with unittest.mock.patch(
                'kfp.dsl.component_factory._detect_kubeflow_imports_in_function',
                return_value=True):
            command = component_factory._get_packages_to_install_command(
                func=mock_func,
                packages_to_install=[],
                install_kubeflow_package=component_factory
                .KubeflowPackageInstallMode.SKIP)

            # Should not contain kubeflow when disabled
            command_str = ' '.join(command)
            self.assertNotIn('kubeflow', command_str)

    def test_auto_installation_no_duplicate_when_already_specified(self):
        """Test that kubeflow is not duplicated if already in
        packages_to_install."""

        def mock_func():
            pass

        import unittest.mock
        with unittest.mock.patch(
                'kfp.dsl.component_factory._detect_kubeflow_imports_in_function',
                return_value=True):
            command = component_factory._get_packages_to_install_command(
                func=mock_func,
                packages_to_install=['kubeflow==2.0.0'],
                install_kubeflow_package=component_factory
                .KubeflowPackageInstallMode.AUTO)

            # Should only appear once (the user-specified version)
            command_str = ' '.join(command)
            kubeflow_count = command_str.count('kubeflow')
            self.assertEqual(kubeflow_count, 1)
            self.assertIn('kubeflow==2.0.0', command_str)

    def test_auto_installation_respects_user_version(self):
        """Test that user-specified kubeflow version is respected."""

        def mock_func():
            pass

        import unittest.mock
        with unittest.mock.patch(
                'kfp.dsl.component_factory._detect_kubeflow_imports_in_function',
                return_value=True):
            command = component_factory._get_packages_to_install_command(
                func=mock_func,
                packages_to_install=['kubeflow>=1.5.0', 'pandas'],
                install_kubeflow_package=component_factory
                .KubeflowPackageInstallMode.AUTO)

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
            install_kubeflow_package=component_factory
            .KubeflowPackageInstallMode.AUTO)

        command_str = ' '.join(command)
        self.assertNotIn('kubeflow', command_str)

    def test_install_mode_always_installs(self):
        """Test that INSTALL mode always installs kubeflow."""

        def component_without_kubeflow():
            return 'test'

        command = component_factory._get_packages_to_install_command(
            func=component_without_kubeflow,
            packages_to_install=[],
            install_kubeflow_package=component_factory
            .KubeflowPackageInstallMode.INSTALL)

        command_str = ' '.join(command)
        self.assertIn('kubeflow', command_str)

    def test_component_decorator_with_kubeflow_auto_install(self):
        """Test that @component decorator properly auto-installs kubeflow."""

        @dsl.component(
            base_image='python:3.9',
            install_kubeflow_package=component_factory
            .KubeflowPackageInstallMode.AUTO)
        def my_component() -> str:
            return 'test'

        # Check that the component was created successfully
        self.assertIsInstance(my_component, python_component.PythonComponent)

        # Since the component doesn't actually import kubeflow, it shouldn't be installed
        container_spec = my_component.component_spec.implementation.container
        command_str = ' '.join(container_spec.command)
        # With AUTO mode and no kubeflow imports, it should not install kubeflow
        pip_install_has_kubeflow = "'kubeflow'" in command_str and 'pip install' in command_str
        self.assertFalse(pip_install_has_kubeflow)

    def test_component_decorator_kubeflow_auto_install_disabled(self):
        """Test that @component decorator respects
        install_kubeflow_package=SKIP."""

        @dsl.component(
            base_image='python:3.9',
            install_kubeflow_package=component_factory
            .KubeflowPackageInstallMode.SKIP)
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
            install_kubeflow_package=component_factory
            .KubeflowPackageInstallMode.AUTO)
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
