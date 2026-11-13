# Copyright 2023 The Kubeflow Authors
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

import runpy
import sys
import unittest
from pathlib import Path
from unittest import mock


class TestPythonMinimum(unittest.TestCase):
    def test_unsupported_python_fails_before_dependency_imports(self):
        for version in ((3, 9, 25), (3, 10, 19)):
            for runtime in ('false', 'true'):
                with self.subTest(version=version, runtime=runtime):
                    with (
                        mock.patch.object(sys, 'version_info', version),
                        mock.patch.dict(
                            'os.environ', {'_KFP_RUNTIME': runtime}
                        ),
                    ):
                        with self.assertRaisesRegex(
                            RuntimeError, 'KFP requires Python 3.11 or later'
                        ):
                            runpy.run_path(
                                str(Path(__file__).with_name('__init__.py')),
                                init_globals={'__path__': []},
                            )

    def test_supported_python_runtime_import_needs_no_dependencies(self):
        with (
            mock.patch.object(sys, 'version_info', (3, 11, 0)),
            mock.patch.dict('os.environ', {'_KFP_RUNTIME': 'true'}),
        ):
            namespace = runpy.run_path(
                str(Path(__file__).with_name('__init__.py')),
                init_globals={'__path__': []},
            )
        self.assertTrue(namespace['TYPE_CHECK'])

    def test_runtime_import_needs_no_dependencies_without_kfp_runtime_env(self):
        with (
            mock.patch.object(sys, 'version_info', (3, 11, 0)),
            mock.patch.dict('os.environ', {}, clear=True),
        ):
            namespace = runpy.run_path(
                str(Path(__file__).with_name('__init__.py')),
                init_globals={'__path__': []},
            )
        self.assertTrue(namespace['TYPE_CHECK'])

    def test_lazy_modules_present_in_dir(self):
        import kfp

        for mod in ['Client', 'dsl', 'compiler', 'components']:
            self.assertIn(mod, dir(kfp))

    def test_dsl_runtime_import_needs_no_compiler_dependencies(self):
        from kfp.dsl import Artifact, Dataset, Input, Output
        from kfp.dsl.executor import Executor

        self.assertIsNotNone(Artifact)
        self.assertIsNotNone(Executor)

    def test_dsl_dir_contains_lazy_compile_time_symbols(self):
        from kfp import dsl

        for symbol in ['component', 'pipeline', 'Condition', 'ParallelFor']:
            self.assertIn(symbol, dir(dsl))


if __name__ == '__main__':
    unittest.main()
