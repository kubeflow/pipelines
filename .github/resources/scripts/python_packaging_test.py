#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
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
"""Regression coverage for Python packaging and dependency tooling."""

import os
from pathlib import Path
import re
import shlex
import shutil
import subprocess
import sys
import tempfile
import textwrap
import unittest

ROOT = Path(__file__).resolve().parents[3]
EXPORTS = (
    'requirements.txt',
    'sdk/python/requirements.txt',
    'kubernetes_platform/python/requirements.txt',
    'api/v2alpha1/python/requirements.txt',
)


def package_version(path: Path) -> str:
    """Read a literal package version without importing SDK dependencies."""
    match = re.search(r'''^(?:__version__|version)\s*=\s*['"]([^'"]+)['"]''',
                      path.read_text(), re.MULTILINE)
    if match is None:
        raise AssertionError(f'No package version found in {path}')
    return match.group(1)


class PythonPackagingTest(unittest.TestCase):

    def test_workspace_versions_and_public_dependency_ranges(self) -> None:
        """Keep the four distributions on one SDK release without exact
        pins."""
        version = package_version(ROOT / 'sdk/python/kfp/version.py')
        for path in (
                'api/v2alpha1/python/pyproject.toml',
                'kubernetes_platform/python/kfp/kubernetes/__init__.py',
                'backend/api/v2beta1/python_http_client/pyproject.toml',
                'backend/api/v2beta1/python_http_client/kfp_server_api/__init__.py',
        ):
            with self.subTest(path=path):
                self.assertEqual(package_version(ROOT / path), version)
        next_major = int(version.split('.')[0]) + 1
        sdk_metadata = (ROOT / 'sdk/python/pyproject.toml').read_text()
        for package in ('kfp-pipeline-spec', 'kfp-server-api'):
            self.assertIn(f'"{package}>={version},<{next_major}"', sdk_metadata)
        self.assertIn(f'"kfp-kubernetes=={version}"', sdk_metadata)
        self.assertIn(f'"kfp>={version},<{next_major}"',
                      (ROOT /
                       'kubernetes_platform/python/pyproject.toml').read_text())

    def test_export_workflow_rejects_each_stale_tracked_file(self) -> None:
        """Run the workflow check against clean and stale export fixtures."""
        workflow = (ROOT /
                    '.github/workflows/check-requirements-txt.yml').read_text()
        self.assertNotIn('working-directory:', workflow)
        command = textwrap.dedent(workflow.split('        run: |\n', 1)[1])
        for stale_path in (None, *EXPORTS):
            with self.subTest(stale_path=stale_path
                             ), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                for relative_path in EXPORTS:
                    path = root / relative_path
                    path.parent.mkdir(parents=True, exist_ok=True)
                    path.write_text('stale\n' if relative_path ==
                                    stale_path else 'current\n')
                subprocess.run(['git', 'init', '--quiet'], cwd=root, check=True)
                subprocess.run(['git', 'add', *EXPORTS], cwd=root, check=True)
                fake_bin = root / 'bin'
                fake_bin.mkdir()
                uv = fake_bin / 'uv'
                uv.write_text(f'#!{sys.executable}\n' + textwrap.dedent('''\
                    from pathlib import Path
                    import sys
                    output = Path(sys.argv[sys.argv.index('-o') + 1])
                    output.write_text('current\\n')
                '''))
                uv.chmod(0o755)
                result = subprocess.run(
                    ['bash', '-e', '-c', command],
                    cwd=root,
                    env={
                        **os.environ, 'PATH':
                            f'{fake_bin}{os.pathsep}{os.environ["PATH"]}'
                    },
                    capture_output=True,
                    text=True,
                    check=False,
                )
                self.assertEqual(result.returncode, int(stale_path is not None),
                                 result.stderr)
                if stale_path is not None:
                    self.assertIn(f'diff --git a/{stale_path}', result.stdout)

    def test_readthedocs_uses_generated_workspace_packages(self) -> None:
        """Build both documentation sites with uv rather than pip-installing
        root."""
        preparation = [
            'uv sync --frozen --extra docs',
            'make -C api fetch-protos',
            'uv run python api/v2alpha1/python/generate_proto.py',
            'uv run python kubernetes_platform/python/generate_proto.py',
        ]
        for config_path, docs_path in (
            ('.readthedocs.yml', 'docs'),
            ('kubernetes_platform/python/docs/.readthedocs.yml',
             'kubernetes_platform/python/docs'),
        ):
            with self.subTest(config_path=config_path):
                config = (ROOT / config_path).read_text()
                self.assertNotIn('python:\n  install:', config)
                commands = [
                    line.removeprefix('    - ') for line in config.split(
                        '  commands:\n', 1)[1].splitlines()
                ]
                self.assertEqual(commands[:-1], preparation)
                build = shlex.split(commands[-1])
                self.assertEqual(build[:3], ['uv', 'run', 'sphinx-build'])
                self.assertEqual(build[-2:],
                                 [docs_path, '$READTHEDOCS_OUTPUT/html'])

    def test_visualization_updater_retains_its_shared_helper(self) -> None:
        """Exercise the retained requirements workflow without running
        Docker."""
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            path = Path('backend/src/apiserver/visualization')
            destination = root / path
            destination.mkdir(parents=True)
            shutil.copy(ROOT / path / 'update_requirements.sh', destination)
            (root / 'hack').mkdir()
            shutil.copy(ROOT / 'hack/update-requirements.sh', root / 'hack')
            (destination / 'requirements.in').write_text('test-package==1.0\n')
            (destination / 'requirements.txt').write_text('old\n')
            fake_bin = root / 'bin'
            fake_bin.mkdir()
            docker = fake_bin / 'docker'
            docker.write_text('#!/bin/sh\ncat\n')
            docker.chmod(0o755)
            result = subprocess.run(
                ['bash', 'update_requirements.sh'],
                cwd=destination,
                env={
                    **os.environ, 'PATH':
                        f'{fake_bin}{os.pathsep}{os.environ["PATH"]}'
                },
                capture_output=True,
                text=True,
                check=False,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual((destination / 'requirements.txt').read_text(),
                             'test-package==1.0\n')

    def test_server_generator_selects_sdk_version_only_for_v2(self) -> None:
        """Regenerate v2 metadata from the SDK while retaining legacy v1
        behavior."""
        for api_version, expected_version in (
            ('v2beta1', '2.17.0'),
            ('v1beta1', '2.99.0'),
        ):
            with self.subTest(api_version=api_version
                             ), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                api = root / 'backend/api'
                api.mkdir(parents=True)
                generator = api / 'build_kfp_server_api_python_package.sh'
                shutil.copy(ROOT / 'backend/api' / generator.name, generator)
                version_file = root / 'sdk/python/kfp/version.py'
                version_file.parent.mkdir(parents=True)
                version_file.write_text("__version__ = '2.17.0'\n")
                (root / 'VERSION').write_text('2.99.0\n')
                (root / 'LICENSE').write_text('test license\n')
                fake_bin = root / 'bin'
                fake_bin.mkdir()
                curl = fake_bin / 'curl'
                curl.write_text('#!/bin/sh\nexit 0\n')
                curl.chmod(0o755)
                java = fake_bin / 'java'
                java.write_text(f'#!{sys.executable}\n' + textwrap.dedent('''\
                    import json
                    from pathlib import Path
                    import sys
                    args = sys.argv[1:]
                    config = json.loads(Path(args[args.index('-c') + 1]).read_text())
                    output = Path(args[args.index('-o') + 1])
                    models = output / 'kfp_server_api/models'
                    models.mkdir(parents=True)
                    (models / '__init__.py').touch()
                    (models.parent / '__init__.py').write_text(
                        '__version__ = ' + repr(config['packageVersion']) + '\\n')
                    for name in ('README.md', 'setup.py', 'tox.ini', 'test-requirements.txt'):
                        (output / name).touch()
                '''))
                java.chmod(0o755)
                result = subprocess.run(
                    ['bash', '-e', str(generator)],
                    cwd=root,
                    env={
                        **os.environ,
                        'PATH':
                            f'{fake_bin}{os.pathsep}{os.environ["PATH"]}',
                        'API_VERSION':
                            api_version,
                    },
                    capture_output=True,
                    text=True,
                    check=False,
                )
                self.assertEqual(result.returncode, 0, result.stderr)
                output = api / api_version / 'python_http_client'
                self.assertEqual(
                    package_version(output / 'kfp_server_api/__init__.py'),
                    expected_version)
                if api_version == 'v2beta1':
                    self.assertEqual(
                        package_version(output / 'pyproject.toml'),
                        expected_version)
                self.assertEqual((root / 'VERSION').read_text(), '2.99.0\n')


if __name__ == '__main__':
    unittest.main()
