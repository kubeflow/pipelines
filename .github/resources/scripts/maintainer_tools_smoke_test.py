# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import hashlib
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

SCRIPT = Path(__file__).with_name('maintainer_tools_smoke.sh')


class SnapshotTests(unittest.TestCase):

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)

    def write(self, path, content):
        destination = self.root / path
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_text(content)

    def snapshot(self, *directories):
        return subprocess.run([
            'bash', '-c', 'source "$1"; shift; snapshot_sources "$@"', 'test',
            str(SCRIPT),
            str(self.root), *directories
        ],
                              check=False,
                              capture_output=True,
                              text=True)

    def api_source_directories(self) -> list[str]:
        """Read the inventory used by the real cross-image comparison."""
        result = subprocess.run([
            'bash', '-c',
            'source "$1"; printf "%s\\n" "${API_SOURCE_DIRECTORIES[@]}"',
            'test',
            str(SCRIPT)
        ],
                                check=True,
                                capture_output=True,
                                text=True)
        return result.stdout.splitlines()

    def test_api_inventory_tracks_sdk_client_and_documentation(self) -> None:
        """Changes to either relocated modules or retained docs affect
        hashes."""
        directories = self.api_source_directories()
        self.assertIn('sdk/python/kfp/server_api', directories)
        self.assertIn('backend/api/v2/python_http_client', directories)
        self.assertIn('backend/api/v2beta1/go_client', directories)
        self.assertIn('backend/api/v2beta1/go_http_client', directories)
        self.assertIn(
            'snapshot_sources "$source_dir" "${API_SOURCE_DIRECTORIES[@]}"',
            SCRIPT.read_text())
        for directory in directories:
            self.write(f'{directory}/generated.txt', 'source')
        for path in (
                'sdk/python/kfp/server_api/api_client.py',
                'backend/api/v2/python_http_client/docs/RunServiceApi.md',
                'backend/api/v2beta1/go_client/aliases.go',
                'backend/api/v2beta1/go_http_client/run_model/aliases.go',
        ):
            with self.subTest(path=path):
                self.write(path, 'before')
                before = self.snapshot(*directories)
                self.assertEqual(before.returncode, 0, before.stderr)
                self.assertIn(path, before.stdout)
                self.write(path, 'after')
                after = self.snapshot(*directories)
                self.assertEqual(after.returncode, 0, after.stderr)
                self.assertNotEqual(before.stdout, after.stdout)

    def test_missing_sdk_client_fails_api_snapshot(self) -> None:
        """Retained docs cannot hide a missing SDK-owned client."""
        directories = self.api_source_directories()
        for directory in directories:
            if directory != 'sdk/python/kfp/server_api':
                self.write(f'{directory}/generated.txt', 'source')
        result = self.snapshot(*directories)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('No generated files in sdk/python/kfp/server_api',
                      result.stderr)

    def test_snapshot_hashes_sorted_relative_source_paths(self):
        self.write('generated/z.go', 'last')
        self.write('generated/a.py', 'first')
        result = self.snapshot('generated')
        self.assertEqual(result.returncode, 0, result.stderr)
        expected = ''.join(
            f'{hashlib.sha256(content).hexdigest()}  {path}\n'
            for path, content in [('generated/a.py',
                                   b'first'), ('generated/z.go', b'last')])
        self.assertEqual(result.stdout, expected)
        self.assertNotIn(str(self.root), result.stdout)

    def test_excludes_build_products_not_generator_metadata(self):
        for path in ('dist/package.tar.gz', 'kfp_server_api.egg-info/PKG-INFO',
                     '__pycache__/module.pyc'):
            self.write(f'generated/{path}', 'volatile')
        self.write('generated/.openapi-generator/VERSION', '4.3.1')
        self.write('generated/kfp_server_api/api.py', 'source')
        result = self.snapshot('generated')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(len(result.stdout.splitlines()), 2)
        self.assertIn('.openapi-generator/VERSION', result.stdout)
        self.assertIn('kfp_server_api/api.py', result.stdout)

    def test_missing_outputs_fail(self):
        result = self.snapshot('missing')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('No generated files', result.stderr)

    def test_source_content_changes_are_visible(self):
        self.write('generated/api.py', 'first')
        before = self.snapshot('generated').stdout
        self.write('generated/api.py', 'changed')
        self.assertNotEqual(self.snapshot('generated').stdout, before)


