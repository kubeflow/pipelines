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
import tarfile
import unittest
from tempfile import TemporaryDirectory
from typing import List

from kfp import dsl
from kfp.dsl import Input, Output, component_factory, structures
from kfp.dsl.component_decorator import component
from kfp.dsl.types import type_utils
from kfp.dsl.types.artifact_types import Artifact, Model
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
            packages_to_install=packages_to_install
        )
        self.assertEqual(
            strip_kfp_version(command),
            strip_kfp_version(
                [
                    'sh',
                    '-c',
                    '\nif ! [ -x "$(command -v pip)" ]; then\n    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'kfp==2.1.2\' \'--no-deps\' \'typing-extensions>=3.7.4,<5; python_version<"3.9"\' && "$0" "$@"\n',
                ]
            ),
        )

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
            kfp_package_path='git+https://github.com/kubeflow/pipelines.git@main#subdirectory=sdk/python',
        )

        self.assertEqual(
            strip_kfp_version(command),
            strip_kfp_version(
                [
                    'sh',
                    '-c',
                    '\nif ! [ -x "$(command -v pip)" ]; then\n    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'git+https://github.com/kubeflow/pipelines.git@main#subdirectory=sdk/python\' && "$0" "$@"\n',
                ]
            ),
        )

    def test_with_no_user_packages_to_install_and_kfp_package_path_and_install_kfp_false(
        self,
    ):
        packages_to_install = []

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install,
            kfp_package_path='git+https://github.com/kubeflow/pipelines.git@main#subdirectory=sdk/python',
            install_kfp_package=False,
        )
        self.assertEqual(command, [])

    def test_with_local_kfp_package_path_uses_no_deps(self):
        with TemporaryDirectory() as kfp_source:
            command = component_factory._get_packages_to_install_command(
                packages_to_install=[],
                kfp_package_path=kfp_source,
            )

        self.assertIn(f"'{kfp_source}' '--no-deps'", command[2])

    def test_with_pull_request_kfp_package_path_uses_no_deps(self):
        kfp_source = 'git+https://github.com/kubeflow/pipelines@refs/pull/13665/merge#egg=kfp&subdirectory=sdk/python'

        command = component_factory._get_packages_to_install_command(
            packages_to_install=[],
            kfp_package_path=kfp_source,
        )

        self.assertIn(f"'{kfp_source}' '--no-deps'", command[2])

    def test_with_user_packages_to_install_and_kfp_package_path_and_install_kfp_false(
        self,
    ):
        packages_to_install = ['sklearn']

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install,
            kfp_package_path='git+https://github.com/kubeflow/pipelines.git@main#subdirectory=sdk/python',
            install_kfp_package=False,
        )

        self.assertEqual(
            strip_kfp_version(command),
            strip_kfp_version(
                [
                    'sh',
                    '-c',
                    '\nif ! [ -x "$(command -v pip)" ]; then\n    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'sklearn\' && "$0" "$@"\n',
                ]
            ),
        )

    def test_with_no_user_packages_to_install_and_kfp_package_path_and_target_image(
        self,
    ):
        packages_to_install = []

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install,
            target_image='gcr.io/my-kfp-image',
            kfp_package_path='./sdk/python',
        )

        self.assertEqual(command, [])

    def test_with_no_user_packages_to_install_and_kfp_package_path_and_target_image_and_install_kfp_false(
        self,
    ):
        packages_to_install = []

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install,
            target_image='gcr.io/my-kfp-image',
            kfp_package_path='./sdk/python',
            install_kfp_package=False,
        )

        self.assertEqual(command, [])

    def test_with_user_packages_to_install_and_no_pip_index_url(self):
        packages_to_install = ['package1', 'package2']

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install
        )

        self.assertEqual(
            strip_kfp_version(command),
            strip_kfp_version(
                [
                    'sh',
                    '-c',
                    '\n'
                    'if ! [ -x "$(command -v pip)" ]; then\n'
                    '    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install '
                    'python3-pip\n'
                    'fi\n'
                    '\n'
                    'PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet '
                    "--no-warn-script-location 'package1' 'package2'  &&  python3 -m pip install "
                    "--quiet --no-warn-script-location kfp '--no-deps' "
                    '\'typing-extensions>=3.7.4,<5; python_version<"3.9"\' && "$0" "$@"\n',
                ]
            ),
        )

    def test_with_packages_to_install_with_pip_index_url(self):
        packages_to_install = ['package1', 'package2']
        pip_index_urls = ['https://myurl.org/simple']

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install,
            pip_index_urls=pip_index_urls,
        )

        self.assertEqual(
            strip_kfp_version(command),
            strip_kfp_version(
                [
                    'sh',
                    '-c',
                    '\n'
                    'if ! [ -x "$(command -v pip)" ]; then\n'
                    '    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install '
                    'python3-pip\n'
                    'fi\n'
                    '\n'
                    'PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet '
                    '--no-warn-script-location --index-url https://myurl.org/simple '
                    "'package1' 'package2'  &&  python3 "
                    '-m pip install --quiet --no-warn-script-location --index-url '
                    'https://myurl.org/simple kfp '
                    '\'--no-deps\' \'typing-extensions>=3.7.4,<5; python_version<"3.9"\' && "$0" '
                    '"$@"\n',
                ]
            ),
        )

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
            strip_kfp_version(
                [
                    'sh',
                    '-c',
                    '\n'
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
                    'python_version<"3.9"\' && "$0" "$@"\n',
                ]
            ),
        )

    def test_with_packages_to_install_with_pip_index_url_and_empty_trusted_host(
        self,
    ):
        packages_to_install = ['package1', 'package2']
        pip_index_urls = ['https://myurl.org/simple']

        command = component_factory._get_packages_to_install_command(
            packages_to_install=packages_to_install,
            pip_index_urls=pip_index_urls,
            pip_trusted_hosts=[],
        )

        self.assertEqual(
            strip_kfp_version(command),
            strip_kfp_version(
                [
                    'sh',
                    '-c',
                    '\n'
                    'if ! [ -x "$(command -v pip)" ]; then\n'
                    '    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install '
                    'python3-pip\n'
                    'fi\n'
                    '\n'
                    'PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet '
                    "--no-warn-script-location --index-url https://myurl.org/simple 'package1' "
                    "'package2'  &&  python3 -m pip install --quiet --no-warn-script-location "
                    "--index-url https://myurl.org/simple kfp '--no-deps' "
                    '\'typing-extensions>=3.7.4,<5; python_version<"3.9"\' && "$0" "$@"\n',
                ]
            ),
        )


