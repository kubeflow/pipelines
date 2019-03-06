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


from kfp.dsl._component import component
from kfp.dsl._metadata import ComponentMeta, ParameterMeta, TypeMeta
from kfp.dsl._types import GCSPath, Integer
import unittest
import mock

class TestPythonComponent(unittest.TestCase):

  def test_component(self):
    """Test component decorator."""

    class MockContainerOp:
      def _set_metadata(self, component_meta):
        self._metadata = component_meta

    @component
    def componentA(a: {'Schema': {'file_type': 'csv'}}, b: {'number': {'step': 'large'}} = 12, c: GCSPath(path_type='file', file_type='tsv') = 'gs://hello/world') -> {'model': Integer()}:
      return MockContainerOp()

    containerOp = componentA(1,2,3)

    golden_meta = ComponentMeta(name='componentA', description='')
    golden_meta.inputs.append(ParameterMeta(name='a', description='', param_type=TypeMeta(name='Schema', properties={'file_type': 'csv'})))
    golden_meta.inputs.append(ParameterMeta(name='b', description='', param_type=TypeMeta(name='number', properties={'step': 'large'}), default=12))
    golden_meta.inputs.append(ParameterMeta(name='c', description='', param_type=TypeMeta(name='GCSPath', properties={'path_type':'file', 'file_type': 'tsv'}), default='gs://hello/world'))
    golden_meta.outputs.append(ParameterMeta(name='model', description='', param_type=TypeMeta(name='Integer')))

    self.assertEqual(containerOp._metadata, golden_meta)
