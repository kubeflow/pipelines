# Copyright 2019 Google LLC
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


import kfp
from kfp.dsl import SuspendOp
import unittest


class TestSuspendOp(unittest.TestCase):

    def test_basic(self):
        """Test basic usage."""
        def my_pipeline(param):
            res = SuspendOp(name="suspend")

            self.assertEqual(res.name, "suspend")
            self.assertEqual(
                res.suspend.duration,
                ''
            )
            self.assertEqual(res.dependent_names, [])

        kfp.compiler.Compiler()._compile(my_pipeline)

    def test_duration_validation(self):
        """Test bad duration value."""
        def my_pipeline(param):
            res = SuspendOp(name="suspend", duration="bad input")

        with self.assertRaises(ValueError):
            kfp.compiler.Compiler()._compile(my_pipeline)
