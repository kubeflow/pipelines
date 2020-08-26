# Copyright 2018-2019 Google LLC
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


import unittest
import sys

import aws_extensions_tests
import pipeline_tests
import pipeline_param_tests
import container_op_tests
import ops_group_tests
import type_tests
import component_tests
import component_bridge_tests
import metadata_tests
import resource_op_tests
import volume_op_tests
import pipeline_volume_tests
import volume_snapshotop_tests
import extensions.test_kubernetes as test_kubernetes

if __name__ == '__main__':
  suite = unittest.TestSuite()
  suite.addTests(unittest.defaultTestLoader.loadTestsFromModule(aws_extensions_tests))
  suite.addTests(unittest.defaultTestLoader.loadTestsFromModule(pipeline_param_tests))
  suite.addTests(unittest.defaultTestLoader.loadTestsFromModule(pipeline_tests))
  suite.addTests(unittest.defaultTestLoader.loadTestsFromModule(container_op_tests))
  suite.addTests(unittest.defaultTestLoader.loadTestsFromModule(ops_group_tests))
  suite.addTests(unittest.defaultTestLoader.loadTestsFromModule(type_tests))
  suite.addTests(unittest.defaultTestLoader.loadTestsFromModule(component_tests))
  suite.addTests(unittest.defaultTestLoader.loadTestsFromModule(component_bridge_tests))
  suite.addTests(unittest.defaultTestLoader.loadTestsFromModule(metadata_tests))
  suite.addTests(
    unittest.defaultTestLoader.loadTestsFromModule(resource_op_tests)
  )
  suite.addTests(
    unittest.defaultTestLoader.loadTestsFromModule(volume_op_tests)
  )
  suite.addTests(
    unittest.defaultTestLoader.loadTestsFromModule(pipeline_volume_tests)
  )
  suite.addTests(
    unittest.defaultTestLoader.loadTestsFromModule(volume_snapshotop_tests)
  )
  suite.addTests(unittest.defaultTestLoader.loadTestsFromModule(test_kubernetes))

  runner = unittest.TextTestRunner()
  if not runner.run(suite).wasSuccessful():
    sys.exit(1)
