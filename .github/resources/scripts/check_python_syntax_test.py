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
import unittest
from unittest import mock

from check_python_syntax import syntax_errors
from check_python_syntax import repository_python_files

ROOT = Path(__file__).resolve().parents[3]
PYTHON_SYNTAX_WORKFLOW_PATH = ROOT / '.github/workflows/python-syntax.yml'


class CheckPythonSyntaxTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.python_syntax_workflow = PYTHON_SYNTAX_WORKFLOW_PATH.read_text(
            encoding='utf-8')

    def test_reports_only_invalid_python(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            valid_path = root / 'valid.py'
            invalid_path = root / 'invalid.py'
            valid_path.write_text('value = 1\n', encoding='utf-8')
            invalid_path.write_text('ddef invalid():\n    pass\n',
                                    encoding='utf-8')

            errors = syntax_errors([valid_path, invalid_path])

        self.assertEqual(len(errors), 1)
        self.assertEqual(errors[0][0], invalid_path)
        self.assertEqual(errors[0][2], 1)

    def test_reports_null_bytes_as_syntax_errors(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            invalid_path = Path(temporary_directory) / 'invalid.py'
            invalid_path.write_bytes(b'value = 1\0\n')

            errors = syntax_errors([invalid_path])

        self.assertEqual(len(errors), 1)
        self.assertEqual(errors[0][0], invalid_path)
        self.assertIn('null byte', errors[0][1])

    def test_lists_only_tracked_python_files(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            subprocess.run(['git', 'init', '--quiet', str(root)], check=True)
            tracked_path = root / 'tracked.py'
            untracked_path = root / 'untracked.py'
            tracked_path.touch()
            untracked_path.touch()
            subprocess.run(
                ['git', '-C', str(root), 'add', 'tracked.py'],
                check=True,
            )

            paths = repository_python_files(root)

        self.assertEqual(paths, [tracked_path])

    def test_decodes_non_utf8_git_paths_with_filesystem_encoding(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            encoded_name = b'\xff.py'
            git_result = subprocess.CompletedProcess(
                args=['git'], returncode=0, stdout=encoded_name + b'\0'
            )
            with mock.patch(
                'check_python_syntax.subprocess.run', return_value=git_result
            ), mock.patch.object(Path, 'is_file', return_value=True):
                paths = repository_python_files(root)

        self.assertEqual(len(paths), 1)
        self.assertEqual(os.fsencode(paths[0].name), encoded_name)

    def test_ci_checks_syntax_for_every_python_change(self):
        self.assertIn("      - '**/*.py'", self.python_syntax_workflow)
        self.assertIn('name: Check Python 3.9 syntax',
                      self.python_syntax_workflow)
        self.assertIn("python-version: '3.9'", self.python_syntax_workflow)
        self.assertIn(
            'run: python3 .github/resources/scripts/check_python_syntax.py',
            self.python_syntax_workflow,
        )


if __name__ == '__main__':
    unittest.main()
