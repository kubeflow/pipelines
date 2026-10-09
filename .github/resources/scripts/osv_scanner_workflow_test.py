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

import hashlib
import os
from pathlib import Path
import subprocess
import sys
from tempfile import TemporaryDirectory
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW_PATH = ROOT / '.github/workflows/osv-scanner.yml'
CI_SCRIPTS_WORKFLOW_PATH = ROOT / '.github/workflows/ci-scripts-tests.yml'
SETUP_ACTION_PATH = ROOT / '.github/actions/setup-osv-scanner/action.yml'


class OsvScannerWorkflowTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.workflow = WORKFLOW_PATH.read_text(encoding='utf-8')
        cls.ci_scripts_workflow = CI_SCRIPTS_WORKFLOW_PATH.read_text(
            encoding='utf-8')

    def test_scanner_release_uses_shared_pinned_installer(self):
        self.assertEqual(
            self.workflow.count('uses: ./.github/actions/setup-osv-scanner'), 2)
        self.assertNotIn('OSV_SCANNER_VERSION', self.workflow)
        self.assertNotIn('./osv-scanner', self.workflow)
        action = yaml.safe_load(SETUP_ACTION_PATH.read_text())
        self.assertEqual(action['runs']['using'], 'composite')
        setup = action['runs']['steps'][0]
        self.assertEqual(
            setup['env'], {
                'OSV_SCANNER_VERSION':
                    '2.5.0',
                'OSV_SCANNER_SHA256_AMD64':
                    'edcfc41d257db36148f065055655fe3fcfc434b0b423ea67468a84c207524e0c',
                'OSV_SCANNER_SHA256_ARM64':
                    'fe152e1a546af223e6c557cc3111a8bb3e5dc02fcbf7dbe95d26567c0f0041f2',
            })
        self.assertIn('sha256sum --check --strict', setup['run'])

    def test_scan_recursively_reports_all_supported_dependencies(self):
        self.assertIn('osv-scanner scan source', self.workflow)
        self.assertIn('            --recursive', self.workflow)
        self.assertIn('            --no-resolve', self.workflow)
        self.assertIn(
            '            --experimental-exclude backend/api/v2beta1/python_http_client',
            self.workflow,
        )
        self.assertIn('            --format sarif', self.workflow)
        self.assertIn('            --output-file osv-results.sarif',
                      self.workflow)
        self.assertIn('            . || scan_exit_code=$?', self.workflow)
        self.assertIn(
            '"${scan_exit_code}" -ne 0 && "${scan_exit_code}" -ne 1',
            self.workflow,
        )

    def test_scan_has_least_privilege_and_manual_dispatch(self):
        self.assertIn('  push:', self.workflow)
        self.assertIn('      - master', self.workflow)
        self.assertIn('  workflow_dispatch:', self.workflow)
        self.assertIn('  contents: read', self.workflow)
        self.assertIn('  security-events: write', self.workflow)
        self.assertEqual(self.workflow.count('      security-events: write'), 2)
        self.assertIn('          persist-credentials: false', self.workflow)
        self.assertNotIn('contents: write', self.workflow)
        self.assertNotIn('pull-requests: write', self.workflow)

    def test_deployed_images_are_discovered_and_scanned(self):
        self.assertIn("KUSTOMIZE_VERSION: '5.8.1'", self.workflow)
        self.assertIn(
            "KUSTOMIZE_SHA256: '029a7f0f4e1932c52a0476cf02a0fd855c0bb85694b82c338fc648dcb53a819d'",
            self.workflow,
        )
        self.assertIn('osv_manifest_images.py', self.workflow)
        self.assertIn('--overlay manifests/kustomize/env/platform-agnostic',
                      self.workflow)
        self.assertIn(
            '--overlay manifests/kustomize/env/platform-agnostic-multi-user',
            self.workflow,
        )
        self.assertIn('osv-scanner scan image "${IMAGE}"', self.workflow)
        self.assertIn('docker-pull-with-retry.sh "${IMAGE}"', self.workflow)
        self.assertIn(
            'category: kubeflow-pipelines-osv-image-${{ matrix.category }}',
            self.workflow)
        self.assertIn('      fail-fast: false', self.workflow)

    def test_exact_duplicate_results_are_removed_before_upload(self):
        self.assertEqual(self.workflow.count('deduplicate_sarif.py'), 2)
        self.assertIn('            --input osv-results.sarif', self.workflow)
        self.assertIn('            --output osv-results-deduplicated.sarif',
                      self.workflow)
        self.assertIn('            --input osv-image-results.sarif',
                      self.workflow)
        self.assertIn(
            '            --output osv-image-results-deduplicated.sarif',
            self.workflow,
        )
        self.assertIn('          sarif_file: osv-results-deduplicated.sarif',
                      self.workflow)
        self.assertIn(
            '          sarif_file: osv-image-results-deduplicated.sarif',
            self.workflow,
        )

    def test_ci_scripts_tests_run_for_osv_workflow_changes(self):
        self.assertIn(
            "      - '.github/actions/setup-osv-scanner/**'",
            self.ci_scripts_workflow,
        )
        self.assertIn(
            "      - '.github/workflows/osv-scanner.yml'",
            self.ci_scripts_workflow,
        )


