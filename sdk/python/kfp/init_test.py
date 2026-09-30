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

from pathlib import Path
import runpy
import sys
import unittest
from unittest import mock


class TestPythonMinimum(unittest.TestCase):

    def test_unsupported_python_fails_before_dependency_imports(self):
        for version in ((3, 9, 25), (3, 10, 19)):
            for runtime in ('false', 'true'):
                with self.subTest(version=version, runtime=runtime):
                    with mock.patch.object(sys, 'version_info', version), \
                            mock.patch.dict('os.environ', {'_KFP_RUNTIME': runtime}):
                        with self.assertRaisesRegex(
                                RuntimeError,
                                'KFP requires Python 3.11 or later'):
                            runpy.run_path(
                                str(Path(__file__).with_name('__init__.py')),
                                init_globals={'__path__': []})

    def test_supported_python_runtime_import_needs_no_dependencies(self):
        with mock.patch.object(sys, 'version_info', (3, 11, 0)), \
                mock.patch.dict('os.environ', {'_KFP_RUNTIME': 'true'}):
            namespace = runpy.run_path(
                str(Path(__file__).with_name('__init__.py')),
                init_globals={'__path__': []})
        self.assertTrue(namespace['TYPE_CHECK'])


if __name__ == '__main__':
    unittest.main()