class NativeGuardTests(unittest.TestCase):

    def test_local_invocation_reaches_usage_guard(self):
        result = subprocess.run(['bash', SCRIPT.name],
                                cwd=SCRIPT.parent,
                                capture_output=True,
                                text=True,
                                check=False)
        self.assertEqual(result.returncode, 1)
        self.assertIn('Usage:', result.stderr)
        self.assertNotIn('Not a directory', result.stderr)

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.log = self.root / 'docker.log'
        for name, body in {
                'uname':
                    '''#!/bin/bash
if [[ $1 == -s ]]; then echo "$TEST_OS"; else echo "$TEST_MACHINE"; fi
''',
                'docker':
                    '''#!/bin/bash
printf '%s\\n' "$*" >> "$TEST_DOCKER_LOG"
if [[ ${*: -1} == kfp-release:ci ]]; then
  echo "$TEST_RELEASE_PLATFORM"
else
  echo "$TEST_GENERATOR_PLATFORM"
fi
''',
        }.items():
            path = self.root / name
            path.write_text(body)
            path.chmod(0o755)

    def validate(self, architecture, **overrides):
        environment = {
            **os.environ,
            'PATH':
                f'{self.root}{os.pathsep}{os.environ["PATH"]}',
            'TEST_DOCKER_LOG':
                str(self.log),
            'TEST_OS':
                'Linux',
            'TEST_MACHINE':
                'aarch64' if architecture == 'arm64' else 'x86_64',
            'TEST_GENERATOR_PLATFORM':
                f'linux/{architecture}',
            'TEST_RELEASE_PLATFORM':
                f'linux/{architecture}',
            **overrides,
        }
        return subprocess.run([
            'bash', '-c', 'source "$1"; validate_native_images "$2"', 'test',
            str(SCRIPT), architecture
        ],
                              env=environment,
                              check=False,
                              capture_output=True,
                              text=True)

    def test_native_images_on_both_architectures(self):
        for architecture in ('amd64', 'arm64'):
            with self.subTest(architecture=architecture):
                result = self.validate(architecture)
                self.assertEqual(result.returncode, 0, result.stderr)
        commands = self.log.read_text()
        self.assertIn('kfp-api-generator:ci', commands)
        self.assertIn('kfp-release:ci', commands)

    def test_wrong_runner_architecture_fails_before_docker(self):
        result = self.validate('arm64', TEST_MACHINE='x86_64')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('native Linux arm64', result.stderr)
        self.assertFalse(self.log.exists())

    def test_non_linux_host_fails_before_docker(self):
        result = self.validate('arm64', TEST_OS='Darwin')
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.log.exists())

    def test_invalid_architecture_fails(self):
        result = self.validate('ppc64le')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Expected amd64 or arm64', result.stderr)

    def test_either_wrong_image_platform_fails(self):
        for variable in ('TEST_GENERATOR_PLATFORM', 'TEST_RELEASE_PLATFORM'):
            with self.subTest(variable=variable):
                result = self.validate('arm64', **{variable: 'linux/amd64'})
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('does not match native Linux arm64',
                              result.stderr)


class ScriptContractTests(unittest.TestCase):

    def test_shell_syntax(self):
        subprocess.run(['bash', '-n', str(SCRIPT)], check=True)

    def test_builds_unified_sdk_instead_of_standalone_client(self) -> None:
        """Both tooling images must check the new client and SDK artifact."""
        script = SCRIPT.read_text()
        self.assertIn('test -s sdk/python/kfp/server_api/api_client.py', script)
        self.assertIn(
            'python3 -m pip wheel --no-deps ./sdk/python '
            '--wheel-dir /tmp/kfp-smoke-dist', script)
        self.assertIn("compgen -G '/tmp/kfp-smoke-dist/kfp-*.whl'", script)
        self.assertNotIn('python_http_client/kfp_server_api/', script)
        self.assertNotIn('python_http_client/dist/', script)

    def test_real_generators_and_release_preparation_are_isolated(self):
        script = SCRIPT.read_text()
        self.assertIn('git -C "$root" archive HEAD', script)
        self.assertIn('backend/api/hack/generator.sh', script)
        self.assertIn('backend/api/build_kfp_server_api_python_package.sh',
                      script)
        self.assertIn('manifests/kustomize/hack/release.sh 3.0.0-smoke', script)
        self.assertIn('git-cliff -c cliff.toml', script)
        self.assertIn('git init -q', script)
        self.assertIn('--user "$(id -u):$(id -g)"', script)
        self.assertIn('--env HOME=/tmp/kfp-smoke-home', script)
        self.assertNotIn('--platform', script)
        self.assertNotIn('docker push', script)
        self.assertNotIn('git push', script)
        self.assertNotIn('git tag ', script)


if __name__ == '__main__':
    unittest.main()
