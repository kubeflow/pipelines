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


import sys
import unittest

import compiler_tests
import component_builder_test
import container_builder_test
import k8s_helper_tests


if __name__ == '__main__':
  suite = unittest.TestSuite()
  suite.addTests(unittest.defaultTestLoader.loadTestsFromModule(compiler_tests))
  suite.addTests(unittest.defaultTestLoader.loadTestsFromModule(component_builder_test))
  suite.addTests(unittest.defaultTestLoader.loadTestsFromModule(container_builder_test))
  suite.addTests(unittest.defaultTestLoader.loadTestsFromModule(k8s_helper_tests))
  runner = unittest.TextTestRunner()
  if not runner.run(suite).wasSuccessful():
    sys.exit(1)