class TestInvalidParameterName(unittest.TestCase):
    def test_output_named_Output(self):

        with self.assertRaisesRegex(
            ValueError, r'"Output" is an invalid parameter name.'
        ):

            @component
            def comp(Output: OutputPath(str)):
                pass

    def test_output_named_Output_with_string_output(self):

        with self.assertRaisesRegex(
            ValueError, r'"Output" is an invalid parameter name.'
        ):

            @component
            def comp(Output: OutputPath(str), text: str) -> str:
                pass


class TestExtractComponentInterfaceListofArtifacts(unittest.TestCase):
    def test_python_component_input(self):

        def comp(i: Input[List[Model]]): ...

        component_spec = component_factory.extract_component_interface(comp)
        self.assertEqual(component_spec.name, 'comp')
        self.assertEqual(component_spec.description, None)
        self.assertEqual(
            component_spec.inputs,
            {
                'i': structures.InputSpec(
                    type='system.Model@0.0.1',
                    default=None,
                    is_artifact_list=True,
                )
            },
        )

    def test_custom_container_component_input(self):

        def comp(i: Input[List[Artifact]]): ...

        component_spec = component_factory.extract_component_interface(
            comp, containerized=True
        )
        self.assertEqual(component_spec.name, 'comp')
        self.assertEqual(component_spec.description, None)
        self.assertEqual(
            component_spec.inputs,
            {
                'i': structures.InputSpec(
                    type='system.Artifact@0.0.1',
                    default=None,
                    is_artifact_list=True,
                )
            },
        )

    def test_pipeline_input(self):

        def comp(i: Input[List[Model]]): ...

        component_spec = component_factory.extract_component_interface(comp)
        self.assertEqual(component_spec.name, 'comp')
        self.assertEqual(component_spec.description, None)
        self.assertEqual(
            component_spec.inputs,
            {
                'i': structures.InputSpec(
                    type='system.Model@0.0.1',
                    default=None,
                    is_artifact_list=True,
                )
            },
        )


