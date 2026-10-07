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
"""Exercise the shared requirements exporter against isolated workspaces."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import textwrap
import unittest

SCRIPT = Path(__file__).with_name('export_python_requirements.sh')
EXPORTS = ('requirements.txt', 'sdk/python/requirements.txt')


class ExportPythonRequirementsTest(unittest.TestCase):

    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name)
        self.workspace = self.root / 'workspace with spaces'
        (self.workspace / 'sdk/python').mkdir(parents=True)
        self.bin = self.root / 'bin'
        self.bin.mkdir()
        self.log = self.root / 'uv-calls.jsonl'
        uv = self.bin / 'uv'
        uv.write_text(f'#!{sys.executable}\n' + textwrap.dedent('''\
            import json
            import os
            from pathlib import Path
            import sys

            args = sys.argv[1:]
            with open(os.environ['UV_CALL_LOG'], 'a') as stream:
                stream.write(json.dumps({'args': args, 'cwd': os.getcwd()}) + '\\n')
            if os.environ.get('EXPORT_FAIL') == 'true':
                sys.exit(7)
            output = Path(args[args.index('-o') + 1])
            output.write_text('resolved==1.0\\n-e ./sdk/python\\n')
        '''))
        uv.chmod(0o755)
        self.env = {
            **os.environ,
            'PATH': f'{self.bin}{os.pathsep}{os.environ["PATH"]}',
            'UV_CALL_LOG': str(self.log),
        }

    def run_export(self, *arguments, cwd=None):
        return subprocess.run(
            ['bash', str(SCRIPT), *map(str, arguments)],
            cwd=cwd or self.root,
            env=self.env,
            text=True,
            capture_output=True,
            check=False)

    def calls(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()]

    def test_exports_only_frozen_runtime_requirements_in_selected_workspace(
            self):
        # A caller can use its trusted exporter even when the target has scripts.
        target_script = (
            self.workspace / '.github/resources/scripts' / SCRIPT.name)
        target_script.parent.mkdir(parents=True)
        target_script.write_text('exit 99\n')
        result = self.run_export(self.workspace)
        self.assertEqual(result.returncode, 0, result.stderr)
        prefix = ['export', '--frozen', '--no-dev', '--no-hashes']
        self.assertEqual(self.calls(), [
            {
                'args':
                    prefix + ['--format', 'requirements-txt', '-o', EXPORTS[0]],
                'cwd':
                    str(self.workspace.resolve())
            },
            {
                'args':
                    prefix + [
                        '--package', 'kfp', '--format', 'requirements-txt',
                        '-o', EXPORTS[1]
                    ],
                'cwd':
                    str(self.workspace.resolve())
            },
        ])
        self.assertEqual(target_script.read_text(), 'exit 99\n')
        for path in EXPORTS:
            self.assertEqual((self.workspace / path).read_text(),
                             'resolved==1.0\n-e ./sdk/python\n')

    def test_defaults_to_current_workspace_and_is_idempotent(self):
        first = self.run_export(cwd=self.workspace)
        self.assertEqual(first.returncode, 0, first.stderr)
        before = [(self.workspace / path).read_bytes() for path in EXPORTS]
        second = self.run_export(cwd=self.workspace)
        self.assertEqual(second.returncode, 0, second.stderr)
        self.assertEqual(
            before, [(self.workspace / path).read_bytes() for path in EXPORTS])

    def test_export_failure_stops_before_second_file(self):
        for path in EXPORTS:
            (self.workspace / path).write_text('original\n')
        self.env['EXPORT_FAIL'] = 'true'
        result = self.run_export(self.workspace)
        self.assertEqual(result.returncode, 7)
        self.assertEqual(len(self.calls()), 1)
        for path in EXPORTS:
            self.assertEqual((self.workspace / path).read_text(), 'original\n')

    def test_invalid_workspace_fails_before_uv(self):
        result = self.run_export(self.root / 'missing')
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.log.exists())

    def test_extra_arguments_fail_before_uv(self):
        result = self.run_export(self.workspace, 'unexpected')
        self.assertEqual(result.returncode, 2)
        self.assertIn('Usage:', result.stderr)
        self.assertFalse(self.log.exists())


if __name__ == '__main__':
    unittest.main()