class SetupOsvScannerTest(unittest.TestCase):

    def setUp(self):
        self.temp_dir = TemporaryDirectory()
        self.addCleanup(self.temp_dir.cleanup)
        self.root = Path(self.temp_dir.name)
        self.runner_temp = self.root / 'runner temp'
        self.runner_temp.mkdir()
        self.stub_bin = self.root / 'stub-bin'
        self.stub_bin.mkdir()
        self.github_path = self.root / 'github-path'
        self.github_path.touch()
        self.curl_log = self.root / 'curl.log'
        self.scanner_log = self.root / 'scanner.log'
        self.payload = self.root / 'scanner-payload'
        self.payload.write_text(
            '#!/usr/bin/env bash\n'
            'printf "%s\\n" "$*" >> "$SCANNER_LOG"\n'
            'exit "${SCANNER_EXIT_CODE:-0}"\n',
            encoding='utf-8')
        curl = self.stub_bin / 'curl'
        curl.write_text(
            '#!/usr/bin/env bash\n'
            'set -eu\n'
            'printf "%s\\n" "$@" > "$CURL_LOG"\n'
            'if [[ "${CURL_EXIT_CODE:-0}" != 0 ]]; then\n'
            '  exit "$CURL_EXIT_CODE"\n'
            'fi\n'
            'while [[ "$#" -gt 0 ]]; do\n'
            '  if [[ "$1" == --output ]]; then\n'
            '    cp "$SCANNER_PAYLOAD" "$2"\n'
            '    exit 0\n'
            '  fi\n'
            '  shift\n'
            'done\n'
            'exit 2\n',
            encoding='utf-8')
        curl.chmod(0o755)
        # BSD sha256sum lacks --strict; use the same checksum contract on all
        # development hosts while still validating the downloaded bytes.
        checksum = self.stub_bin / 'sha256sum'
        checksum.write_text(
            f'#!{sys.executable}\n'
            'import hashlib, pathlib, sys\n'
            'assert sys.argv[1:] == ["--check", "--strict"]\n'
            'expected, name = sys.stdin.read().rstrip("\\n").split("  ", 1)\n'
            'actual = hashlib.sha256(pathlib.Path(name).read_bytes()).hexdigest()\n'
            'print("OK" if expected == actual else "FAILED")\n'
            'sys.exit(0 if expected == actual else 1)\n',
            encoding='utf-8')
        checksum.chmod(0o755)
        self.step = yaml.safe_load(
            SETUP_ACTION_PATH.read_text())['runs']['steps'][0]

    def run_installer(self, architecture, valid_checksum=True, **extra_env):
        env = os.environ.copy()
        env.update(self.step['env'])
        env.update({
            'PATH': f'{self.stub_bin}{os.pathsep}{env["PATH"]}',
            'RUNNER_OS': 'Linux',
            'RUNNER_ARCH': architecture,
            'RUNNER_TEMP': str(self.runner_temp),
            'GITHUB_PATH': str(self.github_path),
            'CURL_LOG': str(self.curl_log),
            'SCANNER_LOG': str(self.scanner_log),
            'SCANNER_PAYLOAD': str(self.payload),
        })
        # Substitute only the selected architecture's fixture checksum so the
        # checksum verifier checks bytes without downloading release binaries.
        if valid_checksum and architecture in ('X64', 'ARM64'):
            suffix = 'AMD64' if architecture == 'X64' else 'ARM64'
            env[f'OSV_SCANNER_SHA256_{suffix}'] = hashlib.sha256(
                self.payload.read_bytes()).hexdigest()
        env.update(extra_env)
        return subprocess.run(['bash', '-c', self.step['run']],
                              cwd=self.root,
                              env=env,
                              capture_output=True,
                              text=True,
                              check=False)

    def test_native_architectures_install_verified_binary_on_path(self):
        for architecture, asset_arch in (('X64', 'amd64'), ('ARM64', 'arm64')):
            with self.subTest(architecture=architecture):
                result = self.run_installer(architecture)
                self.assertEqual(result.returncode, 0, result.stderr)
                expected_url = (
                    'https://github.com/google/osv-scanner/releases/download/'
                    f'v{self.step["env"]["OSV_SCANNER_VERSION"]}/'
                    f'osv-scanner_linux_{asset_arch}')
                self.assertIn(expected_url,
                              self.curl_log.read_text().splitlines())
                binary_dir = Path(self.github_path.read_text().splitlines()[-1])
                self.assertEqual(binary_dir.parent, self.runner_temp)
                self.assertEqual(binary_dir.stat().st_mode & 0o777, 0o700)
                self.assertTrue(os.access(binary_dir / 'osv-scanner', os.X_OK))
                self.assertIn('--version',
                              self.scanner_log.read_text().splitlines())

    def test_checksum_mismatch_does_not_execute_or_publish_binary(self):
        for architecture in ('X64', 'ARM64'):
            with self.subTest(architecture=architecture):
                result = self.run_installer(architecture, valid_checksum=False)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('FAILED', result.stdout)
                self.assertFalse(self.scanner_log.exists())
                self.assertEqual(self.github_path.read_text(), '')

    def test_unsupported_architecture_fails_before_download(self):
        result = self.run_installer('ARM')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Unsupported OSV-Scanner runner architecture: ARM',
                      result.stdout)
        self.assertFalse(self.curl_log.exists())
        self.assertEqual(self.github_path.read_text(), '')

    def test_unsupported_os_fails_before_download(self):
        result = self.run_installer('ARM64', RUNNER_OS='macOS')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('requires a Linux runner', result.stdout)
        self.assertFalse(self.curl_log.exists())
        self.assertEqual(self.github_path.read_text(), '')

    def test_download_failure_does_not_execute_or_publish_binary(self):
        result = self.run_installer('X64', CURL_EXIT_CODE='22')
        self.assertEqual(result.returncode, 22)
        self.assertFalse(self.scanner_log.exists())
        self.assertEqual(self.github_path.read_text(), '')

    def test_binary_execution_failure_does_not_add_to_path(self):
        result = self.run_installer('X64', SCANNER_EXIT_CODE='126')
        self.assertEqual(result.returncode, 126)
        self.assertEqual(self.github_path.read_text(), '')


if __name__ == '__main__':
    unittest.main()
