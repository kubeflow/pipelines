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

import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

import yaml

ACTION = Path(__file__).parents[2] / 'actions/setup-osv-scanner/action.yml'


class SetupOsvScannerTest(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.binaries = self.root / 'bin'
        self.binaries.mkdir()
        for command in ('mktemp', 'chmod', 'grep'):
            (self.binaries / command).symlink_to(shutil.which(command))
        self.environment = {
            **os.environ,
            'PATH': str(self.binaries),
            'RUNNER_TEMP': str(self.root),
            'GITHUB_PATH': str(self.root / 'path'),
            'CALLS': str(self.root / 'calls'),
            'ARCH': 'amd64',
        }
        self.command(
            'curl',
            'echo "download:$*" >> "$CALLS"; '
            '[[ "${FAIL_DOWNLOAD:-0}" == 0 ]] || exit 22; '
            'printf \'#!/bin/bash\\necho executed >> "$CALLS"\\necho "osv-scanner version: ${TEST_VERSION:-2.5.0}"\\n\' > "${@: -1}"',
        )
        self.command(
            'sha256sum',
            'read -r digest binary; '
            'echo "verify:$digest:$*" >> "$CALLS"; '
            '[[ "${FAIL_CHECKSUM:-0}" == 0 ]]',
        )

    def command(self, name, body):
        path = self.binaries / name
        path.write_text('#!/bin/bash\nset -e\n' + body + '\n')
        path.chmod(0o755)

    def run_action(self):
        script = yaml.safe_load(ACTION.read_text())['runs']['steps'][0]['run']
        return subprocess.run(
            ['/bin/bash', '-c', script],
            env=self.environment,
            capture_output=True,
            text=True,
            check=False,
        )

    def test_supported_binaries_are_verified_before_execution(self):
        pins = {
            'amd64': 'edcfc41d257db36148f065055655fe3fcfc434b0b423ea67468a84c207524e0c',
            'arm64': 'fe152e1a546af223e6c557cc3111a8bb3e5dc02fcbf7dbe95d26567c0f0041f2',
        }
        for architecture, checksum in pins.items():
            with self.subTest(architecture=architecture):
                self.environment['ARCH'] = architecture
                result = self.run_action()
                self.assertEqual(result.returncode, 0, result.stderr)
                calls = (self.root / 'calls').read_text()
                self.assertIn(
                    f'/v2.5.0/osv-scanner_linux_{architecture}', calls
                )
                self.assertIn(f'verify:{checksum}:--check --strict', calls)
                self.assertLess(
                    calls.rindex('verify:'), calls.rindex('executed')
                )
                installed = Path(
                    (self.root / 'path').read_text().splitlines()[-1]
                )
                self.assertTrue((installed / 'osv-scanner').is_file())

    def test_bad_checksum_never_executes_or_publishes(self):
        self.environment['FAIL_CHECKSUM'] = '1'
        self.assertNotEqual(self.run_action().returncode, 0)
        self.assertNotIn('executed', (self.root / 'calls').read_text())
        self.assertFalse((self.root / 'path').exists())

    def test_download_failure_never_verifies_or_publishes(self):
        self.environment['FAIL_DOWNLOAD'] = '1'
        self.assertNotEqual(self.run_action().returncode, 0)
        self.assertNotIn('verify:', (self.root / 'calls').read_text())
        self.assertFalse((self.root / 'path').exists())

    def test_wrong_version_never_publishes(self):
        self.environment['TEST_VERSION'] = '2.5.01'
        self.assertNotEqual(self.run_action().returncode, 0)
        self.assertFalse((self.root / 'path').exists())

    def test_unsupported_architecture_never_downloads(self):
        self.environment['ARCH'] = 'riscv64'
        self.assertEqual(self.run_action().returncode, 2)
        self.assertFalse((self.root / 'calls').exists())
        self.assertFalse((self.root / 'path').exists())


if __name__ == '__main__':
    unittest.main()
