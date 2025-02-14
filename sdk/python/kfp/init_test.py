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


@mock.patch.object(sys, 'version_info', new=(3, 7, 12, 'final', 0))
class TestPythonEOLWarning(unittest.TestCase):

    def test(self):
        mod = importlib.import_module('kfp')

        with self.assertWarnsRegex(
                FutureWarning,
                r'KFP will drop support for Python 3.9 on Oct 1, 2025. To use new versions of the KFP SDK after that date, you will need to upgrade to Python >= 3.10. See https://devguide.python.org/versions/ for more details.'
        ):
            # simulate first import from kfp
            importlib.reload(mod)


if __name__ == '__main__':
    unittest.main()