class TestBuiltinGenericParameterAnnotations(unittest.TestCase):
    def test_builtin_list_parameter_annotations(self):

        @dsl.component
        def comp(nums: list[int]) -> list[int]:
            return nums

        spec = comp.component_spec
        self.assertEqual(
            type_utils.get_canonical_name_for_outer_generic(
                spec.inputs['nums'].type
            ),
            'List',
        )
        self.assertFalse(spec.inputs['nums'].is_artifact_list)
        self.assertEqual(
            type_utils.get_canonical_name_for_outer_generic(
                spec.outputs['Output'].type
            ),
            'List',
        )
        self.assertFalse(spec.outputs['Output'].is_artifact_list)

    def test_builtin_dict_parameter_annotations(self):

        @dsl.component
        def comp(mapping: dict[str, int]) -> dict[str, int]:
            return mapping

        spec = comp.component_spec
        self.assertEqual(
            type_utils.get_canonical_name_for_outer_generic(
                spec.inputs['mapping'].type
            ),
            'Dict',
        )
        self.assertFalse(spec.inputs['mapping'].is_artifact_list)
        self.assertEqual(
            type_utils.get_canonical_name_for_outer_generic(
                spec.outputs['Output'].type
            ),
            'Dict',
        )
        self.assertFalse(spec.outputs['Output'].is_artifact_list)


class TestArtifactStringInInputpathOutputpath(unittest.TestCase):
    def test_unknown(self):

        @dsl.component
        def comp(
            i: dsl.InputPath('MyCustomType'),
            o: dsl.OutputPath('MyCustomType'),
        ): ...

        self.assertEqual(
            comp.component_spec.outputs['o'].type, 'system.Artifact@0.0.1'
        )
        self.assertFalse(comp.component_spec.outputs['o'].is_artifact_list)
        self.assertEqual(
            comp.component_spec.inputs['i'].type, 'system.Artifact@0.0.1'
        )
        self.assertFalse(comp.component_spec.inputs['i'].is_artifact_list)

    def test_sdk_artifact_name(self):

        @dsl.component
        def comp(
            i: dsl.InputPath('Dataset'),
            o: dsl.OutputPath('Dataset'),
        ): ...

        self.assertEqual(
            comp.component_spec.outputs['o'].type, 'system.Dataset@0.0.1'
        )
        self.assertFalse(comp.component_spec.outputs['o'].is_artifact_list)
        self.assertEqual(
            comp.component_spec.inputs['i'].type, 'system.Dataset@0.0.1'
        )
        self.assertFalse(comp.component_spec.inputs['i'].is_artifact_list)


class TestOutputListsOfArtifactsTemporarilyBlocked(unittest.TestCase):
    def test_python_component(self):
        with self.assertRaisesRegex(
            ValueError,
            r"Output lists of artifacts are only supported for pipelines\. Got output list of artifacts for output parameter 'output_list' of component 'comp'\.",
        ):

            @dsl.component
            def comp(output_list: Output[List[Artifact]]): ...

    def test_container_component(self):
        with self.assertRaisesRegex(
            ValueError,
            r"Output lists of artifacts are only supported for pipelines\. Got output list of artifacts for output parameter 'output_list' of component 'comp'\.",
        ):

            @dsl.container_component
            def comp(output_list: Output[List[Artifact]]):
                return dsl.ContainerSpec(image='alpine')


