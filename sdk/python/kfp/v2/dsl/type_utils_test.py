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
from kfp.pipeline_spec import pipeline_spec_pb2 as pb

_PARAMETER_TYPES = ['String', 'str', 'Integer', 'int', 'Float', 'Double']
_KNOWN_ARTIFACT_TYPES = ['Model', 'Dataset', 'Schema', 'Metrics']
_UNKNOWN_ARTIFACT_TYPES = [{
    'JsonObject': {
        'data_type': 'proto:tfx.components.trainer.Trai'
    }
}, None, 'Arbtrary Model', 'dummy']


class TypeUtilsTest(unittest.TestCase):

  def test_is_parameter_type(self):
    for type_name in _PARAMETER_TYPES:
      self.assertTrue(type_utils.is_parameter_type(type_name))
    for type_name in _KNOWN_ARTIFACT_TYPES + _UNKNOWN_ARTIFACT_TYPES:
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
    for type_name in _UNKNOWN_ARTIFACT_TYPES:
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
        structures.InputSpec(name='input2', type='Model'),
        structures.InputSpec(name='input3', type=None),
    ]
    # input not found.
    with self.assertRaises(AssertionError) as cm:
      type_utils.get_input_artifact_type_schema('input0', input_specs)
      self.assertEqual('Input not found.', str(cm))

    # input found, but it doesn't map to an artifact type.
    with self.assertRaises(AssertionError) as cm:
      type_utils.get_input_artifact_type_schema('input1', input_specs)
      self.assertEqual('Input is not an artifact type.', str(cm))

    # input found, and a matching artifact type schema returned.
    self.assertEqual(
        'title: kfp.Model\ntype: object\nproperties:\n',
        type_utils.get_input_artifact_type_schema('input2', input_specs))

    # input found, and the default artifact type schema returned.
    self.assertEqual(
        'title: kfp.Artifact\ntype: object\nproperties:\n',
        type_utils.get_input_artifact_type_schema('input3', input_specs))


if __name__ == '__main__':
  unittest.main()
