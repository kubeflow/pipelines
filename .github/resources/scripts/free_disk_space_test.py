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
import subprocess
import tempfile
import textwrap
import unittest

SCRIPT = (
    Path(__file__).parents[2] / 'actions' / 'github-disk-cleanup' /
    'free-disk-space.sh')


class FreeDiskSpaceTest(unittest.TestCase):

    def _run(
        self,
        available_gib: int,
        *,
        minimum_gib: str = '60',
        github_actions: str = 'true',
    ) -> tuple[subprocess.CompletedProcess[str], str]:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            fake_bin = root / 'bin'
            fake_bin.mkdir()
            command_log = root / 'commands.log'

            (fake_bin / 'df').write_text(
                textwrap.dedent(f'''\
                    #!/usr/bin/env bash
                    if [[ "$*" == "-Pk /" ]]; then
                      echo "Filesystem 1024-blocks Used Available Capacity Mounted on"
                      echo "/dev/root 152043520 1 {available_gib * 1024 * 1024} 1% /"
                    else
                      echo "Filesystem Size Used Avail Use% Mounted on"
                      echo "/dev/root 145G 1G {available_gib}G 1% /"
                    fi
                '''),
                encoding='utf-8',
            )
            for command in ('sudo', 'docker'):
                (fake_bin / command).write_text(
                    textwrap.dedent(f'''\
                        #!/usr/bin/env bash
                        echo "{command} $*" >> "$COMMAND_LOG"
                    '''),
                    encoding='utf-8',
                )
            for executable in fake_bin.iterdir():
                executable.chmod(0o755)

            environment = os.environ.copy()
            environment.update({
                'COMMAND_LOG': str(command_log),
                'GITHUB_ACTIONS': github_actions,
                'MIN_FREE_SPACE_GIB': minimum_gib,
                'PATH': f'{fake_bin}{os.pathsep}{environment["PATH"]}',
            })
            result = subprocess.run(
                ['bash', str(SCRIPT)],
                capture_output=True,
                text=True,
                check=False,
                env=environment,
            )
            commands = (
                command_log.read_text(
                    encoding='utf-8') if command_log.exists() else '')
            return result, commands

    def test_skips_cleanup_with_sufficient_headroom(self):
        result, commands = self._run(89)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('Skipping disk cleanup: 89 GiB available', result.stdout)
        self.assertEqual(commands, '')

    def test_runs_cleanup_below_threshold(self):
        result, commands = self._run(40)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('Only 40 GiB available; running disk cleanup',
                      result.stdout)
        self.assertIn('sudo rm -rf /usr/share/dotnet', commands)
        self.assertIn('sudo apt-get autoremove -y', commands)
        self.assertIn('docker system prune -af --volumes', commands)
        self.assertIn('sudo systemctl stop containerd', commands)

    def test_rejects_invalid_threshold(self):
        result, commands = self._run(89, minimum_gib='sixty')

        self.assertEqual(result.returncode, 1)
        self.assertIn('MIN_FREE_SPACE_GIB must be', result.stderr)
        self.assertEqual(commands, '')

    def test_refuses_to_run_outside_github_actions(self):
        result, commands = self._run(40, github_actions='false')

        self.assertEqual(result.returncode, 1)
        self.assertIn('for GitHub Actions runners only', result.stdout)
        self.assertEqual(commands, '')


if __name__ == '__main__':
    unittest.main()
