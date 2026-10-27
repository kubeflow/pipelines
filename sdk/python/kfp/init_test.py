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

import importlib
import sys
import unittest
from unittest import mock
import warnings

import kfp


class TestPythonVersionWarning(unittest.TestCase):

    def test_unsupported_versions_warn(self):
        for version in ((3, 7, 12), (3, 8, 20)):
            with self.subTest(version=version):
                with mock.patch.object(sys, 'version_info', version):
                    with self.assertWarnsRegex(
                            FutureWarning,
                            r'KFP 2\.18 requires Python 3\.9 or newer\.'):
                        importlib.reload(kfp)

    def test_supported_versions_do_not_warn(self):
        for version in ((3, 9, 0), (3, 10, 0), (3, 13, 0)):
            with self.subTest(version=version):
                with mock.patch.object(sys, 'version_info', version):
                    with warnings.catch_warnings(record=True) as caught:
                        warnings.simplefilter('always')
                        importlib.reload(kfp)
                self.assertEqual(caught, [])


if __name__ == '__main__':
    unittest.main()
