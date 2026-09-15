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
import tempfile
import unittest
from kfp.compiler import migration


class TestMigration(unittest.TestCase):

    def test_migrate_v2_imports(self):
        code = (
            "from kfp.v2 import dsl\n"
            "from kfp.v2 import compiler\n"
            "from kfp.v2.dsl import component\n"
            "from kfp.v2.compiler import Compiler\n"
            "import kfp.v2 as kfp_v2\n"
            "import kfp.v2.dsl as dsl_v2\n"
        )
        expected = (
            "from kfp import dsl\n"
            "from kfp import compiler\n"
            "from kfp.dsl import component\n"
            "from kfp.compiler import Compiler\n"
            "import kfp as kfp_v2\n"
            "import kfp.dsl as dsl_v2\n"
        )
        migrated, warnings = migration.migrate_code(code)
        self.assertEqual(migrated, expected)
        self.assertTrue(any("kfp.v2" in w for w in warnings))

    def test_migrate_components_imports(self):
        code = (
            "from kfp.components import InputPath, OutputPath, load_component_from_file\n"
        )
        expected = (
            "from kfp.dsl import InputPath, OutputPath\n"
            "from kfp.components import load_component_from_file\n"
        )
        migrated, warnings = migration.migrate_code(code)
        self.assertEqual(migrated.strip(), expected.strip())

    def test_migrate_factory_imports_and_usages(self):
        code = (
            "from kfp.components import func_to_container_op, create_component_from_func\n"
            "import kfp.components as comp\n"
            "my_op = func_to_container_op(my_func, base_image='python:3.9')\n"
            "other_op = comp.create_component_from_func(other_func)\n"
            "third_op = kfp.components.func_to_container_op(third_func)\n"
        )
        migrated, warnings = migration.migrate_code(code)
        self.assertIn("from kfp.dsl import component", migrated)
        self.assertIn("my_op = component(my_func, base_image='python:3.9')", migrated)
        self.assertIn("other_op = dsl.component(other_func)", migrated)
        self.assertIn("third_op = kfp.dsl.component(third_func)", migrated)

    def test_migrate_condition(self):
        code = (
            "from kfp import dsl\n"
            "from kfp.dsl import Condition, ParallelFor\n"
            "with Condition(param == 'value'):\n"
            "    pass\n"
            "with dsl.Condition(param == 'value'):\n"
            "    pass\n"
        )
        expected = (
            "from kfp import dsl\n"
            "from kfp.dsl import If, ParallelFor\n"
            "with If(param == 'value'):\n"
            "    pass\n"
            "with dsl.If(param == 'value'):\n"
            "    pass\n"
        )
        migrated, warnings = migration.migrate_code(code)
        self.assertEqual(migrated, expected)

    def test_migrate_parallel_for(self):
        code = (
            "from kfp import dsl\n"
            "with dsl.ParallelFor(loop_args=my_list) as item:\n"
            "    pass\n"
            "with ParallelFor(name='loop', loop_args=my_list) as item:\n"
            "    pass\n"
        )
        expected = (
            "from kfp import dsl\n"
            "with dsl.ParallelFor(items=my_list) as item:\n"
            "    pass\n"
            "with ParallelFor(name='loop', items=my_list) as item:\n"
            "    pass\n"
        )
        migrated, warnings = migration.migrate_code(code)
        self.assertEqual(migrated, expected)

    def test_warning_container_op(self):
        code = (
            "from kfp import dsl\n"
            "op = dsl.ContainerOp(\n"
            "    name='my-op',\n"
            "    image='gcr.io/my-image',\n"
            ")\n"
        )
        migrated, warnings = migration.migrate_code(code)
        self.assertEqual(migrated, code)  # ContainerOp not auto-migrated
        self.assertTrue(any("ContainerOp" in w for w in warnings))

    def test_migrate_file_inplace(self):
        with tempfile.TemporaryDirectory() as tempdir:
            source_file = os.path.join(tempdir, "pipeline.py")
            with open(source_file, "w", encoding="utf-8") as f:
                f.write("from kfp.v2 import dsl\n")
            
            migration.migrate(source_path=source_file, inplace=True)
            
            with open(source_file, "r", encoding="utf-8") as f:
                content = f.read()
            self.assertEqual(content, "from kfp import dsl\n")

    def test_migrate_file_output_path(self):
        with tempfile.TemporaryDirectory() as tempdir:
            source_file = os.path.join(tempdir, "pipeline.py")
            output_file = os.path.join(tempdir, "output_pipeline.py")
            with open(source_file, "w", encoding="utf-8") as f:
                f.write("from kfp.v2 import dsl\n")
            
            migration.migrate(source_path=source_file, output_path=output_file)
            
            with open(output_file, "r", encoding="utf-8") as f:
                content = f.read()
            self.assertEqual(content, "from kfp import dsl\n")

    def test_migrate_directory(self):
        with tempfile.TemporaryDirectory() as tempdir:
            src_dir = os.path.join(tempdir, "src")
            out_dir = os.path.join(tempdir, "out")
            os.makedirs(src_dir)
            
            file1 = os.path.join(src_dir, "p1.py")
            file2 = os.path.join(src_dir, "p2.py")
            with open(file1, "w", encoding="utf-8") as f:
                f.write("from kfp.v2 import dsl\n")
            with open(file2, "w", encoding="utf-8") as f:
                f.write("from kfp.v2 import compiler\n")
            
            migration.migrate(source_path=src_dir, output_path=out_dir)
            
            with open(os.path.join(out_dir, "p1.py"), "r", encoding="utf-8") as f:
                self.assertEqual(f.read(), "from kfp import dsl\n")
            with open(os.path.join(out_dir, "p2.py"), "r", encoding="utf-8") as f:
                self.assertEqual(f.read(), "from kfp import compiler\n")


if __name__ == "__main__":
    unittest.main()
