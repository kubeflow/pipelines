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
# limitations under the License.

import sys
import unittest
from kfp.components import structures
from kfp.v2.dsl import type_utils
from kfp.v2.proto import pipeline_spec_pb2 as pb


class TypeUtilsTest(unittest.TestCase):

  _parameter_types = ['String', 'str', 'Integer', 'int', 'Float', 'Double']
  _known_artifact_types = ['Model', 'Dataset', 'Schema', 'Metrics']
  _unknown_artifact_types = [{
      'JsonObject': {
          'data_type': 'proto:tfx.components.trainer.Trai'
      }
  }, None, 'Arbtrary Model', 'dummy']

  def test_is_parameter_type(self):
    for type_name in self._parameter_types:
      self.assertTrue(type_utils.is_parameter_type(type_name))
    for type_name in self._known_artifact_types + self._unknown_artifact_types:
      self.assertFalse(type_utils.is_parameter_type(type_name))

  def test_get_artifact_type_schema(self):
    self.assertEqual('title: kfp.Model\ntype: object\nproperties:\n',
                     type_utils.get_artifact_type_schema('Model'))
    self.assertEqual('title: kfp.Dataset\ntype: object\nproperties:\n',
                     type_utils.get_artifact_type_schema('Dataset'))
    self.assertEqual('title: kfp.Metrics\ntype: object\nproperties:\n',
                     type_utils.get_artifact_type_schema('Metrics'))
    self.assertEqual('title: kfp.Schema\ntype: object\nproperties:\n',
                     type_utils.get_artifact_type_schema('Schema'))
    for type_name in self._unknown_artifact_types:
      self.assertEqual('title: kfp.Artifact\ntype: object\nproperties:\n',
                       type_utils.get_artifact_type_schema(type_name))

  def test_get_parameter_type(self):
    self.assertEqual(pb.PrimitiveType.INT, type_utils.get_parameter_type('Int'))
    self.assertEqual(pb.PrimitiveType.INT,
                     type_utils.get_parameter_type('Integer'))
    self.assertEqual(pb.PrimitiveType.DOUBLE,
                     type_utils.get_parameter_type('Double'))
    self.assertEqual(pb.PrimitiveType.DOUBLE,
                     type_utils.get_parameter_type('Float'))
    self.assertEqual(pb.PrimitiveType.STRING,
                     type_utils.get_parameter_type('String'))
    self.assertEqual(pb.PrimitiveType.STRING,
                     type_utils.get_parameter_type('Str'))
    with self.assertRaises(AttributeError):
      type_utils.get_parameter_type_schema(None)

  def test_get_input_artifact_type_schema(self):
    input_specs = [
        structures.InputSpec(name='input1', type='String'),
        structures.InputSpec(name='input2', type='GCSPath'),
    ]
    # input not found.
    self.assertEqual(
        None, type_utils.get_input_artifact_type_schema('input0', input_specs))
    # input found, but it doesn't map to an artifact type.
    self.assertEqual(
        None, type_utils.get_input_artifact_type_schema('input1', input_specs))
    # input found, and a matching artifact type schema returned.
    self.assertEqual(
        'title: kfp.Artifact\ntype: object\nproperties:\n',
        type_utils.get_input_artifact_type_schema('input2', input_specs))


if __name__ == '__main__':
  unittest.main()
