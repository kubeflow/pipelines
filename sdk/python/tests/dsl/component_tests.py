# Copyright 2018 The Kubeflow Authors
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

from kfp.deprecated.dsl import graph_component
from kfp.deprecated.dsl import Pipeline
from kfp.deprecated.dsl import PipelineParam
import kfp.deprecated.dsl as dsl


@unittest.skip("deprecated")
class TestGraphComponent(unittest.TestCase):

    def test_graphcomponent_basic(self):
        """Test graph_component decorator metadata."""

        @graph_component
        def flip_component(flip_result):
            with dsl.Condition(flip_result == 'heads'):
                flip_component(flip_result)

        with Pipeline('pipeline') as p:
            param = PipelineParam(name='param')
            flip_component(param)
            self.assertEqual(1, len(p.groups))
            self.assertEqual(1, len(p.groups[0].groups))  # pipeline
            self.assertEqual(1, len(
                p.groups[0].groups[0].groups))  # flip_component
            self.assertEqual(1, len(
                p.groups[0].groups[0].groups[0].groups))  # condition
            self.assertEqual(0,
                             len(p.groups[0].groups[0].groups[0].groups[0]
                                 .groups))  # recursive flip_component
            recursive_group = p.groups[0].groups[0].groups[0].groups[0]
            self.assertTrue(recursive_group.recursive_ref is not None)
            self.assertEqual(1, len(recursive_group.inputs))
            self.assertEqual('param', recursive_group.inputs[0].name)
