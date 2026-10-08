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

import fnmatch
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
)
PACKAGE_PATHS = {
    'kfp-pipeline-spec': 'api/v2alpha1/python',
    'kfp-server-api': 'backend/api/v2beta1/python_http_client',
    'kfp': 'sdk/python',
    'kfp-kubernetes': 'kubernetes_platform/python',
}


class PythonPackagingTest(unittest.TestCase):

    def test_backend_compiler_reuses_proto_downloader(self) -> None:
        """Keep compiler dependencies and retry behavior in the API
        Makefile."""
        compiler = (ROOT / 'backend/Dockerfile').read_text().split(
            ' AS compiler\n', 1)[1].split('# 3. Start', 1)[0]
        self.assertIn('COPY api/Makefile ./api/Makefile', compiler)
        self.assertIn(
            'RUN make -C api fetch-protos && \\\n'
            '    python3 /workspace/api/v2alpha1/python/generate_proto.py',
            compiler)
        self.assertNotIn('raw.githubusercontent.com', compiler)
        dependencies = next(line for line in compiler.splitlines()
                            if line.startswith('RUN apt-get')).split()
        for tool in ('make', 'git', 'wget', 'protobuf-compiler'):
            self.assertIn(tool, dependencies)

    def test_lockfile_validation_covers_release_pushes(self) -> None:
        """Check every master/release push while retaining filtered PR
        checks."""
        workflow = (ROOT / '.github/workflows/check-uv-lock.yml').read_text()
        push, pull_request = workflow.split('  push:\n',
                                            1)[1].split('  pull_request:\n', 1)
        self.assertNotIn('paths:', push)
        self.assertNotIn('paths-ignore:', push)
        branch_list = re.search(r'branches: \[([^\]]+)\]', push)
        if branch_list is None:
            self.fail('Lockfile workflow has no push branch list')
        patterns = [
            branch.strip().strip("'\"")
            for branch in branch_list.group(1).split(',')
        ]
        for branch, expected in (
            ('master', True),
            ('release-2.17', True),
            ('release-2.18', True),
            ('release-3.0', True),
            ('feature', False),
        ):
            with self.subTest(branch=branch):
                self.assertEqual(
                    any(
                        fnmatch.fnmatchcase(branch, pattern)
                        for pattern in patterns), expected)
        pull_request = pull_request.split('\njobs:', 1)[0]
        for path in ('**/pyproject.toml', 'uv.lock',
                     '.github/workflows/check-uv-lock.yml'):
            self.assertIn(f"'{path}'", pull_request)

    def test_publishing_setup_is_independent_of_source_tag(self) -> None:
        """Keep legacy tag setup independent and publishing dry-run guarded."""
        workflow = (ROOT / '.github/workflows/publish-packages.yml').read_text()
        self.assertNotIn('uses: ./', workflow)
        self.assertNotIn('uv sync', workflow)
        self.assertIn('uses: actions/setup-python@', workflow)
        self.assertIn('uses: astral-sh/setup-uv@', workflow)
        self.assertEqual(
            workflow.count('ref: ${{ github.event.inputs.tag }}'), 2)
        self.assertIn("if: ${{ github.event.inputs.dry_run == 'false' }}",
                      workflow)
        self.assertIn("if: ${{ github.event.inputs.dry_run == 'true' }}",
                      workflow)
        self.assertIn('TWINE_VERSION: "7.0.0"', workflow)
        self.assertIn('TWINE_PYTHON_VERSION: "3.12"', workflow)

    def test_tag_selection_only_publishes_owned_distributions(self) -> None:
        """Select only kfp for unified tags and preserve all legacy choices."""
        workflow = (ROOT / '.github/workflows/publish-packages.yml').read_text()
        selector = re.search(r"python3 - <<'PY'\n(.*?)^          PY", workflow,
                             re.MULTILINE | re.DOTALL).group(1)
        import json
        for unified in (False, True):
            for selected in ('all', *PACKAGE_PATHS):
                with self.subTest(unified=unified, selected=selected):
                    with tempfile.TemporaryDirectory() as directory:
                        root = Path(directory)
                        if unified:
                            initializer = root / 'sdk/python/kfp/server_api/__init__.py'
                            initializer.parent.mkdir(parents=True)
                            initializer.touch()
                        output = root / 'output'
                        result = subprocess.run(
                            [sys.executable, '-c',
                             textwrap.dedent(selector)],
                            cwd=root,
                            capture_output=True,
                            text=True,
                            env={
                                **os.environ, 'GITHUB_OUTPUT': str(output),
                                'SELECTED_PACKAGE': selected
                            },
                            check=False)
                        incompatible = unified and selected not in ('all',
                                                                    'kfp')
                        self.assertEqual(result.returncode != 0, incompatible,
                                         result.stderr)
                        if incompatible:
                            self.assertIn('not a distribution', result.stderr)
                        else:
                            actual = json.loads(output.read_text().split(
                                '=', 1)[1])
                            expected = (['kfp']
                                        if unified else list(PACKAGE_PATHS))
                            if selected != 'all':
                                expected = [selected]
                            self.assertCountEqual(
                                [package['name'] for package in actual],
                                expected)

    def test_publishing_builds_each_selected_tag_once(self) -> None:
        """Execute tag-owned builds once for old and unified layouts."""
        workflow = (ROOT / '.github/workflows/publish-packages.yml').read_text()
        build = textwrap.dedent(
            workflow.split('      - name: Build selected distribution\n',
                           1)[1].split('        run: |\n',
                                       1)[1].split('      - name:', 1)[0])
        make_directories = {
            'kfp': 'sdk',
            'kfp-pipeline-spec': 'api',
            'kfp-kubernetes': 'kubernetes_platform'
        }
        for legacy in (False, True):
            packages = PACKAGE_PATHS if legacy else {'kfp': 'sdk/python'}
            for package, relative_path in packages.items():
                with self.subTest(legacy=legacy, package=package):
                    with tempfile.TemporaryDirectory() as directory:
                        root = Path(directory)
                        source = root / relative_path
                        source.mkdir(parents=True)
                        metadata = 'setup.py' if legacy else 'pyproject.toml'
                        (source / metadata).touch()
                        if package in make_directories:
                            make_root = root / make_directories[package]
                            local = source.relative_to(make_root)
                            (make_root / 'Makefile').write_text(
                                '.PHONY: python\npython:\n'
                                f'\ttest -f {local}/{metadata}\n'
                                f'\tmkdir -p {local}/dist\n'
                                f'\ttouch {local}/dist/package.whl {local}/dist/package.tar.gz\n'
                                '\tprintf "build\\n" >> ../build-count\n')
                        fake_bin = root / 'bin'
                        fake_bin.mkdir()
                        uv = fake_bin / 'uv'
                        uv.write_text(f'#!{sys.executable}\n' +
                                      textwrap.dedent("""
                            from pathlib import Path
                            import sys
                            source = Path(sys.argv[2])
                            assert sys.argv[1:] == ['build', str(source), '--out-dir', str(source / 'dist')]
                            assert (source / 'setup.py').is_file()
                            output = source / 'dist'
                            output.mkdir()
                            (output / 'package.whl').touch()
                            (output / 'package.tar.gz').touch()
                            Path('build-count').write_text('build\\n')
                        """))
                        uv.chmod(0o755)
                        uvx = fake_bin / 'uvx'
                        uvx.write_text(f'#!{sys.executable}\n' +
                                       textwrap.dedent("""
                            from pathlib import Path
                            import sys
                            assert sys.argv[1:7] == ['--python', '3.12', '--from', 'twine==7.0.0', 'twine', 'check']
                            assert len(sys.argv[7:]) == 2
                            assert all(Path(path).is_file() for path in sys.argv[7:])
                        """))
                        uvx.chmod(0o755)
                        result = subprocess.run(
                            ['bash', '-e', '-c', build],
                            cwd=root,
                            env={
                                **os.environ, 'PATH':
                                    f'{fake_bin}{os.pathsep}{os.environ["PATH"]}',
                                'PACKAGE_NAME':
                                    package,
                                'PACKAGE_PATH':
                                    relative_path,
                                'TWINE_VERSION':
                                    '7.0.0',
                                'TWINE_PYTHON_VERSION':
                                    '3.12'
                            },
                            capture_output=True,
                            text=True,
                            check=False)
                        self.assertEqual(result.returncode, 0, result.stderr)
                        self.assertEqual((root / 'build-count').read_text(),
                                         'build\n')

    def test_workspace_has_one_distribution_and_one_version(self) -> None:
        """Only kfp owns the bundled namespaces and release version."""
        sdk_metadata = (ROOT / 'sdk/python/pyproject.toml').read_text()
        workspace = (ROOT / 'pyproject.toml').read_text()
        for package in ('kfp-pipeline-spec', 'kfp-server-api',
                        'kfp-kubernetes'):
            self.assertNotIn(package, sdk_metadata)
            self.assertNotIn(package, workspace)
        self.assertIn('kubernetes = []', sdk_metadata)
        for path in ('kubernetes', 'server_api'):
            self.assertIn('from kfp.version import __version__',
                          (ROOT /
                           f'sdk/python/kfp/{path}/__init__.py').read_text())
        for path in ('api/v2alpha1/python/pyproject.toml',
                     'backend/api/v2/python_http_client/pyproject.toml',
                     'kubernetes_platform/python/pyproject.toml'):
            self.assertFalse((ROOT / path).exists(), path)
        self.assertNotIn('extend_path',
                         (ROOT / 'sdk/python/kfp/__init__.py').read_text())

    def test_export_workflow_rejects_each_stale_tracked_file(self) -> None:
        """Run the workflow check against clean and stale export fixtures."""
        workflow = (ROOT /
                    '.github/workflows/check-requirements-txt.yml').read_text()
        self.assertNotIn('working-directory:', workflow)
        command = textwrap.dedent(
            workflow.split('        run: |\n', 1)[1].split('      - name:',
                                                           1)[0])
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
                    assert '--no-hashes' in sys.argv
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

    def test_requirements_exports_do_not_hash_editable_packages(self) -> None:
        """Keep pip's incompatible hash-checking mode out of editable
        exports."""
        for relative_path in EXPORTS:
            with self.subTest(path=relative_path):
                requirements = (ROOT / relative_path).read_text()
                self.assertRegex(requirements, r'(?m)^-e \./')
                self.assertNotIn('--hash=', requirements)
                self.assertNotIn('--require-hashes', requirements)
                self.assertIn('--no-hashes', requirements.splitlines()[1])

    def test_retired_publisher_fails_in_executed_and_sourced_modes(
            self) -> None:
        """Never publish a second distribution from the consolidated tree."""
        script = ROOT / 'kubernetes_platform/python/release.sh'
        for command in (['bash', str(script)],
                        ['bash', '-c', 'source "$1"', 'bash',
                         str(script)]):
            result = subprocess.run(
                command, capture_output=True, text=True, check=False)
            self.assertEqual(result.returncode, 1)
            self.assertIn('included in kfp', result.stderr)

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

    def test_server_generator_uses_sdk_version_and_rejects_v1(self) -> None:
        """Regenerate v2 metadata from the SDK and reject removed v1 APIs."""
        for api_version in ('v2', 'v1beta1'):
            with self.subTest(api_version=api_version
                             ), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                api = root / 'backend/api'
                api.mkdir(parents=True)
                generator = api / 'build_kfp_server_api_python_package.sh'
                shutil.copy(ROOT / 'backend/api' / generator.name, generator)
                (api / 'hack').mkdir()
                shutil.copy(ROOT / 'backend/api/hack/generate_python_compat.py',
                            api / 'hack/generate_python_compat.py')
                version_file = root / 'sdk/python/kfp/version.py'
                version_file.parent.mkdir(parents=True)
                (root / 'sdk/python/test').mkdir()
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
                    models = output.joinpath(*config['packageName'].split('.'), 'models')
                    models.mkdir(parents=True)
                    (models / '__init__.py').touch()
                    (models / 'v2_run.py').write_text('class V2Run: pass\\n')
                    (output / 'test').mkdir()
                    (models.parent / '__init__.py').write_text(
                        '__version__ = ' + repr(config['packageVersion']) + '\\n')
                    (output / 'README.md').write_text(
                        '# kfp.server-api\\n## Requirements.\\n'
                        'Python 2.7\\npython setup.py install\\n## Getting Started\\n')
                    for name in ('setup.py', 'tox.ini', 'test-requirements.txt'):
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
                output = api / api_version / 'python_http_client'
                self.assertEqual((root / 'VERSION').read_text(), '2.99.0\n')
                if api_version == 'v1beta1':
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn('Generate the canonical v2 API;',
                                  result.stderr)
                    self.assertFalse(output.exists())
                    continue
                self.assertEqual(result.returncode, 0, result.stderr)
                initializer = root / 'sdk/python/kfp/server_api/__init__.py'
                self.assertIn('from kfp.version import __version__',
                              initializer.read_text())
                self.assertFalse((output / 'test').exists())
                self.assertFalse((output / 'pyproject.toml').exists())
                self.assertFalse((output / 'setup.py').exists())
                readme = (output / 'README.md').read_text()
                self.assertIn('python -m pip install kfp', readme)
                self.assertIn('Python 3.11 or later', readme)
                self.assertNotIn('setup.py install', readme)
                self.assertNotIn('Python 2.7', readme)


if __name__ == '__main__':
    unittest.main()