class TestPythonEOLWarning(unittest.TestCase):
    def test_default_base_image(self):

        with self.assertWarnsRegex(
            FutureWarning,
            r"The default base_image used by the @dsl\.component decorator will switch from 'python:3\.11' to 'python:3\.12' on Oct 1, 2027\. To ensure your existing components work with versions of the KFP SDK released after that date, you should provide an explicit base_image argument and ensure your component works as intended on Python 3\.12\.",
        ):

            @dsl.component
            def foo():
                pass


def _get_compressed_archive(component_fn) -> bytes:
    """Extract the gzip bytes of a compiled component's embedded archive."""
    import base64

    source = component_fn.component_spec.implementation.container.command[-1]
    match = re.search(r"__KFP_EMBEDDED_ARCHIVE_B64\s*=\s*'([^']+)'", source)
    if not match:
        raise ValueError(
            'Could not locate __KFP_EMBEDDED_ARCHIVE_B64 in compiled '
            'component source.'
        )
    archive_b64 = match.group(1)

    return base64.b64decode(archive_b64.encode('ascii'))


def _get_archive(component_fn) -> bytes:
    """Extract the raw tar bytes from a compiled component's embedded
    archive."""
    import gzip

    return gzip.decompress(_get_compressed_archive(component_fn))


class TestEmbeddedArtifactSymlinkResolution(unittest.TestCase):
    """Tests for #13533: embedded artifacts should resolve symlinks."""

    def test_symlinked_file_is_dereferenced_in_archive(self):
        """A symlinked file should be archived as a regular file, not a
        symlink."""
        import io
        import os
        import tarfile
        import tempfile

        with tempfile.TemporaryDirectory() as tmpdir:
            real_dir = os.path.join(tmpdir, 'real')
            os.makedirs(real_dir)
            real_file = os.path.join(real_dir, 'data.txt')
            with open(real_file, 'w') as f:
                f.write('real content')

            link_file = os.path.join(tmpdir, 'link.txt')
            try:
                os.symlink(real_file, link_file)
            except OSError:
                self.skipTest('Cannot create symlinks on this platform')

            @component(
                base_image='python:3.11',
                embedded_artifact_path=link_file,
                install_kfp_package=False,
            )
            def my_comp(cfg: dsl.EmbeddedInput[dsl.Dataset]):
                pass

            raw = _get_archive(my_comp)
            with tarfile.open(fileobj=io.BytesIO(raw), mode='r') as tar:
                members = tar.getmembers()
                self.assertEqual(len(members), 1)
                member = members[0]
                self.assertFalse(
                    member.issym(), 'Symlink was not dereferenced in archive'
                )
                self.assertFalse(
                    member.islnk(), 'Hardlink was not dereferenced in archive'
                )
                self.assertTrue(member.isfile())
                with tar.extractfile(member) as extracted:
                    self.assertIsNotNone(extracted)
                    self.assertEqual(extracted.read().decode(), 'real content')

    def test_directory_with_symlinked_file_is_dereferenced(self):
        """When a directory is embedded and it contains a symlinked file whose
        target is inside the artifact tree, the archive should contain the real
        file content."""
        import io
        import os
        import tarfile
        import tempfile

        with tempfile.TemporaryDirectory() as tmpdir:
            embed_dir = os.path.join(tmpdir, 'assets')
            os.makedirs(embed_dir)

            real_file = os.path.join(embed_dir, 'original.txt')
            with open(real_file, 'w') as f:
                f.write('original content')

            link_in_dir = os.path.join(embed_dir, 'linked.txt')
            try:
                os.symlink(real_file, link_in_dir)
            except OSError:
                self.skipTest('Cannot create symlinks on this platform')

            @component(
                base_image='python:3.11',
                embedded_artifact_path=embed_dir,
                install_kfp_package=False,
            )
            def my_comp(cfg: dsl.EmbeddedInput[dsl.Dataset]):
                pass

            raw = _get_archive(my_comp)
            with tarfile.open(fileobj=io.BytesIO(raw), mode='r') as tar:
                members = tar.getmembers()
                for member in members:
                    self.assertFalse(
                        member.issym(),
                        f'Member {member.name} is still a symlink',
                    )
                    self.assertFalse(
                        member.islnk(),
                        f'Member {member.name} is still a hardlink',
                    )
                    self.assertTrue(
                        member.isfile() or member.isdir(),
                        f'Member {member.name} has unexpected type',
                    )
                linked_member = next(
                    (m for m in members if m.name.endswith('linked.txt')), None
                )
                self.assertIsNotNone(
                    linked_member, 'linked.txt not found in archive'
                )
                with tar.extractfile(linked_member) as extracted:
                    self.assertIsNotNone(extracted)
                    self.assertEqual(
                        extracted.read().decode(), 'original content'
                    )

    def test_directory_symlink_is_rejected(self):
        """A directory symlink should be rejected."""
        import os
        import tempfile

        with tempfile.TemporaryDirectory() as tmpdir:
            embed_dir = os.path.join(tmpdir, 'assets')
            os.makedirs(embed_dir)
            with open(os.path.join(embed_dir, 'file.txt'), 'w') as f:
                f.write('hello')
            cycle_link = os.path.join(embed_dir, 'loop')
            try:
                os.symlink(embed_dir, cycle_link)
            except OSError:
                self.skipTest('Cannot create symlinks on this platform')

            with self.assertRaisesRegex(ValueError, r'[Dd]irectory symlinks'):

                @component(
                    base_image='python:3.11',
                    embedded_artifact_path=embed_dir,
                    install_kfp_package=False,
                )
                def my_comp(cfg: dsl.EmbeddedInput[dsl.Dataset]):
                    pass

    def test_escaping_symlink_raises_error(self):
        """A nested symlink whose target is outside the artifact tree should be
        rejected."""
        import os
        import tempfile

        with tempfile.TemporaryDirectory() as tmpdir:
            outside_file = os.path.join(tmpdir, 'secret.txt')
            with open(outside_file, 'w') as f:
                f.write('sensitive data')

            embed_dir = os.path.join(tmpdir, 'assets')
            sub_dir = os.path.join(embed_dir, 'nested')
            os.makedirs(sub_dir)
            escape_link = os.path.join(sub_dir, 'escaped.txt')
            try:
                os.symlink(outside_file, escape_link)
            except OSError:
                self.skipTest('Cannot create symlinks on this platform')

            with self.assertRaisesRegex(ValueError, r'escapes'):

                @component(
                    base_image='python:3.11',
                    embedded_artifact_path=embed_dir,
                    install_kfp_package=False,
                )
                def my_comp(cfg: dsl.EmbeddedInput[dsl.Dataset]):
                    pass

    def test_dangling_symlink_raises_error(self):
        """A symlink whose target does not exist should be rejected."""
        import os
        import tempfile

        with tempfile.TemporaryDirectory() as tmpdir:
            embed_dir = os.path.join(tmpdir, 'assets')
            os.makedirs(embed_dir)
            dangling_link = os.path.join(embed_dir, 'broken.txt')
            try:
                os.symlink(
                    os.path.join(tmpdir, 'nonexistent.txt'), dangling_link
                )
            except OSError:
                self.skipTest('Cannot create symlinks on this platform')

            with self.assertRaisesRegex(ValueError, r'[Dd]angling'):

                @component(
                    base_image='python:3.11',
                    embedded_artifact_path=embed_dir,
                    install_kfp_package=False,
                )
                def my_comp(cfg: dsl.EmbeddedInput[dsl.Dataset]):
                    pass

    def test_sibling_directory_symlink_is_rejected(self):
        """A symlink to a sibling directory should be rejected."""
        import os
        import tempfile

        with tempfile.TemporaryDirectory() as tmpdir:
            embed_dir = os.path.join(tmpdir, 'assets')
            sub_dir = os.path.join(embed_dir, 'sub')
            os.makedirs(sub_dir)
            with open(os.path.join(sub_dir, 'a.txt'), 'w') as f:
                f.write('content A')

            alias_link = os.path.join(embed_dir, 'alias')
            try:
                os.symlink(sub_dir, alias_link)
            except OSError:
                self.skipTest('Cannot create symlinks on this platform')

            with self.assertRaisesRegex(ValueError, r'[Dd]irectory symlinks'):

                @component(
                    base_image='python:3.11',
                    embedded_artifact_path=embed_dir,
                    install_kfp_package=False,
                )
                def my_comp(cfg: dsl.EmbeddedInput[dsl.Dataset]):
                    pass

    def test_self_referential_symlink_raises_error(self):
        """A self-referential symlink (a -> a) should be rejected as a cycle
        consistently across Python versions."""
        import os
        import tempfile

        with tempfile.TemporaryDirectory() as tmpdir:
            embed_dir = os.path.join(tmpdir, 'assets')
            os.makedirs(embed_dir)
            with open(os.path.join(embed_dir, 'file.txt'), 'w') as f:
                f.write('hello')
            self_link = os.path.join(embed_dir, 'self')
            try:
                os.symlink(self_link, self_link)
            except OSError:
                self.skipTest('Cannot create symlinks on this platform')

            with self.assertRaisesRegex(ValueError, r'[Cc]ycle'):

                @component(
                    base_image='python:3.11',
                    embedded_artifact_path=embed_dir,
                    install_kfp_package=False,
                )
                def my_comp(cfg: dsl.EmbeddedInput[dsl.Dataset]):
                    pass


