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
"""Check compiler validation covers nested and tool-only modules."""

from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

import release_cve_validate_go as validation


class ValidateGoTest(unittest.TestCase):

    def test_all_modules_compile_and_tool_only_modules_verify(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory)
            modules = ['go.mod', 'api/go.mod', 'backend/api/tools/go.mod']
            for name in modules:
                path = source / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text('module example.test/fixture\n')
            commands = []

            def command(args, cwd):
                commands.append((args, cwd))
                output = ''
                if args[0] == 'git':
                    output = '\0'.join(modules) + '\0'
                elif args[:2] == ['go', 'list'
                                 ] and cwd != source / 'backend/api/tools':
                    output = 'example.test/fixture\n'
                return subprocess.CompletedProcess(args, 0, output, '')

            with mock.patch.object(validation, 'run', side_effect=command):
                validation.validate(source)
            for name in ('', 'api'):
                self.assertIn(
                    (['go', 'test', '-mod=readonly', '-run=^$', './...'
                     ], source / name), commands)
            self.assertIn(
                (['go', 'mod', 'verify'], source / 'backend/api/tools'),
                commands)
            self.assertIn(
                (['go', 'test', '-mod=readonly', './backend/src/...'], source),
                commands)

    def test_module_failure_stops_validation(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory)
            (source / 'go.mod').write_text('module example.test/fixture\n')
            with mock.patch.object(
                    validation,
                    'run',
                    side_effect=[
                        subprocess.CompletedProcess([], 0, 'go.mod\0', ''),
                        subprocess.CalledProcessError(1, ['go', 'list'])
                    ]), self.assertRaises(subprocess.CalledProcessError):
                validation.validate(source)


if __name__ == '__main__':
    unittest.main()
