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

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import textwrap
import unittest

SCRIPT = Path(
    __file__).resolve().parents[3] / 'test/presubmit-test-sdk-upgrade.sh'


class SdkUpgradeTest(unittest.TestCase):

    def test_isolated_upgrade_resolves_local_wheels_and_cleans_up(self):
        """Exercise successful and failed upgrades without downloading
        packages."""
        for fail_upgrade in (False, True):
            with self.subTest(fail_upgrade=fail_upgrade
                             ), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                fake_bin = root / 'bin'
                fake_bin.mkdir()
                (root / 'api').mkdir()
                log = root / 'commands.jsonl'
                program = f'#!{sys.executable}\n' + textwrap.dedent('''\
                    import json
                    import os
                    from pathlib import Path
                    import shutil
                    import sys

                    executable = Path(sys.argv[0])
                    args = sys.argv[1:]
                    with open(os.environ['UPGRADE_COMMAND_LOG'], 'a') as log:
                        log.write(json.dumps([str(executable), args]) + '\\n')
                    if args[:2] == ['-m', 'venv']:
                        target = Path(args[2]) / 'bin/python'
                        target.parent.mkdir(parents=True)
                        shutil.copy(executable, target)
                    elif executable.name == 'uv':
                        package = args[args.index('--package') + 1]
                        output = Path(args[args.index('--out-dir') + 1])
                        output.mkdir(exist_ok=True)
                        (output / (package.replace('-', '_') + '-HEAD.whl')).touch()
                    elif args[:3] == ['-m', 'pip', 'show']:
                        print('Version: 2.15.2')
                    elif args[:3] == ['-m', 'pip', 'install']:
                        if any(arg.endswith('.whl') for arg in args):
                            sys.exit(int(os.environ['FAIL_UPGRADE']))
                ''')
                for name in ('python3', 'uv', 'make'):
                    executable = fake_bin / name
                    executable.write_text(program, encoding='utf-8')
                    executable.chmod(0o755)
                environment = {
                    **os.environ,
                    'PATH':
                        f'{fake_bin}{os.pathsep}{os.environ["PATH"]}',
                    'UPGRADE_COMMAND_LOG':
                        str(log),
                    'FAIL_UPGRADE':
                        str(int(fail_upgrade)),
                    'TMPDIR':
                        str(root),
                }
                result = subprocess.run(
                    ['bash', str(SCRIPT)],
                    cwd=root,
                    env=environment,
                    capture_output=True,
                    text=True,
                    check=False,
                )
                self.assertEqual(result.returncode, int(fail_upgrade),
                                 result.stderr)
                commands = [
                    json.loads(line) for line in log.read_text().splitlines()
                ]
                self.assertEqual(commands[0][1][:2], ['-m', 'venv'])
                venv = Path(commands[0][1][2])
                pip_commands = [(executable, args)
                                for executable, args in commands
                                if args[:2] == ['-m', 'pip']]
                self.assertTrue(
                    all(executable == str(venv / 'bin/python')
                        for executable, _ in pip_commands))
                self.assertEqual(pip_commands[1][1],
                                 ['-m', 'pip', 'install', 'kfp'])
                upgrades = [
                    args for _, args in pip_commands
                    if '--force-reinstall' in args
                ]
                self.assertEqual(len(upgrades), 1)
                self.assertCountEqual(
                    [
                        Path(arg).name
                        for arg in upgrades[0]
                        if arg.endswith('.whl')
                    ],
                    [
                        'kfp-HEAD.whl', 'kfp_pipeline_spec-HEAD.whl',
                        'kfp_server_api-HEAD.whl'
                    ],
                )
                self.assertEqual(
                    any(args == ['-c', 'import kfp'] for _, args in commands),
                    not fail_upgrade,
                )
                self.assertFalse(venv.parent.exists())


if __name__ == '__main__':
    unittest.main()
