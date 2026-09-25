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
import tomllib
import unittest

ROOT = Path(__file__).resolve().parents[3]
EXPORTS = (
    'requirements.txt',
    'sdk/python/requirements.txt',
    'kubernetes_platform/python/requirements.txt',
    'api/v2alpha1/python/requirements.txt',
)
PACKAGE_PATHS = {
    'kfp-pipeline-spec': 'api/v2alpha1/python',
    'kfp-server-api': 'backend/api/v2beta1/python_http_client',
    'kfp': 'sdk/python',
    'kfp-kubernetes': 'kubernetes_platform/python',
}


def package_version(path: Path) -> str:
    """Read a literal package version without importing SDK dependencies."""
    match = re.search(r'''^(?:__version__|version)\s*=\s*['"]([^'"]+)['"]''',
                      path.read_text(), re.MULTILINE)
    if match is None:
        raise AssertionError(f'No package version found in {path}')
    return match.group(1)


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
        """Old tags need neither a local action nor a uv workspace."""
        workflow = (ROOT / '.github/workflows/publish-packages.yml').read_text()
        self.assertNotIn('uses: ./', workflow)
        self.assertNotIn('uv sync', workflow)
        self.assertNotIn('uv run ', workflow)
        self.assertEqual(workflow.count('uses: actions/setup-python@'), 4)
        self.assertEqual(workflow.count('uses: astral-sh/setup-uv@'), 4)
        self.assertEqual(
            workflow.count('ref: ${{ github.event.inputs.tag }}'), 4)
        self.assertEqual(
            workflow.count("if: ${{ github.event.inputs.dry_run == 'false' }}"),
            4)
        self.assertEqual(
            workflow.count("if: ${{ github.event.inputs.dry_run == 'true' }}"),
            4)

    def test_publishing_builds_each_selected_tag_once(self) -> None:
        """Execute workflow build steps in legacy and migrated tag fixtures."""
        workflow = (ROOT / '.github/workflows/publish-packages.yml').read_text()
        build_steps = dict(
            re.findall(
                r'      - name: Build (kfp[\w-]*)\n'
                r'        run: \|\n((?:          [^\n]*\n)+)', workflow))
        self.assertEqual(set(build_steps), set(PACKAGE_PATHS))
        make_directories = {
            'kfp-pipeline-spec': 'api',
            'kfp': 'sdk',
            'kfp-kubernetes': 'kubernetes_platform',
        }
        for legacy in (True, False):
            for package, relative_path in PACKAGE_PATHS.items():
                with self.subTest(
                        legacy=legacy, package=package
                ), tempfile.TemporaryDirectory() as directory:
                    root = Path(directory)
                    source = root / relative_path
                    source.mkdir(parents=True)
                    metadata = 'setup.py' if legacy else 'pyproject.toml'
                    (source / metadata).touch()
                    if not legacy:
                        (root / 'uv.lock').touch()
                    if package in make_directories:
                        make_root = root / make_directories[package]
                        distribution_path = source.relative_to(make_root)
                        # Each tag owns its Makefile and build backend.
                        (make_root / 'Makefile').write_text(
                            '.PHONY: python\npython:\n'
                            f'\tmkdir -p {distribution_path}/dist\n'
                            f'\ttest -f {distribution_path}/{metadata}\n'
                            f'\ttouch {distribution_path}/dist/package.tar.gz\n'
                            f'\ttouch {distribution_path}/dist/package.whl\n'
                            '\tprintf "build\\n" >> ../build-count\n')
                    fake_bin = root / 'bin'
                    fake_bin.mkdir()
                    uv = fake_bin / 'uv'
                    uv.write_text(f'#!{sys.executable}\n' + textwrap.dedent('''\
                        from pathlib import Path
                        import sys
                        source = 'backend/api/v2beta1/python_http_client'
                        assert sys.argv[1:] == ['build', source, '--out-dir', source + '/dist']
                        assert any((Path(source) / name).exists()
                                   for name in ('setup.py', 'pyproject.toml'))
                        output = Path(source) / 'dist'
                        output.mkdir()
                        (output / 'package.tar.gz').touch()
                        (output / 'package.whl').touch()
                        with Path('build-count').open('a') as count:
                            count.write('build\\n')
                    '''))
                    uv.chmod(0o755)
                    uvx = fake_bin / 'uvx'
                    uvx.write_text(f'#!{sys.executable}\n' +
                                   textwrap.dedent('''\
                        from pathlib import Path
                        import sys
                        assert sys.argv[1:7] == ['--python', '3.12', '--from', 'twine==7.0.0', 'twine', 'check']
                        assert len(sys.argv[7:]) == 2
                        assert all(Path(path).is_file() for path in sys.argv[7:])
                    '''))
                    uvx.chmod(0o755)
                    result = subprocess.run(
                        [
                            'bash', '-e', '-c',
                            textwrap.dedent(build_steps[package])
                        ],
                        cwd=root,
                        env={
                            **os.environ,
                            'TWINE_VERSION':
                                '7.0.0',
                            'TWINE_PYTHON_VERSION':
                                '3.12',
                            'PATH':
                                f'{fake_bin}{os.pathsep}{os.environ["PATH"]}',
                        },
                        capture_output=True,
                        text=True,
                        check=False,
                    )
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual(
                        sorted(
                            path.name for path in (source / 'dist').iterdir()),
                        ['package.tar.gz', 'package.whl'])
                    self.assertEqual((root / 'build-count').read_text(),
                                     'build\n')

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

    def _prepare_kubernetes_publisher(self, root: Path) -> Path:
        """Copy the real publisher and replace builders/uploaders with
        fixtures."""
        package = root / 'kubernetes_platform/python'
        version_file = package / 'kfp/kubernetes/__init__.py'
        version_file.parent.mkdir(parents=True)
        version_file.write_text(
            "__version__ = '2.17.0'\n"
            "raise AssertionError('Version extraction must not import the package')\n"
        )
        shutil.copy(ROOT / 'kubernetes_platform/python/release.sh', package)
        fake_bin = root / 'bin'
        fake_bin.mkdir()
        scripts = {
            'python3':
                '''\
                import os
                from pathlib import Path
                import shutil
                import sys
                args = sys.argv[1:]
                if args[:2] == ['-m', 'build']:
                    output = Path(args[args.index('--outdir') + 1])
                    output.mkdir(exist_ok=True)
                    (output / 'kfp_kubernetes-2.17.0.tar.gz').touch()
                    (output / 'kfp_kubernetes-2.17.0-py3-none-any.whl').touch()
                elif args[:2] == ['-m', 'venv']:
                    bin_dir = Path(args[2]) / 'bin'
                    bin_dir.mkdir(parents=True)
                    shutil.copy(Path(os.environ['FAKE_BIN']) / 'pip', bin_dir)
                else:
                    raise AssertionError(args)
            ''',
            'pip':
                '''\
                from pathlib import Path
                import sys
                if sys.argv[1] == 'install':
                    assert Path(sys.argv[2]).is_file()
                elif sys.argv[1:] == ['list']:
                    print('kfp-kubernetes 2.17.0')
                else:
                    raise AssertionError(sys.argv)
            ''',
            'uvx':
                '''\
                import os
                from pathlib import Path
                import sys
                assert sys.argv[1:6] == [
                    '--python', '3.12', '--from', 'twine==7.0.0', 'twine']
                command = sys.argv[6]
                assert command in ('check', 'upload')
                assert sys.argv[7:] == ['kfp-kubernetes-2.17.0.tar.gz']
                assert Path(sys.argv[7]).is_file()
                with Path(os.environ['TWINE_CALLS']).open('a') as calls:
                    calls.write(command + '\\n')
                if command == 'check':
                    sys.exit(int(os.environ['CHECK_EXIT_CODE']))
            ''',
        }
        for name, script in scripts.items():
            executable = fake_bin / name
            executable.write_text(f'#!{sys.executable}\n' +
                                  textwrap.dedent(script))
            executable.chmod(0o755)
        shutil.copy(fake_bin / 'python3', fake_bin / 'python')
        grep = fake_bin / 'grep'
        grep.write_text('#!/bin/sh\n'
                        'if [ "$1" = "-oP" ] || [ "$1" = "-P" ]; then\n'
                        '  echo "grep: invalid option -- P" >&2\n'
                        '  exit 2\n'
                        'fi\n'
                        'exec /usr/bin/grep "$@"\n')
        grep.chmod(0o755)
        (root / 'tmp').mkdir()
        return package

    def test_kubernetes_publisher_uses_portable_version_and_isolated_twine(
            self) -> None:
        """Exercise executed/sourced publishing and block upload after bad
        metadata."""
        workflow = (ROOT / '.github/workflows/publish-packages.yml').read_text()
        self.assertIn('TWINE_VERSION: "7.0.0"', workflow)
        self.assertIn('TWINE_PYTHON_VERSION: "3.12"', workflow)
        for sourced in (False, True):
            for check_exit_code in (0, 1):
                with self.subTest(
                        sourced=sourced, check_exit_code=check_exit_code
                ), tempfile.TemporaryDirectory() as directory:
                    root = Path(directory)
                    package = self._prepare_kubernetes_publisher(root)
                    calls = root / 'twine-calls'
                    command = (['bash', '-c', 'source ./release.sh']
                               if sourced else ['bash', '-e', 'release.sh'])
                    result = subprocess.run(
                        command,
                        cwd=package,
                        env={
                            **os.environ,
                            'PATH':
                                f'{root / "bin"}{os.pathsep}{os.environ["PATH"]}',
                            'FAKE_BIN':
                                str(root / 'bin'),
                            'TMPDIR':
                                str(root / 'tmp'),
                            'KFP_KUBERNETES_VERSION':
                                '2.17.0',
                            'TWINE_CALLS':
                                str(calls),
                            'CHECK_EXIT_CODE':
                                str(check_exit_code),
                        },
                        capture_output=True,
                        text=True,
                        check=False,
                    )
                    self.assertEqual(result.returncode, check_exit_code,
                                     result.stdout + result.stderr)
                    self.assertTrue(calls.exists(),
                                    result.stdout + result.stderr)
                    self.assertEqual(calls.read_text().splitlines(),
                                     ['check', 'upload']
                                     if check_exit_code == 0 else ['check'])

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
                    (output / 'README.md').write_text('Python 2.7 and 3.4+\\n')
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
                self.assertEqual(result.returncode, 0, result.stderr)
                output = api / api_version / 'python_http_client'
                self.assertEqual(
                    package_version(output / 'kfp_server_api/__init__.py'),
                    expected_version)
                if api_version == 'v2beta1':
                    self.assertEqual(
                        package_version(output / 'pyproject.toml'),
                        expected_version)
                    metadata = tomllib.loads(
                        (output / 'pyproject.toml').read_text())
                    self.assertEqual(metadata['project']['requires-python'],
                                     '>=3.11')
                    self.assertEqual((output / 'README.md').read_text(),
                                     'Python 3.11 or later\n')
                else:
                    self.assertEqual((output / 'README.md').read_text(),
                                     'Python 2.7 and 3.4+\n')
                self.assertEqual((root / 'VERSION').read_text(), '2.99.0\n')


if __name__ == '__main__':
    unittest.main()
