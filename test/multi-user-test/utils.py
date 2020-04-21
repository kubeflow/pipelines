# Copyright 2020 Google LLC
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
# limitations under the License

from junit_xml import TestSuite, TestCase

# Junit xml utilities
def add_junit_test(test_cases, testname, succ, message='default message', elapsed_sec=0):
  test_case = TestCase(testname, elapsed_sec = elapsed_sec)
  if not succ:
    test_case.add_failure_info(message)
  test_cases.append(test_case)

def write_junit_xml(testname, filename, test_cases):
  with open(filename, 'w') as f:
    ts = TestSuite(testname, test_cases)
    TestSuite.to_file(f, [ts], prettyprint=False)
