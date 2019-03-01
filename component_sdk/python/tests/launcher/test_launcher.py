# Copyright 2018 Google LLC
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

import mock
import unittest
import math
import os

from kfp_component.launcher import launch

TEST_PY_FILE = os.path.join(os.path.dirname(__file__), 'echo.py')

class LauncherTest(unittest.TestCase):

    def test_launch_module_succeed(self):
        self.assertEqual(math.pi, launch('math', 'pi'))

    def test_launch_py_file_succeed(self):
        self.assertEqual('hello', 
            launch(TEST_PY_FILE, ['echo', 'hello']))