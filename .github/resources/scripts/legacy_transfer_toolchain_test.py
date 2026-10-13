#!/usr/bin/env python3

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

from pathlib import Path
import subprocess
import tempfile
import unittest

import yaml


class SourceToolchainTest(unittest.TestCase):

    def test_compiler_bootstrap_preserves_cgo_source_build(self):
        root = Path(__file__).resolve().parents[3]
        workflow = yaml.safe_load(
            (root / '.github/workflows/legacy-native-transfer.yml').read_text())
        steps = workflow['jobs']['source']['steps']
        build = next(i for i, step in enumerate(steps)
                     if 'CGO_ENABLED=1 go build' in step.get('run', ''))
        bootstrap = next(step for step in steps[:build]
                         if 'build-essential' in step.get('run', ''))
        self.assertLessEqual(bootstrap['timeout-minutes'], 5)
        for available in (False, True):
            with self.subTest(available=available), tempfile.TemporaryDirectory(
            ) as directory:
                script = Path(directory) / 'probe.sh'
                script.write_text(
                    'set -eu\n'
                    f'compiler_available={int(available)}\n'
                    'command() { test "$compiler_available" = 1; }\n'
                    'sudo() { printf "%s\\n" "$*"; compiler_available=1; }\n'
                    'gcc() { test "$compiler_available" = 1; }\n' +
                    bootstrap['run'])
                result = subprocess.run(['bash', str(script)],
                                        capture_output=True,
                                        text=True,
                                        check=False)
                self.assertEqual(result.returncode, 0, result.stderr)
                if available:
                    self.assertEqual(result.stdout, '')
                else:
                    self.assertIn('Acquire::Retries=3', result.stdout)
                    self.assertIn(
                        'install --yes --no-install-recommends build-essential',
                        result.stdout)


if __name__ == '__main__':
    unittest.main()
