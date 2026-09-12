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

import functools
import os
import tempfile
import unittest
from click import testing
from kfp.cli import cli


class TestCliMigrate(unittest.TestCase):

    def setUp(self):
        runner = testing.CliRunner()
        self.invoke = functools.partial(
            runner.invoke, cli=cli.cli, catch_exceptions=False, obj={})

    def test_migrate_help(self):
        result = self.invoke(args=['migrate', '--help'])
        self.assertEqual(result.exit_code, 0)
        self.assertIn("Migrates KFP v1 SDK code to v2 SDK code.", result.output)

    def test_migrate_dry_run_file(self):
        with tempfile.TemporaryDirectory() as tempdir:
            source_file = os.path.join(tempdir, "pipeline.py")
            with open(source_file, "w", encoding="utf-8") as f:
                f.write("from kfp.v2 import dsl\n")
            
            result = self.invoke(args=['migrate', source_file])
            self.assertEqual(result.exit_code, 0)
            self.assertIn("Running dry-run for file.", result.output)
            self.assertIn("-from kfp.v2 import dsl", result.output)
            self.assertIn("+from kfp import dsl", result.output)

    def test_migrate_dry_run_directory(self):
        with tempfile.TemporaryDirectory() as tempdir:
            src_dir = os.path.join(tempdir, "src")
            os.makedirs(src_dir)
            
            file1 = os.path.join(src_dir, "pipeline.py")
            with open(file1, "w", encoding="utf-8") as f:
                f.write("from kfp.v2 import dsl\n")
            
            result = self.invoke(args=['migrate', src_dir])
            self.assertEqual(result.exit_code, 0)
            self.assertIn("Running dry-run for directory recursively.", result.output)
            self.assertIn("-from kfp.v2 import dsl", result.output)
            self.assertIn("+from kfp import dsl", result.output)

    def test_migrate_inplace_file(self):
        with tempfile.TemporaryDirectory() as tempdir:
            source_file = os.path.join(tempdir, "pipeline.py")
            with open(source_file, "w", encoding="utf-8") as f:
                f.write("from kfp.v2 import dsl\n")
            
            result = self.invoke(args=['migrate', '-i', source_file])
            self.assertEqual(result.exit_code, 0)
            self.assertIn("File migration completed successfully.", result.output)
            
            with open(source_file, "r", encoding="utf-8") as f:
                content = f.read()
            self.assertEqual(content, "from kfp import dsl\n")

    def test_migrate_output_path_file(self):
        with tempfile.TemporaryDirectory() as tempdir:
            source_file = os.path.join(tempdir, "pipeline.py")
            output_file = os.path.join(tempdir, "output.py")
            with open(source_file, "w", encoding="utf-8") as f:
                f.write("from kfp.v2 import dsl\n")
            
            result = self.invoke(args=['migrate', '-o', output_file, source_file])
            self.assertEqual(result.exit_code, 0)
            self.assertIn("File migration completed successfully.", result.output)
            
            with open(output_file, "r", encoding="utf-8") as f:
                content = f.read()
            self.assertEqual(content, "from kfp import dsl\n")


if __name__ == '__main__':
    unittest.main()
