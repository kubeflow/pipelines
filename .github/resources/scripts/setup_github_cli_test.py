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
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

import yaml

SCRIPTS = Path(__file__).parent


class SetupGitHubCliTest(unittest.TestCase):

    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        self.binaries = self.root / 'bin'
        self.binaries.mkdir()
        for command in ('bash', 'mktemp', 'rm', 'mkdir'):
            (self.binaries / command).symlink_to(shutil.which(command))
        self.environment = {
            **os.environ,
            'PATH': str(self.binaries),
            'RUNNER_TEMP': str(self.root),
            'GITHUB_PATH': str(self.root / 'path'),
            'CALLS': str(self.root / 'calls'),
            'TEST_ARCH': 'x86_64',
            'GITHUB_REPOSITORY': 'kubeflow/pipelines',
            'GITHUB_RUN_ID': '123',
        }
        self.command(
            'uname', 'if [[ "$1" == -s ]]; then echo Linux; '
            'else echo "$TEST_ARCH"; fi')
        self.command(
            'curl', 'echo "download:$*" >> "$CALLS"; '
            '[[ "${FAIL_DOWNLOAD:-0}" == 0 ]] || exit 22; '
            'printf archive > "${@: -1}"')
        self.command(
            'sha256sum', 'read -r digest archive; '
            'echo "verify:$digest" >> "$CALLS"; '
            '[[ "${FAIL_CHECKSUM:-0}" == 0 ]]')
        self.command(
            'tar', 'echo extract >> "$CALLS"; '
            'mkdir -p "$4/bin"; '
            'printf "#!/bin/bash\\necho gh-test\\n" > "$4/bin/gh"; '
            '/bin/chmod +x "$4/bin/gh"')

    def command(self, name, body):
        path = self.binaries / name
        path.write_text('#!/bin/bash\nset -e\n' + body + '\n')
        path.chmod(0o755)

    def run_script(self, name='setup-github-cli.sh'):
        return subprocess.run(['/bin/bash', str(SCRIPTS / name)],
                              env=self.environment,
                              capture_output=True,
                              text=True,
                              check=False)

    def test_existing_cli_is_used_without_download_or_path_change(self):
        self.command('gh', 'echo existing-gh')
        result = self.run_script()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('existing-gh', result.stdout)
        self.assertFalse((self.root / 'calls').exists())
        self.assertFalse((self.root / 'path').exists())

    def test_supported_archives_are_verified_before_publication(self):
        pins = {
            'x86_64': (
                'amd64',
                'bb766f710eef8ede859c18578c72c327597cd4c8a85b06001b1f3843c6019386'
            ),
            'aarch64': (
                'arm64',
                '7862c86c72f43df3a2d93ddde6f473285b4e2af61b494849846827e513ef6484'
            ),
        }
        for architecture, (asset, checksum) in pins.items():
            with self.subTest(architecture=architecture):
                self.environment['TEST_ARCH'] = architecture
                result = self.run_script()
                self.assertEqual(result.returncode, 0, result.stderr)
                calls = (self.root / 'calls').read_text()
                self.assertIn(f'gh_2.102.0_linux_{asset}.tar.gz', calls)
                self.assertIn(f'verify:{checksum}', calls)
                self.assertLess(
                    calls.rindex('verify:'), calls.rindex('extract'))
                directory = Path(
                    (self.root / 'path').read_text().splitlines()[-1])
                self.assertTrue((directory / 'gh').is_file())

    def test_bad_checksum_never_extracts_or_publishes(self):
        self.environment['FAIL_CHECKSUM'] = '1'
        result = self.run_script()
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn('extract', (self.root / 'calls').read_text())
        self.assertFalse((self.root / 'path').exists())
        self.assertEqual(list(self.root.glob('kfp-gh.*')), [])

    def test_failed_download_never_verifies_or_publishes(self):
        self.environment['FAIL_DOWNLOAD'] = '1'
        result = self.run_script()
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn('verify', (self.root / 'calls').read_text())
        self.assertFalse((self.root / 'path').exists())

    def test_unsupported_architecture_fails_before_download(self):
        self.environment['TEST_ARCH'] = 'riscv64'
        result = self.run_script()
        self.assertEqual(result.returncode, 2)
        self.assertFalse((self.root / 'calls').exists())

    def test_missing_cli_fails_before_polling(self):
        self.command('sleep', 'echo unexpected-sleep >> "$CALLS"; exit 99')
        result = self.run_script('wait-for-image-artifacts.sh')
        self.assertEqual(result.returncode, 2)
        self.assertIn('setup-github-cli.sh', result.stderr)
        self.assertFalse((self.root / 'calls').exists())

    def test_deploy_provisions_cli_before_polling(self):
        action = yaml.safe_load((SCRIPTS.parents[1] / 'actions' / 'deploy' /
                                 'action.yml').read_text())
        commands = [step.get('run', '') for step in action['runs']['steps']]
        setup = next(i for i, command in enumerate(commands)
                     if 'setup-github-cli.sh' in command)
        polling = next(i for i, command in enumerate(commands)
                       if 'wait-for-image-artifacts.sh' in command)
        self.assertLess(setup, polling)


if __name__ == '__main__':
    unittest.main()