def _get_archive_members(component_fn) -> List[tarfile.TarInfo]:
    """List the tar members of a compiled component's embedded archive."""
    import io

    with tarfile.open(
        fileobj=io.BytesIO(_get_archive(component_fn)), mode='r'
    ) as tar:
        return tar.getmembers()


def _compile_embedded_component(embedded_artifact_path: str):
    @component(
        base_image='python:3.11',
        embedded_artifact_path=embedded_artifact_path,
        install_kfp_package=False,
    )
    def my_comp(cfg: dsl.EmbeddedInput[dsl.Dataset]):
        pass

    return my_comp


def _get_command(component_fn) -> List[str]:
    return component_fn.component_spec.implementation.container.command


class TestEmbeddedArtifactDeterminism(unittest.TestCase):
    """Tests for #14852: the same embedded files must give the same command.

    The command is part of the cache key, so an archive that changes
    between two compiles makes the step miss the cache.
    """

    # 1980-02-01T00:00:00Z. ZIP cannot store a timestamp before 1980.
    EXPECTED_MEMBER_MTIME = 318211200

    def setUp(self):
        import os
        import tempfile

        temporary_directory = tempfile.TemporaryDirectory()
        self.addCleanup(temporary_directory.cleanup)
        self.embed_dir = os.path.join(temporary_directory.name, 'pkg')
        self.nested_dir = os.path.join(self.embed_dir, 'sub')
        os.makedirs(self.nested_dir)
        self.top_file = os.path.join(self.embed_dir, 'mod.py')
        self.nested_file = os.path.join(self.nested_dir, 'nested.py')
        for path in (self.top_file, self.nested_file):
            with open(path, 'w') as f:
                f.write('X = 1\n')

    def _set_mtime_of_all_paths(self, mtime: int) -> None:
        import os

        for path in (
            self.embed_dir,
            self.nested_dir,
            self.top_file,
            self.nested_file,
        ):
            os.utime(path, (mtime, mtime))

    def test_command_is_same_when_directory_mtimes_change(self):
        # A fresh clone or a CI checkout gives new mtimes to unchanged files.
        self._set_mtime_of_all_paths(1_600_000_000)
        first_command = _get_command(
            _compile_embedded_component(self.embed_dir)
        )

        self._set_mtime_of_all_paths(1_700_000_000)
        second_command = _get_command(
            _compile_embedded_component(self.embed_dir)
        )

        self.assertEqual(first_command, second_command)

    def test_command_is_same_when_single_file_mtime_changes(self):
        import os

        os.utime(self.top_file, (1_600_000_000, 1_600_000_000))
        first_command = _get_command(_compile_embedded_component(self.top_file))

        os.utime(self.top_file, (1_700_000_000, 1_700_000_000))
        second_command = _get_command(
            _compile_embedded_component(self.top_file)
        )

        self.assertEqual(first_command, second_command)

    def test_command_is_same_when_compile_time_changes(self):
        # Two submissions of the same pipeline compile at different times.
        from unittest import mock

        with mock.patch('time.time', return_value=1_600_000_000):
            first_command = _get_command(
                _compile_embedded_component(self.embed_dir)
            )
        with mock.patch('time.time', return_value=1_700_000_000):
            second_command = _get_command(
                _compile_embedded_component(self.embed_dir)
            )

        self.assertEqual(first_command, second_command)

    def test_command_changes_when_file_content_changes(self):
        first_command = _get_command(
            _compile_embedded_component(self.embed_dir)
        )

        with open(self.top_file, 'w') as f:
            f.write('X = 2\n')
        second_command = _get_command(
            _compile_embedded_component(self.embed_dir)
        )

        self.assertNotEqual(first_command, second_command)

    def test_gzip_header_has_no_timestamp_and_no_host_os(self):
        # A laptop compile and a CI compile must give the same header.
        compressed = _get_compressed_archive(
            _compile_embedded_component(self.embed_dir)
        )

        # RFC 1952: magic, deflate, no flags, MTIME 0, XFL 2, OS 255 (unknown).
        self.assertEqual(
            compressed[:10].hex(), '1f8b0800' + '00000000' + '02' + 'ff'
        )

    def test_directory_members_have_fixed_mtime_and_no_owner(self):
        members = _get_archive_members(
            _compile_embedded_component(self.embed_dir)
        )

        self.assertEqual(
            [member.name for member in members],
            ['.', 'mod.py', 'sub', 'sub/nested.py'],
        )
        for member in members:
            with self.subTest(member=member.name):
                self.assertEqual(member.mtime, self.EXPECTED_MEMBER_MTIME)
                self.assertEqual(member.uid, 0)
                self.assertEqual(member.gid, 0)
                self.assertEqual(member.uname, '')
                self.assertEqual(member.gname, '')

    def test_single_file_member_has_fixed_mtime_and_no_owner(self):
        members = _get_archive_members(
            _compile_embedded_component(self.top_file)
        )

        self.assertEqual([member.name for member in members], ['mod.py'])
        member = members[0]
        self.assertEqual(member.mtime, self.EXPECTED_MEMBER_MTIME)
        self.assertEqual(member.uid, 0)
        self.assertEqual(member.gid, 0)
        self.assertEqual(member.uname, '')
        self.assertEqual(member.gname, '')

    def test_file_mode_is_preserved(self):
        # An embedded script must stay executable in the container.
        import os

        os.chmod(self.top_file, 0o755)
        os.chmod(self.nested_file, 0o640)

        members = _get_archive_members(
            _compile_embedded_component(self.embed_dir)
        )

        name_to_mode = {member.name: member.mode for member in members}
        self.assertEqual(name_to_mode['mod.py'], 0o755)
        self.assertEqual(name_to_mode['sub/nested.py'], 0o640)

    def test_extracted_file_can_be_added_to_zip(self):
        # A component that zips its embedded files: ZIP rejects a 1970 mtime.
        import io
        import os
        import tempfile
        import zipfile

        raw = _get_archive(_compile_embedded_component(self.embed_dir))

        with tempfile.TemporaryDirectory() as extract_dir:
            with tarfile.open(fileobj=io.BytesIO(raw), mode='r') as tar:
                tar.extractall(path=extract_dir)
            with zipfile.ZipFile(io.BytesIO(), mode='w') as zip_file:
                zip_file.write(os.path.join(extract_dir, 'mod.py'))


if __name__ == '__main__':
    unittest.main()
